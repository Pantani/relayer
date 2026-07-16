// Package classic adapts the existing IBC Classic provider values to the
// protocol-neutral model without changing the Classic relay runtime.
package classic

import (
	"bytes"
	"fmt"

	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/cosmos/relayer/v2/relayer/protocol"
	"github.com/cosmos/relayer/v2/relayer/provider"
)

// FromPacketInfo converts a Classic packet-flow observation to the neutral
// model. All byte slices are copied so neither representation aliases the
// other.
func FromPacketInfo(eventType string, info provider.PacketInfo) (protocol.PacketObservation, error) {
	if !isClassicPacketEventType(eventType) {
		return protocol.PacketObservation{}, fmt.Errorf("event type %q is not a Classic packet event", eventType)
	}
	observation := protocol.PacketObservation{
		Protocol:  protocol.ProtocolClassic,
		EventType: eventType,
		Height:    info.Height,
		Packet: protocol.PacketEnvelope{
			ID: protocol.PacketID{
				Protocol: protocol.ProtocolClassic,
				Source: protocol.Endpoint{
					PortID:    info.SourcePort,
					ChannelID: info.SourceChannel,
				},
				Destination: protocol.Endpoint{
					PortID:    info.DestPort,
					ChannelID: info.DestChannel,
				},
				Sequence: info.Sequence,
			},
			Timeout: protocol.Timeout{
				Height: protocol.Height{
					RevisionNumber: info.TimeoutHeight.RevisionNumber,
					RevisionHeight: info.TimeoutHeight.RevisionHeight,
				},
				Timestamp:     info.TimeoutTimestamp,
				TimestampUnit: classicTimestampUnit(info.TimeoutTimestamp),
			},
			Payloads: []protocol.Payload{{
				SourcePort:      info.SourcePort,
				DestinationPort: info.DestPort,
				Value:           bytes.Clone(info.Data),
			}},
		},
		ChannelOrder:    info.ChannelOrder,
		Acknowledgement: bytes.Clone(info.Ack),
	}
	if err := observation.Validate(); err != nil {
		return protocol.PacketObservation{}, fmt.Errorf("invalid Classic packet observation: %w", err)
	}
	return observation, nil
}

// ToPacketInfo converts a neutral Classic packet-flow observation back to the
// provider representation.
func ToPacketInfo(observation protocol.PacketObservation) (provider.PacketInfo, error) {
	if observation.Protocol != protocol.ProtocolClassic {
		return provider.PacketInfo{}, fmt.Errorf("cannot convert %q packet observation with Classic adapter", observation.Protocol)
	}
	if observation.Packet.ID.Protocol != protocol.ProtocolClassic {
		return provider.PacketInfo{}, fmt.Errorf("cannot convert %q packet with Classic adapter", observation.Packet.ID.Protocol)
	}
	if !isClassicPacketEventType(observation.EventType) {
		return provider.PacketInfo{}, fmt.Errorf("event type %q is not a Classic packet event", observation.EventType)
	}
	if err := observation.Validate(); err != nil {
		return provider.PacketInfo{}, fmt.Errorf("invalid Classic packet observation: %w", err)
	}
	payload := observation.Packet.Payloads[0]
	return provider.PacketInfo{
		Height:           observation.Height,
		Sequence:         observation.Packet.ID.Sequence,
		SourcePort:       observation.Packet.ID.Source.PortID,
		SourceChannel:    observation.Packet.ID.Source.ChannelID,
		DestPort:         observation.Packet.ID.Destination.PortID,
		DestChannel:      observation.Packet.ID.Destination.ChannelID,
		ChannelOrder:     observation.ChannelOrder,
		Data:             bytes.Clone(payload.Value),
		TimeoutHeight:    providerHeight(observation.Packet.Timeout.Height),
		TimeoutTimestamp: observation.Packet.Timeout.Timestamp,
		Ack:              bytes.Clone(observation.Acknowledgement),
	}, nil
}

// FromPacketProof converts one of the Classic packet proof variants to the
// neutral representation.
func FromPacketProof(kind protocol.ProofKind, proof provider.PacketProof) (protocol.ProofEnvelope, error) {
	if !isClassicPacketProofKind(kind) {
		return protocol.ProofEnvelope{}, fmt.Errorf("proof kind %q is not a Classic packet proof", kind)
	}
	envelope := protocol.ProofEnvelope{
		Protocol: protocol.ProtocolClassic,
		Kind:     kind,
		Height: protocol.Height{
			RevisionNumber: proof.ProofHeight.RevisionNumber,
			RevisionHeight: proof.ProofHeight.RevisionHeight,
		},
		Data: bytes.Clone(proof.Proof),
	}
	if err := envelope.Validate(); err != nil {
		return protocol.ProofEnvelope{}, fmt.Errorf("invalid Classic packet proof: %w", err)
	}
	return envelope.Clone(), nil
}

// ToPacketProof converts a neutral Classic packet proof back to the provider
// representation.
func ToPacketProof(envelope protocol.ProofEnvelope) (provider.PacketProof, error) {
	if envelope.Protocol != protocol.ProtocolClassic {
		return provider.PacketProof{}, fmt.Errorf("cannot convert %q proof with Classic adapter", envelope.Protocol)
	}
	if !isClassicPacketProofKind(envelope.Kind) {
		return provider.PacketProof{}, fmt.Errorf("proof kind %q is not a Classic packet proof", envelope.Kind)
	}
	if err := envelope.Validate(); err != nil {
		return provider.PacketProof{}, fmt.Errorf("invalid Classic packet proof: %w", err)
	}
	return provider.PacketProof{
		Proof:       bytes.Clone(envelope.Data),
		ProofHeight: providerHeight(envelope.Height),
	}, nil
}

func isClassicPacketProofKind(kind protocol.ProofKind) bool {
	switch kind {
	case protocol.ProofPacketCommitment,
		protocol.ProofAcknowledgement,
		protocol.ProofReceiptAbsence,
		protocol.ProofNextSequenceRecv:
		return true
	default:
		return false
	}
}

func isClassicPacketEventType(eventType string) bool {
	switch eventType {
	case chantypes.EventTypeSendPacket,
		chantypes.EventTypeRecvPacket,
		chantypes.EventTypeWriteAck,
		chantypes.EventTypeAcknowledgePacket,
		chantypes.EventTypeTimeoutPacket:
		return true
	default:
		return false
	}
}

func providerHeight(height protocol.Height) clienttypes.Height {
	return clienttypes.Height{
		RevisionNumber: height.RevisionNumber,
		RevisionHeight: height.RevisionHeight,
	}
}

func classicTimestampUnit(timestamp uint64) protocol.TimestampUnit {
	if timestamp == 0 {
		return protocol.TimestampUnitUnspecified
	}
	return protocol.TimestampNanoseconds
}
