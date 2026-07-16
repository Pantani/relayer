package cosmos

import (
	"context"
	"testing"

	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/cosmos/relayer/v2/relayer/chains"
	"github.com/cosmos/relayer/v2/relayer/processor"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestCosmosChannelHandlerCharacterizesOpenLifecycleBranches(t *testing.T) {
	tests := []struct {
		event    string
		wantOpen bool
		wantInfo bool
	}{
		{event: chantypes.EventTypeChannelOpenTry},
		{event: chantypes.EventTypeChannelOpenAck, wantOpen: true, wantInfo: true},
		{event: chantypes.EventTypeChannelOpenConfirm, wantOpen: true, wantInfo: true},
	}
	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			chainProcessor, logs := newCosmosChannelHandlerCharacterization()
			cache := processor.NewIBCMessagesCache()
			info := cosmosCharacterizedChannelInfo()
			key := processor.ChannelInfoChannelKey(info)
			chainProcessor.channelStateCache.SetOpen(key.MsgInitKey(), false, chantypes.NONE)

			chainProcessor.handleChannelMessage(tt.event, info, cache)

			require.Equal(t, "connection-7", chainProcessor.channelConnections["channel-7"])
			require.NotContains(t, chainProcessor.channelStateCache, key.MsgInitKey())
			require.Equal(t, processor.ChannelState{Open: tt.wantOpen, Order: chantypes.ORDERED}, chainProcessor.channelStateCache[key])
			require.Equal(t, info, cache.ChannelHandshake[tt.event][key])
			require.Equal(t, 1, logs.FilterMessage("Observed IBC message").Len())
			require.Equal(t, tt.wantInfo, logs.FilterMessage("Successfully created new channel").Len() == 1)
			assertCosmosChannelHandlerLogFields(t, logs, tt.event, info)
		})
	}
}

func TestCosmosChannelHandlerCharacterizesCloseLifecycleBranches(t *testing.T) {
	for _, event := range []string{
		chantypes.EventTypeChannelClosed,
		chantypes.EventTypeChannelCloseConfirm,
	} {
		t.Run(event, func(t *testing.T) {
			chainProcessor, logs := newCosmosChannelHandlerCharacterization()
			cache := processor.NewIBCMessagesCache()
			info := cosmosCharacterizedChannelInfo()
			key := processor.ChannelInfoChannelKey(info)
			chainProcessor.channelStateCache.SetOpen(key, true, chantypes.UNORDERED)
			chainProcessor.channelStateCache.SetOpen(key.MsgInitKey(), false, chantypes.NONE)

			chainProcessor.handleChannelMessage(event, info, cache)

			require.Len(t, chainProcessor.channelStateCache, 1)
			require.Equal(t, processor.ChannelState{Open: false, Order: chantypes.ORDERED}, chainProcessor.channelStateCache[key])
			require.Equal(t, info, cache.ChannelHandshake[event][key])
			require.Equal(t, 1, logs.FilterMessage("Observed IBC message").Len())
			require.Zero(t, logs.FilterMessage("Successfully created new channel").Len())
		})
	}
}

func TestCosmosChannelHandlerCharacterizesCloseWithoutMatchingState(t *testing.T) {
	chainProcessor, _ := newCosmosChannelHandlerCharacterization()
	cache := processor.NewIBCMessagesCache()
	info := cosmosCharacterizedChannelInfo()
	key := processor.ChannelInfoChannelKey(info)
	unrelated := processor.ChannelKey{ChannelID: "channel-other", PortID: info.PortID}
	chainProcessor.channelStateCache.SetOpen(unrelated, true, chantypes.UNORDERED)

	chainProcessor.handleChannelMessage(chantypes.EventTypeChannelClosed, info, cache)

	require.Len(t, chainProcessor.channelStateCache, 1)
	require.NotContains(t, chainProcessor.channelStateCache, key)
	require.Equal(t, processor.ChannelState{Open: true, Order: chantypes.UNORDERED}, chainProcessor.channelStateCache[unrelated])
	require.Equal(t, info, cache.ChannelHandshake[chantypes.EventTypeChannelClosed][key])
}

func TestCosmosChannelHandlerCharacterizesInitInsertionAndExistingFullDedup(t *testing.T) {
	chainProcessor, _ := newCosmosChannelHandlerCharacterization()
	cache := processor.NewIBCMessagesCache()
	info := cosmosCharacterizedChannelInfo()
	info.CounterpartyChannelID = ""
	key := processor.ChannelInfoChannelKey(info)

	chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenInit, info, cache)
	require.Equal(t, processor.ChannelState{Open: false, Order: chantypes.ORDERED}, chainProcessor.channelStateCache[key])
	require.Equal(t, info, cache.ChannelHandshake[chantypes.EventTypeChannelOpenInit][key])

	fullInfo := cosmosCharacterizedChannelInfo()
	fullKey := processor.ChannelInfoChannelKey(fullInfo)
	chainProcessor.channelStateCache = make(processor.ChannelStateCache)
	chainProcessor.channelStateCache.SetOpen(fullKey, true, chantypes.UNORDERED)
	chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenInit, info, cache)

	require.Len(t, chainProcessor.channelStateCache, 1)
	require.NotContains(t, chainProcessor.channelStateCache, key)
	require.Equal(t, processor.ChannelState{Open: true, Order: chantypes.UNORDERED}, chainProcessor.channelStateCache[fullKey])
}

func TestCosmosChannelHandlerCharacterizesUnhandledEventsDeleteInitAndRetain(t *testing.T) {
	for _, event := range []string{chantypes.EventTypeChannelCloseInit, "unknown-channel-event"} {
		t.Run(event, func(t *testing.T) {
			chainProcessor, logs := newCosmosChannelHandlerCharacterization()
			cache := processor.NewIBCMessagesCache()
			info := cosmosCharacterizedChannelInfo()
			key := processor.ChannelInfoChannelKey(info)
			chainProcessor.channelStateCache.SetOpen(key.MsgInitKey(), false, chantypes.ORDERED)

			chainProcessor.handleChannelMessage(event, info, cache)

			require.Empty(t, chainProcessor.channelStateCache)
			require.Equal(t, info, cache.ChannelHandshake[event][key])
			require.Equal(t, "connection-7", chainProcessor.channelConnections["channel-7"])
			require.Equal(t, 1, logs.FilterMessage("Observed IBC message").Len())
		})
	}
}

func TestCosmosChannelHandlerCharacterizesOverwriteAndStateOrderPreservation(t *testing.T) {
	chainProcessor, _ := newCosmosChannelHandlerCharacterization()
	cache := processor.NewIBCMessagesCache()
	first := cosmosCharacterizedChannelInfo()
	key := processor.ChannelInfoChannelKey(first)
	chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenAck, first, cache)

	second := first
	second.Height = 99
	second.ConnID = "connection-overwrite"
	second.Order = chantypes.NONE
	second.Version = "ics20-overwrite"
	chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenAck, second, cache)

	require.Len(t, cache.ChannelHandshake[chantypes.EventTypeChannelOpenAck], 1)
	require.Equal(t, second, cache.ChannelHandshake[chantypes.EventTypeChannelOpenAck][key])
	require.Equal(t, "connection-overwrite", chainProcessor.channelConnections[first.ChannelID])
	require.Equal(t, processor.ChannelState{Open: true, Order: chantypes.ORDERED}, chainProcessor.channelStateCache[key])
}

func TestCosmosChannelHandlerCharacterizesZeroInfoAsValidMapKey(t *testing.T) {
	chainProcessor, logs := newCosmosChannelHandlerCharacterization()
	cache := processor.NewIBCMessagesCache()
	zero := provider.ChannelInfo{}
	zeroKey := processor.ChannelKey{}

	chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenInit, zero, cache)

	require.Equal(t, "", chainProcessor.channelConnections[""])
	require.Equal(t, processor.ChannelState{Open: false, Order: chantypes.NONE}, chainProcessor.channelStateCache[zeroKey])
	require.Equal(t, zero, cache.ChannelHandshake[chantypes.EventTypeChannelOpenInit][zeroKey])
	assertCosmosChannelHandlerLogFields(t, logs, chantypes.EventTypeChannelOpenInit, zero)
}

func TestCosmosChannelHandlerCharacterizesNilHandshakeCachePartialEffects(t *testing.T) {
	chainProcessor, logs := newCosmosChannelHandlerCharacterization()
	cache := processor.IBCMessagesCache{}
	info := cosmosCharacterizedChannelInfo()
	key := processor.ChannelInfoChannelKey(info)
	chainProcessor.channelStateCache.SetOpen(key.MsgInitKey(), false, chantypes.NONE)

	require.Panics(t, func() {
		chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenAck, info, cache)
	})

	require.Equal(t, "connection-7", chainProcessor.channelConnections["channel-7"])
	require.NotContains(t, chainProcessor.channelStateCache, key.MsgInitKey())
	require.Equal(t, processor.ChannelState{Open: true, Order: chantypes.ORDERED}, chainProcessor.channelStateCache[key])
	require.Nil(t, cache.ChannelHandshake)
	require.Equal(t, 1, logs.FilterMessage("Successfully created new channel").Len())
	require.Zero(t, logs.FilterMessage("Observed IBC message").Len())
}

func TestCosmosChannelHandlerCharacterizesNilLoggerEarlyAckPanic(t *testing.T) {
	chainProcessor := &CosmosChainProcessor{
		channelConnections: make(map[string]string),
		channelStateCache:  make(processor.ChannelStateCache),
	}
	cache := processor.NewIBCMessagesCache()
	info := cosmosCharacterizedChannelInfo()
	key := processor.ChannelInfoChannelKey(info)
	chainProcessor.channelStateCache.SetOpen(key.MsgInitKey(), false, chantypes.NONE)

	require.Panics(t, func() {
		chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenAck, info, cache)
	})

	require.Equal(t, "connection-7", chainProcessor.channelConnections["channel-7"])
	require.Contains(t, chainProcessor.channelStateCache, key.MsgInitKey())
	require.Equal(t, processor.ChannelState{Open: true, Order: chantypes.ORDERED}, chainProcessor.channelStateCache[key])
	require.Empty(t, cache.ChannelHandshake)
}

func TestCosmosChannelHandlerCharacterizesNilChannelConnectionsPanicFirst(t *testing.T) {
	chainProcessor, logs := newCosmosChannelHandlerCharacterization()
	chainProcessor.channelConnections = nil
	cache := processor.NewIBCMessagesCache()
	info := cosmosCharacterizedChannelInfo()

	require.Panics(t, func() {
		chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenInit, info, cache)
	})

	require.Empty(t, chainProcessor.channelStateCache)
	require.Empty(t, cache.ChannelHandshake)
	require.Zero(t, logs.Len())
}

func TestCosmosChannelHandlerCharacterizesNilStateCacheAfterConnectionMutation(t *testing.T) {
	chainProcessor, logs := newCosmosChannelHandlerCharacterization()
	chainProcessor.channelStateCache = nil
	cache := processor.NewIBCMessagesCache()
	info := cosmosCharacterizedChannelInfo()

	require.Panics(t, func() {
		chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenInit, info, cache)
	})

	require.Equal(t, "connection-7", chainProcessor.channelConnections["channel-7"])
	require.Nil(t, chainProcessor.channelStateCache)
	require.Empty(t, cache.ChannelHandshake)
	require.Zero(t, logs.Len())
}

func TestCosmosChannelHandlerCharacterizesNilAndTypedNilDispatch(t *testing.T) {
	chainProcessor, logs := newCosmosChannelHandlerCharacterization()
	cache := processor.NewIBCMessagesCache()

	chainProcessor.handleMessage(context.Background(), chains.IbcMessage{Info: nil}, cache)
	require.Empty(t, chainProcessor.channelConnections)
	require.Empty(t, cache.ChannelHandshake)
	require.Zero(t, logs.Len())

	var typedNil *chains.ChannelInfo
	require.Panics(t, func() {
		chainProcessor.handleMessage(context.Background(), chains.IbcMessage{
			EventType: chantypes.EventTypeChannelOpenInit,
			Info:      typedNil,
		}, cache)
	})
	require.Empty(t, chainProcessor.channelConnections)
	require.Empty(t, cache.ChannelHandshake)
}

func TestCosmosChannelHandlerCharacterizesOpenInfoLogFields(t *testing.T) {
	chainProcessor, logs := newCosmosChannelHandlerCharacterization()
	info := cosmosCharacterizedChannelInfo()
	chainProcessor.handleChannelMessage(
		chantypes.EventTypeChannelOpenConfirm,
		info,
		processor.NewIBCMessagesCache(),
	)

	fields := logs.FilterMessage("Successfully created new channel").All()[0].ContextMap()
	require.Equal(t, "cosmos-name", fields["chain_name"])
	require.Equal(t, "cosmos-1", fields["chain_id"])
	require.Equal(t, info.ChannelID, fields["channel_id"])
	require.Equal(t, info.ConnID, fields["connection_id"])
	require.Equal(t, info.PortID, fields["port_id"])
}

func newCosmosChannelHandlerCharacterization() (*CosmosChainProcessor, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)
	chainProvider := &CosmosProvider{PCfg: CosmosProviderConfig{ChainName: "cosmos-name", ChainID: "cosmos-1"}}
	return NewCosmosChainProcessor(zap.New(core), chainProvider, nil), logs
}

func cosmosCharacterizedChannelInfo() provider.ChannelInfo {
	return provider.ChannelInfo{
		Height:                42,
		PortID:                "transfer",
		ChannelID:             "channel-7",
		CounterpartyPortID:    "transfer-counterparty",
		CounterpartyChannelID: "channel-8",
		ConnID:                "connection-7",
		CounterpartyConnID:    "connection-8",
		Order:                 chantypes.ORDERED,
		Version:               "ics20-1",
	}
}

func assertCosmosChannelHandlerLogFields(
	t *testing.T,
	logs *observer.ObservedLogs,
	event string,
	info provider.ChannelInfo,
) {
	t.Helper()
	fields := logs.FilterMessage("Observed IBC message").All()[0].ContextMap()
	require.Equal(t, "cosmos-name", fields["chain_name"])
	require.Equal(t, "cosmos-1", fields["chain_id"])
	require.Equal(t, event, fields["event_type"])
	require.Equal(t, info.ChannelID, fields["channel_id"])
	require.Equal(t, info.PortID, fields["port_id"])
	require.Equal(t, info.CounterpartyChannelID, fields["counterparty_channel_id"])
	require.Equal(t, info.CounterpartyPortID, fields["counterparty_port_id"])
	require.Equal(t, info.ConnID, fields["connection_id"])
}
