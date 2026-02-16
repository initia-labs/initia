package v1_4_0

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"github.com/initia-labs/initia/app/upgrades"
	movetypes "github.com/initia-labs/initia/x/move/types"

	vmapi "github.com/initia-labs/movevm/api"
	vmprecom "github.com/initia-labs/movevm/precompile"
	vmtypes "github.com/initia-labs/movevm/types"
)

const upgradeName = "v1.4.0"

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

			// update modules
			moduleBytesArray, err := vmprecom.ReadStdlib()
			if err != nil {
				return nil, err
			}

			var modules []vmtypes.Module
			for _, module := range moduleBytesArray {
				// initiation-2 network upgrade, skip minitswap.move module
				if sdk.UnwrapSDKContext(ctx).ChainID() == "initiation-2" {
					_, name, err := vmapi.ReadModuleInfo(module)
					if err != nil {
						return nil, err
					}
					if name == "minitswap" {
						continue
					}
				}

				modules = append(modules, vmtypes.NewModule(module))
			}

			err = app.GetMoveKeeper().PublishModuleBundle(ctx, vmtypes.StdAddress, vmtypes.NewModuleBundle(modules...), movetypes.UpgradePolicy_COMPATIBLE)
			if err != nil {
				return nil, err
			}

			return vm, nil
		},
	)
}

func updateTotalEscrowAmount(ctx context.Context, app upgrades.InitiaApp) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	totalEscrows := sdk.NewCoins()

	// update total escrow amount by iterating all ibc channels
	var err error
	app.GetIBCKeeper().ChannelKeeper.IterateChannels(sdkCtx, func(channel channeltypes.IdentifiedChannel) bool {
		if channel.PortId != transfertypes.PortID {
			return false
		}

		escrowAddr := transfertypes.GetEscrowAddress(channel.PortId, channel.ChannelId)
		err = app.GetMoveKeeper().MoveBankKeeper().IterateAccountBalances(ctx, escrowAddr, func(c sdk.Coin) (bool, error) {
			totalEscrows = totalEscrows.Add(c)
			return false, nil
		})
		if err != nil {
			return true
		}

		return false
	})
	if err != nil {
		return err
	}

	for _, coin := range totalEscrows {
		app.GetTransferKeeper().SetTotalEscrowForDenom(sdkCtx, coin)
	}

	return nil
}
