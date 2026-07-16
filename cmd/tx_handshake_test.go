package cmd

import (
	"testing"
	"time"

	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/protocol"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type keyCheckingProvider struct {
	provider.ChainProvider
	chainID string
	key     string
	exists  bool
}

func (p *keyCheckingProvider) ChainId() string {
	return p.chainID
}

func (p *keyCheckingProvider) Key() string {
	return p.key
}

func (p *keyCheckingProvider) KeyExists(string) bool {
	return p.exists
}

func TestHandshakeCommandContracts(t *testing.T) {
	t.Parallel()

	state := &appState{viper: viper.New()}
	tests := []struct {
		name    string
		command *cobra.Command
		use     string
		aliases []string
		flags   []string
	}{
		{
			name:    "clients",
			command: createClientsCmd(state),
			use:     "clients path_name",
			flags:   []string{flagUpdateAfterExpiry, flagMaxClockDrift, flagOverride, flagMemo},
		},
		{
			name:    "client",
			command: createClientCmd(state),
			use:     "client src_chain_name dst_chain_name path_name",
			flags:   []string{flagClientUnbondingPeriod, flagClientTrustingPeriodPercentage, flagMemo},
		},
		{
			name:    "connection",
			command: createConnectionCmd(state),
			use:     "connection path_name",
			aliases: []string{"conn"},
			flags:   []string{flagTimeout, flagMaxRetries, flagInitialBlockHistory, flagMemo},
		},
		{
			name:    "link",
			command: linkCmd(state),
			use:     "link path_name",
			aliases: []string{"connect"},
			flags:   []string{flagSrcPort, flagDstPort, flagOrder, flagVersion, flagInitialBlockHistory},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.use, test.command.Use)
			require.Equal(t, test.aliases, test.command.Aliases)
			for _, flagName := range test.flags {
				require.NotNil(t, test.command.Flags().Lookup(flagName), flagName)
			}
		})
	}
}

func TestReadSingleClientOptions(t *testing.T) {
	t.Parallel()

	cmd := createClientCmd(&appState{viper: viper.New()})
	setCommandFlags(t, cmd, map[string]string{
		flagUpdateAfterExpiry:              "false",
		flagUpdateAfterMisbehaviour:        "false",
		flagClientTrustingPeriod:           "24h",
		flagClientTrustingPeriodPercentage: "72",
		flagClientUnbondingPeriod:          "48h",
		flagOverride:                       "true",
		flagMaxClockDrift:                  "45s",
	})

	options, err := readSingleClientOptions(cmd)
	require.NoError(t, err)
	require.False(t, options.allowUpdateAfterExpiry)
	require.False(t, options.allowUpdateAfterMisbehaviour)
	require.Equal(t, 24*time.Hour, options.customTrustingPeriod)
	require.Equal(t, int64(72), options.customTrustingPeriodPercentage)
	require.Equal(t, 48*time.Hour, options.overrideUnbondingPeriod)
	require.True(t, options.override)
	require.Equal(t, 45*time.Second, options.maxClockDrift)
}

func TestReadLinkOptions(t *testing.T) {
	t.Parallel()

	cmd := linkCmd(&appState{viper: viper.New()})
	setCommandFlags(t, cmd, map[string]string{
		flagSrcPort:       "transfer-src",
		flagDstPort:       "transfer-dst",
		flagOrder:         "ordered",
		flagVersion:       "ics20-2",
		flagTimeout:       "17s",
		flagMaxRetries:    "11",
		flagOverride:      "true",
		flagMaxClockDrift: "31s",
	})

	channel, err := readChannelCreationOptions(cmd)
	require.NoError(t, err)
	require.Equal(t, channelCreationOptions{
		srcPort: "transfer-src",
		dstPort: "transfer-dst",
		order:   "ordered",
		version: "ics20-2",
	}, channel)
	execution, err := readHandshakeExecutionOptions(cmd)
	require.NoError(t, err)
	require.Equal(t, handshakeExecutionOptions{
		timeout:       17 * time.Second,
		retries:       11,
		override:      true,
		maxClockDrift: 31 * time.Second,
	}, execution)
}

func TestClientIDForPath(t *testing.T) {
	t.Parallel()

	path := &relayer.Path{
		Src: &relayer.PathEnd{ChainID: "chain-a"},
		Dst: &relayer.PathEnd{ChainID: "chain-b"},
	}

	srcID, dstID := clientIDForPath(path, "chain-a", "07-tendermint-10")
	require.Equal(t, "07-tendermint-10", srcID)
	require.Empty(t, dstID)

	srcID, dstID = clientIDForPath(path, "chain-b", "07-tendermint-20")
	require.Empty(t, srcID)
	require.Equal(t, "07-tendermint-20", dstID)
}

func TestResolveSingleClientPathRejectsV2BeforePathMutation(t *testing.T) {
	t.Parallel()

	src := relayer.NewChain(zap.NewNop(), &keyCheckingProvider{chainID: "chain-a-1"}, false)
	dst := relayer.NewChain(zap.NewNop(), &keyCheckingProvider{chainID: "chain-b-1"}, false)
	config := Config{
		Chains: relayer.Chains{
			"chain-a": src,
			"chain-b": dst,
		},
		Paths: relayer.Paths{
			"v2-path": {
				Protocol: protocol.ProtocolV2,
				Src:      &relayer.PathEnd{ChainID: "chain-a-1"},
				Dst:      &relayer.PathEnd{ChainID: "chain-b-1"},
			},
		},
	}

	gotSrc, gotDst, path, err := resolveSingleClientPath(&config, []string{"chain-a", "chain-b", "v2-path"})

	require.ErrorIs(t, err, relayer.ErrV2RuntimeNotImplemented)
	require.Nil(t, gotSrc)
	require.Nil(t, gotDst)
	require.Nil(t, path)
	require.Nil(t, src.PathEnd)
	require.Nil(t, dst.PathEnd)
}

func TestLinkChainsForPathRejectsV2BeforeChainLookup(t *testing.T) {
	t.Parallel()

	config := Config{Paths: relayer.Paths{
		"v2-path": {
			Protocol: protocol.ProtocolV2,
			Src:      &relayer.PathEnd{ChainID: "chain-a-1"},
			Dst:      &relayer.PathEnd{ChainID: "chain-b-1"},
		},
	}}

	pair, err := linkChainsForPath(&config, "v2-path")

	require.ErrorIs(t, err, relayer.ErrV2RuntimeNotImplemented)
	require.Empty(t, pair)
}

func TestEnsureSelectedChainKeysPreservesErrors(t *testing.T) {
	t.Parallel()

	srcProvider := &keyCheckingProvider{chainID: "chain-a-1", key: "alice"}
	dstProvider := &keyCheckingProvider{chainID: "chain-b-1", key: "bob", exists: true}
	src := &relayer.Chain{ChainProvider: srcProvider}
	dst := &relayer.Chain{ChainProvider: dstProvider}

	require.EqualError(t, ensureSelectedChainKeys(src, dst), "key alice not found on src chain chain-a-1")
	srcProvider.exists = true
	dstProvider.exists = false
	require.EqualError(t, ensureSelectedChainKeys(src, dst), "key bob not found on dst chain chain-b-1")
}

func setCommandFlags(t *testing.T, cmd *cobra.Command, values map[string]string) {
	t.Helper()
	for name, value := range values {
		require.NoError(t, cmd.Flags().Set(name, value), name)
	}
}
