package interchaintest

import (
	"context"
	"testing"

	"github.com/cosmos/interchaintest/v11/ibc"
	"github.com/stretchr/testify/require"
)

func TestAddKeyArgs(t *testing.T) {
	require.Equal(t,
		[]string{"keys", "add", "chain-a", "rly", "--coin-type", "118", "--signing-algorithm", "eth_secp256k1"},
		addKeyArgs("chain-a", "rly", "118", "eth_secp256k1"),
	)
	require.Equal(t,
		[]string{"keys", "add", "chain-a", "rly", "--coin-type", "118"},
		addKeyArgs("chain-a", "rly", "118", ""),
	)
}

func TestRestoreKeyArgs(t *testing.T) {
	cfg := ibc.ChainConfig{ChainID: "chain-a", CoinType: "354", SigningAlgorithm: "sr25519"}
	require.Equal(t,
		[]string{"keys", "restore", "chain-a", "rly", "words", "--coin-type", "354", "--signing-algorithm", "sr25519"},
		restoreKeyArgs(cfg, "rly", "words"),
	)

	cfg.SigningAlgorithm = ""
	require.Equal(t,
		[]string{"keys", "restore", "chain-a", "rly", "words", "--coin-type", "354"},
		restoreKeyArgs(cfg, "rly", "words"),
	)
}

func TestCreateClientArgs(t *testing.T) {
	opts := ibc.CreateClientOptions{
		TrustingPeriod:           "24h",
		TrustingPeriodPercentage: 66,
		MaxClockDrift:            "30s",
		Override:                 true,
	}
	require.Equal(t,
		[]string{"tx", "client", "chain-a", "chain-b", "path", "--client-tp", "24h", "--client-tp-percentage", "66", "--max-clock-drift", "30s", "--override"},
		createClientArgs([]string{"tx", "client", "chain-a", "chain-b", "path"}, opts),
	)
}

func TestLinkPathArgsOmitsEmptyClientOptions(t *testing.T) {
	channelOpts := ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          ibc.Unordered,
		Version:        "ics20-1",
	}
	require.Equal(t,
		[]string{"tx", "link", "path", "--src-port", "transfer", "--dst-port", "transfer", "--order", "unordered", "--version", "ics20-1"},
		linkPathArgs("path", channelOpts, ibc.CreateClientOptions{}),
	)
}

func TestPathUpdateArgs(t *testing.T) {
	srcChain, dstChain := "chain-a", "chain-b"
	srcClient, dstClient := "07-tendermint-0", "07-tendermint-1"
	srcConn, dstConn := "connection-0", "connection-1"
	opts := ibc.PathUpdateOptions{
		ChannelFilter: &ibc.ChannelFilter{Rule: "allowlist", ChannelList: []string{"channel-0", "channel-1"}},
		SrcChainID:    &srcChain,
		DstChainID:    &dstChain,
		SrcClientID:   &srcClient,
		DstClientID:   &dstClient,
		SrcConnID:     &srcConn,
		DstConnID:     &dstConn,
	}
	require.Equal(t,
		[]string{"paths", "update", "path", "--filter-rule", "allowlist", "--filter-channels", "channel-0,channel-1", "--src-chain-id", "chain-a", "--dst-chain-id", "chain-b", "--src-client-id", "07-tendermint-0", "--dst-client-id", "07-tendermint-1", "--src-connection-id", "connection-0", "--dst-connection-id", "connection-1"},
		pathUpdateArgs("path", opts),
	)
}

func TestUnsupportedInProcessRelayerFeaturesReturnErrors(t *testing.T) {
	r := &Relayer{}
	require.Error(t, r.SetClientContractHash(context.Background(), nil, ibc.ChainConfig{}, "hash"))
	require.Error(t, r.PauseRelayer(context.Background()))
	require.Error(t, r.ResumeRelayer(context.Background()))
}
