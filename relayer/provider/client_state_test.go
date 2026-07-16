package provider

import (
	"testing"

	"github.com/cosmos/gogoproto/proto"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v11/modules/light-clients/06-solomachine"
	tendermint "github.com/cosmos/ibc-go/v11/modules/light-clients/07-tendermint"
	attestations "github.com/cosmos/ibc-go/v11/modules/light-clients/attestations"
	"github.com/stretchr/testify/require"
)

type unsupportedClientState struct{}

func (*unsupportedClientState) Reset()         {}
func (*unsupportedClientState) String() string { return "unsupported" }
func (*unsupportedClientState) ProtoMessage()  {}
func (*unsupportedClientState) ClientType() string {
	return "unsupported"
}
func (*unsupportedClientState) Validate() error { return nil }

var _ proto.Message = (*unsupportedClientState)(nil)
var _ ibcexported.ClientState = (*unsupportedClientState)(nil)

func TestClientStateLatestHeight(t *testing.T) {
	testCases := []struct {
		name  string
		state ibcexported.ClientState
		want  clienttypes.Height
	}{
		{
			name:  "tendermint",
			state: &tendermint.ClientState{LatestHeight: clienttypes.NewHeight(4, 12)},
			want:  clienttypes.NewHeight(4, 12),
		},
		{
			name:  "solo machine",
			state: &solomachine.ClientState{Sequence: 8},
			want:  clienttypes.NewHeight(0, 8),
		},
		{
			name:  "attestations",
			state: &attestations.ClientState{LatestHeight: 19},
			want:  clienttypes.NewHeight(0, 19),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ClientStateLatestHeight(testCase.state)
			require.NoError(t, err)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestClientStateLatestHeightRejectsUnsupportedAndNilStates(t *testing.T) {
	for _, state := range []ibcexported.ClientState{
		(*tendermint.ClientState)(nil),
		(*unsupportedClientState)(nil),
		&unsupportedClientState{},
	} {
		_, err := ClientStateLatestHeight(state)
		require.ErrorContains(t, err, "unsupported IBC client state")
	}
}
