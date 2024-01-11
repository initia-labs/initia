package main

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"

	initiaapp "github.com/initia-labs/initia/app"
)

func postSetup(svrCtx *server.Context, clientCtx client.Context, ctx context.Context, g *errgroup.Group, app servertypes.Application) error {
	initiaApp := app.(*initiaapp.InitiaApp)
	g.Go(func() error {
		if initiaApp.OraclePrometheusServer != nil {
			go func() {
				initiaApp.OraclePrometheusServer.Start()
			}()

			ctx.Done()
			initiaApp.OraclePrometheusServer.Close()
		}

		return nil
	})

	g.Go(func() error {
		errCh := make(chan error)
		go func() {
			errCh <- initiaApp.OracleService.Start(ctx)
		}()

		select {
		case <-ctx.Done():
			return initiaApp.OracleService.Stop(ctx)
		case err := <-errCh:
			return err
		}
	})

	return nil
}
