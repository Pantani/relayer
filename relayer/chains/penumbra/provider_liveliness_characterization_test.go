package penumbra

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
	"go.uber.org/zap/zaptest/observer"
)

func TestPenumbraLivelinessCharacterizesNoBackupImmediateReturn(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	provider := &PenumbraProvider{
		log:  zap.New(core),
		PCfg: PenumbraProviderConfig{ChainName: "penumbra-name", RPCAddr: "rpc-primary"},
	}
	started := time.Now()

	provider.startLivelinessChecks(context.Background(), time.Millisecond)

	require.Less(t, time.Since(started), time.Second)
	require.Equal(t, []string{"No backup RPCs defined"}, penumbraLivelinessMessages(logs))
	require.Equal(t, map[string]any{"chain": "penumbra-name"}, logs.All()[0].ContextMap())
}

func TestPenumbraLivelinessCharacterizesNoBackupNilLoggerSafeReturn(t *testing.T) {
	provider := &PenumbraProvider{PCfg: PenumbraProviderConfig{RPCAddr: ""}}

	require.NotPanics(t, func() {
		provider.startLivelinessChecks(nil, time.Millisecond)
	})
}

func TestPenumbraLivelinessCharacterizesCanceledInventoryAndDuplicates(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	provider := &PenumbraProvider{
		log: zap.New(core),
		PCfg: PenumbraProviderConfig{
			ChainName:      "penumbra-name",
			RPCAddr:        "",
			BackupRPCAddrs: []string{"rpc-backup", "rpc-backup"},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider.startLivelinessChecks(ctx, 25*time.Millisecond)

	require.Equal(t, []string{"Available RPC clients"}, penumbraLivelinessMessages(logs))
	require.Equal(t, map[string]any{
		"chain": "penumbra-name",
		"count": int64(3),
	}, logs.All()[0].ContextMap())
}

func TestPenumbraLivelinessCharacterizesCancellationStopsGoroutine(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	provider := &PenumbraProvider{
		log: zap.New(core),
		PCfg: PenumbraProviderConfig{
			ChainName:      "penumbra-name",
			RPCAddr:        "rpc-primary",
			BackupRPCAddrs: []string{"rpc-backup"},
		},
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
	requirePenumbraLivelinessDone(t, done)
	require.Equal(t, []string{"Available RPC clients"}, penumbraLivelinessMessages(logs))
}

func TestPenumbraLivelinessCharacterizesNilLoggerPanicWithBackup(t *testing.T) {
	provider := &PenumbraProvider{PCfg: PenumbraProviderConfig{
		RPCAddr:        "rpc-primary",
		BackupRPCAddrs: []string{"rpc-backup"},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.Panics(t, func() {
		provider.startLivelinessChecks(ctx, time.Millisecond)
	})
}

func TestPenumbraLivelinessCharacterizesNilContextPanicWithBackup(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	provider := &PenumbraProvider{
		log: zap.New(core),
		PCfg: PenumbraProviderConfig{
			RPCAddr:        "rpc-primary",
			BackupRPCAddrs: []string{"rpc-backup"},
		},
	}

	require.Panics(t, func() {
		provider.startLivelinessChecks(nil, time.Millisecond)
	})
	require.Equal(t, []string{"Available RPC clients"}, penumbraLivelinessMessages(logs))
}

func TestPenumbraLivelinessCharacterizesHealthyTickWithoutRotation(t *testing.T) {
	t.Parallel()
	primary, primaryRequests := penumbraLivelinessRPCServer(t, false)
	backup, backupRequests := penumbraLivelinessRPCServer(t, false)
	core, logs := observer.New(zap.DebugLevel)
	provider := &PenumbraProvider{
		log: zap.New(core),
		PCfg: PenumbraProviderConfig{
			ChainID:        "chain-p-1",
			ChainName:      "penumbra-name",
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
	requirePenumbraLivelinessDone(t, done)
	require.GreaterOrEqual(t, time.Since(started), 10*time.Second)
	require.Equal(t, int32(0), backupRequests.Load())
	require.Equal(t, []string{"Available RPC clients"}, penumbraLivelinessMessages(logs))
	require.Equal(t, initialLightProvider, provider.LightProvider)
	require.Equal(t, primary.URL, provider.PCfg.RPCAddr)
}

func TestPenumbraLivelinessCharacterizesTickRotationOrderAndClientSwap(t *testing.T) {
	t.Parallel()
	current, currentRequests := penumbraLivelinessRPCServer(t, true)
	backup, backupRequests := penumbraLivelinessRPCServer(t, false)
	core, logs := observer.New(zap.DebugLevel)
	provider := &PenumbraProvider{
		log: zap.New(core),
		PCfg: PenumbraProviderConfig{
			ChainID:        "chain-p-1",
			ChainName:      "penumbra-name",
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
		return penumbraLivelinessHasMessage(logs, "Successfully connected to new RPC")
	}, 13*time.Second, 10*time.Millisecond)
	cancel()
	requirePenumbraLivelinessDone(t, done)
	require.GreaterOrEqual(t, time.Since(started), 10*time.Second)
	require.Equal(t, []string{current.URL, current.URL, backup.URL}, penumbraLivelinessAttemptRPCs(logs))
	disconnected := logs.FilterMessage("RPC client disconnected").All()[0].ContextMap()
	require.Equal(t, "penumbra-name", disconnected["chain"])
	require.Contains(t, disconnected["error"], "status unavailable")
	connected := logs.FilterMessage("Successfully connected to new RPC").All()[0].ContextMap()
	require.Equal(t, "penumbra-name", connected["chain"])
	require.Equal(t, backup.URL, connected["rpc"])
	require.Equal(t, int32(3), currentRequests.Load())
	require.Equal(t, int32(1), backupRequests.Load())
	require.Equal(t, current.URL, provider.PCfg.RPCAddr)
	require.Equal(t, []string{current.URL, backup.URL}, provider.PCfg.BackupRPCAddrs)
	require.NotNil(t, provider.LightProvider)
	_, err := provider.ConsensusClient.Status(context.Background())
	require.NoError(t, err)
	require.Equal(t, int32(2), backupRequests.Load())
	require.Equal(t, 2, penumbraLivelinessDebugFailureCount(logs))
}

func TestPenumbraLivelinessCharacterizesAllFailPartialClientMutation(t *testing.T) {
	t.Parallel()
	current, currentRequests := penumbraLivelinessRPCServer(t, true)
	last, lastRequests := penumbraLivelinessRPCServer(t, true)
	core, logs := observer.New(zap.DebugLevel)
	provider := &PenumbraProvider{
		log: zap.New(core),
		PCfg: PenumbraProviderConfig{
			ChainID:        "chain-p-1",
			ChainName:      "penumbra-name",
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
		return penumbraLivelinessHasMessage(logs, "All configured RPC endpoints return non-200 response")
	}, 13*time.Second, 10*time.Millisecond)
	cancel()
	requirePenumbraLivelinessDone(t, done)
	require.GreaterOrEqual(t, time.Since(started), 10*time.Second)
	require.Equal(t, []string{current.URL, current.URL, last.URL}, penumbraLivelinessAttemptRPCs(logs))
	terminal := logs.FilterMessage("All configured RPC endpoints return non-200 response").All()[0].ContextMap()
	require.Equal(t, "penumbra-name", terminal["chain"])
	require.Contains(t, terminal["error"], "status unavailable")
	require.Equal(t, int32(3), currentRequests.Load())
	require.Equal(t, int32(1), lastRequests.Load())
	require.Nil(t, provider.LightProvider)
	require.Equal(t, current.URL, provider.PCfg.RPCAddr)
	require.Equal(t, []string{current.URL, last.URL}, provider.PCfg.BackupRPCAddrs)
	_, err := provider.ConsensusClient.Status(context.Background())
	require.Error(t, err)
	require.Equal(t, int32(2), lastRequests.Load())
	require.Equal(t, 3, penumbraLivelinessDebugFailureCount(logs))
	require.Equal(t, 1, penumbraLivelinessErrorCount(logs, "All configured RPC endpoints return non-200 response"))
}

func requirePenumbraLivelinessDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		require.FailNow(t, "liveliness goroutine did not stop after cancellation")
	}
}

func penumbraLivelinessMessages(logs *observer.ObservedLogs) []string {
	entries := logs.All()
	messages := make([]string, 0, len(entries))
	for _, entry := range entries {
		messages = append(messages, entry.Message)
	}
	return messages
}

func penumbraLivelinessRPCServer(t *testing.T, fail bool) (*httptest.Server, *atomic.Int32) {
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

func penumbraLivelinessHasMessage(logs *observer.ObservedLogs, message string) bool {
	for _, entry := range logs.All() {
		if entry.Message == message {
			return true
		}
	}
	return false
}

func penumbraLivelinessAttemptRPCs(logs *observer.ObservedLogs) []string {
	var rpcs []string
	for _, entry := range logs.All() {
		if entry.Message == "Attempting to connect to new RPC" {
			rpcs = append(rpcs, entry.ContextMap()["rpc"].(string))
		}
	}
	return rpcs
}

func penumbraLivelinessDebugFailureCount(logs *observer.ObservedLogs) int {
	count := 0
	for _, entry := range logs.All() {
		if entry.Level == zap.DebugLevel && entry.Message == "Failed to connect to RPC client" {
			count++
		}
	}
	return count
}

func penumbraLivelinessErrorCount(logs *observer.ObservedLogs, message string) int {
	count := 0
	for _, entry := range logs.All() {
		if entry.Level == zap.ErrorLevel && entry.Message == message {
			count++
		}
	}
	return count
}
