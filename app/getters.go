package app

import (
	"cosmossdk.io/log"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	ibctestingtypes "github.com/initia-labs/initia/x/ibc/testing/types"
	icaauthkeeper "github.com/initia-labs/initia/x/intertx/keeper"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"

	ophostkeeper "github.com/initia-labs/OPinit/x/ophost/keeper"

	marketmapkeeper "github.com/skip-mev/connect/v2/x/marketmap/keeper"
)

// GetLogger returns the logger for the app.
func (app *InitiaApp) GetLogger() log.Logger {
	return app.Logger()
}

// GetBaseApp returns the base app for the app.
func (app *InitiaApp) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

// GetAccountKeeper returns the account keeper for the app.
func (app *InitiaApp) GetAccountKeeper() *authkeeper.AccountKeeper {
	return app.AccountKeeper
}

// GetStakingKeeper implements the TestingApp interface.
func (app *InitiaApp) GetStakingKeeper() ibctestingtypes.StakingKeeper {
	return app.StakingKeeper
}

// GetMoveKeeper returns the move keeper for the app.
func (app *InitiaApp) GetMoveKeeper() *movekeeper.Keeper {
	return app.MoveKeeper
}

// GetUpgradeKeeper returns the upgrade keeper for the app.
func (app *InitiaApp) GetUpgradeKeeper() *upgradekeeper.Keeper {
	return app.UpgradeKeeper
}

// GetIBCKeeper returns the ibc keeper for the app.
func (app *InitiaApp) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// GetICAControllerKeeper returns the ica controller keeper for the app.
func (app *InitiaApp) GetICAControllerKeeper() *icacontrollerkeeper.Keeper {
	return app.ICAControllerKeeper
}

// GetICAAuthKeeper returns the ica auth keeper for the app.
func (app *InitiaApp) GetICAAuthKeeper() *icaauthkeeper.Keeper {
	return app.ICAAuthKeeper
}

// GetScopedIBCKeeper returns the scoped ibc keeper for the app.
func (app *InitiaApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

// TxConfig returns the tx config for the app.
func (app *InitiaApp) TxConfig() client.TxConfig {
	return app.txConfig
}

// GetConfigurator returns the configurator for the app.
func (app *InitiaApp) GetConfigurator() module.Configurator {
	return app.configurator
}

// GetModuleManager returns the module manager for the app.
func (app *InitiaApp) GetModuleManager() *module.Manager {
	return app.ModuleManager
}

// CheckStateContextGetter returns a function that returns a new Context for state checking.
func (app *InitiaApp) CheckStateContextGetter() func() sdk.Context {
	return func() sdk.Context {
		return app.GetContextForCheckTx(nil)
	}
}

// GetTransferKeeper returns the IBC transfer keeper for the app.
func (app *InitiaApp) GetTransferKeeper() *ibctransferkeeper.Keeper {
	return app.TransferKeeper
}

// GetOPHostKeeper returns the ophost keeper for the app.
func (app *InitiaApp) GetOPHostKeeper() *ophostkeeper.Keeper {
	return app.OPHostKeeper
}

// GetMarketMapKeeper returns the marketmap keeper for the app.
func (app *InitiaApp) GetMarketMapKeeper() *marketmapkeeper.Keeper {
	return app.MarketMapKeeper
}
