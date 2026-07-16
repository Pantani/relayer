package v2

import (
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/relayer/v2/relayer/protocol"
)

const (
	// MaximumEncodedPacketSize bounds allocation before protobuf decoding. The
	// semantic payload limit remains protocol.MaximumPayloadsSize.
	MaximumEncodedPacketSize = 512 * 1024

	// MaximumEncodedAcknowledgementSize bounds untrusted event attributes. IBC
	// v2 does not define an equivalent total app-ack size in Validate.
	MaximumEncodedAcknowledgementSize = 512 * 1024

	minimumChannelIdentifierLength = 8
	maximumIdentifierLength        = 64
	minimumPortIdentifierLength    = 2
	maximumPortIdentifierLength    = 128
)

var (
	// ErrEncodedValueTooLarge reports an event value above its operational cap.
	ErrEncodedValueTooLarge = errors.New("encoded value is too large")
	// ErrInvalidHex reports malformed hexadecimal event data.
	ErrInvalidHex = errors.New("invalid hex")
	// ErrInvalidProtobuf reports a protobuf wire decoding failure.
	ErrInvalidProtobuf = errors.New("invalid protobuf")
	// ErrInvalidPacketContract reports a decoded packet that violates v11.2.0 invariants.
	ErrInvalidPacketContract = errors.New("invalid IBC v2 packet contract")
	// ErrInvalidAcknowledgement reports a decoded acknowledgement that violates v11.2.0 invariants.
	ErrInvalidAcknowledgement = errors.New("invalid IBC v2 acknowledgement")
)

var validIdentifier = regexp.MustCompile(`^[a-zA-Z0-9._+\-#\[\]<>]+$`)

// DecodePacketHex decodes and validates an encoded_packet_hex event value.
func DecodePacketHex(value string) (protocol.PacketEnvelope, error) {
	encoded, err := decodeBoundedHex(value, MaximumEncodedPacketSize)
	if err != nil {
		return protocol.PacketEnvelope{}, err
	}

	packet := wirePacket{}
	if err := proto.Unmarshal(encoded, &packet); err != nil {
		return protocol.PacketEnvelope{}, fmt.Errorf("%w: packet decode failed", ErrInvalidProtobuf)
	}
	if err := validateWirePacket(packet); err != nil {
		return protocol.PacketEnvelope{}, fmt.Errorf("%w: %v", ErrInvalidPacketContract, err)
	}

	envelope, err := FromContractPacket(contractPacketFromWire(packet))
	if err != nil {
		return protocol.PacketEnvelope{}, fmt.Errorf("%w: %v", ErrInvalidPacketContract, err)
	}
	return envelope, nil
}

// DecodeAcknowledgementHex decodes and validates an
// encoded_acknowledgement_hex event value.
func DecodeAcknowledgementHex(value string) (protocol.Acknowledgement, error) {
	encoded, err := decodeBoundedHex(value, MaximumEncodedAcknowledgementSize)
	if err != nil {
		return protocol.Acknowledgement{}, err
	}

	wireAck := wireAcknowledgement{}
	if err := proto.Unmarshal(encoded, &wireAck); err != nil {
		return protocol.Acknowledgement{}, fmt.Errorf("%w: acknowledgement decode failed", ErrInvalidProtobuf)
	}
	ack := acknowledgementFromWire(wireAck)
	if err := ack.Validate(); err != nil {
		return protocol.Acknowledgement{}, fmt.Errorf("%w: %v", ErrInvalidAcknowledgement, err)
	}
	return ack, nil
}

func decodeBoundedHex(value string, maximumSize int) ([]byte, error) {
	if len(value) > maximumSize*2 {
		return nil, fmt.Errorf("%w: encoded length exceeds %d bytes", ErrEncodedValueTooLarge, maximumSize)
	}
	if len(value)%2 != 0 {
		return nil, fmt.Errorf("%w: odd encoded length", ErrInvalidHex)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("%w: decode failed", ErrInvalidHex)
	}
	return decoded, nil
}

func validateWirePacket(packet wirePacket) error {
	if len(packet.Payloads) != 1 {
		return fmt.Errorf("payload length must be exactly one")
	}
	if packet.Sequence == 0 {
		return fmt.Errorf("packet sequence cannot be zero")
	}
	if packet.TimeoutTimestamp == 0 {
		return fmt.Errorf("packet timeout timestamp cannot be zero")
	}
	if err := validateWirePacketIdentifiers(packet); err != nil {
		return err
	}
	return validateWirePayload(packet.Payloads[0])
}

func validateWirePacketIdentifiers(packet wirePacket) error {
	if err := validateIdentifier(packet.SourceClient, minimumChannelIdentifierLength, maximumIdentifierLength); err != nil {
		return fmt.Errorf("invalid source client: %w", err)
	}
	if err := validateIdentifier(packet.DestinationClient, minimumChannelIdentifierLength, maximumIdentifierLength); err != nil {
		return fmt.Errorf("invalid destination client: %w", err)
	}
	return nil
}

func validateWirePayload(payload *wirePayload) error {
	if payload == nil {
		return fmt.Errorf("payload cannot be nil")
	}
	if err := validateWirePayloadIdentifiers(*payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.Version) == "" {
		return fmt.Errorf("payload version cannot be empty")
	}
	if strings.TrimSpace(payload.Encoding) == "" {
		return fmt.Errorf("payload encoding cannot be empty")
	}
	if len(payload.Value) == 0 {
		return fmt.Errorf("payload value cannot be empty")
	}
	if len(payload.Value) > protocol.MaximumPayloadsSize {
		return fmt.Errorf("payload value exceeds %d bytes", protocol.MaximumPayloadsSize)
	}
	return nil
}

func validateWirePayloadIdentifiers(payload wirePayload) error {
	if err := validateIdentifier(payload.SourcePort, minimumPortIdentifierLength, maximumPortIdentifierLength); err != nil {
		return fmt.Errorf("invalid source port: %w", err)
	}
	if err := validateIdentifier(payload.DestinationPort, minimumPortIdentifierLength, maximumPortIdentifierLength); err != nil {
		return fmt.Errorf("invalid destination port: %w", err)
	}
	return nil
}

func validateIdentifier(identifier string, minimumLength, maximumLength int) error {
	if strings.TrimSpace(identifier) == "" {
		return fmt.Errorf("identifier cannot be blank")
	}
	if len(identifier) < minimumLength || len(identifier) > maximumLength {
		return fmt.Errorf("identifier length must be between %d and %d", minimumLength, maximumLength)
	}
	if !validIdentifier.MatchString(identifier) {
		return fmt.Errorf("identifier contains unsupported characters")
	}
	return nil
}

func contractPacketFromWire(packet wirePacket) ContractPacket {
	payloads := make([]ContractPayload, len(packet.Payloads))
	for index, payload := range packet.Payloads {
		if payload != nil {
			payloads[index] = contractPayloadFromWire(*payload)
		}
	}
	return ContractPacket{
		Sequence:          packet.Sequence,
		SourceClient:      packet.SourceClient,
		DestinationClient: packet.DestinationClient,
		TimeoutTimestamp:  packet.TimeoutTimestamp,
		Payloads:          payloads,
	}
}

func contractPayloadFromWire(payload wirePayload) ContractPayload {
	return ContractPayload{
		SourcePort:      payload.SourcePort,
		DestinationPort: payload.DestinationPort,
		Version:         payload.Version,
		Encoding:        payload.Encoding,
		Value:           append([]byte(nil), payload.Value...),
	}
}

func acknowledgementFromWire(wireAck wireAcknowledgement) protocol.Acknowledgement {
	appAcknowledgements := make([][]byte, len(wireAck.AppAcknowledgements))
	for index, value := range wireAck.AppAcknowledgements {
		appAcknowledgements[index] = append([]byte(nil), value...)
	}
	return protocol.Acknowledgement{
		Protocol:            protocol.ProtocolV2,
		AppAcknowledgements: appAcknowledgements,
	}
}
