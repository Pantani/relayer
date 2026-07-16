package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/protocol"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestValidatePathRejectsInvalidV2BeforeRPC(t *testing.T) {
	t.Parallel()

	config, src, dst := v2ValidationConfig()
	path := validConfigV2Path()
	path.Src.ConnectionID = "connection-0"

	err := config.ValidatePath(context.Background(), &bytes.Buffer{}, path)
	require.EqualError(t, err, "path protocol v2 cannot set source connection-id")
	require.Zero(t, src.heightQueries+dst.heightQueries)
	require.Zero(t, src.clientQueries+dst.clientQueries)
	require.Zero(t, src.connectionQueries+dst.connectionQueries)
}

func TestValidateV2PathQueriesClientsOnly(t *testing.T) {
	t.Parallel()

	config, src, dst := v2ValidationConfig()
	err := config.ValidatePath(context.Background(), &bytes.Buffer{}, validConfigV2Path())

	require.NoError(t, err)
	require.Equal(t, 2, src.heightQueries+dst.heightQueries)
	require.Equal(t, 2, src.clientQueries+dst.clientQueries)
	require.Zero(t, src.connectionQueries+dst.connectionQueries)
}

func TestValidateConfigChecksProtocolStructure(t *testing.T) {
	t.Parallel()

	path := validConfigV2Path()
	path.Filter.ChannelList = []string{"channel-0"}
	config := Config{
		Global: GlobalConfig{Timeout: "10s"},
		Paths:  relayer.Paths{"v2-path": path},
	}

	err := config.validateConfig()
	require.ErrorContains(t, err, "path protocol v2 cannot set src-channel-filter")
}

func TestValidateConfigReportsMalformedPathWithoutPanic(t *testing.T) {
	t.Parallel()

	config := Config{
		Global: GlobalConfig{Timeout: "10s"},
		Paths:  relayer.Paths{"broken": nil},
	}

	require.EqualError(t, config.validateConfig(), "error initializing the relayer config for path broken: path is nil")
}

func TestAddPathProtocolConflicts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		oldPath  *relayer.Path
		newPath  *relayer.Path
		wantErr  string
		protocol protocol.Protocol
	}{
		{
			name:    "implicit classic to v2",
			oldPath: configClassicPath(protocol.ProtocolUnspecified),
			newPath: validConfigV2Path(),
			wantErr: "path with ID demo and conflicting protocol (classic) already exists",
		},
		{
			name:    "v2 to classic",
			oldPath: validConfigV2Path(),
			newPath: configClassicPath(protocol.ProtocolClassic),
			wantErr: "path with ID demo and conflicting protocol (v2) already exists",
		},
		{
			name:     "explicit classic preserved",
			oldPath:  configClassicPath(protocol.ProtocolClassic),
			newPath:  configClassicPath(protocol.ProtocolUnspecified),
			protocol: protocol.ProtocolClassic,
		},
		{
			name:     "explicit classic added",
			oldPath:  configClassicPath(protocol.ProtocolUnspecified),
			newPath:  configClassicPath(protocol.ProtocolClassic),
			protocol: protocol.ProtocolClassic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{Paths: relayer.Paths{"demo": tt.oldPath}}
			err := config.AddPath("demo", tt.newPath)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.protocol, config.Paths["demo"].Protocol)
		})
	}
}

func TestAddPathMerklePrefixConflicts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		old     relayer.MerklePrefix
		updated relayer.MerklePrefix
		wantErr bool
	}{
		{name: "fill", updated: relayer.MerklePrefix{"ibc"}},
		{name: "same", old: relayer.MerklePrefix{"ibc"}, updated: relayer.MerklePrefix{"ibc"}},
		{name: "change", old: relayer.MerklePrefix{"ibc"}, updated: relayer.MerklePrefix{"store"}, wantErr: true},
		{name: "remove", old: relayer.MerklePrefix{"ibc"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldPath := validConfigV2Path()
			oldPath.Src.MerklePrefix = tt.old
			newPath := validConfigV2Path()
			newPath.Src.MerklePrefix = tt.updated
			config := Config{Paths: relayer.Paths{"demo": oldPath}}

			err := config.AddPath("demo", newPath)
			if tt.wantErr {
				require.ErrorContains(t, err, "conflicting source merkle prefix")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.updated, config.Paths["demo"].Src.MerklePrefix)
		})
	}
}

func TestChainsFromPathRejectsV2BeforeChainLookup(t *testing.T) {
	t.Parallel()

	config := Config{Paths: relayer.Paths{"v2-path": validConfigV2Path()}}
	chains, src, dst, err := config.ChainsFromPath("v2-path")

	require.Nil(t, chains)
	require.Empty(t, src)
	require.Empty(t, dst)
	require.ErrorIs(t, err, relayer.ErrV2RuntimeNotImplemented)
}

func v2ValidationConfig() (Config, *pathValidationProvider, *pathValidationProvider) {
	src := &pathValidationProvider{chainID: "chain-a-1", height: 42}
	dst := &pathValidationProvider{chainID: "chain-b-1", height: 43}
	return Config{
		Chains: relayer.Chains{
			"chain-a": relayer.NewChain(zap.NewNop(), src, false),
			"chain-b": relayer.NewChain(zap.NewNop(), dst, false),
		},
	}, src, dst
}

func validConfigV2Path() *relayer.Path {
	return &relayer.Path{
		Protocol: protocol.ProtocolV2,
		Src: &relayer.PathEnd{
			ChainID:      "chain-a-1",
			ClientID:     "07-tendermint-0",
			MerklePrefix: relayer.MerklePrefix{"ibc"},
		},
		Dst: &relayer.PathEnd{
			ChainID:      "chain-b-1",
			ClientID:     "07-tendermint-1",
			MerklePrefix: relayer.MerklePrefix{"ibc"},
		},
	}
}

func configClassicPath(value protocol.Protocol) *relayer.Path {
	return &relayer.Path{
		Protocol: value,
		Src: &relayer.PathEnd{
			ChainID:      "chain-a-1",
			ClientID:     "07-tendermint-0",
			ConnectionID: "connection-0",
		},
		Dst: &relayer.PathEnd{
			ChainID:      "chain-b-1",
			ClientID:     "07-tendermint-1",
			ConnectionID: "connection-1",
		},
	}
}
