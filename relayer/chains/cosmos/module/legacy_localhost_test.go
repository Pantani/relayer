package module

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
	"github.com/stretchr/testify/require"
)

func TestRegisterInterfacesUnpacksLegacyLocalhostClient(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	AppModuleBasic{}.RegisterInterfaces(registry)

	legacyAny := &codectypes.Any{
		TypeUrl: "/" + legacyLocalhostClientStateTypeURL,
		Value:   []byte{0x0a, 0x04, 0x08, 0x01, 0x10, 0x02},
	}
	var clientState ibcexported.ClientState

	require.NoError(t, registry.UnpackAny(legacyAny, &clientState))
	legacyState, ok := clientState.(*legacyLocalhostClientState)
	require.True(t, ok)
	require.Equal(t, clienttypes.NewHeight(1, 2), legacyState.LatestHeight)
}
