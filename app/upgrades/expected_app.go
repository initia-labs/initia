package upgrades

import (
	"cosmossdk.io/log"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	ratelimitkeeper "github.com/cosmos/ibc-apps/modules/rate-limiting/v8/keeper"
	transferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	govkeeper "github.com/initia-labs/initia/x/gov/keeper"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"

	ophostkeeper "github.com/initia-labs/OPinit/x/ophost/keeper"

	forwardingkeeper "github.com/noble-assets/forwarding/v2/keeper"

	marketmapkeeper "github.com/skip-mev/connect/v2/x/marketmap/keeper"
)

type InitiaApp interface {
	GetAccountKeeper() *authkeeper.AccountKeeper
	GetMoveKeeper() *movekeeper.Keeper
	GetUpgradeKeeper() *upgradekeeper.Keeper
	GetOPHostKeeper() *ophostkeeper.Keeper
	GetMarketMapKeeper() *marketmapkeeper.Keeper
	GetTransferKeeper() *transferkeeper.Keeper
	GetIBCKeeper() *ibckeeper.Keeper
	GetForwardingKeeper() *forwardingkeeper.Keeper
	GetGovKeeper() *govkeeper.Keeper
	GetRatelimitKeeper() *ratelimitkeeper.Keeper

	GetLogger() log.Logger
	GetConfigurator() module.Configurator
	GetModuleManager() *module.Manager
	SetStoreLoader(loader baseapp.StoreLoader)
}

const (
	MainnetChainID = "interwoven-1"
	TestnetChainID = "initiation-2"
)
