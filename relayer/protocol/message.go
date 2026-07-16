package protocol

import "fmt"

// MessageKind identifies an IBC packet message.
type MessageKind string

const (
	MessageSendPacket      MessageKind = "send_packet"
	MessageRecvPacket      MessageKind = "recv_packet"
	MessageAcknowledgement MessageKind = "acknowledgement"
	MessageTimeout         MessageKind = "timeout"
)

// Acknowledgement contains the acknowledgement form selected by Protocol.
type Acknowledgement struct {
	Protocol            Protocol
	Value               []byte
	AppAcknowledgements [][]byte
}

// SendPacketRequest represents the pre-sequence v2 MsgSendPacket shape. The
// destination client and packet sequence are assigned by the on-chain keeper.
type SendPacketRequest struct {
	Protocol     Protocol
	SourceClient string
	Timeout      Timeout
	Payloads     []Payload
}

// Validate checks the contract shared with ibc-go v11.2.0 MsgSendPacket.
func (r SendPacketRequest) Validate() error {
	if r.Protocol != ProtocolV2 {
		return fmt.Errorf("send packet request requires protocol v2")
	}
	if r.SourceClient == "" {
		return fmt.Errorf("send packet request requires a source client")
	}
	if !r.Timeout.Height.IsZero() {
		return fmt.Errorf("send packet request cannot use timeout height")
	}
	if r.Timeout.Timestamp == 0 || r.Timeout.TimestampUnit != TimestampSeconds {
		return fmt.Errorf("send packet timeout must be non-zero seconds")
	}
	return validateSendPayloads(r.Payloads)
}

// Clone returns an independent send packet request.
func (r SendPacketRequest) Clone() SendPacketRequest {
	r.Payloads = clonePayloads(r.Payloads)
	return r
}

func validateSendPayloads(payloads []Payload) error {
	if len(payloads) != 1 {
		return fmt.Errorf("send packet request must contain exactly one payload")
	}
	if payloadsSize(payloads) > MaximumPayloadsSize {
		return fmt.Errorf("send packet payloads cannot exceed %d bytes", MaximumPayloadsSize)
	}
	return validateV2Payload(payloads[0])
}

// Validate rejects mixed or empty acknowledgement bodies.
func (a Acknowledgement) Validate() error {
	if err := a.Protocol.Validate(); err != nil {
		return err
	}
	if a.Protocol == ProtocolV2 {
		return validateV2Acknowledgement(a)
	}
	return validateClassicAcknowledgement(a)
}

// Clone returns an independent acknowledgement.
func (a Acknowledgement) Clone() Acknowledgement {
	clone := a
	clone.Value = cloneBytes(a.Value)
	clone.AppAcknowledgements = cloneByteSlices(a.AppAcknowledgements)
	return clone
}

func validateClassicAcknowledgement(ack Acknowledgement) error {
	if len(ack.Value) == 0 || len(ack.AppAcknowledgements) != 0 {
		return fmt.Errorf("classic acknowledgement requires one opaque value")
	}
	return nil
}

func validateV2Acknowledgement(ack Acknowledgement) error {
	if len(ack.Value) != 0 || len(ack.AppAcknowledgements) != 1 {
		return fmt.Errorf("v2 acknowledgement requires exactly one app acknowledgement")
	}
	if len(ack.AppAcknowledgements[0]) == 0 {
		return fmt.Errorf("v2 app acknowledgement cannot be empty")
	}
	return nil
}

// MessageEnvelope describes a packet message without an SDK dependency.
type MessageEnvelope struct {
	Protocol        Protocol
	Kind            MessageKind
	Send            *SendPacketRequest
	Packet          *PacketEnvelope
	Proof           *ProofEnvelope
	Acknowledgement *Acknowledgement
	Signer          string
}

// Validate enforces the packet message matrix.
func (m MessageEnvelope) Validate() error {
	if err := validateMessageCommon(m); err != nil {
		return err
	}
	switch m.Kind {
	case MessageSendPacket:
		return validateSendMessage(m)
	case MessageRecvPacket:
		return validateRecvMessage(m)
	case MessageAcknowledgement:
		return validateAcknowledgementMessage(m)
	case MessageTimeout:
		return validateTimeoutMessage(m)
	default:
		return fmt.Errorf("message kind %q is not supported", m.Kind)
	}
}

func validateMessageCommon(message MessageEnvelope) error {
	if err := message.Protocol.Validate(); err != nil {
		return err
	}
	if message.Signer == "" {
		return fmt.Errorf("message signer cannot be empty")
	}
	return nil
}

func validateSendMessage(message MessageEnvelope) error {
	if message.Packet != nil || message.Proof != nil || message.Acknowledgement != nil {
		return fmt.Errorf("send message cannot include packet, proof, or acknowledgement")
	}
	if message.Send == nil || message.Send.Protocol != message.Protocol {
		return fmt.Errorf("send message requires a matching send packet request")
	}
	return message.Send.Validate()
}

func validateRecvMessage(message MessageEnvelope) error {
	if message.Send != nil || message.Acknowledgement != nil {
		return fmt.Errorf("recv message cannot include send request or acknowledgement")
	}
	if err := validateRelayPacket(message); err != nil {
		return err
	}
	return validateMessageProof(message, ProofPacketCommitment)
}

func validateAcknowledgementMessage(message MessageEnvelope) error {
	if message.Send != nil {
		return fmt.Errorf("acknowledgement message cannot include a send request")
	}
	if err := validateRelayPacket(message); err != nil {
		return err
	}
	if message.Acknowledgement == nil {
		return fmt.Errorf("acknowledgement message requires an acknowledgement")
	}
	if message.Acknowledgement.Protocol != message.Protocol {
		return fmt.Errorf("message and acknowledgement protocols do not match")
	}
	if err := message.Acknowledgement.Validate(); err != nil {
		return err
	}
	return validateMessageProof(message, ProofAcknowledgement)
}

func validateTimeoutMessage(message MessageEnvelope) error {
	if message.Send != nil || message.Acknowledgement != nil {
		return fmt.Errorf("timeout message cannot include send request or acknowledgement")
	}
	if err := validateRelayPacket(message); err != nil {
		return err
	}
	if message.Proof == nil {
		return fmt.Errorf("timeout message requires a proof")
	}
	if message.Protocol == ProtocolClassic && message.Proof.Kind == ProofNextSequenceRecv {
		return validateMessageProof(message, ProofNextSequenceRecv)
	}
	return validateMessageProof(message, ProofReceiptAbsence)
}

func validateRelayPacket(message MessageEnvelope) error {
	if message.Packet == nil {
		return fmt.Errorf("relay message requires a packet")
	}
	if message.Packet.ID.Protocol != message.Protocol {
		return fmt.Errorf("message and packet protocols do not match")
	}
	return message.Packet.Validate()
}

func validateMessageProof(message MessageEnvelope, kind ProofKind) error {
	if message.Proof == nil {
		return fmt.Errorf("message requires a %s proof", kind)
	}
	if message.Proof.Protocol != message.Protocol || message.Proof.Kind != kind {
		return fmt.Errorf("message requires a matching %s proof", kind)
	}
	return message.Proof.Validate()
}

func cloneByteSlices(values [][]byte) [][]byte {
	clones := make([][]byte, len(values))
	for i, value := range values {
		clones[i] = cloneBytes(value)
	}
	return clones
}
