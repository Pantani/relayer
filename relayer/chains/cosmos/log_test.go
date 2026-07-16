package cosmos

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	typestx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetFeePayerHandlesMalformedTransactions(t *testing.T) {
	log := zap.NewNop()
	assertEmptyWithoutPanic := func(tx *typestx.Tx) {
		require.NotPanics(t, func() {
			require.Empty(t, getFeePayer(log, nil, tx))
		})
	}

	assertEmptyWithoutPanic(nil)
	assertEmptyWithoutPanic(&typestx.Tx{})
	assertEmptyWithoutPanic(&typestx.Tx{AuthInfo: &typestx.AuthInfo{}})
	assertEmptyWithoutPanic(&typestx.Tx{
		Body:     &typestx.TxBody{},
		AuthInfo: &typestx.AuthInfo{Fee: &typestx.Fee{}},
	})
	assertEmptyWithoutPanic(&typestx.Tx{
		Body: &typestx.TxBody{Messages: []*codectypes.Any{{TypeUrl: "/unknown.Msg"}}},
	})

	require.Equal(t, "cosmos1explicit", getFeePayer(log, nil, &typestx.Tx{
		AuthInfo: &typestx.AuthInfo{Fee: &typestx.Fee{Payer: "cosmos1explicit"}},
	}))
}
