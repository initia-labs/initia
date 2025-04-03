package app

import (
	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client/flags"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	"github.com/initia-labs/initia/app/params"

	cryptocodec "github.com/initia-labs/initia/crypto/codec"
)

func MakeEncodingConfig() params.EncodingConfig {
	encodingConfig := params.MakeEncodingConfig()

	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	cryptocodec.RegisterLegacyAminoCodec(encodingConfig.Amino)
	cryptocodec.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	basicManager := NewBasicManager()
	basicManager.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	basicManager.RegisterLegacyAminoCodec(encodingConfig.Amino)
	return encodingConfig
}

func AutoCliOpts(encodingConfig params.EncodingConfig) autocli.AppOptions {
	appModules := make(map[string]appmodule.AppModule, 0)
	moduleOptions := make(map[string]interface{}, 0)

	modules := modulesForAutoCli(encodingConfig.Codec, encodingConfig.TxConfig, encodingConfig.InterfaceRegistry, authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()), authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()))

	for _, m := range modules {
		if moduleWithName, ok := m.(module.HasName); ok {
			moduleName := moduleWithName.Name()
			if _, ok := m.(interface {
				AutoCLIOptions() *autocliv1.ModuleOptions
			}); ok {
				moduleOptions[moduleName] = m
			} else {
				continue
			}

			if appModule, ok := moduleWithName.(appmodule.AppModule); ok {
				appModules[moduleName] = appModule
			}
		}
	}

	return autocli.AppOptions{
		Modules:               appModules,
		ModuleOptions:         runtimeservices.ExtractAutoCLIOptions(moduleOptions),
		AddressCodec:          authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}

// EmptyAppOptions is a stub implementing AppOptions
type EmptyAppOptions struct{}

// Get implements AppOptions
func (ao EmptyAppOptions) Get(o string) interface{} {
	if o == flags.FlagHome {
		return DefaultNodeHome
	}

	return nil
}
