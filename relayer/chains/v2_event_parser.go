package chains

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/cosmos/relayer/v2/relayer/protocol"
	protocolv2 "github.com/cosmos/relayer/v2/relayer/protocol/v2"
)

var ErrPacketEventMismatch = errors.New("packet event does not match encoded packet")

func parseV2PacketEvent(event protocol.EventEnvelope) (V2PacketEvent, error) {
	packetHex, err := requiredEventAttribute(event.Attributes, protocolv2.AttributeKeyEncodedPacketHex)
	if err != nil {
		return V2PacketEvent{}, err
	}
	packet, err := protocolv2.DecodePacketHex(packetHex)
	if err != nil {
		return V2PacketEvent{}, err
	}
	if err := matchV2PacketEvent(event.Attributes, packet); err != nil {
		return V2PacketEvent{}, err
	}
	acknowledgement, appAcknowledgement, err := decodeV2EventAcknowledgement(event)
	if err != nil {
		return V2PacketEvent{}, err
	}
	observation := protocol.NewPacketObservation(protocol.ProtocolV2, event.Type, event.Height, packet, "", appAcknowledgement)
	if err := observation.Validate(); err != nil {
		return V2PacketEvent{}, fmt.Errorf("%w: %v", protocolv2.ErrInvalidPacketContract, err)
	}
	return V2PacketEvent{Event: event.Clone(), Observation: observation, Acknowledgement: acknowledgement}, nil
}

func decodeV2EventAcknowledgement(event protocol.EventEnvelope) (*protocol.Acknowledgement, []byte, error) {
	if event.Type != protocolv2.EventTypeWriteAck {
		return nil, nil, nil
	}
	value, err := requiredEventAttribute(event.Attributes, protocolv2.AttributeKeyEncodedAckHex)
	if err != nil {
		return nil, nil, err
	}
	acknowledgement, err := protocolv2.DecodeAcknowledgementHex(value)
	if err != nil {
		return nil, nil, err
	}
	owned := acknowledgement.Clone()
	return &owned, acknowledgement.AppAcknowledgements[0], nil
}

func matchV2PacketEvent(attributes []protocol.EventAttribute, packet protocol.PacketEnvelope) error {
	if err := matchV2StringAttribute(attributes, protocolv2.AttributeKeySrcClient, packet.ID.Source.ClientID); err != nil {
		return err
	}
	if err := matchV2StringAttribute(attributes, protocolv2.AttributeKeyDstClient, packet.ID.Destination.ClientID); err != nil {
		return err
	}
	if err := matchV2UintAttribute(attributes, protocolv2.AttributeKeySequence, packet.ID.Sequence); err != nil {
		return err
	}
	return matchV2UintAttribute(attributes, protocolv2.AttributeKeyTimeoutTimestamp, packet.Timeout.Timestamp)
}

func matchV2StringAttribute(attributes []protocol.EventAttribute, key, expected string) error {
	value, err := requiredEventAttribute(attributes, key)
	if err != nil {
		return err
	}
	if value != expected {
		return fmt.Errorf("%w: attribute %q", ErrPacketEventMismatch, key)
	}
	return nil
}

func matchV2UintAttribute(attributes []protocol.EventAttribute, key string, expected uint64) error {
	value, err := requiredEventAttribute(attributes, key)
	if err != nil {
		return err
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed != expected {
		return fmt.Errorf("%w: attribute %q", ErrPacketEventMismatch, key)
	}
	return nil
}
