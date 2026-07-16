package mock

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cosmos/relayer/v2/relayer/processor"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	mockBackpressureRuns    = 32
	mockCacheDataQueueSize  = 100
	mockBackpressureTimeout = time.Second
)

func TestMockChainProcessorBackpressureStress(t *testing.T) {
	for run := 0; run < mockBackpressureRuns; run++ {
		runMockBackpressureCancellation(t, run)
	}
}

func runMockBackpressureCancellation(t *testing.T, run int) {
	ctx, cancel := context.WithCancel(t.Context())
	pp := newMockBackpressurePathProcessor(run)
	fillMockBackpressureQueue(t, ctx, pp, run)
	entered := make(chan struct{})
	result := make(chan error, 1)
	var once sync.Once
	mcp := &MockChainProcessor{
		log:     zap.NewNop(),
		chainID: mockBackpressureChainID(run),
		getMockMessages: func() []TransactionMessage {
			once.Do(func() { close(entered) })
			return nil
		},
	}
	mcp.SetPathProcessors(processor.PathProcessors{pp})

	go func() { result <- mcp.Run(ctx, 0, nil) }()
	waitForMockSignal(t, entered)
	cancel()
	require.NoError(t, receiveMockResult(t, result))
}

func newMockBackpressurePathProcessor(run int) *processor.PathProcessor {
	chainID := mockBackpressureChainID(run)
	pathEnd1 := processor.PathEnd{PathName: "mock-backpressure", ChainID: chainID, ClientID: "client-a"}
	pathEnd2 := processor.PathEnd{PathName: "mock-backpressure", ChainID: "counterparty", ClientID: "client-b"}
	return processor.NewPathProcessor(zap.NewNop(), pathEnd1, pathEnd2, nil, "", time.Hour, time.Hour, 1, 0, 0)
}

func fillMockBackpressureQueue(t *testing.T, ctx context.Context, pp *processor.PathProcessor, run int) {
	t.Helper()
	for i := 0; i < mockCacheDataQueueSize; i++ {
		data := processor.ChainProcessorCacheData{IBCMessagesCache: processor.NewIBCMessagesCache()}
		require.NoError(t, pp.HandleNewDataContext(ctx, mockBackpressureChainID(run), data))
	}
}

func mockBackpressureChainID(run int) string {
	return fmt.Sprintf("mock-chain-%d", run)
}

func waitForMockSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(mockBackpressureTimeout):
		t.Fatal("timed out waiting for mock query cycle")
	}
}

func receiveMockResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(mockBackpressureTimeout):
		t.Fatal("mock chain processor did not stop under backpressure")
		return nil
	}
}
