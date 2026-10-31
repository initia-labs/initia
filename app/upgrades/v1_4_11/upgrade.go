package v1_4_11

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	abci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/initia-labs/initia/app/upgrades"
)

const UpgradeName = "v1.4.11"

// The chain halted at 23365925. No upgrade plan can be stored in a halted chain, so the
// first block the new binary produces applies the handler itself (see WrapPreBlocker).
var (
	RestartChainID       = upgrades.MainnetChainID
	RestartHeight  int64 = 23365926
)

// RegisterUpgradeHandlers returns upgrade handlers
func RegisterUpgradeHandlers(app upgrades.InitiaApp) {
	app.GetUpgradeKeeper().SetUpgradeHandler(
		UpgradeName,
		func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			if err := run(ctx, app); err != nil {
				return nil, err
			}

			return vm, nil
		},
	)
}

// WrapPreBlocker applies the upgrade at RestartHeight on RestartChainID before the module
// pre-blockers run, once. ApplyUpgrade records the done height and clears the plan, so the
// x/upgrade pre-blocker that follows sees nothing to do.
func WrapPreBlocker(app upgrades.InitiaApp, next sdk.PreBlocker) sdk.PreBlocker {
	return func(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
		if ctx.ChainID() == RestartChainID && ctx.BlockHeight() == RestartHeight {
			done, err := app.GetUpgradeKeeper().GetDoneHeight(ctx, UpgradeName)
			if err != nil {
				return nil, err
			}

			if done == 0 {
				if !app.GetUpgradeKeeper().HasHandler(UpgradeName) {
					return nil, fmt.Errorf("upgrade handler %s is not registered", UpgradeName)
				}

				ctx.Logger().Info("applying restart upgrade", "name", UpgradeName, "height", RestartHeight)
				if err := app.GetUpgradeKeeper().ApplyUpgrade(ctx, upgradetypes.Plan{
					Name:   UpgradeName,
					Height: RestartHeight,
					Info:   "interwoven-1 restart after the 2026-10-23 halt",
				}); err != nil {
					return nil, err
				}
			}
		}

		return next(ctx, req)
	}
}
