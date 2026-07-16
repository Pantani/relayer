package mock

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
	"github.com/cosmos/relayer/v2/relayer/processor"
	"github.com/cosmos/relayer/v2/relayer/provider"

	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"go.uber.org/zap"
)

const (
	minQueryLoopDuration     = 1 * time.Second
	inSyncNumBlocksThreshold = 2
)

type MockChainProcessor struct {
	log *zap.Logger

	chainID string

	// subscribers to this chain processor, where relevant IBC messages will be published
	pathProcessors []*processor.PathProcessor

	// indicates whether queries are in sync with latest height of the chain
	inSync bool

	getMockMessages func() []TransactionMessage

	chainProvider provider.ChainProvider
}

// types used for parsing IBC messages from transactions, then passed to message handlers for mutating the MockChainProcessor state if necessary and retaining applicable messages for sending to the Path Processors
type TransactionMessage struct {
	EventType  string
	PacketInfo *chantypes.Packet
}

func NewMockChainProcessor(ctx context.Context, log *zap.Logger, chainID string, getMockMessages func() []TransactionMessage) *MockChainProcessor {
	chainProviderCfg := cosmos.CosmosProviderConfig{
		Key:            "mock-key",
		ChainID:        chainID,
		AccountPrefix:  "mock",
		KeyringBackend: "test",
		Timeout:        "10s",
	}
	chainProvider, _ := chainProviderCfg.NewProvider(zap.NewNop(), "/tmp", true, "mock-chain-name-"+chainID)
	_ = chainProvider.Init(ctx)
	_, _ = chainProvider.AddKey(chainProvider.Key(), 118, string(hd.Secp256k1Type))
	return &MockChainProcessor{
		log:             log,
		chainID:         chainID,
		getMockMessages: getMockMessages,
		chainProvider:   chainProvider,
	}
}

func (mcp *MockChainProcessor) SetPathProcessors(pathProcessors processor.PathProcessors) {
	mcp.pathProcessors = pathProcessors
}

// Provider returns the ChainProvider, which provides the methods for querying, assembling IBC messages, and sending transactions.
func (mcp *MockChainProcessor) Provider() provider.ChainProvider {
	return mcp.chainProvider
}

type queryCyclePersistence struct {
	latestHeight       int64
	latestQueriedBlock int64
}

func (mcp *MockChainProcessor) Run(ctx context.Context, initialBlockHistory uint64, _ *processor.StuckPacket) error {
	// this will be used for persistence across query cycle loop executions
	persistence := queryCyclePersistence{
		// would be query of latest height, mocking 20
		latestHeight: 20,
	}

	// this will make initial QueryLoop iteration look back initialBlockHistory blocks in history
	latestQueriedBlock := persistence.latestHeight - int64(initialBlockHistory)

	if latestQueriedBlock < 0 {
		persistence.latestQueriedBlock = 0
	} else {
		persistence.latestQueriedBlock = latestQueriedBlock
	}

	mcp.log.Info("entering main query loop", zap.String("chain_id", mcp.chainID))

	ticker := time.NewTicker(minQueryLoopDuration)
	defer ticker.Stop()

	// QueryLoop:
	for {
		mcp.queryCycle(ctx, &persistence)
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// minQueryLoopDuration never changes for MockChainProcessor, but it will for CosmosChainProcessor, so mocking that behavior
			ticker.Reset(minQueryLoopDuration)
		}
	}
}

func (mcp *MockChainProcessor) queryCycle(ctx context.Context, persistence *queryCyclePersistence) {
	persistence.latestHeight++
	mcp.updateSyncState(persistence)
	mcp.log.Debug("queried latest height",
		zap.String("chain_id", mcp.chainID),
		zap.Int64("latest_height", persistence.latestHeight),
	)

	for i := persistence.latestQueriedBlock + 1; i <= persistence.latestHeight; i++ {
		if !mcp.publishCacheData(ctx, mcp.cacheDataForBlock(i)) {
			return
		}
		persistence.latestQueriedBlock = i
	}
}

func (mcp *MockChainProcessor) updateSyncState(persistence *queryCyclePersistence) {
	if !mcp.inSync {
		if (persistence.latestHeight - persistence.latestQueriedBlock) < inSyncNumBlocksThreshold {
			mcp.inSync = true
			mcp.log.Info("chain is in sync", zap.String("chain_id", mcp.chainID))
		} else {
			mcp.log.Warn("chain is not yet in sync",
				zap.String("chain_id", mcp.chainID),
				zap.Int64("latest_queried_block", persistence.latestQueriedBlock),
				zap.Int64("latest_height", persistence.latestHeight),
			)
		}
	}
}

func (mcp *MockChainProcessor) cacheDataForBlock(height int64) processor.ChainProcessorCacheData {
	ibcMessagesCache := processor.NewIBCMessagesCache()
	for _, message := range mcp.getMockMessages() {
		if handler, ok := messageHandlers[message.EventType]; ok {
			handler(msgHandlerParams{
				height:           height,
				mcp:              mcp,
				packetInfo:       message.PacketInfo,
				ibcMessagesCache: ibcMessagesCache,
			})
		}
	}
	channelStateCache := make(processor.ChannelStateCache)
	for channelKey := range ibcMessagesCache.PacketFlow {
		channelStateCache.SetOpen(channelKey, true, chantypes.NONE)
	}
	return processor.ChainProcessorCacheData{
		LatestBlock: provider.LatestBlock{
			Height: uint64(height),
			Time:   time.Now(),
		},
		IBCMessagesCache:  ibcMessagesCache,
		InSync:            mcp.inSync,
		ChannelStateCache: channelStateCache,
	}
}

func (mcp *MockChainProcessor) publishCacheData(
	ctx context.Context,
	cacheData processor.ChainProcessorCacheData,
) bool {
	for _, pp := range mcp.pathProcessors {
		mcp.log.Info("sending messages to path processor", zap.String("chain_id", mcp.chainID))
		if err := pp.HandleNewDataContext(ctx, mcp.chainID, cacheData); err != nil {
			return false
		}
	}
	return true
}
