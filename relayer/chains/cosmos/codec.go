package cosmos

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/contrib/x/crisis"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authz "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant/module"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/cosmos/cosmos-sdk/x/mint"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/ibc-go/v11/modules/apps/transfer"
	ibc "github.com/cosmos/ibc-go/v11/modules/core"

	"github.com/cosmos/relayer/v2/relayer/chains/cosmos/keys/sr25519"
	cosmosmodule "github.com/cosmos/relayer/v2/relayer/chains/cosmos/module"
	"github.com/cosmos/relayer/v2/relayer/chains/cosmos/stride"
	ethermintcodecs "github.com/cosmos/relayer/v2/relayer/codecs/ethermint"
	ics29codecs "github.com/cosmos/relayer/v2/relayer/codecs/ics29"
	injectivecodecs "github.com/cosmos/relayer/v2/relayer/codecs/injective"
)

var ModuleBasics = []module.AppModuleBasic{
	auth.AppModuleBasic{},
	authz.AppModuleBasic{},
	bank.AppModuleBasic{},
	// TODO: add osmosis governance proposal types here
	// TODO: add other proposal types here
	gov.NewAppModuleBasic(
		[]govclient.ProposalHandler{
			paramsclient.ProposalHandler,
		},
	),
	crisis.AppModuleBasic{},
	distribution.AppModuleBasic{},
	feegrant.AppModuleBasic{},
	mint.AppModuleBasic{},
	params.AppModuleBasic{},
	slashing.AppModuleBasic{},
	staking.AppModuleBasic{},
	upgrade.AppModuleBasic{},
	transfer.AppModuleBasic{},
	ibc.AppModuleBasic{},
	cosmosmodule.AppModuleBasic{},
	stride.AppModuleBasic{},
}

type Codec struct {
	InterfaceRegistry types.InterfaceRegistry
	Marshaler         codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

func MakeCodec(moduleBasics []module.AppModuleBasic, extraCodecs []string, accBech32Prefix, valBech32Prefix string) Codec {
	modBasic := module.NewBasicManager(moduleBasics...)
	encodingConfig := MakeCodecConfig(accBech32Prefix, valBech32Prefix)
	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	modBasic.RegisterLegacyAminoCodec(encodingConfig.Amino)
	modBasic.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ics29codecs.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ics29codecs.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	sr25519.RegisterLegacyAminoCodec(encodingConfig.Amino)
	sr25519.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	for _, c := range extraCodecs {
		switch c {
		case "ethermint":
			ethermintcodecs.RegisterInterfaces(encodingConfig.InterfaceRegistry)
			encodingConfig.Amino.RegisterConcrete(&ethermintcodecs.PubKey{}, ethermintcodecs.PubKeyName, nil)
			encodingConfig.Amino.RegisterConcrete(&ethermintcodecs.PrivKey{}, ethermintcodecs.PrivKeyName, nil)
		case "injective":
			injectivecodecs.RegisterInterfaces(encodingConfig.InterfaceRegistry)
			encodingConfig.Amino.RegisterConcrete(&injectivecodecs.PubKey{}, injectivecodecs.PubKeyName, nil)
			encodingConfig.Amino.RegisterConcrete(&injectivecodecs.PrivKey{}, injectivecodecs.PrivKeyName, nil)
		}
	}

	return encodingConfig
}

func MakeCodecConfig(accBech32Prefix, valBech32Prefix string) Codec {
	interfaceRegistry, err := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          address.NewBech32Codec(accBech32Prefix),
			ValidatorAddressCodec: address.NewBech32Codec(valBech32Prefix),
		},
	})
	if err != nil {
		panic(err)
	}
	marshaler := codec.NewProtoCodec(interfaceRegistry)

	done := SetSDKConfigContext(accBech32Prefix)
	defer done()

	return Codec{
		InterfaceRegistry: interfaceRegistry,
		Marshaler:         marshaler,
		TxConfig:          tx.NewTxConfig(marshaler, tx.DefaultSignModes),
		Amino:             codec.NewLegacyAmino(),
	}
}
