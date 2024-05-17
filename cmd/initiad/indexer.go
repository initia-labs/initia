package main

import (
	"context"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/initia-labs/indexer"
	indexercfg "github.com/initia-labs/indexer/config"
	"github.com/initia-labs/indexer/service/collector"
	"github.com/initia-labs/indexer/service/cron/validator"
	initiaapp "github.com/initia-labs/initia/app"
)

const (
	FlagIndexer = "indexer"
)

func addIndexFlag(cmd *cobra.Command) {
	indexercfg.AddIndexerFlag(cmd)
}

func preSetupIndexer(svrCtx *server.Context, clientCtx client.Context, ctx context.Context, g *errgroup.Group, _app types.Application) error {
	app := _app.(*initiaapp.InitiaApp)

	// listen auth key: need only auth key for now
	keysToListen := []storetypes.StoreKey{app.GetKey(authtypes.StoreKey)}

	app.CommitMultiStore().AddListeners(keysToListen)

	// if indexer is disabled, it returns nil
	idxer, err := indexer.NewIndexer(svrCtx.Viper, app)
	if err != nil {
		return err
	}
	// if idxer is nil, it means indexer is disabled
	if idxer == nil {
		return nil
	}

	idxer.GetHandler().RegisterService(collector.CollectorSvc)
	idxer.GetHandler().RegisterCronjob(validator.Tag, validator.Expr, validator.JobInit, validator.JobFunc)

	err = idxer.Validate()
	if err != nil {
		return err
	}

	err = idxer.Start(nil)
	if err != nil {
		return err
	}

	streamingManager := storetypes.StreamingManager{
		ABCIListeners: []storetypes.ABCIListener{idxer},
		StopNodeOnErr: true,
	}
	app.SetStreamingManager(streamingManager)

	return nil
}

var startCmdOptions = server.StartCmdOptions{
	DBOpener: nil,
	PreSetup: preSetupIndexer,
	AddFlags: addIndexFlag,
}
