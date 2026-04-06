package v1_4_0

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/initia-labs/initia/app/upgrades"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
)

const upgradeName = "v1.4.2"

// RegisterUpgradeHandlers returns upgrade handlers
func RegisterUpgradeHandlers(app upgrades.InitiaApp) {

	// apply store upgrade only if this upgrade is scheduled at a height
	if upgradeInfo, err := app.GetUpgradeKeeper().ReadUpgradeInfoFromDisk(); err == nil {
		if upgradeInfo.Name == upgradeName && !app.GetUpgradeKeeper().IsSkipHeight(upgradeInfo.Height) {
			storeUpgrades := storetypes.StoreUpgrades{
				Deleted: []string{"auction"},
			}

			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
		}
	}

	app.GetUpgradeKeeper().SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			if err := updateTotalEscrowAmount(ctx, app); err != nil {
				return nil, err
			}

			// setup clamm contract address
			if err := setupClammModuleAddress(ctx, app); err != nil {
				return nil, err
			}

			// bind the opinit IBC port for ophost module
			if !app.GetOPHostKeeper().IsBound(ctx, ophosttypes.PortID) {
				if err := app.GetOPHostKeeper().BindPort(ctx, ophosttypes.PortID); err != nil {
					return nil, err
				}
			}

			// update modules (skip)
			// if err := upgrades.UpgradeMoveModules(ctx, app); err != nil {
			// 	return nil, err
			// }

			return vm, nil
		},
	)
}
