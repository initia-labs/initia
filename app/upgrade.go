package app

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/types"
	icqtypes "github.com/cosmos/ibc-apps/modules/async-icq/v8/types"

	fetchpricetypes "github.com/initia-labs/initia/x/ibc/fetchprice/types"

	alertstypes "github.com/skip-mev/slinky/x/alerts/types"
	incentivestypes "github.com/skip-mev/slinky/x/incentives/types"
)

const upgradeName = "0.2.0-beta.7"

// RegisterUpgradeHandlers returns upgrade handlers
func (app *InitiaApp) RegisterUpgradeHandlers(cfg module.Configurator) {
	app.UpgradeKeeper.SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {

			// remove fetchprice ibc module from version map
			delete(vm, "fetchprice")

			return app.ModuleManager.RunMigrations(ctx, app.configurator, vm)
		},
	)

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	if upgradeInfo.Name == upgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{
				fetchpricetypes.StoreKey,
				packetforwardtypes.StoreKey,
				icqtypes.StoreKey,
			},
			Deleted: []string{
				"fetchpriceconsumer",
				"fetchpriceprovider",
				alertstypes.StoreKey,
				incentivestypes.StoreKey,
			},
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
