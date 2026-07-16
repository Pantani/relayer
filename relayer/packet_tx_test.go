package relayer

import (
	"context"
	"testing"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
	"github.com/cosmos/relayer/v2/relayer/chains/cosmos"
	"github.com/stretchr/testify/require"
)

func TestTransferClientLatestHeightUsesStatelessLocalhostHeight(t *testing.T) {
	chain := &Chain{
		ChainProvider: &cosmos.CosmosProvider{
			PCfg: cosmos.CosmosProviderConfig{ChainID: "local-chain-7"},
		},
		PathEnd: &PathEnd{ClientID: ibcexported.LocalhostClientID},
	}

	height, err := transferClientLatestHeight(context.Background(), chain, 42)
	require.NoError(t, err)
	require.Equal(t, clienttypes.NewHeight(7, 42), height)
}

func TestValidateTransferTimeoutOffsetRejectsNegativeDuration(t *testing.T) {
	require.ErrorContains(t, validateTransferTimeoutOffset(-time.Second), "cannot be negative")
	require.NoError(t, validateTransferTimeoutOffset(0))
	require.NoError(t, validateTransferTimeoutOffset(time.Second))
}

func TestTransferClientLatestHeightRejectsNegativeLocalhostHeight(t *testing.T) {
	chain := &Chain{
		ChainProvider: &cosmos.CosmosProvider{
			PCfg: cosmos.CosmosProviderConfig{ChainID: "local-chain-7"},
		},
		PathEnd: &PathEnd{ClientID: ibcexported.LocalhostClientID},
	}

	_, err := transferClientLatestHeight(context.Background(), chain, -1)
	require.ErrorContains(t, err, "cannot be negative")
}
