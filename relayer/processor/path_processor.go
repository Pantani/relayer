package processor

import (
	"context"
	"fmt"
	"time"

	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"go.uber.org/zap"
)

const (
	// durationErrorRetry determines how long to wait before retrying
	// in the case of failure to send transactions with IBC messages.
	durationErrorRetry = 5 * time.Second

	// Amount of time to wait when sending transactions before giving up
	// and continuing on. Messages will be retried later if they are still
	// relevant.
	messageSendTimeout = 60 * time.Second

	// Amount of time to wait for a proof to be queried before giving up.
	// The proof query will be retried later if the message still needs
	// to be relayed.
	packetProofQueryTimeout = 5 * time.Second

	// Amount of time to wait for interchain queries.
	interchainQueryTimeout = 60 * time.Second

	// Amount of time between flushes if the previous flush failed.
	flushFailureRetry = 5 * time.Second

	// If the message was assembled successfully, but sending the message failed,
	// how many blocks should pass before retrying.
	blocksToRetrySendAfter = 5

	// How many times to retry sending a message before giving up on it.
	maxMessageSendRetries = 5

	// How many times to retry sending a message if channel is not opened.
	maxMessageSendRetriesIfChannelNotOpen = 1

	// How many blocks of history to retain ibc headers in the cache for.
	ibcHeadersToCache = 10

	// How many blocks of history before determining that a query needs to be
	// made to retrieve the client consensus state in order to assemble a
	// MsgUpdateClient message.
	clientConsensusHeightUpdateThresholdBlocks = 2
)

// PathProcessor is a process that handles incoming IBC messages from a pair of chains.
// It determines what messages need to be relayed, and sends them.
type PathProcessor struct {
	log *zap.Logger

	pathEnd1 *pathEndRuntime
	pathEnd2 *pathEndRuntime

	memo string

	clientUpdateThresholdTime time.Duration

	messageLifecycle MessageLifecycle

	initialFlushComplete bool
	flushTimer           *time.Timer
	flushInterval        time.Duration

	// Signals to retry.
	retryProcess chan struct{}

	sentInitialMsg bool

	// true if this is a localhost IBC connection
	isLocalhost bool

	maxMsgs                    uint64
	memoLimit, maxReceiverSize int

	metrics *PrometheusMetrics
}

// PathProcessors is a slice of PathProcessor instances
type PathProcessors []*PathProcessor

func (p PathProcessors) IsRelayedChannel(k ChannelKey, chainID string) bool {
	for _, pp := range p {
		if pp.IsRelayedChannel(chainID, k) {
			return true
		}
	}
	return false
}

func NewPathProcessor(
	log *zap.Logger,
	pathEnd1 PathEnd,
	pathEnd2 PathEnd,
	metrics *PrometheusMetrics,
	memo string,
	clientUpdateThresholdTime time.Duration,
	flushInterval time.Duration,
	maxMsgs uint64,
	memoLimit, maxReceiverSize int,
) *PathProcessor {
	isLocalhost := pathEnd1.ClientID == ibcexported.LocalhostClientID

	pp := &PathProcessor{
		log:                       log,
		pathEnd1:                  newPathEndRuntime(log, pathEnd1, metrics),
		pathEnd2:                  newPathEndRuntime(log, pathEnd2, metrics),
		retryProcess:              make(chan struct{}, 2),
		memo:                      memo,
		clientUpdateThresholdTime: clientUpdateThresholdTime,
		flushInterval:             flushInterval,
		metrics:                   metrics,
		isLocalhost:               isLocalhost,
		maxMsgs:                   maxMsgs,
		memoLimit:                 memoLimit,
		maxReceiverSize:           maxReceiverSize,
	}
	if flushInterval == 0 {
		pp.disablePeriodicFlush()
	}
	return pp
}

// disablePeriodicFlush will "disable" periodic flushing by using a large value.
func (pp *PathProcessor) disablePeriodicFlush() {
	pp.flushInterval = 200 * 24 * 365 * time.Hour
}

func (pp *PathProcessor) SetMessageLifecycle(messageLifecycle MessageLifecycle) {
	pp.messageLifecycle = messageLifecycle
	if !pp.shouldFlush() {
		// disable flushing when termination conditions are set, e.g. connection/channel handshakes
		pp.disablePeriodicFlush()
	}
}

func (pp *PathProcessor) shouldFlush() bool {
	if pp.messageLifecycle == nil {
		return true
	}
	if _, ok := pp.messageLifecycle.(*FlushLifecycle); ok {
		return true
	}
	return false
}

// TEST USE ONLY
func (pp *PathProcessor) PathEnd1Messages(channelKey ChannelKey, message string) PacketSequenceCache {
	return pp.pathEnd1.messageCache.PacketFlow[channelKey][message]
}

// TEST USE ONLY
func (pp *PathProcessor) PathEnd2Messages(channelKey ChannelKey, message string) PacketSequenceCache {
	return pp.pathEnd2.messageCache.PacketFlow[channelKey][message]
}

type channelPair struct {
	pathEnd1ChannelKey ChannelKey
	pathEnd2ChannelKey ChannelKey
}

// RelevantClientID returns the relevant client ID or panics
func (pp *PathProcessor) RelevantClientID(chainID string) string {
	if pp.pathEnd1.info.ChainID == chainID {
		return pp.pathEnd1.info.ClientID
	}
	if pp.pathEnd2.info.ChainID == chainID {
		return pp.pathEnd2.info.ClientID
	}
	panic(fmt.Errorf("no relevant client ID for chain ID: %s", chainID))
}

// OnConnectionMessage allows the caller to handle connection handshake messages with a callback.
func (pp *PathProcessor) OnConnectionMessage(chainID string, eventType string, onMsg func(provider.ConnectionInfo)) {
	if pp.pathEnd1.info.ChainID == chainID {
		pp.pathEnd1.connSubscribers[eventType] = append(pp.pathEnd1.connSubscribers[eventType], onMsg)
	} else if pp.pathEnd2.info.ChainID == chainID {
		pp.pathEnd2.connSubscribers[eventType] = append(pp.pathEnd2.connSubscribers[eventType], onMsg)
	}
}

func (pp *PathProcessor) channelPairs() []channelPair {
	// Channel keys are from pathEnd1's perspective
	channels := make(map[ChannelKey]ChannelState)
	for k, cs := range pp.pathEnd1.channelStateCache {
		channels[k] = cs
	}
	for k, cs := range pp.pathEnd2.channelStateCache {
		channels[k.Counterparty()] = cs
	}
	pairs := make([]channelPair, len(channels))
	i := 0
	for k := range channels {
		pairs[i] = channelPair{
			pathEnd1ChannelKey: k,
			pathEnd2ChannelKey: k.Counterparty(),
		}
		i++
	}
	return pairs
}

// Path Processors are constructed before ChainProcessors, so reference needs to be added afterwards
// This can be done inside the ChainProcessor constructor for simplification
func (pp *PathProcessor) SetChainProviderIfApplicable(chainProvider provider.ChainProvider) bool {
	if chainProvider == nil {
		return false
	}
	if pp.pathEnd1.info.ChainID == chainProvider.ChainId() {
		pp.pathEnd1.chainProvider = chainProvider

		if pp.isLocalhost {
			pp.pathEnd2.chainProvider = chainProvider
		}

		return true
	} else if pp.pathEnd2.info.ChainID == chainProvider.ChainId() {
		pp.pathEnd2.chainProvider = chainProvider

		if pp.isLocalhost {
			pp.pathEnd1.chainProvider = chainProvider
		}

		return true
	}
	return false
}

func (pp *PathProcessor) IsRelayedChannel(chainID string, channelKey ChannelKey) bool {
	if pp.pathEnd1.info.ChainID == chainID {
		return pp.pathEnd1.ShouldRelayChannel(ChainChannelKey{ChainID: chainID, CounterpartyChainID: pp.pathEnd2.info.ChainID, ChannelKey: channelKey})
	} else if pp.pathEnd2.info.ChainID == chainID {
		return pp.pathEnd2.ShouldRelayChannel(ChainChannelKey{ChainID: chainID, CounterpartyChainID: pp.pathEnd1.info.ChainID, ChannelKey: channelKey})
	}
	return false
}

func (pp *PathProcessor) IsRelevantClient(chainID string, clientID string) bool {
	if pp.pathEnd1.info.ChainID == chainID {
		return pp.pathEnd1.info.ClientID == clientID
	} else if pp.pathEnd2.info.ChainID == chainID {
		return pp.pathEnd2.info.ClientID == clientID
	}
	return false
}

func (pp *PathProcessor) IsRelevantConnection(chainID string, connectionID string) bool {
	if pp.pathEnd1.info.ChainID == chainID {
		return pp.pathEnd1.isRelevantConnection(connectionID)
	} else if pp.pathEnd2.info.ChainID == chainID {
		return pp.pathEnd2.isRelevantConnection(connectionID)
	}
	return false
}

func (pp *PathProcessor) IsRelevantChannel(chainID string, channelID string) bool {
	if pp.pathEnd1.info.ChainID == chainID {
		return pp.pathEnd1.isRelevantChannel(channelID)
	} else if pp.pathEnd2.info.ChainID == chainID {
		return pp.pathEnd2.isRelevantChannel(channelID)
	}
	return false
}

// ProcessBacklogIfReady gives ChainProcessors a way to trigger the path processor process
// as soon as they are in sync for the first time, even if they do not have new messages.
func (pp *PathProcessor) ProcessBacklogIfReady() {
	select {
	case pp.retryProcess <- struct{}{}:
		// All good.
	default:
		// Log that the channel is saturated;
		// something is wrong if we are retrying this quickly.
		pp.log.Error("Failed to enqueue path processor retry, retries already scheduled")
	}
}

// HandleNewData preserves the original blocking enqueue contract.
// Context-aware producers should use HandleNewDataContext.
func (pp *PathProcessor) HandleNewData(chainID string, cacheData ChainProcessorCacheData) {
	_ = pp.HandleNewDataContext(context.Background(), chainID, cacheData)
}

// HandleNewDataContext blocks until the data is enqueued or ctx is canceled.
func (pp *PathProcessor) HandleNewDataContext(
	ctx context.Context,
	chainID string,
	cacheData ChainProcessorCacheData,
) error {
	if pp.isLocalhost {
		return pp.handleLocalhostData(ctx, cacheData)
	}

	if pp.pathEnd1.info.ChainID == chainID {
		return enqueueCacheData(ctx, pp.pathEnd1.incomingCacheData, cacheData)
	}
	if pp.pathEnd2.info.ChainID == chainID {
		return enqueueCacheData(ctx, pp.pathEnd2.incomingCacheData, cacheData)
	}
	return nil
}

func enqueueCacheData(
	ctx context.Context,
	queue chan<- ChainProcessorCacheData,
	cacheData ChainProcessorCacheData,
) error {
	select {
	case queue <- cacheData:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (pp *PathProcessor) handleFlush(ctx context.Context) {
	flushTimer := pp.flushInterval
	if err := pp.flush(ctx); err != nil {
		pp.log.Warn("Flush not complete",
			zap.String("chain_id_1", pp.pathEnd1.chainProvider.ChainId()),
			zap.String("client_id_1", pp.pathEnd1.info.ClientID),
			zap.String("chain_id_2", pp.pathEnd2.chainProvider.ChainId()),
			zap.String("client_id_2", pp.pathEnd2.info.ClientID),
			zap.Error(err))
		flushTimer = flushFailureRetry
	}
	pp.flushTimer.Stop()
	pp.flushTimer = time.NewTimer(flushTimer)
}

// processAvailableSignals will block if signals are not yet available, otherwise it will process one of the available signals.
// It returns whether or not the pathProcessor should quit.
func (pp *PathProcessor) processAvailableSignals(ctx context.Context, cancel func()) bool {
	select {
	case <-ctx.Done():
		pp.log.Debug("Context done, quitting PathProcessor",
			zap.String("chain_id_1", pp.pathEnd1.info.ChainID),
			zap.String("chain_id_2", pp.pathEnd2.info.ChainID),
			zap.String("client_id_1", pp.pathEnd1.info.ClientID),
			zap.String("client_id_2", pp.pathEnd2.info.ClientID),
			zap.Error(ctx.Err()),
		)
		return true
	case t := <-pp.pathEnd1.finishedProcessing:
		pp.pathEnd1.trackFinishedProcessingMessage(t)
	case t := <-pp.pathEnd2.finishedProcessing:
		pp.pathEnd2.trackFinishedProcessingMessage(t)
	case d := <-pp.pathEnd1.incomingCacheData:
		// we have new data from ChainProcessor for pathEnd1
		pp.pathEnd1.mergeCacheData(
			ctx,
			cancel,
			d,
			pp.pathEnd2.info.ChainID,
			pp.pathEnd2.inSync,
			pp.messageLifecycle,
			pp.pathEnd2,
			pp.memoLimit,
			pp.maxReceiverSize,
		)

	case d := <-pp.pathEnd2.incomingCacheData:
		// we have new data from ChainProcessor for pathEnd2
		pp.pathEnd2.mergeCacheData(
			ctx,
			cancel,
			d,
			pp.pathEnd1.info.ChainID,
			pp.pathEnd1.inSync,
			pp.messageLifecycle,
			pp.pathEnd1,
			pp.memoLimit,
			pp.maxReceiverSize,
		)

	case <-pp.retryProcess:
		// No new data to merge in, just retry handling.
	case <-pp.flushTimer.C:
		for len(pp.pathEnd1.incomingCacheData) > 0 {
			d := <-pp.pathEnd1.incomingCacheData
			// we have new data from ChainProcessor for pathEnd1
			pp.pathEnd1.mergeCacheData(
				ctx,
				cancel,
				d,
				pp.pathEnd2.info.ChainID,
				pp.pathEnd2.inSync,
				pp.messageLifecycle,
				pp.pathEnd2,
				pp.memoLimit,
				pp.maxReceiverSize,
			)
		}
		for len(pp.pathEnd2.incomingCacheData) > 0 {
			d := <-pp.pathEnd2.incomingCacheData
			// we have new data from ChainProcessor for pathEnd2
			pp.pathEnd2.mergeCacheData(
				ctx,
				cancel,
				d,
				pp.pathEnd1.info.ChainID,
				pp.pathEnd1.inSync,
				pp.messageLifecycle,
				pp.pathEnd1,
				pp.memoLimit,
				pp.maxReceiverSize,
			)
		}
		// Periodic flush to clear out any old packets
		pp.handleFlush(ctx)
	}
	return false
}

// Run executes the main path process.
func (pp *PathProcessor) Run(ctx context.Context, cancel func()) {
	var retryTimer *time.Timer

	pp.flushTimer = time.NewTimer(time.Hour)

	for {
		// block until we have any signals to process
		if pp.processAvailableSignals(ctx, cancel) {
			return
		}

		for len(pp.pathEnd1.incomingCacheData) > 0 || len(pp.pathEnd2.incomingCacheData) > 0 || len(pp.retryProcess) > 0 || len(pp.pathEnd1.finishedProcessing) > 0 || len(pp.pathEnd2.finishedProcessing) > 0 {
			// signals are available, so this will not need to block.
			if pp.processAvailableSignals(ctx, cancel) {
				return
			}
		}

		if !pp.pathEnd1.inSync || !pp.pathEnd2.inSync {
			continue
		}

		if pp.shouldFlush() && !pp.initialFlushComplete {
			pp.handleFlush(ctx)
			pp.initialFlushComplete = true
		} else if pp.shouldTerminateForFlushComplete() {
			cancel()
			return
		}

		// process latest message cache state from both pathEnds
		if err := pp.processLatestMessages(ctx, cancel); err != nil {
			// in case of IBC message send errors, schedule retry after durationErrorRetry
			if retryTimer != nil {
				retryTimer.Stop()
			}
			if ctx.Err() == nil {
				retryTimer = time.AfterFunc(durationErrorRetry, pp.ProcessBacklogIfReady)
			}
		}
	}
}

func (pp *PathProcessor) handleLocalhostData(
	ctx context.Context,
	cacheData ChainProcessorCacheData,
) error {
	pathEnd1Cache, pathEnd2Cache := pp.splitLocalhostData(cacheData)
	if err := enqueueCacheData(ctx, pp.pathEnd1.incomingCacheData, pathEnd1Cache); err != nil {
		return err
	}
	return enqueueCacheData(ctx, pp.pathEnd2.incomingCacheData, pathEnd2Cache)
}

func (pp *PathProcessor) splitLocalhostData(
	cacheData ChainProcessorCacheData,
) (ChainProcessorCacheData, ChainProcessorCacheData) {
	pathEnd1Cache := newLocalhostCacheData(cacheData)
	pathEnd2Cache := newLocalhostCacheData(cacheData)
	pp.splitLocalhostPackets(cacheData.IBCMessagesCache.PacketFlow, &pathEnd1Cache, &pathEnd2Cache)
	pp.splitLocalhostHandshakes(cacheData.IBCMessagesCache.ChannelHandshake, &pathEnd1Cache, &pathEnd2Cache)
	pathEnd1Cache.ChannelStateCache, pathEnd2Cache.ChannelStateCache = splitLocalhostChannelStates(cacheData.ChannelStateCache)
	return pathEnd1Cache, pathEnd2Cache
}

func newLocalhostCacheData(cacheData ChainProcessorCacheData) ChainProcessorCacheData {
	return ChainProcessorCacheData{
		IBCMessagesCache:     NewIBCMessagesCache(),
		InSync:               cacheData.InSync,
		ClientState:          cacheData.ClientState,
		ConnectionStateCache: cacheData.ConnectionStateCache,
		ChannelStateCache:    cacheData.ChannelStateCache,
		LatestBlock:          cacheData.LatestBlock,
		LatestHeader:         cacheData.LatestHeader,
		IBCHeaderCache:       cacheData.IBCHeaderCache,
	}
}

func (pp *PathProcessor) splitLocalhostPackets(
	packetFlow ChannelPacketMessagesCache,
	pathEnd1Cache, pathEnd2Cache *ChainProcessorCacheData,
) {
	// split up data and send lower channel-id data to pathEnd1 and higher channel-id data to pathEnd2.
	for k, v := range packetFlow {
		chan1, err := chantypes.ParseChannelSequence(k.ChannelID)
		if err != nil {
			pp.log.Error("Failed to parse channel ID int from string", zap.Error(err))
			continue
		}

		chan2, err := chantypes.ParseChannelSequence(k.CounterpartyChannelID)
		if err != nil {
			pp.log.Error("Failed to parse channel ID int from string", zap.Error(err))
			continue
		}

		if chan1 < chan2 {
			pathEnd1Cache.IBCMessagesCache.PacketFlow[k] = v
		} else {
			pathEnd2Cache.IBCMessagesCache.PacketFlow[k] = v
		}
	}
}

func (pp *PathProcessor) splitLocalhostHandshakes(
	handshakes ChannelMessagesCache,
	pathEnd1Cache, pathEnd2Cache *ChainProcessorCacheData,
) {
	for eventType, c := range handshakes {
		for k, v := range c {
			pp.splitLocalhostHandshake(eventType, k, v, pathEnd1Cache, pathEnd2Cache)
		}
	}
}

func (pp *PathProcessor) splitLocalhostHandshake(
	eventType string,
	key ChannelKey,
	info provider.ChannelInfo,
	pathEnd1Cache, pathEnd2Cache *ChainProcessorCacheData,
) {
	switch eventType {
	case chantypes.EventTypeChannelOpenInit, chantypes.EventTypeChannelOpenAck, chantypes.EventTypeChannelCloseInit:
		pp.addLocalhostPathEnd1Handshake(eventType, key, info, pathEnd1Cache)
	case chantypes.EventTypeChannelOpenTry, chantypes.EventTypeChannelOpenConfirm, chantypes.EventTypeChannelCloseConfirm:
		pp.addLocalhostPathEnd2Handshake(eventType, key, info, pathEnd2Cache)
	default:
		pp.log.Error("Invalid IBC channel event type", zap.String("event_type", eventType))
	}
}

func (pp *PathProcessor) addLocalhostPathEnd1Handshake(
	eventType string,
	key ChannelKey,
	info provider.ChannelInfo,
	cacheData *ChainProcessorCacheData,
) {
	cache := cacheData.IBCMessagesCache.ChannelHandshake
	ensureChannelMessageCache(cache, eventType)
	info.Order = pp.localhostPathEnd1Order(key, info.Order)
	cache[eventType][key] = info
}

func (pp *PathProcessor) addLocalhostPathEnd2Handshake(
	eventType string,
	key ChannelKey,
	info provider.ChannelInfo,
	cacheData *ChainProcessorCacheData,
) {
	cache := cacheData.IBCMessagesCache.ChannelHandshake
	ensureChannelMessageCache(cache, eventType)
	info.Order = pp.localhostPathEnd2Order(key, info.Order)
	cache[eventType][key] = info
}

func ensureChannelMessageCache(cache ChannelMessagesCache, eventType string) {
	if _, ok := cache[eventType]; !ok {
		cache[eventType] = make(ChannelMessageCache)
	}
}

func (pp *PathProcessor) localhostPathEnd1Order(key ChannelKey, order chantypes.Order) chantypes.Order {
	if cached, ok := pp.pathEnd1.channelOrderCache[key.ChannelID]; ok {
		order = cached
	}
	if cached, ok := pp.pathEnd2.channelOrderCache[key.CounterpartyChannelID]; ok {
		order = cached
	}
	// TODO this is insanely hacky, need to figure out how to handle the ordering dilemma on ordered chans
	if order == chantypes.NONE {
		return chantypes.ORDERED
	}
	return order
}

func (pp *PathProcessor) localhostPathEnd2Order(key ChannelKey, order chantypes.Order) chantypes.Order {
	if cached, ok := pp.pathEnd2.channelOrderCache[key.ChannelID]; ok {
		order = cached
	}
	if cached, ok := pp.pathEnd1.channelOrderCache[key.CounterpartyChannelID]; ok {
		order = cached
	}
	return order
}

func splitLocalhostChannelStates(states ChannelStateCache) (ChannelStateCache, ChannelStateCache) {
	pathEnd1States := make(ChannelStateCache)
	pathEnd2States := make(ChannelStateCache)

	for k, v := range states {
		splitLocalhostChannelState(k, v, pathEnd1States, pathEnd2States)
	}
	return pathEnd1States, pathEnd2States
}

func splitLocalhostChannelState(
	key ChannelKey,
	state ChannelState,
	pathEnd1States, pathEnd2States ChannelStateCache,
) {
	chan1, err := chantypes.ParseChannelSequence(key.ChannelID)
	chan2, secErr := chantypes.ParseChannelSequence(key.CounterpartyChannelID)
	if err != nil && secErr != nil {
		return
	}
	// A missing counterparty ID means the handshake has not progressed past TRY.
	if secErr != nil && err == nil {
		pathEnd1States[key] = state
		return
	}
	// Lower channel IDs belong to pathEnd1; higher IDs belong to pathEnd2.
	if chan1 > chan2 {
		pathEnd2States[key] = state
		return
	}
	pathEnd1States[key] = state
}
