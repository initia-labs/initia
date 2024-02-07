package main

import (
	"context"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/initia-labs/audience"
	audiencecfg "github.com/initia-labs/audience/config"
	"github.com/initia-labs/indexer/cron/validator"
	"github.com/initia-labs/indexer/service/collector"
	"github.com/initia-labs/indexer/service/dashboard"
	initiaapp "github.com/initia-labs/initia/app"
)

func addIndexFlag(cmd *cobra.Command) {
	audiencecfg.AddAudienceFlag(cmd)
}

func preSetupIndexer(svrCtx *server.Context, clientCtx client.Context, ctx context.Context, g *errgroup.Group, _app types.Application) error {
	app := _app.(*initiaapp.InitiaApp)

	// if indexer is disabled, it returns nil
	idxer, err := audience.NewAudience(svrCtx.Viper, app)
	if err != nil {
		return err
	}
	// if idxer is nil, it means indexer is disabled
	if idxer == nil {
		return nil
	}

	err = idxer.Validate()
	if err != nil {
		return err
	}

	err = idxer.GetHandler().RegisterService(collector.CollectorSvc)
	if err != nil {
		return err
	}
	err = idxer.GetHandler().RegisterService(dashboard.DashboardSvc)
	if err != nil {
		return err
	}
	err = idxer.GetHandler().RegisterCronjob(validator.Tag, validator.Expr, validator.JobInit, validator.JobFunc)
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
	DBOpener:  nil,
	PreSetup:  preSetupIndexer,
	PostSetup: postSetup,
	AddFlags:  addIndexFlag,
}
