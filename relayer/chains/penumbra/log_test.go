package penumbra

import (
	"bytes"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typestx "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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

	require.Equal(t, "penumbra1explicit", getFeePayer(log, nil, &typestx.Tx{
		AuthInfo: &typestx.AuthInfo{Fee: &typestx.Fee{Payer: "penumbra1explicit"}},
	}))
}

func TestLogSuccessTxUsesProviderInterfaceRegistry(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	provider := &PenumbraProvider{
		log:   zap.New(core),
		PCfg:  PenumbraProviderConfig{ChainID: "penumbra-test"},
		Codec: makeCodec(moduleBasics, nil, "penumbra"),
	}
	message, err := codectypes.NewAnyWithValue(&transfertypes.MsgTransfer{Sender: "penumbra1sender"})
	require.NoError(t, err)
	txAny, err := codectypes.NewAnyWithValue(&typestx.Tx{
		Body: &typestx.TxBody{Messages: []*codectypes.Any{message}},
		AuthInfo: &typestx.AuthInfo{Fee: &typestx.Fee{
			Payer: "penumbra1fee",
		}},
	})
	require.NoError(t, err)

	provider.LogSuccessTx(&sdk.TxResponse{Tx: txAny, TxHash: "AABBCC"}, nil)

	entries := observed.FilterField(zap.String("fee_payer", "penumbra1fee")).All()
	require.Len(t, entries, 1)
	require.Equal(t, "Successful transaction", entries[0].Message)
}

func TestGetFeePayerUsesConfiguredAddressCodec(t *testing.T) {
	providerCodec := makeCodec(moduleBasics, nil, "penumbra")
	fromAddress, err := address.NewBech32Codec("penumbra").BytesToString(bytes.Repeat([]byte{1}, 20))
	require.NoError(t, err)
	message, err := codectypes.NewAnyWithValue(&banktypes.MsgSend{FromAddress: fromAddress})
	require.NoError(t, err)
	tx := &typestx.Tx{
		Body:     &typestx.TxBody{Messages: []*codectypes.Any{message}},
		AuthInfo: &typestx.AuthInfo{Fee: &typestx.Fee{}},
	}

	payer := getFeePayer(zap.NewNop(), codec.NewProtoCodec(providerCodec.InterfaceRegistry), tx)
	require.Equal(t, fromAddress, payer)
}

func TestLogSuccessTxHandlesNilResponse(t *testing.T) {
	provider := &PenumbraProvider{log: zap.NewNop()}
	require.NotPanics(t, func() { provider.LogSuccessTx(nil, nil) })
}

func TestMsgRegisterCounterpartyPayeeReturnsUnsupportedError(t *testing.T) {
	provider := &PenumbraProvider{}
	message, err := provider.MsgRegisterCounterpartyPayee("transfer", "channel-0", "relayer", "counterparty")
	require.Nil(t, message)
	require.ErrorContains(t, err, "does not support ICS-29")
}
