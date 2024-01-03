package app

import (
	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	"github.com/initia-labs/initia/app/params"
	moveconfig "github.com/initia-labs/initia/x/move/config"
)

func MakeEncodingConfig() params.EncodingConfig {
	tempApp := NewInitiaApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, moveconfig.DefaultMoveConfig(), simtestutil.EmptyAppOptions{})
	encodingConfig := params.EncodingConfig{
		InterfaceRegistry: tempApp.InterfaceRegistry(),
		Codec:             tempApp.AppCodec(),
		TxConfig:          tempApp.TxConfig(),
		Amino:             tempApp.LegacyAmino(),
	}

	return encodingConfig
}

func AutoCliOpts() autocli.AppOptions {
	tempApp := NewInitiaApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, moveconfig.DefaultMoveConfig(), simtestutil.EmptyAppOptions{})
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

func BasicManager() module.BasicManager {
	tempApp := NewInitiaApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, moveconfig.DefaultMoveConfig(), simtestutil.EmptyAppOptions{})
	return tempApp.BasicModuleManager
}
