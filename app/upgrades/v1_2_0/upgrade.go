package v1_2_0

import (
	"context"
	"errors"
	"slices"

	"cosmossdk.io/collections"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/initia-labs/initia/app/upgrades"
	movetypes "github.com/initia-labs/initia/x/move/types"

	vmprecom "github.com/initia-labs/movevm/precompile"
	vmtypes "github.com/initia-labs/movevm/types"

	marketmapkeeper "github.com/skip-mev/connect/v2/x/marketmap/keeper"
	marketmaptypes "github.com/skip-mev/connect/v2/x/marketmap/types"
)

const upgradeName = "v1.2.0"

// RegisterUpgradeHandlers returns upgrade handlers
func RegisterUpgradeHandlers(app upgrades.InitiaApp) {
	app.GetUpgradeKeeper().SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			moduleBytesArray, err := vmprecom.ReadStdlib()
			if err != nil {
				return nil, err
			}

			var modules []vmtypes.Module
			for _, module := range moduleBytesArray {
				modules = append(modules, vmtypes.NewModule(module))
			}

			err = app.GetMoveKeeper().PublishModuleBundle(ctx, vmtypes.StdAddress, vmtypes.NewModuleBundle(modules...), movetypes.UpgradePolicy_COMPATIBLE)
			if err != nil {
				return nil, err
			}

			err = updateMarketMap(ctx, app.GetMarketMapKeeper())
			if err != nil {
				return nil, err
			}

			return vm, nil
		},
	)
}

func updateMarketMap(ctx context.Context, k *marketmapkeeper.Keeper) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	authorities := params.GetMarketAuthorities()
	if len(authorities) == 0 {
		return nil
	}
	market, err := k.GetMarket(ctx, "USDC/USD")
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	providerExists := false
	market.ProviderConfigs = slices.DeleteFunc(market.ProviderConfigs, func(cfg marketmaptypes.ProviderConfig) bool {
		providerExists = providerExists || cfg.Name == "kraken_api"
		return cfg.Name == "kraken_api"
	})
	if !providerExists {
		return nil
	}

	ms := marketmapkeeper.NewMsgServer(k)
	_, err = ms.UpdateMarkets(ctx, &marketmaptypes.MsgUpdateMarkets{
		Authority: authorities[0],
		UpdateMarkets: []marketmaptypes.Market{
			market,
		},
	})

	return err
}
