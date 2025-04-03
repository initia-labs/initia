package app

import (
	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/cosmos/cosmos-sdk/client/flags"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"

	"github.com/initia-labs/initia/app/params"
	moveconfig "github.com/initia-labs/initia/x/move/config"

	oracleconfig "github.com/skip-mev/connect/v2/oracle/config"

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

func newTempApp() *InitiaApp {
	return NewInitiaApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		moveconfig.DefaultMoveConfig(),
		oracleconfig.NewDefaultAppConfig(),
		EmptyAppOptions{},
	)
}

func AutoCliOpts() autocli.AppOptions {
	tempApp := newTempApp()
	modules := make(map[string]appmodule.AppModule, 0)
	for _, m := range tempApp.ModuleManager.Modules {
		if moduleWithName, ok := m.(module.HasName); ok {
			moduleName := moduleWithName.Name()
			if appModule, ok := moduleWithName.(appmodule.AppModule); ok {
				modules[moduleName] = appModule
			}
		}
	}

	return autocli.AppOptions{
		Modules:               modules,
		ModuleOptions:         runtimeservices.ExtractAutoCLIOptions(tempApp.ModuleManager.Modules),
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
