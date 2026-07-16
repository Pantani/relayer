package provider

import (
	"context"
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	tmclient "github.com/cosmos/ibc-go/v11/modules/light-clients/07-tendermint"
	"github.com/stretchr/testify/require"
)

func TestClientsMatchSkipsUnsupportedClientType(t *testing.T) {
	existingClient := clienttypes.IdentifiedClientState{
		ClientId: "09-localhost",
		ClientState: &codectypes.Any{
			TypeUrl: "/ibc.lightclients.localhost.v2.ClientState",
			Value:   []byte{},
		},
	}

	clientID, err := ClientsMatch(context.Background(), nil, nil, existingClient, &tmclient.ClientState{})
	require.NoError(t, err)
	require.Empty(t, clientID)
}

func TestClientsMatchRejectsMalformedTendermintClient(t *testing.T) {
	existingClient := clienttypes.IdentifiedClientState{
		ClientId: "07-tendermint-0",
		ClientState: &codectypes.Any{
			TypeUrl: tendermintClientStateTypeURL,
			Value:   []byte{0xff},
		},
	}

	clientID, err := ClientsMatch(context.Background(), nil, nil, existingClient, &tmclient.ClientState{})
	require.Error(t, err)
	require.Empty(t, clientID)
}
