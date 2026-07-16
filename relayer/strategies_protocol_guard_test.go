package relayer

import (
	"context"
	"testing"

	"github.com/cosmos/relayer/v2/relayer/protocol"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStartRelayerRejectsV2BeforeProviderAccess(t *testing.T) {
	path := &Path{
		Protocol: protocol.ProtocolV2,
		Src: &PathEnd{
			ChainID:      "chain-a",
			ClientID:     "07-tendermint-0",
			MerklePrefix: MerklePrefix{"ibc"},
		},
		Dst: &PathEnd{
			ChainID:      "chain-b",
			ClientID:     "07-tendermint-1",
			MerklePrefix: MerklePrefix{"ibc"},
		},
	}

	errorChan := StartRelayer(
		context.Background(),
		zap.NewNop(),
		nil,
		[]NamedPath{{Name: "v2-path", Path: path}},
		DefaultMaxMsgLength,
		0,
		0,
		"",
		DefaultClientUpdateThreshold,
		DefaultFlushInterval,
		nil,
		ProcessorEvents,
		0,
		nil,
		nil,
	)

	err, open := <-errorChan
	require.True(t, open)
	require.ErrorIs(t, err, ErrV2RuntimeNotImplemented)
	require.ErrorContains(t, err, `path "v2-path"`)

	_, open = <-errorChan
	require.False(t, open)
}

func TestEventProcessorPathsPreserveClassicFilters(t *testing.T) {
	paths := eventProcessorPaths([]NamedPath{{
		Name: "classic-path",
		Path: &Path{
			Src: &PathEnd{ChainID: "chain-a", ClientID: "client-a"},
			Dst: &PathEnd{ChainID: "chain-b", ClientID: "client-b"},
			Filter: ChannelFilter{
				Rule:        "allowlist",
				ChannelList: []string{"channel-1", "channel-2"},
			},
		},
	}})

	require.Len(t, paths, 1)
	require.Equal(t, "chain-a", paths[0].src.ChainID)
	require.Equal(t, "client-a", paths[0].src.ClientID)
	require.Equal(t, "chain-b", paths[0].dst.ChainID)
	require.Equal(t, "client-b", paths[0].dst.ClientID)
	require.Equal(t, "allowlist", paths[0].src.Rule)
	require.Equal(t, "allowlist", paths[0].dst.Rule)
	require.Len(t, paths[0].src.FilterList, 2)
	require.Len(t, paths[0].dst.FilterList, 2)
	require.Equal(t, "channel-1", paths[0].src.FilterList[0].ChannelKey.ChannelID)
	require.Equal(t, "chain-a", paths[0].dst.FilterList[0].CounterpartyChainID)
	require.Equal(t, "channel-1", paths[0].dst.FilterList[0].ChannelKey.CounterpartyChannelID)

	emptyFilters := eventProcessorPaths([]NamedPath{{
		Name: "unfiltered-classic-path",
		Path: &Path{
			Src: &PathEnd{ChainID: "chain-a", ClientID: "client-a"},
			Dst: &PathEnd{ChainID: "chain-b", ClientID: "client-b"},
		},
	}})
	require.Nil(t, emptyFilters[0].src.FilterList)
	require.Nil(t, emptyFilters[0].dst.FilterList)
}
