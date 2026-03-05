package upgrades

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	movetypes "github.com/initia-labs/initia/x/move/types"

	vmapi "github.com/initia-labs/movevm/api"
	vmprecom "github.com/initia-labs/movevm/precompile"
	vmtypes "github.com/initia-labs/movevm/types"
)

// fetch latest modules and publish
func UpgradeMoveModules(ctx context.Context, app InitiaApp) error {
	// update modules
	moduleBytesArray, err := vmprecom.ReadStdlib()
	if err != nil {
		return err
	}

	var modules []vmtypes.Module
	for _, module := range moduleBytesArray {
		// initiation-2 network upgrade, skip minitswap.move module
		if sdk.UnwrapSDKContext(ctx).ChainID() == TestnetChainID {
			_, name, err := vmapi.ReadModuleInfo(module)
			if err != nil {
				return err
			}
			if name == "minitswap" {
				continue
			}
		}

		modules = append(modules, vmtypes.NewModule(module))
	}

	err = app.GetMoveKeeper().PublishModuleBundle(ctx, vmtypes.StdAddress, vmtypes.NewModuleBundle(modules...), movetypes.UpgradePolicy_COMPATIBLE)
	if err != nil {
		return err
	}

	return nil
}
