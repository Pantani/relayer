package chains

import (
	"encoding/hex"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v11/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestParseClassicPacketEvent(t *testing.T) {
	event := sdk.StringEvent{
		Type: chantypes.EventTypeSendPacket,
		Attributes: []sdk.Attribute{
			{Key: chantypes.AttributeKeySequence, Value: "7"},
			{Key: chantypes.AttributeKeySrcPort, Value: "transfer"},
			{Key: chantypes.AttributeKeySrcChannel, Value: "channel-1"},
			{Key: chantypes.AttributeKeyDstPort, Value: "transfer"},
			{Key: chantypes.AttributeKeyDstChannel, Value: "channel-9"},
			{Key: chantypes.AttributeKeyChannelOrdering, Value: "ORDER_UNORDERED"},
			{Key: chantypes.AttributeKeyTimeoutHeight, Value: "2-100"},
			{Key: chantypes.AttributeKeyTimeoutTimestamp, Value: "123456789"},
			{Key: chantypes.AttributeKeyDataHex, Value: "010203"},
		},
	}

	message := ParseIBCMessageFromEvent(zap.NewNop(), event, "chain-a", 42)
	require.NotNil(t, message)
	require.Equal(t, chantypes.EventTypeSendPacket, message.EventType)

	packet, ok := message.Info.(*PacketInfo)
	require.True(t, ok)
	require.Equal(t, uint64(42), packet.Height)
	require.Equal(t, uint64(7), packet.Sequence)
	require.Equal(t, "transfer", packet.SourcePort)
	require.Equal(t, "channel-1", packet.SourceChannel)
	require.Equal(t, "transfer", packet.DestPort)
	require.Equal(t, "channel-9", packet.DestChannel)
	require.Equal(t, "ORDER_UNORDERED", packet.ChannelOrder)
	require.Equal(t, clienttypes.NewHeight(2, 100), packet.TimeoutHeight)
	require.Equal(t, uint64(123456789), packet.TimeoutTimestamp)
	require.Equal(t, []byte{1, 2, 3}, packet.Data)
}

func TestParseClassicChannelEvent(t *testing.T) {
	event := sdk.StringEvent{
		Type: chantypes.EventTypeChannelOpenAck,
		Attributes: []sdk.Attribute{
			{Key: chantypes.AttributeKeyPortID, Value: "transfer"},
			{Key: chantypes.AttributeKeyChannelID, Value: "channel-1"},
			{Key: chantypes.AttributeCounterpartyPortID, Value: "transfer"},
			{Key: chantypes.AttributeCounterpartyChannelID, Value: "channel-9"},
			{Key: chantypes.AttributeKeyConnectionID, Value: "connection-3"},
			{Key: chantypes.AttributeKeyVersion, Value: "ics20-1"},
		},
	}

	message := ParseIBCMessageFromEvent(zap.NewNop(), event, "chain-a", 43)
	require.NotNil(t, message)
	channel, ok := message.Info.(*ChannelInfo)
	require.True(t, ok)
	require.Equal(t, uint64(43), channel.Height)
	require.Equal(t, "transfer", channel.PortID)
	require.Equal(t, "channel-1", channel.ChannelID)
	require.Equal(t, "transfer", channel.CounterpartyPortID)
	require.Equal(t, "channel-9", channel.CounterpartyChannelID)
	require.Equal(t, "connection-3", channel.ConnID)
	require.Equal(t, "ics20-1", channel.Version)
}

func TestParseClassicConnectionEvent(t *testing.T) {
	event := sdk.StringEvent{
		Type: conntypes.EventTypeConnectionOpenTry,
		Attributes: []sdk.Attribute{
			{Key: conntypes.AttributeKeyConnectionID, Value: "connection-3"},
			{Key: conntypes.AttributeKeyClientID, Value: "07-tendermint-1"},
			{Key: conntypes.AttributeKeyCounterpartyConnectionID, Value: "connection-8"},
			{Key: conntypes.AttributeKeyCounterpartyClientID, Value: "07-tendermint-4"},
		},
	}

	message := ParseIBCMessageFromEvent(zap.NewNop(), event, "chain-a", 44)
	require.NotNil(t, message)
	connection, ok := message.Info.(*ConnectionInfo)
	require.True(t, ok)
	require.Equal(t, uint64(44), connection.Height)
	require.Equal(t, "connection-3", connection.ConnID)
	require.Equal(t, "07-tendermint-1", connection.ClientID)
	require.Equal(t, "connection-8", connection.CounterpartyConnID)
	require.Equal(t, "07-tendermint-4", connection.CounterpartyClientID)
}

func TestParseClassicClientEvent(t *testing.T) {
	event := sdk.StringEvent{
		Type: clienttypes.EventTypeUpdateClient,
		Attributes: []sdk.Attribute{
			{Key: clienttypes.AttributeKeyClientID, Value: "07-tendermint-1"},
			{Key: clienttypes.AttributeKeyConsensusHeight, Value: "2-99"},
			{Key: legacyAttributeKeyHeader, Value: "010203"},
		},
	}

	message := ParseIBCMessageFromEvent(zap.NewNop(), event, "chain-a", 45)
	require.NotNil(t, message)
	client, ok := message.Info.(*ClientInfo)
	require.True(t, ok)
	require.Equal(t, "07-tendermint-1", client.ClientID)
	require.Equal(t, clienttypes.NewHeight(2, 99), client.ConsensusHeight)
	require.Equal(t, []byte{1, 2, 3}, client.Header)
}

func TestIgnoreNonIBCEvent(t *testing.T) {
	event := sdk.StringEvent{Type: "coin_spent"}
	require.Nil(t, ParseIBCMessageFromEvent(zap.NewNop(), event, "chain-a", 46))
}

func TestParsePacketAttributeValues(t *testing.T) {
	tests := []struct {
		name string
		attr sdk.Attribute
		want PacketInfo
	}{
		{
			name: "sequence",
			attr: sdk.Attribute{Key: chantypes.AttributeKeySequence, Value: "7"},
			want: PacketInfo{Sequence: 7},
		},
		{
			name: "timeout timestamp",
			attr: sdk.Attribute{Key: chantypes.AttributeKeyTimeoutTimestamp, Value: "123456789"},
			want: PacketInfo{TimeoutTimestamp: 123456789},
		},
		{
			name: "legacy data",
			attr: sdk.Attribute{Key: legacyAttributeKeyPacketData, Value: "legacy-data"},
			want: PacketInfo{Data: []byte("legacy-data")},
		},
		{
			name: "hex data",
			attr: sdk.Attribute{Key: chantypes.AttributeKeyDataHex, Value: "010203"},
			want: PacketInfo{Data: []byte{1, 2, 3}},
		},
		{
			name: "legacy acknowledgement",
			attr: sdk.Attribute{Key: legacyAttributeKeyPacketAck, Value: "legacy-ack"},
			want: PacketInfo{Ack: []byte("legacy-ack")},
		},
		{
			name: "hex acknowledgement",
			attr: sdk.Attribute{Key: chantypes.AttributeKeyAckHex, Value: "040506"},
			want: PacketInfo{Ack: []byte{4, 5, 6}},
		},
		{
			name: "timeout height",
			attr: sdk.Attribute{Key: chantypes.AttributeKeyTimeoutHeight, Value: "2-100"},
			want: PacketInfo{TimeoutHeight: clienttypes.NewHeight(2, 100)},
		},
		{
			name: "source port",
			attr: sdk.Attribute{Key: chantypes.AttributeKeySrcPort, Value: "transfer"},
			want: PacketInfo{SourcePort: "transfer"},
		},
		{
			name: "source channel",
			attr: sdk.Attribute{Key: chantypes.AttributeKeySrcChannel, Value: "channel-1"},
			want: PacketInfo{SourceChannel: "channel-1"},
		},
		{
			name: "destination port",
			attr: sdk.Attribute{Key: chantypes.AttributeKeyDstPort, Value: "transfer"},
			want: PacketInfo{DestPort: "transfer"},
		},
		{
			name: "destination channel",
			attr: sdk.Attribute{Key: chantypes.AttributeKeyDstChannel, Value: "channel-9"},
			want: PacketInfo{DestChannel: "channel-9"},
		},
		{
			name: "channel ordering",
			attr: sdk.Attribute{Key: chantypes.AttributeKeyChannelOrdering, Value: "ORDER_UNORDERED"},
			want: PacketInfo{ChannelOrder: "ORDER_UNORDERED"},
		},
		{
			name: "unknown attribute",
			attr: sdk.Attribute{Key: "unrelated", Value: "ignored"},
			want: PacketInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var packet PacketInfo
			packet.parsePacketAttribute(zap.NewNop(), tt.attr)
			require.Equal(t, tt.want, packet)
		})
	}
}

func TestParsePacketAttributeErrors(t *testing.T) {
	baseline := PacketInfo{
		Sequence:         9,
		TimeoutTimestamp: 8,
		TimeoutHeight:    clienttypes.NewHeight(3, 4),
		Data:             []byte("existing-data"),
		Ack:              []byte("existing-ack"),
	}
	sequenceReset := baseline
	sequenceReset.Sequence = 0
	timestampReset := baseline
	timestampReset.TimeoutTimestamp = 0

	_, sequenceErr := strconv.ParseUint("bad-sequence", 10, 64)
	_, timestampErr := strconv.ParseUint("bad-timestamp", 10, 64)
	_, dataErr := hex.DecodeString("not-hex-data")
	_, ackErr := hex.DecodeString("not-hex-ack")
	_, revisionErr := strconv.ParseUint("badrevision", 10, 64)
	_, heightErr := strconv.ParseUint("badheight", 10, 64)

	tests := []struct {
		name       string
		attr       sdk.Attribute
		want       PacketInfo
		message    string
		wantFields []zap.Field
	}{
		{
			name:       "sequence",
			attr:       sdk.Attribute{Key: chantypes.AttributeKeySequence, Value: "bad-sequence"},
			want:       sequenceReset,
			message:    "Error parsing packet sequence",
			wantFields: []zap.Field{zap.String("value", "bad-sequence"), zap.Error(sequenceErr)},
		},
		{
			name:       "timeout timestamp",
			attr:       sdk.Attribute{Key: chantypes.AttributeKeyTimeoutTimestamp, Value: "bad-timestamp"},
			want:       timestampReset,
			message:    "Error parsing packet timestamp",
			wantFields: []zap.Field{zap.Uint64("sequence", 9), zap.String("value", "bad-timestamp"), zap.Error(timestampErr)},
		},
		{
			name:       "hex data",
			attr:       sdk.Attribute{Key: chantypes.AttributeKeyDataHex, Value: "not-hex-data"},
			want:       baseline,
			message:    "Error parsing packet data",
			wantFields: []zap.Field{zap.Uint64("sequence", 9), zap.Error(dataErr)},
		},
		{
			name:       "hex acknowledgement",
			attr:       sdk.Attribute{Key: chantypes.AttributeKeyAckHex, Value: "not-hex-ack"},
			want:       baseline,
			message:    "Error parsing packet ack",
			wantFields: []zap.Field{zap.Uint64("sequence", 9), zap.String("value", "not-hex-ack"), zap.Error(ackErr)},
		},
		{
			name:       "timeout height shape",
			attr:       sdk.Attribute{Key: chantypes.AttributeKeyTimeoutHeight, Value: "2-100-extra"},
			want:       baseline,
			message:    "Error parsing packet height timeout",
			wantFields: []zap.Field{zap.Uint64("sequence", 9), zap.String("value", "2-100-extra")},
		},
		{
			name:       "timeout height revision",
			attr:       sdk.Attribute{Key: chantypes.AttributeKeyTimeoutHeight, Value: "badrevision-100"},
			want:       baseline,
			message:    "Error parsing packet timeout height revision number",
			wantFields: []zap.Field{zap.Uint64("sequence", 9), zap.String("value", "badrevision"), zap.Error(revisionErr)},
		},
		{
			name:       "timeout height revision height",
			attr:       sdk.Attribute{Key: chantypes.AttributeKeyTimeoutHeight, Value: "2-badheight"},
			want:       baseline,
			message:    "Error parsing packet timeout height revision height",
			wantFields: []zap.Field{zap.Uint64("sequence", 9), zap.String("value", "badheight"), zap.Error(heightErr)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zap.ErrorLevel)
			packet := baseline
			packet.parsePacketAttribute(zap.New(core), tt.attr)
			require.Equal(t, tt.want, packet)
			entries := logs.FilterMessage(tt.message)
			require.Len(t, entries.All(), 1)
			for _, field := range tt.wantFields {
				require.Len(t, entries.FilterField(field).All(), 1)
			}
		})
	}
}
