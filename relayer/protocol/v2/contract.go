// Package v2 mirrors the ibc-go v11.2.0 packet contract without importing
// ibc-go/v11 or its Cosmos SDK and CometBFT dependency graph.
package v2

import (
	"fmt"

	"github.com/cosmos/relayer/v2/relayer/protocol"
)

const (
	EventTypeSendPacket        = "send_packet"
	EventTypeRecvPacket        = "recv_packet"
	EventTypeTimeoutPacket     = "timeout_packet"
	EventTypeAcknowledgePacket = "acknowledge_packet"
	EventTypeWriteAck          = "write_acknowledgement"

	AttributeKeySrcClient        = "packet_source_client"
	AttributeKeyDstClient        = "packet_dest_client"
	AttributeKeySequence         = "packet_sequence"
	AttributeKeyTimeoutTimestamp = "packet_timeout_timestamp"
	AttributeKeyEncodedPacketHex = "encoded_packet_hex"
	AttributeKeyEncodedAckHex    = "encoded_acknowledgement_hex"
)

// ContractPayload mirrors channel/v2 types.Payload fields at v11.2.0.
type ContractPayload struct {
	SourcePort      string
	DestinationPort string
	Version         string
	Encoding        string
	Value           []byte
}

// ContractPacket mirrors channel/v2 types.Packet fields at v11.2.0.
type ContractPacket struct {
	Sequence          uint64
	SourceClient      string
	DestinationClient string
	TimeoutTimestamp  uint64
	Payloads          []ContractPayload
}

// ContractMsgSendPacket mirrors channel/v2 types.MsgSendPacket at v11.2.0.
type ContractMsgSendPacket struct {
	SourceClient     string
	TimeoutTimestamp uint64
	Payloads         []ContractPayload
	Signer           string
}

// FromContractMsgSendPacket preserves the pre-sequence send request shape.
func FromContractMsgSendPacket(message ContractMsgSendPacket) (protocol.MessageEnvelope, error) {
	request := protocol.SendPacketRequest{
		Protocol:     protocol.ProtocolV2,
		SourceClient: message.SourceClient,
		Timeout: protocol.Timeout{
			Timestamp:     message.TimeoutTimestamp,
			TimestampUnit: protocol.TimestampSeconds,
		},
		Payloads: fromContractPayloads(message.Payloads),
	}
	envelope := protocol.MessageEnvelope{
		Protocol: protocol.ProtocolV2,
		Kind:     protocol.MessageSendPacket,
		Send:     &request,
		Signer:   message.Signer,
	}
	if err := envelope.Validate(); err != nil {
		return protocol.MessageEnvelope{}, err
	}
	return envelope, nil
}

// FromContractPacket converts the local v11.2.0 shape to the neutral model.
func FromContractPacket(packet ContractPacket) (protocol.PacketEnvelope, error) {
	envelope := protocol.PacketEnvelope{
		ID: protocol.PacketID{
			Protocol:    protocol.ProtocolV2,
			Source:      protocol.Endpoint{ClientID: packet.SourceClient},
			Destination: protocol.Endpoint{ClientID: packet.DestinationClient},
			Sequence:    packet.Sequence,
		},
		Timeout: protocol.Timeout{
			Timestamp:     packet.TimeoutTimestamp,
			TimestampUnit: protocol.TimestampSeconds,
		},
		Payloads: fromContractPayloads(packet.Payloads),
	}
	if err := envelope.Validate(); err != nil {
		return protocol.PacketEnvelope{}, err
	}
	return envelope, nil
}

// ToContractPacket converts only a valid v2 packet envelope.
func ToContractPacket(envelope protocol.PacketEnvelope) (ContractPacket, error) {
	if envelope.ID.Protocol != protocol.ProtocolV2 {
		return ContractPacket{}, fmt.Errorf("v2 adapter cannot convert protocol %q", envelope.ID.Protocol)
	}
	if err := envelope.Validate(); err != nil {
		return ContractPacket{}, err
	}
	return ContractPacket{
		Sequence:          envelope.ID.Sequence,
		SourceClient:      envelope.ID.Source.ClientID,
		DestinationClient: envelope.ID.Destination.ClientID,
		TimeoutTimestamp:  envelope.Timeout.Timestamp,
		Payloads:          toContractPayloads(envelope.Payloads),
	}, nil
}

func fromContractPayloads(payloads []ContractPayload) []protocol.Payload {
	converted := make([]protocol.Payload, len(payloads))
	for i, payload := range payloads {
		converted[i] = protocol.Payload{
			SourcePort:      payload.SourcePort,
			DestinationPort: payload.DestinationPort,
			Version:         payload.Version,
			Encoding:        payload.Encoding,
			Value:           append([]byte(nil), payload.Value...),
		}
	}
	return converted
}

func toContractPayloads(payloads []protocol.Payload) []ContractPayload {
	converted := make([]ContractPayload, len(payloads))
	for i, payload := range payloads {
		converted[i] = ContractPayload{
			SourcePort:      payload.SourcePort,
			DestinationPort: payload.DestinationPort,
			Version:         payload.Version,
			Encoding:        payload.Encoding,
			Value:           append([]byte(nil), payload.Value...),
		}
	}
	return converted
}
