package upgrades

import (
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	movekeeper "github.com/initia-labs/initia/x/move/keeper"
)

type InitiaApp interface {
	GetAccountKeeper() *authkeeper.AccountKeeper
	GetMoveKeeper() *movekeeper.Keeper
	GetUpgradeKeeper() *upgradekeeper.Keeper

	GetConfigurator() module.Configurator
	GetModuleManager() *module.Manager
}
