package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path"

	tmcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"cosmossdk.io/log"
	confixcmd "cosmossdk.io/tools/confix/cmd"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	cosmosgenutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	initiaapp "github.com/initia-labs/initia/app"
	"github.com/initia-labs/initia/app/params"
	initiacmdflags "github.com/initia-labs/initia/cmd/flags"
	movecmd "github.com/initia-labs/initia/cmd/move"
	cryptokeyring "github.com/initia-labs/initia/crypto/keyring"
	"github.com/initia-labs/initia/x/genutil"
	genutilcli "github.com/initia-labs/initia/x/genutil/client/cli"
	moveconfig "github.com/initia-labs/initia/x/move/config"

	oracleconfig "github.com/skip-mev/connect/v2/oracle/config"
)

// NewRootCmd creates a new root command for initiad. It is called once in the
// main function.
func NewRootCmd() (*cobra.Command, params.EncodingConfig) {
	sdkConfig := sdk.GetConfig()
	sdkConfig.SetCoinType(initiaapp.CoinType)

	accountPubKeyPrefix := initiaapp.AccountAddressPrefix + "pub"
	validatorAddressPrefix := initiaapp.AccountAddressPrefix + "valoper"
	validatorPubKeyPrefix := initiaapp.AccountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := initiaapp.AccountAddressPrefix + "valcons"
	consNodePubKeyPrefix := initiaapp.AccountAddressPrefix + "valconspub"

	sdkConfig.SetBech32PrefixForAccount(initiaapp.AccountAddressPrefix, accountPubKeyPrefix)
	sdkConfig.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	sdkConfig.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	sdkConfig.SetAddressVerifier(initiaapp.VerifyAddressLen())
	sdkConfig.Seal()

	encodingConfig := initiaapp.MakeEncodingConfig()
	basicManager := initiaapp.BasicManager()

	// Get the executable name and configure the viper instance so that environmental
	// variables are checked based off that name. The underscore character is used
	// as a separator
	executableName, err := os.Executable()
	if err != nil {
		panic(err)
	}

	basename := path.Base(executableName)

	// Configure the viper instance
	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		// client tx config to avoid tx decode failure
		WithTxConfig(params.NewClientTxConfig(encodingConfig.Codec)).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(initiaapp.DefaultNodeHome).
		WithViper(initiaapp.EnvPrefix).
		WithKeyringOptions(cryptokeyring.Option())

	rootCmd := &cobra.Command{
		Use:   basename,
		Short: "initia App",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			// read envs before reading persistent flags
			// TODO - should we handle this for tx flags & query flags?
			initClientCtx, err := readEnv(initClientCtx)
			if err != nil {
				return err
			}

			// read persistent flags if they changed, and override the env configs.
			initClientCtx, err = client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			// unsafe-reset-all is not working without viper set
			viper.Set(tmcli.HomeFlag, initClientCtx.HomeDir)

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			initiaappTemplate, initiaappConfig := initAppConfig()
			customTMConfig := initTendermintConfig()

			return server.InterceptConfigsPreRunHandler(cmd, initiaappTemplate, initiaappConfig, customTMConfig)
		},
	}

	initRootCmd(rootCmd, encodingConfig, basicManager)

	// add keyring to autocli opts
	autoCliOpts := initiaapp.AutoCliOpts()
	initClientCtx, _ = config.ReadFromClientConfig(initClientCtx)
	autoCliOpts.ClientCtx = initClientCtx

	if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	return rootCmd, encodingConfig
}

func initRootCmd(rootCmd *cobra.Command, encodingConfig params.EncodingConfig, basicManager module.BasicManager) {
	a := appCreator{encodingConfig: encodingConfig}

	rootCmd.AddCommand(
		genutilcli.InitCmd(basicManager, initiaapp.DefaultNodeHome),
		debug.Cmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(a.newApp, initiaapp.DefaultNodeHome),
		snapshot.Cmd(a.newApp),
	)

	server.AddCommandsWithStartCmdOptions(
		rootCmd,
		initiaapp.DefaultNodeHome,
		a.newApp,
		a.appExport,
		server.StartCmdOptions{
			AddFlags: func(startCmd *cobra.Command) {
				crisis.AddModuleInitFlags(startCmd)
				initiacmdflags.AddCometBFTFlags(startCmd)
			},
			PostSetup: func(svrCtx *server.Context, clientCtx client.Context, ctx context.Context, g *errgroup.Group) error {
				g.Go(func() error {
					errCh := make(chan error, 1)
					go func() {
						err := a.app.StartOracleClient(ctx)
						errCh <- err
					}()

					select {
					case err := <-errCh:
						return err
					case <-ctx.Done():
						return nil
					}
				})

				return nil
			},
		},
	)

	// add keybase, auxiliary RPC, query, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		genesisCommand(encodingConfig, basicManager),
		queryCommand(),
		txCommand(),
		keys.Commands(),
	)

	// add move commands
	rootCmd.AddCommand(movecmd.MoveCommand(encodingConfig.InterfaceRegistry.SigningContext().AddressCodec(), false))
}

func genesisCommand(encodingConfig params.EncodingConfig, basicManager module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "genesis",
		Short:                      "Application's genesis-related subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	ac := encodingConfig.TxConfig.SigningContext().AddressCodec()
	vc := encodingConfig.TxConfig.SigningContext().ValidatorAddressCodec()
	gentxModule := basicManager[genutiltypes.ModuleName].(genutil.AppModuleBasic)

	cmd.AddCommand(
		genutilcli.CollectGenTxsCmd(banktypes.GenesisBalancesIterator{}, initiaapp.DefaultNodeHome, gentxModule.GenTxValidator, ac, vc),
		genutilcli.GenTxCmd(basicManager, encodingConfig.TxConfig, banktypes.GenesisBalancesIterator{}, initiaapp.DefaultNodeHome, ac, vc),
		cosmosgenutilcli.ValidateGenesisCmd(basicManager),
		genutilcli.AddGenesisAccountCmd(initiaapp.DefaultNodeHome, encodingConfig.InterfaceRegistry.SigningContext().AddressCodec()),
	)

	return cmd
}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.QueryEventForTxCmd(),
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		server.QueryBlocksCmd(),
		authcmd.QueryTxCmd(),
		server.QueryBlockResultsCmd(),
	)

	return cmd
}

func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		authcmd.GetSimulateCmd(),
	)

	return cmd
}

type appCreator struct {
	app            *initiaapp.InitiaApp
	encodingConfig params.EncodingConfig
}

// newApp is an AppCreator
func (a *appCreator) newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	baseappOptions := server.DefaultBaseappOptions(appOpts)

	oracleConfig, err := oracleconfig.ReadConfigFromAppOpts(appOpts)
	if err != nil {
		logger.Error("failed to read oracle config", "error", err)
		os.Exit(-1)
	}

	a.app = initiaapp.NewInitiaApp(
		logger, db, traceStore, true,
		moveconfig.GetConfig(appOpts),
		oracleConfig,
		appOpts,
		baseappOptions...,
	)

	return a.app
}

func (a appCreator) appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {

	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home not set")
	}

	var initiaApp *initiaapp.InitiaApp
	if height != -1 {
		initiaApp = initiaapp.NewInitiaApp(logger, db, traceStore, false, moveconfig.DefaultMoveConfig(), oracleconfig.NewDefaultAppConfig(), appOpts)

		if err := initiaApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	} else {
		initiaApp = initiaapp.NewInitiaApp(logger, db, traceStore, true, moveconfig.DefaultMoveConfig(), oracleconfig.NewDefaultAppConfig(), appOpts)
	}

	return initiaApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

func readEnv(clientCtx client.Context) (client.Context, error) {
	if outputFormat := clientCtx.Viper.GetString(tmcli.OutputFlag); outputFormat != "" {
		clientCtx = clientCtx.WithOutputFormat(outputFormat)
	}

	if homeDir := clientCtx.Viper.GetString(flags.FlagHome); homeDir != "" {
		clientCtx = clientCtx.WithHomeDir(homeDir)
	}

	if clientCtx.Viper.GetBool(flags.FlagDryRun) {
		clientCtx = clientCtx.WithSimulation(true)
	}

	if keyringDir := clientCtx.Viper.GetString(flags.FlagKeyringDir); keyringDir != "" {
		clientCtx = clientCtx.WithKeyringDir(clientCtx.Viper.GetString(flags.FlagKeyringDir))
	}

	if chainID := clientCtx.Viper.GetString(flags.FlagChainID); chainID != "" {
		clientCtx = clientCtx.WithChainID(chainID)
	}

	if keyringBackend := clientCtx.Viper.GetString(flags.FlagKeyringBackend); keyringBackend != "" {
		kr, err := client.NewKeyringFromBackend(clientCtx, keyringBackend)
		if err != nil {
			return clientCtx, err
		}

		clientCtx = clientCtx.WithKeyring(kr)
	}

	if nodeURI := clientCtx.Viper.GetString(flags.FlagNode); nodeURI != "" {
		clientCtx = clientCtx.WithNodeURI(nodeURI)

		client, err := client.NewClientFromNode(nodeURI)
		if err != nil {
			return clientCtx, err
		}

		clientCtx = clientCtx.WithClient(client)
	}

	return clientCtx, nil
}
