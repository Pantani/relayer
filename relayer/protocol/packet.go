package protocol

import (
	"fmt"
	"strings"
)

// MaximumPayloadsSize matches ibc-go v11.2.0's 256 KiB packet limit.
const MaximumPayloadsSize = 256 * 1024

// Endpoint holds the identifiers used by one side of a packet key.
type Endpoint struct {
	ClientID  string
	PortID    string
	ChannelID string
}

// PacketID is a comparable protocol-qualified packet key.
type PacketID struct {
	Protocol    Protocol
	Source      Endpoint
	Destination Endpoint
	Sequence    uint64
}

// Validate checks the packet key's protocol-specific shape.
func (id PacketID) Validate() error {
	if err := id.Protocol.Validate(); err != nil {
		return err
	}
	if id.Sequence == 0 {
		return fmt.Errorf("packet sequence must be greater than zero")
	}
	if id.Protocol == ProtocolClassic {
		return validateClassicPacketID(id)
	}
	return validateV2PacketID(id)
}

func validateClassicPacketID(id PacketID) error {
	if id.Source.ClientID != "" || id.Destination.ClientID != "" {
		return fmt.Errorf("classic packet ID cannot contain client IDs")
	}
	if id.Source.PortID == "" || id.Source.ChannelID == "" {
		return fmt.Errorf("classic packet ID requires source port and channel")
	}
	if id.Destination.PortID == "" || id.Destination.ChannelID == "" {
		return fmt.Errorf("classic packet ID requires destination port and channel")
	}
	return nil
}

func validateV2PacketID(id PacketID) error {
	if id.Source.ClientID == "" || id.Destination.ClientID == "" {
		return fmt.Errorf("v2 packet ID requires source and destination clients")
	}
	if id.Source.PortID != "" || id.Source.ChannelID != "" {
		return fmt.Errorf("v2 packet ID cannot contain source port or channel")
	}
	if id.Destination.PortID != "" || id.Destination.ChannelID != "" {
		return fmt.Errorf("v2 packet ID cannot contain destination port or channel")
	}
	return nil
}

// Counterparty returns the packet ID from the opposite perspective.
func (id PacketID) Counterparty() PacketID {
	id.Source, id.Destination = id.Destination, id.Source
	return id
}

// Payload is an application payload carried by an IBC packet.
type Payload struct {
	SourcePort      string
	DestinationPort string
	Version         string
	Encoding        string
	Value           []byte
}

// NewPayload creates a payload with an owned copy of value.
func NewPayload(sourcePort, destinationPort, version, encoding string, value []byte) Payload {
	return Payload{
		SourcePort:      sourcePort,
		DestinationPort: destinationPort,
		Version:         version,
		Encoding:        encoding,
		Value:           cloneBytes(value),
	}
}

// Clone returns a deep copy of the payload.
func (p Payload) Clone() Payload {
	p.Value = cloneBytes(p.Value)
	return p
}

// PacketEnvelope is the protocol-neutral packet representation.
type PacketEnvelope struct {
	ID       PacketID
	Timeout  Timeout
	Payloads []Payload
}

// NewPacketEnvelope creates a packet with owned payload storage.
func NewPacketEnvelope(id PacketID, timeout Timeout, payloads []Payload) PacketEnvelope {
	return PacketEnvelope{ID: id, Timeout: timeout, Payloads: clonePayloads(payloads)}
}

// Clone returns a deep copy of the packet.
func (p PacketEnvelope) Clone() PacketEnvelope {
	p.Payloads = clonePayloads(p.Payloads)
	return p
}

// Validate checks the packet's common and protocol-specific invariants.
func (p PacketEnvelope) Validate() error {
	if err := p.ID.Validate(); err != nil {
		return err
	}
	if len(p.Payloads) != 1 {
		return fmt.Errorf("packet must contain exactly one payload")
	}
	if payloadsSize(p.Payloads) > MaximumPayloadsSize {
		return fmt.Errorf("packet payloads cannot exceed %d bytes", MaximumPayloadsSize)
	}
	if p.ID.Protocol == ProtocolClassic {
		return validateClassicPacket(p)
	}
	return validateV2Packet(p)
}

func validateClassicPacket(p PacketEnvelope) error {
	if p.Timeout.Height.IsZero() && p.Timeout.Timestamp == 0 {
		return fmt.Errorf("classic packet requires a height or timestamp timeout")
	}
	if !p.Timeout.Height.IsZero() {
		if err := p.Timeout.Height.Validate(); err != nil {
			return err
		}
	}
	if err := validateTimestamp(p.Timeout, TimestampNanoseconds); err != nil {
		return err
	}
	payload := p.Payloads[0]
	if payload.SourcePort != p.ID.Source.PortID || payload.DestinationPort != p.ID.Destination.PortID {
		return fmt.Errorf("classic payload ports must match packet ID")
	}
	return nil
}

func validateV2Packet(p PacketEnvelope) error {
	if !p.Timeout.Height.IsZero() {
		return fmt.Errorf("v2 packet cannot contain a timeout height")
	}
	if p.Timeout.Timestamp == 0 {
		return fmt.Errorf("v2 packet timeout timestamp must be greater than zero")
	}
	if err := validateTimestamp(p.Timeout, TimestampSeconds); err != nil {
		return err
	}
	return validateV2Payload(p.Payloads[0])
}

func validateTimestamp(timeout Timeout, expected TimestampUnit) error {
	if timeout.Timestamp == 0 && timeout.TimestampUnit != TimestampUnitUnspecified {
		return fmt.Errorf("zero timeout timestamp cannot have a unit")
	}
	if timeout.Timestamp > 0 && timeout.TimestampUnit != expected {
		return fmt.Errorf("timeout timestamp must use %s", expected)
	}
	return nil
}

func validateV2Payload(payload Payload) error {
	if payload.SourcePort == "" || payload.DestinationPort == "" {
		return fmt.Errorf("v2 payload requires source and destination ports")
	}
	if strings.TrimSpace(payload.Version) == "" {
		return fmt.Errorf("v2 payload version cannot be empty")
	}
	if strings.TrimSpace(payload.Encoding) == "" {
		return fmt.Errorf("v2 payload encoding cannot be empty")
	}
	if len(payload.Value) == 0 {
		return fmt.Errorf("v2 payload value cannot be empty")
	}
	return nil
}

func payloadsSize(payloads []Payload) int {
	total := 0
	for _, payload := range payloads {
		total += len(payload.Value)
	}
	return total
}

func clonePayloads(payloads []Payload) []Payload {
	if payloads == nil {
		return nil
	}
	cloned := make([]Payload, len(payloads))
	for i, payload := range payloads {
		cloned[i] = payload.Clone()
	}
	return cloned
}

func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	return append([]byte(nil), value...)
}

// PacketObservation keeps event metadata separate from the wire packet.
type PacketObservation struct {
	Protocol        Protocol
	EventType       string
	Height          uint64
	Packet          PacketEnvelope
	ChannelOrder    string
	Acknowledgement []byte
}

// NewPacketObservation creates an observation with owned byte storage.
func NewPacketObservation(protocol Protocol, eventType string, height uint64, packet PacketEnvelope, channelOrder string, acknowledgement []byte) PacketObservation {
	return PacketObservation{
		Protocol:        protocol,
		EventType:       eventType,
		Height:          height,
		Packet:          packet.Clone(),
		ChannelOrder:    channelOrder,
		Acknowledgement: cloneBytes(acknowledgement),
	}
}

// Clone returns a deep copy of the observation.
func (o PacketObservation) Clone() PacketObservation {
	o.Packet = o.Packet.Clone()
	o.Acknowledgement = cloneBytes(o.Acknowledgement)
	return o
}

// Validate checks observation metadata without imposing event-specific ack rules.
func (o PacketObservation) Validate() error {
	if err := o.Protocol.Validate(); err != nil {
		return err
	}
	if o.Protocol != o.Packet.ID.Protocol {
		return fmt.Errorf("observation protocol does not match packet protocol")
	}
	if o.EventType == "" {
		return fmt.Errorf("packet observation event type cannot be empty")
	}
	if o.Protocol == ProtocolV2 && o.ChannelOrder != "" {
		return fmt.Errorf("v2 packet observation cannot contain channel order")
	}
	return o.Packet.Validate()
}
