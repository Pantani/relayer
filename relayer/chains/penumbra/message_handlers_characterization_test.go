package penumbra

import (
	"testing"

	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/cosmos/relayer/v2/relayer/chains"
	"github.com/cosmos/relayer/v2/relayer/processor"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestPenumbraChannelHandlerCharacterizesOpenLifecycleBranches(t *testing.T) {
	tests := []struct {
		event    string
		wantOpen bool
	}{
		{event: chantypes.EventTypeChannelOpenTry},
		{event: chantypes.EventTypeChannelOpenAck, wantOpen: true},
		{event: chantypes.EventTypeChannelOpenConfirm, wantOpen: true},
	}
	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			chainProcessor, logs := newPenumbraChannelHandlerCharacterization()
			cache := processor.NewIBCMessagesCache()
			info := penumbraCharacterizedChannelInfo()
			key := processor.ChannelInfoChannelKey(info)
			chainProcessor.channelStateCache.SetOpen(key.MsgInitKey(), false, chantypes.NONE)

			chainProcessor.handleChannelMessage(tt.event, info, cache)

			require.Equal(t, "connection-7", chainProcessor.channelConnections["channel-7"])
			require.NotContains(t, chainProcessor.channelStateCache, key.MsgInitKey())
			require.Equal(t, processor.ChannelState{Open: tt.wantOpen, Order: chantypes.ORDERED}, chainProcessor.channelStateCache[key])
			require.Equal(t, info, cache.ChannelHandshake[tt.event][key])
			require.Equal(t, 1, logs.FilterMessage("Observed IBC message").Len())
			require.Zero(t, logs.FilterMessage("Successfully created new channel").Len())
			assertPenumbraChannelHandlerLogFields(t, logs, tt.event, info)
		})
	}
}

func TestPenumbraChannelHandlerCharacterizesCloseLifecycleBranches(t *testing.T) {
	for _, event := range []string{
		chantypes.EventTypeChannelClosed,
		chantypes.EventTypeChannelCloseConfirm,
	} {
		t.Run(event, func(t *testing.T) {
			chainProcessor, logs := newPenumbraChannelHandlerCharacterization()
			cache := processor.NewIBCMessagesCache()
			info := penumbraCharacterizedChannelInfo()
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

func TestPenumbraChannelHandlerCharacterizesCloseWithoutMatchingState(t *testing.T) {
	chainProcessor, _ := newPenumbraChannelHandlerCharacterization()
	cache := processor.NewIBCMessagesCache()
	info := penumbraCharacterizedChannelInfo()
	key := processor.ChannelInfoChannelKey(info)
	unrelated := processor.ChannelKey{ChannelID: "channel-other", PortID: info.PortID}
	chainProcessor.channelStateCache.SetOpen(unrelated, true, chantypes.UNORDERED)

	chainProcessor.handleChannelMessage(chantypes.EventTypeChannelClosed, info, cache)

	require.Len(t, chainProcessor.channelStateCache, 1)
	require.NotContains(t, chainProcessor.channelStateCache, key)
	require.Equal(t, processor.ChannelState{Open: true, Order: chantypes.UNORDERED}, chainProcessor.channelStateCache[unrelated])
	require.Equal(t, info, cache.ChannelHandshake[chantypes.EventTypeChannelClosed][key])
}

func TestPenumbraChannelHandlerCharacterizesInitInsertionAndExistingFullDedup(t *testing.T) {
	chainProcessor, _ := newPenumbraChannelHandlerCharacterization()
	cache := processor.NewIBCMessagesCache()
	info := penumbraCharacterizedChannelInfo()
	info.CounterpartyChannelID = ""
	key := processor.ChannelInfoChannelKey(info)

	chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenInit, info, cache)
	require.Equal(t, processor.ChannelState{Open: false, Order: chantypes.ORDERED}, chainProcessor.channelStateCache[key])
	require.Equal(t, info, cache.ChannelHandshake[chantypes.EventTypeChannelOpenInit][key])

	fullInfo := penumbraCharacterizedChannelInfo()
	fullKey := processor.ChannelInfoChannelKey(fullInfo)
	chainProcessor.channelStateCache = make(processor.ChannelStateCache)
	chainProcessor.channelStateCache.SetOpen(fullKey, true, chantypes.UNORDERED)
	chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenInit, info, cache)

	require.Len(t, chainProcessor.channelStateCache, 1)
	require.NotContains(t, chainProcessor.channelStateCache, key)
	require.Equal(t, processor.ChannelState{Open: true, Order: chantypes.UNORDERED}, chainProcessor.channelStateCache[fullKey])
}

func TestPenumbraChannelHandlerCharacterizesUnhandledEventsDeleteInitAndRetain(t *testing.T) {
	for _, event := range []string{chantypes.EventTypeChannelCloseInit, "unknown-channel-event"} {
		t.Run(event, func(t *testing.T) {
			chainProcessor, logs := newPenumbraChannelHandlerCharacterization()
			cache := processor.NewIBCMessagesCache()
			info := penumbraCharacterizedChannelInfo()
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

func TestPenumbraChannelHandlerCharacterizesOverwriteAndStateOrderPreservation(t *testing.T) {
	chainProcessor, _ := newPenumbraChannelHandlerCharacterization()
	cache := processor.NewIBCMessagesCache()
	first := penumbraCharacterizedChannelInfo()
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

func TestPenumbraChannelHandlerCharacterizesZeroInfoAsValidMapKey(t *testing.T) {
	chainProcessor, logs := newPenumbraChannelHandlerCharacterization()
	cache := processor.NewIBCMessagesCache()
	zero := provider.ChannelInfo{}
	zeroKey := processor.ChannelKey{}

	chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenInit, zero, cache)

	require.Equal(t, "", chainProcessor.channelConnections[""])
	require.Equal(t, processor.ChannelState{Open: false, Order: chantypes.NONE}, chainProcessor.channelStateCache[zeroKey])
	require.Equal(t, zero, cache.ChannelHandshake[chantypes.EventTypeChannelOpenInit][zeroKey])
	assertPenumbraChannelHandlerLogFields(t, logs, chantypes.EventTypeChannelOpenInit, zero)
}

func TestPenumbraChannelHandlerCharacterizesNilHandshakeCachePartialEffects(t *testing.T) {
	chainProcessor, logs := newPenumbraChannelHandlerCharacterization()
	cache := processor.IBCMessagesCache{}
	info := penumbraCharacterizedChannelInfo()
	key := processor.ChannelInfoChannelKey(info)
	chainProcessor.channelStateCache.SetOpen(key.MsgInitKey(), false, chantypes.NONE)

	require.Panics(t, func() {
		chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenAck, info, cache)
	})

	require.Equal(t, "connection-7", chainProcessor.channelConnections["channel-7"])
	require.NotContains(t, chainProcessor.channelStateCache, key.MsgInitKey())
	require.Equal(t, processor.ChannelState{Open: true, Order: chantypes.ORDERED}, chainProcessor.channelStateCache[key])
	require.Nil(t, cache.ChannelHandshake)
	require.Zero(t, logs.Len())
}

func TestPenumbraChannelHandlerCharacterizesNilLoggerLateAckPanic(t *testing.T) {
	chainProcessor := &PenumbraChainProcessor{
		channelConnections: make(map[string]string),
		channelStateCache:  make(processor.ChannelStateCache),
	}
	cache := processor.NewIBCMessagesCache()
	info := penumbraCharacterizedChannelInfo()
	key := processor.ChannelInfoChannelKey(info)
	chainProcessor.channelStateCache.SetOpen(key.MsgInitKey(), false, chantypes.NONE)

	require.Panics(t, func() {
		chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenAck, info, cache)
	})

	require.Equal(t, "connection-7", chainProcessor.channelConnections["channel-7"])
	require.NotContains(t, chainProcessor.channelStateCache, key.MsgInitKey())
	require.Equal(t, processor.ChannelState{Open: true, Order: chantypes.ORDERED}, chainProcessor.channelStateCache[key])
	require.Equal(t, info, cache.ChannelHandshake[chantypes.EventTypeChannelOpenAck][key])
}

func TestPenumbraChannelHandlerCharacterizesNilChannelConnectionsPanicFirst(t *testing.T) {
	chainProcessor, logs := newPenumbraChannelHandlerCharacterization()
	chainProcessor.channelConnections = nil
	cache := processor.NewIBCMessagesCache()
	info := penumbraCharacterizedChannelInfo()

	require.Panics(t, func() {
		chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenInit, info, cache)
	})

	require.Empty(t, chainProcessor.channelStateCache)
	require.Empty(t, cache.ChannelHandshake)
	require.Zero(t, logs.Len())
}

func TestPenumbraChannelHandlerCharacterizesNilStateCacheAfterConnectionMutation(t *testing.T) {
	chainProcessor, logs := newPenumbraChannelHandlerCharacterization()
	chainProcessor.channelStateCache = nil
	cache := processor.NewIBCMessagesCache()
	info := penumbraCharacterizedChannelInfo()

	require.Panics(t, func() {
		chainProcessor.handleChannelMessage(chantypes.EventTypeChannelOpenInit, info, cache)
	})

	require.Equal(t, "connection-7", chainProcessor.channelConnections["channel-7"])
	require.Nil(t, chainProcessor.channelStateCache)
	require.Empty(t, cache.ChannelHandshake)
	require.Zero(t, logs.Len())
}

func TestPenumbraChannelHandlerCharacterizesNilAndTypedNilDispatch(t *testing.T) {
	chainProcessor, logs := newPenumbraChannelHandlerCharacterization()
	cache := processor.NewIBCMessagesCache()

	chainProcessor.handleMessage(chains.IbcMessage{Info: nil}, cache)
	require.Empty(t, chainProcessor.channelConnections)
	require.Empty(t, cache.ChannelHandshake)
	require.Zero(t, logs.Len())

	var typedNil *chains.ChannelInfo
	require.Panics(t, func() {
		chainProcessor.handleMessage(chains.IbcMessage{
			EventType: chantypes.EventTypeChannelOpenInit,
			Info:      typedNil,
		}, cache)
	})
	require.Empty(t, chainProcessor.channelConnections)
	require.Empty(t, cache.ChannelHandshake)
}

func newPenumbraChannelHandlerCharacterization() (*PenumbraChainProcessor, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)
	chainProvider := &PenumbraProvider{PCfg: PenumbraProviderConfig{ChainName: "penumbra-name", ChainID: "penumbra-1"}}
	return NewPenumbraChainProcessor(zap.New(core), chainProvider), logs
}

func penumbraCharacterizedChannelInfo() provider.ChannelInfo {
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

func assertPenumbraChannelHandlerLogFields(
	t *testing.T,
	logs *observer.ObservedLogs,
	event string,
	info provider.ChannelInfo,
) {
	t.Helper()
	fields := logs.FilterMessage("Observed IBC message").All()[0].ContextMap()
	require.Equal(t, "penumbra-name", fields["chain_name"])
	require.Equal(t, "penumbra-1", fields["chain_id"])
	require.Equal(t, event, fields["event_type"])
	require.Equal(t, info.ChannelID, fields["channel_id"])
	require.Equal(t, info.PortID, fields["port_id"])
	require.Equal(t, info.CounterpartyChannelID, fields["counterparty_channel_id"])
	require.Equal(t, info.CounterpartyPortID, fields["counterparty_port_id"])
	require.Equal(t, info.ConnID, fields["connection_id"])
}
