package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
	"cosmossdk.io/core/address"
	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	abci "github.com/cometbft/cometbft/abci/types"
	tmjson "github.com/cometbft/cometbft/libs/json"
	tmos "github.com/cometbft/cometbft/libs/os"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	"github.com/cosmos/cosmos-sdk/version"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	"github.com/cosmos/cosmos-sdk/x/auth/posthandler"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	// this line is used by starport scaffolding # stargate/app/moduleImport

	"github.com/initia-labs/initia/app/keepers"
	"github.com/initia-labs/initia/app/params"
	upgrades_v1_1_1 "github.com/initia-labs/initia/app/upgrades/v1_1_1"
	cryptocodec "github.com/initia-labs/initia/crypto/codec"
	initiatx "github.com/initia-labs/initia/tx"
	moveconfig "github.com/initia-labs/initia/x/move/config"
	movetypes "github.com/initia-labs/initia/x/move/types"
	rewardtypes "github.com/initia-labs/initia/x/reward/types"

	// block-sdk dependencies

	blockchecktx "github.com/skip-mev/block-sdk/v2/abci/checktx"
	"github.com/skip-mev/block-sdk/v2/block"
	blockservice "github.com/skip-mev/block-sdk/v2/block/service"

	// connect oracle dependencies

	oracleconfig "github.com/skip-mev/connect/v2/oracle/config"
	oracleclient "github.com/skip-mev/connect/v2/service/clients/oracle"

	// unnamed import of statik for swagger UI support
	_ "github.com/initia-labs/initia/client/docs/statik"
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string
)

var (
	_ servertypes.Application = (*InitiaApp)(nil)
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, "."+AppName)
}

// InitiaApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type InitiaApp struct {
	*baseapp.BaseApp
	keepers.AppKeepers

	// address codecs
	ac, vc, cc address.Codec

	// codecs
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry types.InterfaceRegistry

	// connect oracle client
	oracleClient oracleclient.OracleClient

	// the module manager
	ModuleManager      *module.Manager
	BasicModuleManager module.BasicManager

	// the configurator
	configurator module.Configurator

	// Override of BaseApp's CheckTx
	checkTxHandler blockchecktx.CheckTx
}

// NewInitiaApp returns a reference to an initialized Initia.
func NewInitiaApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	moveConfig moveconfig.MoveConfig,
	oracleConfig oracleconfig.AppConfig,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *InitiaApp {
	// load the configs
	mempoolMaxTxs := cast.ToInt(appOpts.Get(server.FlagMempoolMaxTxs))
	queryGasLimit := cast.ToInt(appOpts.Get(server.FlagQueryGasLimit))

	logger.Info("mempool max txs", "max_txs", mempoolMaxTxs)
	logger.Info("query gas limit", "gas_limit", queryGasLimit)

	encodingConfig := params.MakeEncodingConfig()
	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	cryptocodec.RegisterLegacyAminoCodec(encodingConfig.Amino)
	cryptocodec.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	appCodec := encodingConfig.Codec
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry
	txConfig := encodingConfig.TxConfig

	bApp := baseapp.NewBaseApp(AppName, logger, db, encodingConfig.TxConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetTxEncoder(txConfig.TxEncoder())

	// app opts
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))
	skipGenesisInvariants := cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))
	invCheckPeriod := cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod))
	skipUpgradeHeights := make(map[int64]bool)
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	app := &InitiaApp{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		txConfig:          txConfig,
		interfaceRegistry: interfaceRegistry,

		// codecs
		ac: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		vc: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		cc: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}

	i := 0
	moduleAddrs := make([]sdk.AccAddress, len(maccPerms))
	for name := range maccPerms {
		moduleAddrs[i] = authtypes.NewModuleAddress(name)
		i += 1
	}

	moduleAccountAddresses := app.ModuleAccountAddrs()
	blockedModuleAccountAddrs := app.BlockedModuleAccountAddrs(moduleAccountAddresses)

	// Setup keepers
	app.AppKeepers = keepers.NewAppKeeper(
		app.ac, app.vc, app.cc,
		appCodec,
		bApp,
		legacyAmino,
		maccPerms,
		blockedModuleAccountAddrs,
		skipUpgradeHeights,
		homePath,
		invCheckPeriod,
		logger,
		moveConfig,
		appOpts,
	)

	/****  Module Options ****/

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.ModuleManager = module.NewManager(appModules(app, skipGenesisInvariants)...)

	// BasicModuleManager defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration and genesis verification.
	// By default it is composed of all the module from the module manager.
	// Additionally, app module basics can be overwritten by passing them as argument.
	app.BasicModuleManager = newBasicManagerFromManager(app)

	// NOTE: upgrade module is required to be prioritized
	app.ModuleManager.SetOrderPreBlockers(
		upgradetypes.ModuleName,
	)

	// set order of module operations
	app.ModuleManager.SetOrderBeginBlockers(orderBeginBlockers()...)
	app.ModuleManager.SetOrderEndBlockers(orderEndBlockers()...)
	genesisModuleOrder := orderInitBlockers()
	app.ModuleManager.SetOrderInitGenesis(genesisModuleOrder...)
	app.ModuleManager.SetOrderExportGenesis(genesisModuleOrder...)

	// register invariants for crisis module
	app.ModuleManager.RegisterInvariants(app.CrisisKeeper)

	// register the service configurator
	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	err := app.ModuleManager.RegisterServices(app.configurator)
	if err != nil {
		tmos.Exit(err.Error())
	}

	// Only register upgrade handlers when loading the latest version of the app.
	// This optimization skips unnecessary handler registration during app initialization.
	//
	// The cosmos upgrade handler attempts to create ${HOME}/.initia/data to check for upgrade info,
	// but this isn't required during initial encoding config setup.
	if loadLatest {
		upgrades_v1_1_1.RegisterUpgradeHandlers(app)
	}

	autocliv1.RegisterQueryServer(app.GRPCQueryRouter(), runtimeservices.NewAutoCLIQueryService(app.ModuleManager.Modules))

	reflectionSvc, err := runtimeservices.NewReflectionService()
	if err != nil {
		tmos.Exit(err.Error())
	}
	reflectionv1.RegisterReflectionServiceServer(app.GRPCQueryRouter(), reflectionSvc)

	// initialize stores
	app.MountKVStores(app.GetKVStoreKey())
	app.MountTransientStores(app.GetTransientStoreKey())
	app.MountMemoryStores(app.GetMemoryStoreKey())

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.setPostHandler()
	app.SetEndBlocker(app.EndBlocker)

	// register context decorator for message router
	app.RegisterMessageRouterContextDecorator()

	// setup BlockSDK

	mempool, anteHandler, checkTx, prepareProposalHandler, processProposalHandler, err := setupBlockSDK(app, mempoolMaxTxs)
	if err != nil {
		tmos.Exit(err.Error())
	}

	app.SetMempool(mempool)
	app.SetAnteHandler(anteHandler)
	app.SetCheckTx(checkTx)

	// setup connect

	oracleClient, prepareProposalHandler, processProposalHandler, preBlocker, extendedVoteHandler, verifyVoteExtensionHandler, err := setupSlinky(
		app,
		oracleConfig,
		prepareProposalHandler,
		processProposalHandler,
	)
	if err != nil {
		tmos.Exit(err.Error())
	}

	// set oracle client
	app.SetOracleClient(oracleClient)

	// override baseapp handlers
	app.SetPrepareProposal(prepareProposalHandler)
	app.SetProcessProposal(processProposalHandler)
	app.SetPreBlocker(preBlocker)
	app.SetExtendVoteHandler(extendedVoteHandler)
	app.SetVerifyVoteExtensionHandler(verifyVoteExtensionHandler)

	// At startup, after all modules have been registered, check that all prot
	// annotations are correct.
	protoFiles, err := proto.MergedRegistry()
	if err != nil {
		tmos.Exit(err.Error())
	}
	err = msgservice.ValidateProtoAnnotations(protoFiles)
	if err != nil {
		// Once we switch to using protoreflect-based antehandlers, we might
		// want to panic here instead of logging a warning.
		fmt.Fprintln(os.Stderr, err.Error())
	}

	// Load the latest state from disk if necessary, and initialize the base-app. From this point on
	// no more modifications to the base-app can be made
	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}

	return app
}

// CheckTx will check the transaction with the provided checkTxHandler. We override the default
// handler so that we can verify bid transactions before they are inserted into the mempool.
// With the POB CheckTx, we can verify the bid transaction and all of the bundled transactions
// before inserting the bid transaction into the mempool.
func (app *InitiaApp) CheckTx(req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	return app.checkTxHandler(req)
}

// SetCheckTx sets the checkTxHandler for the app.
func (app *InitiaApp) SetCheckTx(handler blockchecktx.CheckTx) {
	app.checkTxHandler = handler
}

func (app *InitiaApp) SetOracleClient(oracleClient oracleclient.OracleClient) {
	app.oracleClient = oracleClient
}

func (app *InitiaApp) setPostHandler() {
	postHandler, err := posthandler.NewPostHandler(
		posthandler.HandlerOptions{},
	)
	if err != nil {
		tmos.Exit(err.Error())
	}

	app.SetPostHandler(postHandler)
}

// Name returns the name of the App
func (app *InitiaApp) Name() string { return app.BaseApp.Name() }

// BeginBlocker application updates every begin block
func (app *InitiaApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.ModuleManager.BeginBlock(ctx)
}

// EndBlocker application updates every end block
func (app *InitiaApp) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.ModuleManager.EndBlock(ctx)
}

// InitChainer application update at chain initialization
func (app *InitiaApp) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState GenesisState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		tmos.Exit(err.Error())
	}
	if err := app.UpgradeKeeper.SetModuleVersionMap(ctx, app.ModuleManager.GetVersionMap()); err != nil {
		tmos.Exit(err.Error())
	}
	return app.ModuleManager.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads a particular height
func (app *InitiaApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *InitiaApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		addrStr, _ := app.ac.BytesToString(authtypes.NewModuleAddress(acc).Bytes())
		modAccAddrs[addrStr] = true
	}

	return modAccAddrs
}

// BlockedModuleAccountAddrs returns all the app's blocked module account
// addresses.
func (app *InitiaApp) BlockedModuleAccountAddrs(modAccAddrs map[string]bool) map[string]bool {
	modules := []string{
		govtypes.ModuleName,
		rewardtypes.ModuleName,
	}

	// remove module accounts that are ALLOWED to received funds
	for _, module := range modules {
		moduleAddr, _ := app.ac.BytesToString(authtypes.NewModuleAddress(module).Bytes())
		delete(modAccAddrs, moduleAddr)
	}

	return modAccAddrs
}

// LegacyAmino returns SimApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *InitiaApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns Initia's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *InitiaApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns Initia's InterfaceRegistry
func (app *InitiaApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *InitiaApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register new tx query routes from grpc-gateway.
	initiatx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register new tendermint queries routes from grpc-gateway.
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register node gRPC service for grpc-gateway.
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register the Block SDK mempool API routes.
	blockservice.RegisterGRPCGatewayRoutes(apiSvr.ClientCtx, apiSvr.GRPCGatewayRouter)

	// Register grpc-gateway routes for all modules.
	app.BasicModuleManager.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(apiSvr.Router)
	}
}

// Simulate customize gas simulation to add fee deduction gas amount.
func (app *InitiaApp) Simulate(txBytes []byte) (sdk.GasInfo, *sdk.Result, error) {
	gasInfo, result, err := app.BaseApp.Simulate(txBytes)
	gasInfo.GasUsed += FeeDeductionGasAmount
	return gasInfo, result, err
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *InitiaApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.Simulate, app.interfaceRegistry)
	initiatx.RegisterTxQuery(app.GRPCQueryRouter(), app.DynamicFeeKeeper)

	// Register the Block SDK mempool transaction service.
	mempool, ok := app.Mempool().(block.Mempool)
	if !ok {
		panic("mempool is not a block.Mempool")
	}

	blockservice.RegisterMempoolService(app.GRPCQueryRouter(), mempool)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *InitiaApp) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterTendermintService(
		clientCtx, app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry, app.Query,
	)
}

func (app *InitiaApp) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(rtr *mux.Router) {
	statikFS, err := fs.New()
	if err != nil {
		tmos.Exit(err.Error())
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}
	return dupMaccPerms
}

// VerifyAddressLen ensures that the address matches the expected length
func VerifyAddressLen() func(addr []byte) error {
	return func(addr []byte) error {
		addrLen := len(addr)
		if addrLen != 20 && addrLen != movetypes.AddressBytesLength {
			return sdkerrors.ErrInvalidAddress
		}
		return nil
	}
}

// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
func (app *InitiaApp) DefaultGenesis() map[string]json.RawMessage {
	return app.BasicModuleManager.DefaultGenesis(app.appCodec)
}

// Close closes the underlying baseapp, the oracle service, and the prometheus server if required.
// This method blocks on the closure of both the prometheus server, and the oracle-service
func (app *InitiaApp) Close() error {
	if err := app.BaseApp.Close(); err != nil {
		return err
	}

	// close the oracle service
	if app.oracleClient != nil {
		if err := app.oracleClient.Stop(); err != nil {
			return err
		}
	}

	return nil
}

// StartOracleClient starts the oracle client
func (app *InitiaApp) StartOracleClient(ctx context.Context) error {
	if app.oracleClient != nil {
		return app.oracleClient.Start(ctx)
	}

	return nil
}

// RegisterMessageRouterContextDecorator registers a context decorator for the message router
func (app *InitiaApp) RegisterMessageRouterContextDecorator() {
	// dispatchable fungible asset is allowed in the context of bank and ibc transfer modules
	allowedMsgTypes := []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.bank.v1beta1.MsgMultiSend",
		"/ibc.applications.transfer.v1.MsgTransfer",
		"/ibc.applications.transfer.v1.MsgRecvPacket",
		"/ibc.applications.transfer.v1.MsgTimeout",
		"/ibc.applications.transfer.v1.MsgTimeoutOnClose",
		"/ibc.applications.transfer.v1.MsgAcknowledgement",
		"/opinit.ophost.v1.MsgInitiateTokenDeposit",
		"/opinit.ophost.v1.MsgFinalizeTokenWithdrawal",
		"/noble.forwarding.v1.MsgClearAccount",
	}
	app.MsgServiceRouter().SetContextDecorator(func(ctx sdk.Context, msg sdk.Msg) sdk.Context {
		if slices.Contains(allowedMsgTypes, sdk.MsgTypeURL(msg)) {
			ctx = ctx.WithValue(movetypes.AllowDispatchableContextKey, true)
		}

		return ctx
	})
}
