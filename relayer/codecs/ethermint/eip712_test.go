package ethermint

import (
	"bytes"
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	bank "github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"
)

func TestGetEIP712TypedDataForMsgReportsBothDecodeErrors(t *testing.T) {
	originalProtoCodec, originalAminoCodec := protoCodec, aminoCodec
	protoCodec, aminoCodec = nil, nil
	t.Cleanup(func() {
		protoCodec, aminoCodec = originalProtoCodec, originalAminoCodec
	})

	_, err := GetEIP712TypedDataForMsg([]byte("not a sign doc"))
	require.EqualError(t, err, "could not decode sign doc as either Amino or Protobuf.\n amino: missing codec: codecs have not been properly initialized using SetEncodingConfig\n protobuf: missing codec: codecs have not been properly initialized using SetEncodingConfig")
}

func TestIsValidEIP712Payload(t *testing.T) {
	valid := apitypes.TypedData{
		Types:       apitypes.Types{"Tx": {}},
		PrimaryType: "Tx",
		Domain: apitypes.TypedDataDomain{
			Name:    "Cosmos Web3",
			ChainId: math.NewHexOrDecimal256(1),
		},
		Message: apitypes.TypedDataMessage{"memo": ""},
	}

	tests := map[string]struct {
		mutate func(apitypes.TypedData) apitypes.TypedData
		want   bool
	}{
		"valid":         {mutate: func(data apitypes.TypedData) apitypes.TypedData { return data }, want: true},
		"empty message": {mutate: withoutTypedDataMessage, want: false},
		"empty types":   {mutate: withoutTypedDataTypes, want: false},
		"empty primary": {mutate: withoutTypedDataPrimaryType, want: false},
		"empty domain":  {mutate: withoutTypedDataDomain, want: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, isValidEIP712Payload(tc.mutate(valid)))
		})
	}
}

func TestValidatePayloadMessagesEmpty(t *testing.T) {
	require.EqualError(t, validatePayloadMessages(nil), "unable to build EIP-712 payload: transaction does contain any messages")
}

func TestValidatePayloadMessages(t *testing.T) {
	setTestCodecs(t)

	addressA := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	addressB := sdk.AccAddress(bytes.Repeat([]byte{2}, 20))
	addressC := sdk.AccAddress(bytes.Repeat([]byte{3}, 20))
	coins := sdk.NewCoins(sdk.NewInt64Coin("stake", 10))
	fromA := banktypes.NewMsgSend(addressA, addressB, coins)
	fromAAgain := banktypes.NewMsgSend(addressA, addressC, coins)
	fromB := banktypes.NewMsgSend(addressB, addressC, coins)
	multiSend := banktypes.NewMsgMultiSend(
		banktypes.Input{Address: addressA.String(), Coins: coins},
		[]banktypes.Output{{Address: addressB.String(), Coins: coins}},
	)
	multipleSigners := &banktypes.MsgMultiSend{
		Inputs: []banktypes.Input{
			{Address: addressA.String(), Coins: coins},
			{Address: addressB.String(), Coins: coins},
		},
		Outputs: []banktypes.Output{{Address: addressC.String(), Coins: coins.Add(coins...)}},
	}

	tests := map[string]struct {
		msgs    []sdk.Msg
		wantErr string
	}{
		"one message":          {msgs: []sdk.Msg{fromA}},
		"same type and signer": {msgs: []sdk.Msg{fromA, fromAAgain}},
		"different type": {
			msgs:    []sdk.Msg{fromA, multiSend},
			wantErr: "unable to build EIP-712 payload: different types of messages detected",
		},
		"different signer": {
			msgs:    []sdk.Msg{fromA, fromB},
			wantErr: "unable to build EIP-712 payload: multiple signers detected",
		},
		"multiple signers": {
			msgs:    []sdk.Msg{multipleSigners},
			wantErr: "unable to build EIP-712 payload: expect exactly 1 signer",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validatePayloadMessages(tc.msgs)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tc.wantErr)
		})
	}
}

func TestDecodeProtobufSignDocGolden(t *testing.T) {
	setTestCodecs(t)

	from := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	to := sdk.AccAddress(bytes.Repeat([]byte{2}, 20))
	msg := banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin("stake", 10)))
	typedData, err := decodeProtobufSignDoc(protobufSignDocBytes(t, msg))

	require.NoError(t, err)
	require.Equal(t, "Tx", typedData.PrimaryType)
	require.Equal(t, "Cosmos Web3", typedData.Domain.Name)
	chainID, err := typedData.Domain.ChainId.MarshalText()
	require.NoError(t, err)
	require.Equal(t, "0x2329", string(chainID))
	require.Equal(t, apitypes.TypedDataMessage{
		"account_number": "7",
		"chain_id":       "evmos_9001-2",
		"fee": map[string]interface{}{
			"amount":   []interface{}{map[string]interface{}{"amount": "10", "denom": "stake"}},
			"feePayer": from.String(),
			"gas":      "200000",
		},
		"memo": "golden",
		"msgs": []interface{}{map[string]interface{}{
			"type": "cosmos-sdk/MsgSend",
			"value": map[string]interface{}{
				"amount":       []interface{}{map[string]interface{}{"amount": "10", "denom": "stake"}},
				"from_address": from.String(),
				"to_address":   to.String(),
			},
		}},
		"sequence": "3",
	}, typedData.Message)
}

func TestDecodeProtobufSignDocRejectsMissingFee(t *testing.T) {
	setTestCodecs(t)

	from := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	to := sdk.AccAddress(bytes.Repeat([]byte{2}, 20))
	msg := banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin("stake", 10)))
	packedMsg, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	bodyBytes, err := (&txtypes.TxBody{Messages: []*codectypes.Any{packedMsg}}).Marshal()
	require.NoError(t, err)
	authInfoBytes, err := (&txtypes.AuthInfo{SignerInfos: []*txtypes.SignerInfo{{Sequence: 3}}}).Marshal()
	require.NoError(t, err)
	signDocBytes, err := (&txtypes.SignDoc{
		BodyBytes: bodyBytes, AuthInfoBytes: authInfoBytes, ChainId: "evmos_9001-2",
	}).Marshal()
	require.NoError(t, err)

	_, err = decodeProtobufSignDoc(signDocBytes)
	require.EqualError(t, err, "auth info fee is required")
}

func TestGetMsgTypeRequiresInitializedAminoCodec(t *testing.T) {
	originalAminoCodec := aminoCodec
	aminoCodec = nil
	t.Cleanup(func() {
		aminoCodec = originalAminoCodec
	})

	require.Panics(t, func() {
		_, _ = getMsgType(&banktypes.MsgSend{})
	})
}

func withoutTypedDataMessage(data apitypes.TypedData) apitypes.TypedData {
	data.Message = nil
	return data
}

func withoutTypedDataTypes(data apitypes.TypedData) apitypes.TypedData {
	data.Types = nil
	return data
}

func withoutTypedDataPrimaryType(data apitypes.TypedData) apitypes.TypedData {
	data.PrimaryType = ""
	return data
}

func withoutTypedDataDomain(data apitypes.TypedData) apitypes.TypedData {
	data.Domain = apitypes.TypedDataDomain{}
	return data
}

func setTestCodecs(t *testing.T) {
	t.Helper()
	originalProtoCodec, originalAminoCodec := protoCodec, aminoCodec
	originalRegressionCodec := legacytx.RegressionTestingAminoCodec
	encodingConfig := moduletestutil.MakeTestEncodingConfig(bank.AppModuleBasic{})
	protoCodec, aminoCodec = encodingConfig.Codec, encodingConfig.Amino
	legacytx.RegressionTestingAminoCodec = encodingConfig.Amino
	t.Cleanup(func() {
		protoCodec, aminoCodec = originalProtoCodec, originalAminoCodec
		legacytx.RegressionTestingAminoCodec = originalRegressionCodec
	})
}

func protobufSignDocBytes(t *testing.T, msg sdk.Msg) []byte {
	t.Helper()
	packedMsg, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)

	bodyBytes, err := (&txtypes.TxBody{Messages: []*codectypes.Any{packedMsg}, Memo: "golden"}).Marshal()
	require.NoError(t, err)
	authInfoBytes, err := (&txtypes.AuthInfo{
		SignerInfos: []*txtypes.SignerInfo{{Sequence: 3}},
		Fee: &txtypes.Fee{
			Amount:   sdk.NewCoins(sdk.NewInt64Coin("stake", 10)),
			GasLimit: 200000,
		},
	}).Marshal()
	require.NoError(t, err)

	signDocBytes, err := (&txtypes.SignDoc{
		BodyBytes:     bodyBytes,
		AuthInfoBytes: authInfoBytes,
		ChainId:       "evmos_9001-2",
		AccountNumber: 7,
	}).Marshal()
	require.NoError(t, err)
	return signDocBytes
}
