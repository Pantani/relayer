package chains

import (
	"errors"
	"fmt"

	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/cosmos/relayer/v2/relayer/protocol"
	protocolv2 "github.com/cosmos/relayer/v2/relayer/protocol/v2"
)

var (
	ErrAmbiguousPacketProtocol = errors.New("ambiguous packet protocol")
	ErrMalformedPacketEvent    = errors.New("malformed packet event")
)

var classicPacketAttributeKeys = []string{
	chantypes.AttributeKeySrcPort,
	chantypes.AttributeKeySrcChannel,
	chantypes.AttributeKeyDstPort,
	chantypes.AttributeKeyDstChannel,
	chantypes.AttributeKeyTimeoutHeight,
	legacyAttributeKeyPacketData,
	chantypes.AttributeKeyDataHex,
	legacyAttributeKeyPacketAck,
	chantypes.AttributeKeyAckHex,
	chantypes.AttributeKeyChannelOrdering,
	chantypes.AttributeKeyConnection,
}

var v2PacketAttributeKeys = []string{
	protocolv2.AttributeKeySrcClient,
	protocolv2.AttributeKeyDstClient,
	protocolv2.AttributeKeyEncodedPacketHex,
	protocolv2.AttributeKeyEncodedAckHex,
}

var requiredClassicPacketAttributes = []string{
	chantypes.AttributeKeySrcPort,
	chantypes.AttributeKeySrcChannel,
	chantypes.AttributeKeyDstPort,
	chantypes.AttributeKeyDstChannel,
	chantypes.AttributeKeySequence,
	chantypes.AttributeKeyTimeoutHeight,
	chantypes.AttributeKeyTimeoutTimestamp,
}

var requiredV2PacketAttributes = []string{
	protocolv2.AttributeKeySrcClient,
	protocolv2.AttributeKeyDstClient,
	protocolv2.AttributeKeySequence,
	protocolv2.AttributeKeyTimeoutTimestamp,
	protocolv2.AttributeKeyEncodedPacketHex,
}

func classifyPacketEvent(eventType string, attributes []protocol.EventAttribute) (protocol.Protocol, error) {
	if !isPacketEventType(eventType) {
		return protocol.ProtocolUnspecified, fmt.Errorf("%w: event type %q", ErrMalformedPacketEvent, eventType)
	}
	classic := hasAnyEventAttribute(attributes, classicPacketAttributeKeys)
	v2 := hasAnyEventAttribute(attributes, v2PacketAttributeKeys)
	if classic && v2 {
		return protocol.ProtocolUnspecified, ErrAmbiguousPacketProtocol
	}
	if classic {
		return classifyClassicPacketEvent(attributes)
	}
	if v2 {
		return classifyV2PacketEvent(eventType, attributes)
	}
	return protocol.ProtocolUnspecified, ErrMalformedPacketEvent
}

func classifyClassicPacketEvent(attributes []protocol.EventAttribute) (protocol.Protocol, error) {
	if err := requireSingletonAttributes(attributes, requiredClassicPacketAttributes); err != nil {
		return protocol.ProtocolUnspecified, err
	}
	return protocol.ProtocolClassic, nil
}

func classifyV2PacketEvent(eventType string, attributes []protocol.EventAttribute) (protocol.Protocol, error) {
	if err := requireSingletonAttributes(attributes, requiredV2PacketAttributes); err != nil {
		return protocol.ProtocolUnspecified, err
	}
	if err := validateV2AcknowledgementAttribute(eventType, attributes); err != nil {
		return protocol.ProtocolUnspecified, err
	}
	return protocol.ProtocolV2, nil
}

func validateV2AcknowledgementAttribute(eventType string, attributes []protocol.EventAttribute) error {
	count := eventAttributeCount(attributes, protocolv2.AttributeKeyEncodedAckHex)
	if eventType == protocolv2.EventTypeWriteAck && count != 1 {
		return malformedAttributeCount(protocolv2.AttributeKeyEncodedAckHex, count)
	}
	if eventType != protocolv2.EventTypeWriteAck && count != 0 {
		return malformedAttributeCount(protocolv2.AttributeKeyEncodedAckHex, count)
	}
	return nil
}

func requireSingletonAttributes(attributes []protocol.EventAttribute, required []string) error {
	for _, key := range required {
		if count := eventAttributeCount(attributes, key); count != 1 {
			return malformedAttributeCount(key, count)
		}
	}
	return nil
}

func malformedAttributeCount(key string, count int) error {
	return fmt.Errorf("%w: attribute %q has %d values", ErrMalformedPacketEvent, key, count)
}

func eventAttributeCount(attributes []protocol.EventAttribute, key string) int {
	count := 0
	for _, attribute := range attributes {
		if attribute.Key == key {
			count++
		}
	}
	return count
}

func hasAnyEventAttribute(attributes []protocol.EventAttribute, keys []string) bool {
	for _, key := range keys {
		if eventAttributeCount(attributes, key) != 0 {
			return true
		}
	}
	return false
}

func isPacketEventType(eventType string) bool {
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

func requiredEventAttribute(attributes []protocol.EventAttribute, key string) (string, error) {
	if eventAttributeCount(attributes, key) != 1 {
		return "", malformedAttributeCount(key, eventAttributeCount(attributes, key))
	}
	for _, attribute := range attributes {
		if attribute.Key == key {
			return attribute.Value, nil
		}
	}
	return "", malformedAttributeCount(key, 0)
}
