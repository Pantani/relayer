package processor

import (
	"context"
	"errors"
	"testing"
	"time"

	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const backpressureTestTimeout = time.Second

func TestHandleNewDataContextCanceledWhenQueueIsFull(t *testing.T) {
	pp := newBackpressurePathProcessor(false)
	fillCacheDataQueue(pp.pathEnd1.incomingCacheData)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := pp.HandleNewDataContext(ctx, pp.pathEnd1.info.ChainID, cacheDataAt(101))

	require.ErrorIs(t, err, context.Canceled)
	require.Len(t, pp.pathEnd1.incomingCacheData, cap(pp.pathEnd1.incomingCacheData))
}

func TestHandleNewDataContextEnqueuesWithoutLoss(t *testing.T) {
	pp := newBackpressurePathProcessor(false)
	want := cacheDataAt(42)

	require.NoError(t, pp.HandleNewDataContext(t.Context(), pp.pathEnd1.info.ChainID, want))

	require.Equal(t, want, receiveCacheData(t, pp.pathEnd1.incomingCacheData))
}

func TestHandleNewDataContextCancelsLocalhostBetweenEnqueues(t *testing.T) {
	pp := newBackpressurePathProcessor(true)
	fillCacheDataQueue(pp.pathEnd2.incomingCacheData)
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)

	go func() {
		result <- pp.HandleNewDataContext(ctx, pp.pathEnd1.info.ChainID, cacheDataAt(77))
	}()

	first := receiveCacheData(t, pp.pathEnd1.incomingCacheData)
	cancel()
	require.Equal(t, uint64(77), first.LatestBlock.Height)
	require.ErrorIs(t, receiveError(t, result), context.Canceled)
	require.Len(t, pp.pathEnd2.incomingCacheData, cap(pp.pathEnd2.incomingCacheData))
}

func TestEventProcessorRunReturnsUnderBackpressure(t *testing.T) {
	pp := newBackpressurePathProcessor(false)
	fillCacheDataQueue(pp.pathEnd1.incomingCacheData)
	cp := &backpressureChainProcessor{pp: pp, started: make(chan struct{})}
	ep := EventProcessor{chainProcessors: ChainProcessors{cp}}
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)

	go func() { result <- ep.Run(ctx) }()
	waitForSignal(t, cp.started)
	cancel()
	require.NoError(t, receiveError(t, result))
}

type backpressureChainProcessor struct {
	pp      *PathProcessor
	started chan struct{}
}

func (cp *backpressureChainProcessor) Run(ctx context.Context, _ uint64, _ *StuckPacket) error {
	close(cp.started)
	err := cp.pp.HandleNewDataContext(ctx, cp.pp.pathEnd1.info.ChainID, cacheDataAt(101))
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func (*backpressureChainProcessor) Provider() provider.ChainProvider {
	return nil
}

func (*backpressureChainProcessor) SetPathProcessors(PathProcessors) {}

func newBackpressurePathProcessor(localhost bool) *PathProcessor {
	clientID := "client-a"
	if localhost {
		clientID = ibcexported.LocalhostClientID
	}
	pathEnd1 := PathEnd{PathName: "backpressure", ChainID: "chain-a", ClientID: clientID}
	pathEnd2 := PathEnd{PathName: "backpressure", ChainID: "chain-b", ClientID: "client-b"}
	return NewPathProcessor(zap.NewNop(), pathEnd1, pathEnd2, nil, "", time.Hour, time.Hour, 1, 0, 0)
}

func cacheDataAt(height uint64) ChainProcessorCacheData {
	return ChainProcessorCacheData{
		IBCMessagesCache: NewIBCMessagesCache(),
		LatestBlock:      provider.LatestBlock{Height: height},
	}
}

func fillCacheDataQueue(queue chan<- ChainProcessorCacheData) {
	for i := 0; i < cap(queue); i++ {
		queue <- cacheDataAt(uint64(i))
	}
}

func receiveCacheData(t *testing.T, queue <-chan ChainProcessorCacheData) ChainProcessorCacheData {
	t.Helper()
	select {
	case data := <-queue:
		return data
	case <-time.After(backpressureTestTimeout):
		t.Fatal("timed out waiting for cache data")
		return ChainProcessorCacheData{}
	}
}

func receiveError(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(backpressureTestTimeout):
		t.Fatal("timed out waiting for result")
		return nil
	}
}

func waitForSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(backpressureTestTimeout):
		t.Fatal("timed out waiting for signal")
	}
}
