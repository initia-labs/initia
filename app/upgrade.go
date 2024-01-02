package app

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return h.mm.RunMigrations(ctx, h.configurator, vm)
	}
}
