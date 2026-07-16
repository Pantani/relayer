package chains

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	conntypes "github.com/cosmos/ibc-go/v11/modules/core/03-connection/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
