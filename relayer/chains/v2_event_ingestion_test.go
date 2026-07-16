package chains

import (
	"errors"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/cosmos/relayer/v2/relayer/protocol"
	protocolv2 "github.com/cosmos/relayer/v2/relayer/protocol/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	v11PacketHex = "0807120f30372d74656e6465726d696e742d301a0f30372d74656e6465726d696e742d31207b2a340a087472616e7366657212087472616e736665721a0769637332302d3122106170706c69636174696f6e2f6a736f6e2a03010203"
	v11AckHex    = "0a02aabb"
)

func TestV2LifecycleEventMatrix(t *testing.T) {
	eventTypes := []string{
		protocolv2.EventTypeSendPacket,
		protocolv2.EventTypeRecvPacket,
		protocolv2.EventTypeWriteAck,
		protocolv2.EventTypeAcknowledgePacket,
		protocolv2.EventTypeTimeoutPacket,
	}

	for _, eventType := range eventTypes {
		t.Run(eventType, func(t *testing.T) {
			events := []abci.Event{messageActionEvent("/fixture.MsgZero", "0"), v2PacketEvent(eventType, "0")}
			batch := ParseIBCEventBatch(zap.NewNop(), events, testEventMetadata())

			require.Empty(t, batch.Issues)
			require.Len(t, batch.Envelopes, 1)
			require.Len(t, batch.V2Packets, 1)
			require.Empty(t, batch.ClassicMessages)
			requireV2PacketEvent(t, batch.V2Packets[0], eventType)
			require.Empty(t, IbcMessagesFromEvents(zap.NewNop(), events, "chain-a", 44))
		})
	}
}

func TestRawV2EventPreservesOrderDuplicatesAndIndex(t *testing.T) {
	packetEvent := v2PacketEvent(protocolv2.EventTypeSendPacket, "0")
	packetEvent.Attributes = append(packetEvent.Attributes[:2], append([]abci.EventAttribute{
		{Key: "custom", Value: "first", Index: true},
		{Key: "custom", Value: "second", Index: false},
	}, packetEvent.Attributes[2:]...)...)
	events := []abci.Event{messageActionEvent("/fixture.MsgZero", "0"), packetEvent}
	expected := protocolAttributesFromABCI(packetEvent.Attributes)

	batch := ParseIBCEventBatch(zap.NewNop(), events, testEventMetadata())

	require.Empty(t, batch.Issues)
	require.Len(t, batch.V2Packets, 1)
	require.Equal(t, expected, batch.V2Packets[0].Event.Attributes)
	require.Equal(t, []string{"first", "second"}, batch.V2Packets[0].Event.AttributeValues("custom"))
	events[1].Attributes[0].Value = "mutated"
	require.Equal(t, "07-tendermint-0", batch.V2Packets[0].Event.Attributes[0].Value)
}

func TestMessageActionCorrelationByIndex(t *testing.T) {
	events := []abci.Event{
		v2PacketEvent(protocolv2.EventTypeRecvPacket, "1"),
		messageActionEvent("/fixture.MsgZero", "0"),
		v2PacketEvent(protocolv2.EventTypeSendPacket, "0"),
		keeperModuleEvent("0"),
		messageActionEvent("/fixture.MsgOne", "1"),
	}

	batch := ParseIBCEventBatch(zap.NewNop(), events, testEventMetadata())

	require.Empty(t, batch.Issues)
	require.Len(t, batch.V2Packets, 2)
	require.Equal(t, protocol.MessageAction{Present: true, Index: 1, Type: "/fixture.MsgOne"}, batch.V2Packets[0].Event.Action)
	require.Equal(t, uint32(0), batch.V2Packets[0].Event.EventIndex)
	require.Equal(t, protocol.MessageAction{Present: true, Index: 0, Type: "/fixture.MsgZero"}, batch.V2Packets[1].Event.Action)
	require.Equal(t, uint32(2), batch.V2Packets[1].Event.EventIndex)
}

func TestLegacyActionCorrelationDoesNotUseKeeperModuleEvent(t *testing.T) {
	packetZero := v2PacketEvent(protocolv2.EventTypeSendPacket, "")
	packetOne := v2PacketEvent(protocolv2.EventTypeRecvPacket, "")
	events := []abci.Event{
		messageActionEvent("/fixture.Legacy", ""),
		packetZero,
		keeperModuleEvent(""),
		packetOne,
	}

	batch := ParseIBCEventBatch(zap.NewNop(), events, testEventMetadata())

	require.Empty(t, batch.Issues)
	require.Len(t, batch.V2Packets, 2)
	want := protocol.MessageAction{Present: true, Index: 0, Type: "/fixture.Legacy"}
	require.Equal(t, want, batch.V2Packets[0].Event.Action)
	require.Equal(t, want, batch.V2Packets[1].Event.Action)
}

func TestMessageActionFailures(t *testing.T) {
	tests := []struct {
		name   string
		events []abci.Event
		target error
	}{
		{name: "missing action", events: []abci.Event{v2PacketEvent(protocolv2.EventTypeSendPacket, "0")}, target: ErrMessageActionRequired},
		{name: "invalid index", events: []abci.Event{messageActionEvent("/fixture.Msg", "x"), v2PacketEvent(protocolv2.EventTypeSendPacket, "x")}, target: ErrInvalidMessageIndex},
		{name: "negative index", events: []abci.Event{messageActionEvent("/fixture.Msg", "-1"), v2PacketEvent(protocolv2.EventTypeSendPacket, "-1")}, target: ErrInvalidMessageIndex},
		{name: "overflow index", events: []abci.Event{messageActionEvent("/fixture.Msg", "4294967296"), v2PacketEvent(protocolv2.EventTypeSendPacket, "4294967296")}, target: ErrInvalidMessageIndex},
		{name: "conflicting event index", events: conflictingPacketIndexEvents(), target: ErrConflictingMessageIndex},
		{name: "conflicting action attributes", events: conflictingActionAttributeEvents(), target: ErrConflictingMessageAction},
		{name: "conflicting actions for index", events: conflictingIndexedActionEvents(), target: ErrConflictingMessageAction},
		{name: "empty action", events: []abci.Event{messageActionEvent("", "0"), v2PacketEvent(protocolv2.EventTypeSendPacket, "0")}, target: ErrMessageActionRequired},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			batch := ParseIBCEventBatch(zap.NewNop(), test.events, testEventMetadata())
			requireBatchIssueIs(t, batch, test.target)
			require.Empty(t, batch.V2Packets)
		})
	}
}

func TestIdenticalMessageActionDuplicatesAreIdempotent(t *testing.T) {
	actionEvent := messageActionEvent("/fixture.Msg", "0")
	actionEvent.Attributes = append(actionEvent.Attributes,
		abci.EventAttribute{Key: sdk.AttributeKeyAction, Value: "/fixture.Msg"},
		abci.EventAttribute{Key: messageIndexAttribute, Value: "0"},
	)
	events := []abci.Event{actionEvent, messageActionEvent("/fixture.Msg", "0"), v2PacketEvent(protocolv2.EventTypeSendPacket, "0")}

	batch := ParseIBCEventBatch(zap.NewNop(), events, testEventMetadata())

	require.Empty(t, batch.Issues)
	require.Len(t, batch.V2Packets, 1)
	require.Equal(t, protocol.MessageAction{Present: true, Index: 0, Type: "/fixture.Msg"}, batch.V2Packets[0].Event.Action)
}

func TestPacketProtocolClassificationMatrix(t *testing.T) {
	v2Attributes := protocolAttributesFromABCI(v2PacketAttributes("0"))
	classicAttributes := protocolAttributesFromABCI(classicPacketAttributes("0"))
	ambiguous := append(append([]protocol.EventAttribute{}, v2Attributes...), classicAttributes[1])
	incomplete := []protocol.EventAttribute{{Key: protocolv2.AttributeKeySrcClient, Value: "07-tendermint-0"}}
	duplicate := append(append([]protocol.EventAttribute{}, v2Attributes...), v2Attributes[0])
	tests := []struct {
		name       string
		attributes []protocol.EventAttribute
		want       protocol.Protocol
		target     error
	}{
		{name: "classic", attributes: classicAttributes, want: protocol.ProtocolClassic},
		{name: "v2", attributes: v2Attributes, want: protocol.ProtocolV2},
		{name: "ambiguous", attributes: ambiguous, target: ErrAmbiguousPacketProtocol},
		{name: "incomplete", attributes: incomplete, target: ErrMalformedPacketEvent},
		{name: "duplicate singleton", attributes: duplicate, target: ErrMalformedPacketEvent},
		{name: "missing signature", target: ErrMalformedPacketEvent},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := classifyPacketEvent(protocolv2.EventTypeSendPacket, test.attributes)
			if test.target != nil {
				require.ErrorIs(t, err, test.target)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestRejectedPacketIssuePreservesLosslessEvidence(t *testing.T) {
	tests := []struct {
		name   string
		event  abci.Event
		target error
	}{
		{name: "ambiguous", event: ambiguousV2PacketEvent("0"), target: ErrAmbiguousPacketProtocol},
		{name: "incomplete", event: incompleteV2PacketEvent("0"), target: ErrMalformedPacketEvent},
		{name: "duplicate singleton", event: duplicateV2PacketEvent("0"), target: ErrMalformedPacketEvent},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expected := protocolAttributesFromABCI(test.event.Attributes)
			events := []abci.Event{messageActionEvent("/fixture.Msg", "0"), test.event}
			batch := ParseIBCEventBatch(zap.NewNop(), events, testEventMetadata())
			issue := batchIssueFor(t, batch, test.target)

			require.NotNil(t, issue.Event)
			require.Equal(t, protocol.ProtocolUnspecified, issue.Event.Protocol)
			require.Equal(t, uint32(1), issue.EventIndex)
			require.Equal(t, test.event.Type, issue.EventType)
			require.Equal(t, expected, issue.Event.Attributes)
			require.Empty(t, batch.Envelopes)
			events[1].Attributes[0].Value = "mutated"
			require.Equal(t, expected, issue.Event.Attributes)
		})
	}
}

func TestCorrelationIssuePrecedesClassificationIssue(t *testing.T) {
	tests := []struct {
		name        string
		packetEvent abci.Event
		first       error
	}{
		{name: "invalid index", packetEvent: ambiguousV2PacketEvent("x"), first: ErrInvalidMessageIndex},
		{name: "conflicting index", packetEvent: ambiguousV2PacketEventWithConflictingIndex(), first: ErrConflictingMessageIndex},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			batch := ParseIBCEventBatch(zap.NewNop(), []abci.Event{messageActionEvent("/fixture.Msg", "0"), test.packetEvent}, testEventMetadata())

			require.Len(t, batch.Issues, 2)
			require.ErrorIs(t, batch.Issues[0].Err, test.first)
			require.ErrorIs(t, batch.Issues[1].Err, ErrAmbiguousPacketProtocol)
			require.NotNil(t, batch.Issues[0].Event)
			require.NotNil(t, batch.Issues[1].Event)
			require.Equal(t, batch.Issues[0].Event.Attributes, batch.Issues[1].Event.Attributes)
			require.Empty(t, batch.Envelopes)
			require.Empty(t, batch.V2Packets)
		})
	}
}

func TestInvalidActionPoisonsPreviouslyIndexedAction(t *testing.T) {
	tests := []struct {
		name    string
		invalid abci.Event
		target  error
	}{
		{name: "empty", invalid: messageActionEvent("", "0"), target: ErrMessageActionRequired},
		{name: "conflicting duplicate", invalid: conflictingActionEvent("0"), target: ErrConflictingMessageAction},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := []abci.Event{
				messageActionEvent("/fixture.MsgA", "0"),
				test.invalid,
				v2PacketEvent(protocolv2.EventTypeSendPacket, "0"),
			}
			batch := ParseIBCEventBatch(zap.NewNop(), events, testEventMetadata())
			requireBatchIssueIs(t, batch, test.target)
			require.Empty(t, batch.V2Packets)
		})
	}
}

func TestInvalidLegacyActionPoisonsUntilNextValidAction(t *testing.T) {
	tests := []struct {
		name    string
		invalid abci.Event
		target  error
	}{
		{name: "empty", invalid: messageActionEvent("", ""), target: ErrMessageActionRequired},
		{name: "conflicting duplicate", invalid: conflictingActionEvent(""), target: ErrConflictingMessageAction},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			poisonedEvents := []abci.Event{
				messageActionEvent("/fixture.MsgA", ""),
				test.invalid,
				v2PacketEvent(protocolv2.EventTypeSendPacket, ""),
			}
			poisoned := ParseIBCEventBatch(zap.NewNop(), poisonedEvents, testEventMetadata())
			requireBatchIssueIs(t, poisoned, test.target)
			require.Empty(t, poisoned.V2Packets)

			recoveredEvents := append(poisonedEvents[:2],
				messageActionEvent("/fixture.MsgB", ""),
				v2PacketEvent(protocolv2.EventTypeSendPacket, ""),
			)
			recovered := ParseIBCEventBatch(zap.NewNop(), recoveredEvents, testEventMetadata())
			requireBatchIssueIs(t, recovered, test.target)
			require.Len(t, recovered.V2Packets, 1)
			require.Equal(t, "/fixture.MsgB", recovered.V2Packets[0].Event.Action.Type)
		})
	}
}

func TestV2PacketAttributeMismatchMatrix(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "source", key: protocolv2.AttributeKeySrcClient, value: "07-tendermint-9"},
		{name: "destination", key: protocolv2.AttributeKeyDstClient, value: "07-tendermint-9"},
		{name: "sequence", key: protocolv2.AttributeKeySequence, value: "8"},
		{name: "invalid sequence", key: protocolv2.AttributeKeySequence, value: "x"},
		{name: "timeout", key: protocolv2.AttributeKeyTimeoutTimestamp, value: "124"},
		{name: "overflow timeout", key: protocolv2.AttributeKeyTimeoutTimestamp, value: "18446744073709551616"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			packetEvent := v2PacketEvent(protocolv2.EventTypeSendPacket, "0")
			replaceABCIAttribute(&packetEvent, test.key, test.value)
			batch := ParseIBCEventBatch(zap.NewNop(), []abci.Event{messageActionEvent("/fixture.Msg", "0"), packetEvent}, testEventMetadata())
			requireBatchIssueIs(t, batch, ErrPacketEventMismatch)
			require.Empty(t, batch.V2Packets)
		})
	}
}

func TestV2AcknowledgementRules(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		ackHex    *string
		target    error
	}{
		{name: "write ack valid", eventType: protocolv2.EventTypeWriteAck, ackHex: stringPointer(v11AckHex)},
		{name: "write ack missing", eventType: protocolv2.EventTypeWriteAck, target: ErrMalformedPacketEvent},
		{name: "write ack invalid hex", eventType: protocolv2.EventTypeWriteAck, ackHex: stringPointer("xyz"), target: protocolv2.ErrInvalidHex},
		{name: "write ack empty contract", eventType: protocolv2.EventTypeWriteAck, ackHex: stringPointer(""), target: protocolv2.ErrInvalidAcknowledgement},
		{name: "write ack two app acks", eventType: protocolv2.EventTypeWriteAck, ackHex: stringPointer("0a02aabb0a026f6b"), target: protocolv2.ErrInvalidAcknowledgement},
		{name: "send has unexpected ack", eventType: protocolv2.EventTypeSendPacket, ackHex: stringPointer(v11AckHex), target: ErrMalformedPacketEvent},
		{name: "recv has unexpected ack", eventType: protocolv2.EventTypeRecvPacket, ackHex: stringPointer(v11AckHex), target: ErrMalformedPacketEvent},
		{name: "acknowledge has unexpected ack", eventType: protocolv2.EventTypeAcknowledgePacket, ackHex: stringPointer(v11AckHex), target: ErrMalformedPacketEvent},
		{name: "timeout has unexpected ack", eventType: protocolv2.EventTypeTimeoutPacket, ackHex: stringPointer(v11AckHex), target: ErrMalformedPacketEvent},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			packetEvent := v2PacketEventWithAck(test.eventType, "0", test.ackHex)
			batch := ParseIBCEventBatch(zap.NewNop(), []abci.Event{messageActionEvent("/fixture.Msg", "0"), packetEvent}, testEventMetadata())
			if test.target != nil {
				requireBatchIssueIs(t, batch, test.target)
				require.Empty(t, batch.V2Packets)
				return
			}
			require.Empty(t, batch.Issues)
			require.Len(t, batch.V2Packets, 1)
			require.Equal(t, []byte{0xaa, 0xbb}, batch.V2Packets[0].Observation.Acknowledgement)
			require.Equal(t, [][]byte{{0xaa, 0xbb}}, batch.V2Packets[0].Acknowledgement.AppAcknowledgements)
		})
	}
}

func TestMixedClassicAndV2BatchKeepsV2SidecarOnly(t *testing.T) {
	events := []abci.Event{
		messageActionEvent("/fixture.Msg", "0"),
		classicPacketEvent(protocolv2.EventTypeSendPacket, "0"),
		v2PacketEvent(protocolv2.EventTypeSendPacket, "0"),
	}

	batch := ParseIBCEventBatch(zap.NewNop(), events, testEventMetadata())

	require.Empty(t, batch.Issues)
	require.Len(t, batch.ClassicMessages, 1)
	require.Len(t, batch.V2Packets, 1)
	require.Len(t, batch.Envelopes, 2)
	require.Equal(t, protocol.ProtocolClassic, batch.Envelopes[0].Protocol)
	require.Equal(t, protocol.ProtocolV2, batch.Envelopes[1].Protocol)
	require.Len(t, IbcMessagesFromEvents(zap.NewNop(), events, "chain-a", 44), 1)
	require.Empty(t, IbcMessagesFromEvents(zap.NewNop(), []abci.Event{events[0], events[2]}, "chain-a", 44))
}

func TestAmbiguousPacketProducesOnlyIssue(t *testing.T) {
	packetEvent := v2PacketEvent(protocolv2.EventTypeSendPacket, "0")
	packetEvent.Attributes = append(packetEvent.Attributes, abci.EventAttribute{Key: chantypes.AttributeKeySrcPort, Value: "transfer"})
	batch := ParseIBCEventBatch(zap.NewNop(), []abci.Event{messageActionEvent("/fixture.Msg", "0"), packetEvent}, testEventMetadata())

	requireBatchIssueIs(t, batch, ErrAmbiguousPacketProtocol)
	require.Empty(t, batch.Envelopes)
	require.Empty(t, batch.ClassicMessages)
	require.Empty(t, batch.V2Packets)
}

func TestDirectPacketParserRejectsV2AndAmbiguousEvents(t *testing.T) {
	classicEvent := sdk.StringEvent{Type: chantypes.EventTypeSendPacket, Attributes: sdkAttributes(classicPacketAttributes("0"))}
	require.NotNil(t, ParseIBCMessageFromEvent(zap.NewNop(), classicEvent, "chain-a", 44))

	for _, eventType := range []string{
		protocolv2.EventTypeSendPacket,
		protocolv2.EventTypeRecvPacket,
		protocolv2.EventTypeWriteAck,
		protocolv2.EventTypeAcknowledgePacket,
		protocolv2.EventTypeTimeoutPacket,
	} {
		v2Event := v2PacketEvent(eventType, "0")
		stringEvent := sdk.StringEvent{Type: eventType, Attributes: sdkAttributes(v2Event.Attributes)}
		require.Nil(t, ParseIBCMessageFromEvent(zap.NewNop(), stringEvent, "chain-a", 44))
	}

	ambiguous := ambiguousV2PacketEvent("0")
	stringEvent := sdk.StringEvent{Type: ambiguous.Type, Attributes: sdkAttributes(ambiguous.Attributes)}
	require.Nil(t, ParseIBCMessageFromEvent(zap.NewNop(), stringEvent, "chain-a", 44))
}

func TestClassicRawBatchRegression(t *testing.T) {
	events := []abci.Event{messageActionEvent("/fixture.Msg", "0"), classicPacketEvent(chantypes.EventTypeSendPacket, "0")}
	batch := ParseIBCEventBatch(zap.NewNop(), events, testEventMetadata())

	require.Empty(t, batch.Issues)
	require.Len(t, batch.ClassicMessages, 1)
	require.Empty(t, batch.V2Packets)
	packet, ok := batch.ClassicMessages[0].Info.(*PacketInfo)
	require.True(t, ok)
	require.Equal(t, uint64(44), packet.Height)
	require.Equal(t, uint64(7), packet.Sequence)
	require.Equal(t, "transfer", packet.SourcePort)
	require.Equal(t, "channel-1", packet.SourceChannel)
	require.Equal(t, "transfer", packet.DestPort)
	require.Equal(t, "channel-9", packet.DestChannel)
	require.Equal(t, []byte{1, 2, 3}, packet.Data)
}

func requireV2PacketEvent(t *testing.T, packetEvent V2PacketEvent, eventType string) {
	t.Helper()
	require.Equal(t, protocol.ProtocolV2, packetEvent.Event.Protocol)
	require.Equal(t, eventType, packetEvent.Event.Type)
	require.Equal(t, uint64(44), packetEvent.Event.Height)
	require.Equal(t, "ABC123", packetEvent.Event.TxHash)
	require.Equal(t, uint32(1), packetEvent.Event.EventIndex)
	require.Equal(t, protocol.MessageAction{Present: true, Index: 0, Type: "/fixture.MsgZero"}, packetEvent.Event.Action)
	require.Equal(t, protocol.ProtocolV2, packetEvent.Observation.Protocol)
	require.Equal(t, eventType, packetEvent.Observation.EventType)
	require.Equal(t, "07-tendermint-0", packetEvent.Observation.Packet.ID.Source.ClientID)
	require.Equal(t, "07-tendermint-1", packetEvent.Observation.Packet.ID.Destination.ClientID)
	require.Equal(t, uint64(7), packetEvent.Observation.Packet.ID.Sequence)
	require.Equal(t, uint64(123), packetEvent.Observation.Packet.Timeout.Timestamp)
	require.Equal(t, protocol.TimestampSeconds, packetEvent.Observation.Packet.Timeout.TimestampUnit)
	require.Len(t, packetEvent.Observation.Packet.Payloads, 1)
	if eventType == protocolv2.EventTypeWriteAck {
		require.Equal(t, []byte{0xaa, 0xbb}, packetEvent.Observation.Acknowledgement)
		require.NotNil(t, packetEvent.Acknowledgement)
		return
	}
	require.Nil(t, packetEvent.Observation.Acknowledgement)
	require.Nil(t, packetEvent.Acknowledgement)
}

func requireBatchIssueIs(t *testing.T, batch IBCEventBatch, target error) {
	t.Helper()
	batchIssueFor(t, batch, target)
}

func batchIssueFor(t *testing.T, batch IBCEventBatch, target error) IBCEventIssue {
	t.Helper()
	for _, issue := range batch.Issues {
		if errors.Is(issue.Err, target) {
			return issue
		}
	}
	require.FailNowf(t, "missing batch issue", "target %v, issues: %+v", target, batch.Issues)
	return IBCEventIssue{}
}

func testEventMetadata() IBCEventMetadata {
	return IBCEventMetadata{ChainID: "chain-a", Height: 44, TxHash: "ABC123"}
}

func messageActionEvent(action, messageIndex string) abci.Event {
	attributes := []abci.EventAttribute{{Key: sdk.AttributeKeyAction, Value: action, Index: true}}
	if messageIndex != "" {
		attributes = append(attributes, abci.EventAttribute{Key: messageIndexAttribute, Value: messageIndex})
	}
	return abci.Event{Type: sdk.EventTypeMessage, Attributes: attributes}
}

func keeperModuleEvent(messageIndex string) abci.Event {
	attributes := []abci.EventAttribute{{Key: sdk.AttributeKeyModule, Value: "ibc_channel_v2"}}
	if messageIndex != "" {
		attributes = append(attributes, abci.EventAttribute{Key: messageIndexAttribute, Value: messageIndex})
	}
	return abci.Event{Type: sdk.EventTypeMessage, Attributes: attributes}
}

func v2PacketEvent(eventType, messageIndex string) abci.Event {
	ack := (*string)(nil)
	if eventType == protocolv2.EventTypeWriteAck {
		ack = stringPointer(v11AckHex)
	}
	return v2PacketEventWithAck(eventType, messageIndex, ack)
}

func v2PacketEventWithAck(eventType, messageIndex string, ackHex *string) abci.Event {
	attributes := v2PacketAttributes("")
	if ackHex != nil {
		attributes = append(attributes, abci.EventAttribute{Key: protocolv2.AttributeKeyEncodedAckHex, Value: *ackHex})
	}
	if messageIndex != "" {
		attributes = append(attributes, abci.EventAttribute{Key: messageIndexAttribute, Value: messageIndex})
	}
	return abci.Event{Type: eventType, Attributes: attributes}
}

func v2PacketAttributes(messageIndex string) []abci.EventAttribute {
	attributes := []abci.EventAttribute{
		{Key: protocolv2.AttributeKeySrcClient, Value: "07-tendermint-0", Index: true},
		{Key: protocolv2.AttributeKeyDstClient, Value: "07-tendermint-1"},
		{Key: protocolv2.AttributeKeySequence, Value: "7", Index: true},
		{Key: protocolv2.AttributeKeyTimeoutTimestamp, Value: "123"},
		{Key: protocolv2.AttributeKeyEncodedPacketHex, Value: v11PacketHex, Index: true},
	}
	if messageIndex != "" {
		attributes = append(attributes, abci.EventAttribute{Key: messageIndexAttribute, Value: messageIndex})
	}
	return attributes
}

func classicPacketEvent(eventType, messageIndex string) abci.Event {
	return abci.Event{Type: eventType, Attributes: classicPacketAttributes(messageIndex)}
}

func classicPacketAttributes(messageIndex string) []abci.EventAttribute {
	attributes := []abci.EventAttribute{
		{Key: chantypes.AttributeKeySequence, Value: "7"},
		{Key: chantypes.AttributeKeySrcPort, Value: "transfer"},
		{Key: chantypes.AttributeKeySrcChannel, Value: "channel-1"},
		{Key: chantypes.AttributeKeyDstPort, Value: "transfer"},
		{Key: chantypes.AttributeKeyDstChannel, Value: "channel-9"},
		{Key: chantypes.AttributeKeyTimeoutHeight, Value: "2-100"},
		{Key: chantypes.AttributeKeyTimeoutTimestamp, Value: "123456789"},
		{Key: chantypes.AttributeKeyDataHex, Value: "010203"},
	}
	if messageIndex != "" {
		attributes = append(attributes, abci.EventAttribute{Key: messageIndexAttribute, Value: messageIndex})
	}
	return attributes
}

func conflictingPacketIndexEvents() []abci.Event {
	packetEvent := v2PacketEvent(protocolv2.EventTypeSendPacket, "0")
	packetEvent.Attributes = append(packetEvent.Attributes, abci.EventAttribute{Key: messageIndexAttribute, Value: "1"})
	return []abci.Event{messageActionEvent("/fixture.Msg", "0"), packetEvent}
}

func conflictingActionAttributeEvents() []abci.Event {
	actionEvent := messageActionEvent("/fixture.MsgA", "0")
	actionEvent.Attributes = append(actionEvent.Attributes, abci.EventAttribute{Key: sdk.AttributeKeyAction, Value: "/fixture.MsgB"})
	return []abci.Event{actionEvent, v2PacketEvent(protocolv2.EventTypeSendPacket, "0")}
}

func conflictingIndexedActionEvents() []abci.Event {
	return []abci.Event{
		messageActionEvent("/fixture.MsgA", "0"),
		messageActionEvent("/fixture.MsgB", "0"),
		v2PacketEvent(protocolv2.EventTypeSendPacket, "0"),
	}
}

func conflictingActionEvent(messageIndex string) abci.Event {
	event := messageActionEvent("/fixture.MsgB", messageIndex)
	event.Attributes = append(event.Attributes, abci.EventAttribute{Key: sdk.AttributeKeyAction, Value: "/fixture.MsgC"})
	return event
}

func ambiguousV2PacketEvent(messageIndex string) abci.Event {
	event := v2PacketEvent(protocolv2.EventTypeSendPacket, messageIndex)
	event.Attributes = append(event.Attributes, abci.EventAttribute{Key: chantypes.AttributeKeySrcPort, Value: "transfer", Index: true})
	return event
}

func ambiguousV2PacketEventWithConflictingIndex() abci.Event {
	event := ambiguousV2PacketEvent("0")
	event.Attributes = append(event.Attributes, abci.EventAttribute{Key: messageIndexAttribute, Value: "1"})
	return event
}

func incompleteV2PacketEvent(messageIndex string) abci.Event {
	attributes := []abci.EventAttribute{{Key: protocolv2.AttributeKeySrcClient, Value: "07-tendermint-0", Index: true}}
	attributes = append(attributes, abci.EventAttribute{Key: messageIndexAttribute, Value: messageIndex})
	return abci.Event{Type: protocolv2.EventTypeSendPacket, Attributes: attributes}
}

func duplicateV2PacketEvent(messageIndex string) abci.Event {
	event := v2PacketEvent(protocolv2.EventTypeSendPacket, messageIndex)
	event.Attributes = append(event.Attributes, abci.EventAttribute{Key: protocolv2.AttributeKeySequence, Value: "7", Index: false})
	return event
}

func replaceABCIAttribute(event *abci.Event, key, value string) {
	for i := range event.Attributes {
		if event.Attributes[i].Key == key {
			event.Attributes[i].Value = value
			return
		}
	}
}

func protocolAttributesFromABCI(attributes []abci.EventAttribute) []protocol.EventAttribute {
	converted := make([]protocol.EventAttribute, len(attributes))
	for i, attribute := range attributes {
		converted[i] = protocol.EventAttribute{Key: attribute.Key, Value: attribute.Value, Index: attribute.Index}
	}
	return converted
}

func sdkAttributes(attributes []abci.EventAttribute) []sdk.Attribute {
	converted := make([]sdk.Attribute, len(attributes))
	for i, attribute := range attributes {
		converted[i] = sdk.Attribute{Key: attribute.Key, Value: attribute.Value}
	}
	return converted
}

func stringPointer(value string) *string {
	return &value
}
