// Package ics29 preserves the legacy ICS-29 transaction wire contract after
// the module was removed from ibc-go v11.
package ics29

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the legacy ICS-29 message names.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgPayPacketFee{}, "cosmos-sdk/MsgPayPacketFee")
	legacy.RegisterAminoMsg(cdc, &MsgPayPacketFeeAsync{}, "cosmos-sdk/MsgPayPacketFeeAsync")
	legacy.RegisterAminoMsg(cdc, &MsgRegisterPayee{}, "cosmos-sdk/MsgRegisterPayee")
	legacy.RegisterAminoMsg(cdc, &MsgRegisterCounterpartyPayee{}, "cosmos-sdk/MsgRegisterCounterpartyPayee")
}

// RegisterInterfaces registers the legacy messages under their original type
// URLs. This is an off-chain compatibility codec, not an SDK module keeper.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgPayPacketFee{},
		&MsgPayPacketFeeAsync{},
		&MsgRegisterPayee{},
		&MsgRegisterCounterpartyPayee{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
