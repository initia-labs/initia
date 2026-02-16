package upgrades

import (
	"cosmossdk.io/log"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	transferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	movekeeper "github.com/initia-labs/initia/x/move/keeper"

	marketmapkeeper "github.com/skip-mev/connect/v2/x/marketmap/keeper"
)

type InitiaApp interface {
	GetAccountKeeper() *authkeeper.AccountKeeper
	GetMoveKeeper() *movekeeper.Keeper
	GetUpgradeKeeper() *upgradekeeper.Keeper
	GetMarketMapKeeper() *marketmapkeeper.Keeper
	GetTransferKeeper() *transferkeeper.Keeper
	GetIBCKeeper() *ibckeeper.Keeper

	GetLogger() log.Logger
	GetConfigurator() module.Configurator
	GetModuleManager() *module.Manager
	SetStoreLoader(loader baseapp.StoreLoader)
}
