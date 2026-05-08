package v1_5_0

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/initia-labs/initia/app/upgrades"
)

const upgradeName = "v1.5.0"

// RegisterUpgradeHandlers registers the v1.5.0 upgrade.
//
// v1.5.0 migrates the chain to ibc-go v10. The upgrade:
//   - Deletes the legacy `capability`, `feeibc` (29-fee), and `crisis` module
//     stores. capability + feeibc were removed in ibc-go v10, crisis was
//     removed in cosmos-sdk v0.53.
func RegisterUpgradeHandlers(app upgrades.InitiaApp) {
	if upgradeInfo, err := app.GetUpgradeKeeper().ReadUpgradeInfoFromDisk(); err == nil {
		if upgradeInfo.Name == upgradeName && !app.GetUpgradeKeeper().IsSkipHeight(upgradeInfo.Height) {
			storeUpgrades := storetypes.StoreUpgrades{
				Deleted: []string{"capability", "feeibc", "crisis"},
			}
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
		}
	}

	app.GetUpgradeKeeper().SetUpgradeHandler(
		upgradeName,
		func(_ context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return vm, nil
		},
	)
}
