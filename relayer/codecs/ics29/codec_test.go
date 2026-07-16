package ics29

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
)

func TestLegacyWireContract(t *testing.T) {
	testCases := []struct {
		name     string
		message  proto.Message
		typeName string
		wireHex  string
	}{
		{
			name:     "register payee",
			message:  &MsgRegisterPayee{PortId: "transfer", ChannelId: "channel-7", Relayer: "relayer", Payee: "payee"},
			typeName: "ibc.applications.fee.v1.MsgRegisterPayee",
			wireHex:  "0a087472616e7366657212096368616e6e656c2d371a0772656c6179657222057061796565",
		},
		{
			name:     "register counterparty payee",
			message:  &MsgRegisterCounterpartyPayee{PortId: "transfer", ChannelId: "channel-7", Relayer: "relayer", CounterpartyPayee: "counterparty"},
			typeName: "ibc.applications.fee.v1.MsgRegisterCounterpartyPayee",
			wireHex:  "0a087472616e7366657212096368616e6e656c2d371a0772656c61796572220c636f756e7465727061727479",
		},
		{
			name:     "pay packet fee",
			message:  &MsgPayPacketFee{SourcePortId: "transfer", SourceChannelId: "channel-7", Signer: "signer", Relayers: []string{"relayer-a", "relayer-b"}},
			typeName: "ibc.applications.fee.v1.MsgPayPacketFee",
			wireHex:  "0a0012087472616e736665721a096368616e6e656c2d3722067369676e65722a0972656c617965722d612a0972656c617965722d62",
		},
		{
			name: "pay packet fee async",
			message: &MsgPayPacketFeeAsync{
				PacketId:  chantypes.PacketId{PortId: "transfer", ChannelId: "channel-7", Sequence: 42},
				PacketFee: PacketFee{RefundAddress: "refund", Relayers: []string{"relayer-a"}},
			},
			typeName: "ibc.applications.fee.v1.MsgPayPacketFeeAsync",
			wireHex:  "0a170a087472616e7366657212096368616e6e656c2d37182a12150a001206726566756e641a0972656c617965722d61",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			wire, err := proto.Marshal(testCase.message)
			require.NoError(t, err)
			require.Equal(t, testCase.typeName, proto.MessageName(testCase.message))
			require.Equal(t, testCase.wireHex, hex.EncodeToString(wire))
			require.Equal(t, "/"+testCase.typeName, sdk.MsgTypeURL(testCase.message.(sdk.Msg)))
		})
	}
}

func TestMsgRegisterCounterpartyPayeeValidation(t *testing.T) {
	relayerAddress := sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String()
	message := NewMsgRegisterCounterpartyPayee("transfer", "channel-7", relayerAddress, "counterparty-address")
	require.NoError(t, message.ValidateBasic())
	require.Equal(t, relayerAddress, message.GetSigners()[0].String())

	message.CounterpartyPayee = "   "
	err := message.ValidateBasic()
	require.ErrorIs(t, err, ErrCounterpartyPayeeEmpty)
	codespace, code, _ := errorsmod.ABCIInfo(err, false)
	require.Equal(t, "feeibc", codespace)
	require.Equal(t, uint32(7), code)
	require.True(t, errors.Is(err, ErrCounterpartyPayeeEmpty))
	message.CounterpartyPayee = strings.Repeat("a", maximumCounterpartyPayeeLength+1)
	require.Error(t, message.ValidateBasic())
}
