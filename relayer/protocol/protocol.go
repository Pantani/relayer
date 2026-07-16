package protocol

import "fmt"

// Protocol identifies an IBC protocol generation.
type Protocol string

const (
	ProtocolUnspecified Protocol = ""
	ProtocolClassic     Protocol = "classic"
	ProtocolV2          Protocol = "v2"
)

// ParseProtocol parses an explicit protocol value without normalizing it.
func ParseProtocol(value string) (Protocol, error) {
	protocol := Protocol(value)
	if err := protocol.Validate(); err != nil {
		return ProtocolUnspecified, err
	}
	return protocol, nil
}

// Validate reports whether the protocol is supported.
func (p Protocol) Validate() error {
	switch p {
	case ProtocolClassic, ProtocolV2:
		return nil
	default:
		return fmt.Errorf("protocol %q is not supported", p)
	}
}

// Capabilities describes protocol-level behavior available to the relayer.
type Capabilities struct {
	ClientRouting             bool
	ConnectionHandshake       bool
	ChannelHandshake          bool
	OrderedDelivery           bool
	TimeoutHeight             bool
	TimeoutTimestamp          bool
	AsyncAcknowledgement      bool
	PerClientRelayerAllowlist bool
	MaxPayloads               int
}

// CapabilitiesFor returns the capabilities fixed for a protocol.
func CapabilitiesFor(p Protocol) (Capabilities, error) {
	if err := p.Validate(); err != nil {
		return Capabilities{}, err
	}
	if p == ProtocolClassic {
		return classicCapabilities(), nil
	}
	return v2Capabilities(), nil
}

func classicCapabilities() Capabilities {
	return Capabilities{
		ConnectionHandshake:  true,
		ChannelHandshake:     true,
		OrderedDelivery:      true,
		TimeoutHeight:        true,
		TimeoutTimestamp:     true,
		AsyncAcknowledgement: true,
		MaxPayloads:          1,
	}
}

func v2Capabilities() Capabilities {
	return Capabilities{
		ClientRouting:             true,
		TimeoutTimestamp:          true,
		AsyncAcknowledgement:      true,
		PerClientRelayerAllowlist: true,
		MaxPayloads:               1,
	}
}

// Height is a dependency-neutral IBC height.
type Height struct {
	RevisionNumber uint64
	RevisionHeight uint64
}

// IsZero reports whether both height components are zero.
func (h Height) IsZero() bool {
	return h.RevisionNumber == 0 && h.RevisionHeight == 0
}

// Validate rejects heights without a revision height.
func (h Height) Validate() error {
	if h.RevisionHeight == 0 {
		return fmt.Errorf("revision height must be greater than zero")
	}
	return nil
}

// TimestampUnit makes wire-protocol timestamp units explicit.
type TimestampUnit string

const (
	TimestampUnitUnspecified TimestampUnit = ""
	TimestampNanoseconds     TimestampUnit = "nanoseconds"
	TimestampSeconds         TimestampUnit = "seconds"
)

// Timeout is a dependency-neutral packet timeout.
type Timeout struct {
	Height        Height
	Timestamp     uint64
	TimestampUnit TimestampUnit
}
