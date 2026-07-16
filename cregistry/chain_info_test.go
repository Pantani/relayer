package cregistry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestGetAllRPCEndpoints(t *testing.T) {
	testCases := map[string]struct {
		chainInfo         ChainInfo
		expectedEndpoints []string
		expectedError     error
	}{
		"endpoint with TLS": {
			chainInfo:         ChainInfoWithRPCEndpoint("https://test.com"),
			expectedEndpoints: []string{"https://test.com:443"},
			expectedError:     nil,
		},
		"endpoint without TLS": {
			chainInfo:         ChainInfoWithRPCEndpoint("http://test.com:26657"),
			expectedEndpoints: []string{"http://test.com:26657"},
			expectedError:     nil,
		},
		"endpoint with TLS and with path": {
			chainInfo:         ChainInfoWithRPCEndpoint("https://test.com/rpc"),
			expectedEndpoints: []string{"https://test.com:443/rpc"},
			expectedError:     nil,
		},
		"endpoint with TLS and non-standard port": {
			chainInfo:         ChainInfoWithRPCEndpoint("https://test.com:8443"),
			expectedEndpoints: []string{"https://test.com:8443"},
			expectedError:     nil,
		},
		"proxied endpoint with TLS and non-standard port": {
			chainInfo:         ChainInfoWithRPCEndpoint("https://test.com:8443/rpc"),
			expectedEndpoints: []string{"https://test.com:8443/rpc"},
			expectedError:     nil,
		},
		"proxied endpoint without TLS and without path": {
			chainInfo:         ChainInfoWithRPCEndpoint("http://test.com"),
			expectedEndpoints: []string{"http://test.com:80"},
			expectedError:     nil,
		},
		"proxied endpoint without TLS and with path": {
			chainInfo:         ChainInfoWithRPCEndpoint("http://test.com/rpc"),
			expectedEndpoints: []string{"http://test.com:80/rpc"},
			expectedError:     nil,
		},
		"unsupported or invalid url scheme error": {
			chainInfo:         ChainInfoWithRPCEndpoint("ftp://test.com/rpc"),
			expectedEndpoints: nil,
			expectedError:     errors.New("invalid or unsupported url scheme: ftp"),
		},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			endpoints, err := tc.chainInfo.GetAllRPCEndpoints()
			require.Equal(t, tc.expectedError, err)
			require.Equal(t, tc.expectedEndpoints, endpoints)
		})
	}
}

func ChainInfoWithRPCEndpoint(endpoint string) ChainInfo {
	return ChainInfo{
		Apis: struct {
			RPC []struct {
				Address  string `json:"address"`
				Provider string `json:"provider"`
			} `json:"rpc"`
			Rest []struct {
				Address  string `json:"address"`
				Provider string `json:"provider"`
			} `json:"rest"`
		}{
			RPC: []struct {
				Address  string `json:"address"`
				Provider string `json:"provider"`
			}{
				{
					Address:  endpoint,
					Provider: "test",
				},
			},
		},
	}
}

func TestGetBackupRPCEndpointsPreservesSelectionOrder(t *testing.T) {
	const (
		first  = "https://first.example.com:443"
		second = "https://second.example.com:443"
		third  = "https://third.example.com:443"
	)
	tests := []struct {
		name       string
		primaryRPC string
		count      uint64
		want       []string
	}{
		{name: "first two without primary", count: 2, want: []string{first, second}},
		{name: "stop when primary is first", primaryRPC: first, count: 2, want: []string{}},
		{name: "stop after endpoint before primary", primaryRPC: second, count: 2, want: []string{first}},
		{name: "count is currently ignored", count: 1, want: []string{first, second}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainInfo := chainInfoWithRPCEndpoints(zap.NewNop(), first, second, third)
			endpoints, err := chainInfo.GetBackupRPCEndpoints(
				context.Background(), true, tt.primaryRPC, tt.count,
			)
			require.NoError(t, err)
			require.Equal(t, tt.want, endpoints)
		})
	}
}

func TestGetBackupRPCEndpointsEmptyResults(t *testing.T) {
	tests := []struct {
		name     string
		forceAdd bool
		wantErr  string
	}{
		{name: "healthy endpoints required", wantErr: "no working RPCs found, consider using --force-add"},
		{name: "force add permits empty registry", forceAdd: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainInfo := ChainInfo{log: zap.NewNop()}
			endpoints, err := chainInfo.GetBackupRPCEndpoints(
				context.Background(), tt.forceAdd, "", 2,
			)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr)
			}
			require.Nil(t, endpoints)
		})
	}
}

func TestGetBackupRPCEndpointsLogsSelection(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	chainInfo := chainInfoWithRPCEndpoints(
		zap.New(core),
		"https://first.example.com:443",
		"https://second.example.com:443",
	)
	chainInfo.ChainName = "chain-a"

	endpoints, err := chainInfo.GetBackupRPCEndpoints(context.Background(), true, "", 2)

	require.NoError(t, err)
	require.Equal(t, []string{"https://first.example.com:443", "https://second.example.com:443"}, endpoints)
	entries := logs.FilterMessage("Backup Endpoints selected")
	require.Len(t, entries.FilterField(zap.String("chain_name", "chain-a")).All(), 1)
	require.Len(t, entries.FilterField(zap.Strings("endpoints", endpoints)).All(), 1)
}

func chainInfoWithRPCEndpoints(log *zap.Logger, endpoints ...string) ChainInfo {
	chainInfo := ChainInfo{log: log}
	for _, endpoint := range endpoints {
		chainInfo.Apis.RPC = append(chainInfo.Apis.RPC, struct {
			Address  string `json:"address"`
			Provider string `json:"provider"`
		}{Address: endpoint, Provider: "test"})
	}
	return chainInfo
}
