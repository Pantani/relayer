package sr25519

import (
	"github.com/cosmos/cosmos-sdk/codec"
	legacycodec "github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

func init() {
	// Private-key armor uses the SDK singleton rather than the provider codec.
	// Register both concrete names so existing sr25519 armor remains portable.
	RegisterLegacyAminoCodec(legacycodec.Cdc)
}

// RegisterLegacyAminoCodec preserves the key names used by existing keyrings.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&PubKey{}, PubKeyName, nil)
	cdc.RegisterConcrete(&PrivKey{}, PrivKeyName, nil)
}

// RegisterInterfaces allows persisted protobuf Any key records to be reopened.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*cryptotypes.PubKey)(nil), &PubKey{})
	registry.RegisterImplementations((*cryptotypes.PrivKey)(nil), &PrivKey{})
}
