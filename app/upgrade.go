package app

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	dynamicfeetypes "github.com/initia-labs/initia/x/dynamic-fee/types"
)

const upgradeName = "v1.0.0-rc.0"

// RegisterUpgradeHandlers returns upgrade handlers
func (app *InitiaApp) RegisterUpgradeHandlers(cfg module.Configurator) {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	if upgradeInfo.Name == upgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{
				dynamicfeetypes.StoreKey,
			},
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}

	app.UpgradeKeeper.SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			params := dynamicfeetypes.DefaultParams()
			params.MaxChangeRate = math.LegacyZeroDec()

			err := app.DynamicFeeKeeper.SetParams(ctx, params)
			if err != nil {
				return nil, errors.Wrap(err, "failed to set dynamic fee params")
			}
			return vm, nil
		},
	)
}
