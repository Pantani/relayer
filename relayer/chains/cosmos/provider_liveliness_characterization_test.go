package cosmos

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestCosmosLivelinessCharacterizesCanceledInventoryAndDuplicates(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	provider := &CosmosProvider{
		log: zap.New(core),
		PCfg: CosmosProviderConfig{
			ChainName:      "cosmos-name",
			RPCAddr:        "",
			BackupRPCAddrs: []string{"rpc-backup", "rpc-backup"},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider.startLivelinessChecks(ctx, 25*time.Millisecond)

	require.Equal(t, []string{"Available RPC clients"}, cosmosLivelinessMessages(logs))
	require.Equal(t, map[string]any{
		"chain": "cosmos-name",
		"count": int64(3),
	}, logs.All()[0].ContextMap())
}

func TestCosmosLivelinessCharacterizesCancellationStopsGoroutine(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	provider := &CosmosProvider{
		log:  zap.New(core),
		PCfg: CosmosProviderConfig{ChainName: "cosmos-name", RPCAddr: "rpc-primary"},
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		provider.startLivelinessChecks(ctx, time.Millisecond)
		close(done)
	}()

	require.Eventually(t, func() bool {
		return logs.Len() == 1
	}, time.Second, time.Millisecond)
	cancel()
	requireCosmosLivelinessDone(t, done)
	require.Equal(t, []string{"Available RPC clients"}, cosmosLivelinessMessages(logs))
}

func TestCosmosLivelinessCharacterizesNilLoggerPanicBeforeCancellation(t *testing.T) {
	provider := &CosmosProvider{PCfg: CosmosProviderConfig{RPCAddr: "rpc-primary"}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.Panics(t, func() {
		provider.startLivelinessChecks(ctx, time.Millisecond)
	})
}

func TestCosmosLivelinessCharacterizesNilContextPanicAfterInventoryLog(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	provider := &CosmosProvider{
		log:  zap.New(core),
		PCfg: CosmosProviderConfig{RPCAddr: "rpc-primary"},
	}

	require.Panics(t, func() {
		provider.startLivelinessChecks(nil, time.Millisecond)
	})
	require.Equal(t, []string{"Available RPC clients"}, cosmosLivelinessMessages(logs))
}

func TestCosmosLivelinessCharacterizesHealthyTickWithoutRotation(t *testing.T) {
	t.Parallel()
	primary, primaryRequests := cosmosLivelinessRPCServer(t, false)
	backup, backupRequests := cosmosLivelinessRPCServer(t, false)
	core, logs := observer.New(zap.DebugLevel)
	provider := &CosmosProvider{
		log: zap.New(core),
		PCfg: CosmosProviderConfig{
			ChainID:        "chain-a-1",
			ChainName:      "cosmos-name",
			RPCAddr:        primary.URL,
			BackupRPCAddrs: []string{backup.URL},
		},
	}
	require.NoError(t, provider.setRpcClient(true, primary.URL, 250*time.Millisecond))
	require.NoError(t, provider.setLightProvider(primary.URL))
	initialLightProvider := provider.LightProvider
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	started := time.Now()
	go func() {
		provider.startLivelinessChecks(ctx, 250*time.Millisecond)
		close(done)
	}()

	require.Eventually(t, func() bool {
		return primaryRequests.Load() == 1
	}, 13*time.Second, 10*time.Millisecond)
	cancel()
	requireCosmosLivelinessDone(t, done)
	require.GreaterOrEqual(t, time.Since(started), 10*time.Second)
	require.Equal(t, int32(0), backupRequests.Load())
	require.Equal(t, []string{"Available RPC clients"}, cosmosLivelinessMessages(logs))
	require.Equal(t, initialLightProvider, provider.LightProvider)
	require.Equal(t, primary.URL, provider.PCfg.RPCAddr)
}

func TestCosmosLivelinessCharacterizesTickRotationOrderAndClientSwap(t *testing.T) {
	t.Parallel()
	current, currentRequests := cosmosLivelinessRPCServer(t, true)
	backup, backupRequests := cosmosLivelinessRPCServer(t, false)
	core, logs := observer.New(zap.DebugLevel)
	provider := &CosmosProvider{
		log: zap.New(core),
		PCfg: CosmosProviderConfig{
			ChainID:        "chain-a-1",
			ChainName:      "cosmos-name",
			RPCAddr:        current.URL,
			BackupRPCAddrs: []string{current.URL, backup.URL},
		},
	}
	require.NoError(t, provider.setRpcClient(true, current.URL, 250*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	started := time.Now()
	go func() {
		provider.startLivelinessChecks(ctx, 250*time.Millisecond)
		close(done)
	}()

	require.Eventually(t, func() bool {
		return cosmosLivelinessHasMessage(logs, "Successfully connected to new RPC")
	}, 13*time.Second, 10*time.Millisecond)
	cancel()
	requireCosmosLivelinessDone(t, done)
	require.GreaterOrEqual(t, time.Since(started), 10*time.Second)
	require.Equal(t, []string{current.URL, current.URL, backup.URL}, cosmosLivelinessAttemptRPCs(logs))
	disconnected := logs.FilterMessage("RPC client disconnected").All()[0].ContextMap()
	require.Equal(t, "cosmos-name", disconnected["chain"])
	require.Contains(t, disconnected["error"], "status unavailable")
	connected := logs.FilterMessage("Successfully connected to new RPC").All()[0].ContextMap()
	require.Equal(t, "cosmos-name", connected["chain"])
	require.Equal(t, backup.URL, connected["rpc"])
	require.Equal(t, int32(3), currentRequests.Load())
	require.Equal(t, int32(1), backupRequests.Load())
	require.Equal(t, current.URL, provider.PCfg.RPCAddr)
	require.Equal(t, []string{current.URL, backup.URL}, provider.PCfg.BackupRPCAddrs)
	require.NotNil(t, provider.LightProvider)
	_, err := provider.ConsensusClient.GetStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, int32(2), backupRequests.Load())
	require.Equal(t, 2, cosmosLivelinessLevelCount(logs, zap.ErrorLevel, "Failed to connect to RPC client"))
}

func TestCosmosLivelinessCharacterizesAllFailPartialClientMutation(t *testing.T) {
	t.Parallel()
	current, currentRequests := cosmosLivelinessRPCServer(t, true)
	last, lastRequests := cosmosLivelinessRPCServer(t, true)
	core, logs := observer.New(zap.DebugLevel)
	provider := &CosmosProvider{
		log: zap.New(core),
		PCfg: CosmosProviderConfig{
			ChainID:        "chain-a-1",
			ChainName:      "cosmos-name",
			RPCAddr:        current.URL,
			BackupRPCAddrs: []string{current.URL, last.URL},
		},
	}
	require.NoError(t, provider.setRpcClient(true, current.URL, 250*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	started := time.Now()
	go func() {
		provider.startLivelinessChecks(ctx, 250*time.Millisecond)
		close(done)
	}()

	require.Eventually(t, func() bool {
		return cosmosLivelinessHasMessage(logs, "All configured RPC endpoints return non-200 response")
	}, 13*time.Second, 10*time.Millisecond)
	cancel()
	requireCosmosLivelinessDone(t, done)
	require.GreaterOrEqual(t, time.Since(started), 10*time.Second)
	require.Equal(t, []string{current.URL, current.URL, last.URL}, cosmosLivelinessAttemptRPCs(logs))
	terminal := logs.FilterMessage("All configured RPC endpoints return non-200 response").All()[0].ContextMap()
	require.Equal(t, "cosmos-name", terminal["chain"])
	require.Contains(t, terminal["error"], "status unavailable")
	require.Equal(t, int32(3), currentRequests.Load())
	require.Equal(t, int32(1), lastRequests.Load())
	require.Nil(t, provider.LightProvider)
	require.Equal(t, current.URL, provider.PCfg.RPCAddr)
	require.Equal(t, []string{current.URL, last.URL}, provider.PCfg.BackupRPCAddrs)
	_, err := provider.ConsensusClient.GetStatus(context.Background())
	require.Error(t, err)
	require.Equal(t, int32(2), lastRequests.Load())
	require.Equal(t, 3, cosmosLivelinessLevelCount(logs, zap.ErrorLevel, "Failed to connect to RPC client"))
	require.Equal(t, 1, cosmosLivelinessLevelCount(logs, zap.ErrorLevel, "All configured RPC endpoints return non-200 response"))
}

func requireCosmosLivelinessDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		require.FailNow(t, "liveliness goroutine did not stop after cancellation")
	}
}

func cosmosLivelinessMessages(logs *observer.ObservedLogs) []string {
	entries := logs.All()
	messages := make([]string, 0, len(entries))
	for _, entry := range entries {
		messages = append(messages, entry.Message)
	}
	return messages
}

func cosmosLivelinessRPCServer(t *testing.T, fail bool) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	requests := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		var request map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		response := map[string]any{"jsonrpc": "2.0", "id": request["id"]}
		if fail {
			response["error"] = map[string]any{"code": -32000, "message": "status unavailable"}
		} else {
			response["result"] = map[string]any{}
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	t.Cleanup(server.Close)
	return server, requests
}

func cosmosLivelinessHasMessage(logs *observer.ObservedLogs, message string) bool {
	for _, entry := range logs.All() {
		if entry.Message == message {
			return true
		}
	}
	return false
}

func cosmosLivelinessAttemptRPCs(logs *observer.ObservedLogs) []string {
	var rpcs []string
	for _, entry := range logs.All() {
		if entry.Message == "Attempting to connect to new RPC" {
			rpcs = append(rpcs, entry.ContextMap()["rpc"].(string))
		}
	}
	return rpcs
}

func cosmosLivelinessLevelCount(logs *observer.ObservedLogs, level zapcore.Level, message string) int {
	count := 0
	for _, entry := range logs.All() {
		if entry.Level == level && entry.Message == message {
			count++
		}
	}
	return count
}
