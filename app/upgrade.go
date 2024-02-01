package app

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// UpgradeHandler h for software upgrade proposal
type UpgradeHandler struct {
	*InitiaApp
}

// NewUpgradeHandler return new instance of UpgradeHandler
func NewUpgradeHandler(app *InitiaApp) UpgradeHandler {
	return UpgradeHandler{app}
}

func (h UpgradeHandler) CreateUpgradeHandler() upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {

		// remove fetchprice ibc module from version map
		delete(vm, "fetchprice")

		return h.ModuleManager.RunMigrations(ctx, h.configurator, vm)
	}
}
