package gov

// DONTCOVER

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"golang.org/x/exp/maps"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	modulev1 "cosmossdk.io/api/cosmos/gov/module/v1"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/initia-labs/initia/x/gov/keeper"
	customtypes "github.com/initia-labs/initia/x/gov/types"
)

const ConsensusVersion = 1

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.HasGenesis     = AppModule{}
	_ module.HasServices    = AppModule{}
	_ module.HasInvariants  = AppModule{}

	_ appmodule.AppModule     = AppModule{}
	_ appmodule.HasEndBlocker = AppModule{}
)

// AppModuleBasic defines the basic application module used by the gov module.
type AppModuleBasic struct {
	cdc                    codec.Codec
	legacyProposalHandlers []govclient.ProposalHandler // proposal handlers which live in governance cli and rest
}

// NewAppModuleBasic creates a new AppModuleBasic object
func NewAppModuleBasic(cdc codec.Codec, legacyProposalHandlers ...govclient.ProposalHandler) AppModuleBasic {
	return AppModuleBasic{
		cdc:                    cdc,
		legacyProposalHandlers: legacyProposalHandlers,
	}
}

// Name returns the gov module's name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the gov module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	v1.RegisterLegacyAminoCodec(cdc)
	v1beta1.RegisterLegacyAminoCodec(cdc)
	customtypes.RegisterLegacyAminoCodec(cdc)
}

// DefaultGenesis returns default genesis state as raw bytes for the gov
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(customtypes.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the gov module.
func (b AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var data customtypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return customtypes.ValidateGenesis(&data, b.cdc.InterfaceRegistry().SigningContext().AddressCodec())
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the gov module.
func (a AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := v1.RegisterQueryHandlerClient(context.Background(), mux, v1.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}

	if err := customtypes.RegisterQueryHandlerClient(context.Background(), mux, customtypes.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root tx command for the gov module.
func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	legacyProposalCLIHandlers := getProposalCLIHandlers(a.legacyProposalHandlers)

	return cli.NewTxCmd(legacyProposalCLIHandlers)
}

func getProposalCLIHandlers(handlers []govclient.ProposalHandler) []*cobra.Command {
	proposalCLIHandlers := make([]*cobra.Command, 0, len(handlers))
	for _, proposalHandler := range handlers {
		proposalCLIHandlers = append(proposalCLIHandlers, proposalHandler.CLIHandler())
	}
	return proposalCLIHandlers
}

// RegisterInterfaces implements InterfaceModule.RegisterInterfaces
func (a AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	v1.RegisterInterfaces(registry)
	v1beta1.RegisterInterfaces(registry)
	customtypes.RegisterInterfaces(registry)
}

// AppModule implements an application module for the gov module.
type AppModule struct {
	AppModuleBasic

	keeper        *keeper.Keeper
	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(
	cdc codec.Codec,
	keeper *keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc: cdc},
		keeper:         keeper,
		accountKeeper:  ak,
		bankKeeper:     bk,
	}
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

func init() {
	appmodule.Register(
		&modulev1.Module{},
		appmodule.Provide(ProvideModule),
		appmodule.Invoke(InvokeAddRoutes, InvokeSetHooks))
}

type ModuleInputs struct {
	depinject.In

	Config           *modulev1.Module
	Cdc              codec.Codec
	StoreService     store.KVStoreService
	ModuleKey        depinject.OwnModuleKey
	MsgServiceRouter baseapp.MessageRouter

	AccountKeeper      types.AccountKeeper
	BankKeeper         types.BankKeeper
	StakingKeeper      customtypes.StakingKeeper
	DistributionKeeper types.DistributionKeeper
}

type ModuleOutputs struct {
	depinject.Out

	Module       appmodule.AppModule
	Keeper       *keeper.Keeper
	HandlerRoute v1beta1.HandlerRoute
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	defaultConfig := types.DefaultConfig()
	if in.Config.MaxMetadataLen != 0 {
		defaultConfig.MaxMetadataLen = in.Config.MaxMetadataLen
	}

	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(types.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	k := keeper.NewKeeper(
		in.Cdc,
		in.StoreService,
		in.AccountKeeper,
		in.BankKeeper,
		in.StakingKeeper,
		in.DistributionKeeper,
		in.MsgServiceRouter,
		defaultConfig,
		authority.String(),
	)

	m := NewAppModule(in.Cdc, k, in.AccountKeeper, in.BankKeeper)
	hr := v1beta1.HandlerRoute{Handler: v1beta1.ProposalHandler, RouteKey: types.RouterKey}

	return ModuleOutputs{Module: m, Keeper: k, HandlerRoute: hr}
}

func InvokeAddRoutes(keeper *keeper.Keeper, routes []v1beta1.HandlerRoute) {
	if keeper == nil || routes == nil {
		return
	}

	// Default route order is a lexical sort by RouteKey.
	// Explicit ordering can be added to the module config if required.
	slices.SortFunc(routes, func(x, y v1beta1.HandlerRoute) int {
		return strings.Compare(x.RouteKey, y.RouteKey)
	})

	router := v1beta1.NewRouter()
	for _, r := range routes {
		router.AddRoute(r.RouteKey, r.Handler)
	}
	keeper.SetLegacyRouter(router)
}

func InvokeSetHooks(keeper *keeper.Keeper, govHooks map[string]types.GovHooksWrapper) error {
	if keeper == nil || govHooks == nil {
		return nil
	}

	// Default ordering is lexical by module name.
	// Explicit ordering can be added to the module config if required.
	modNames := maps.Keys(govHooks)
	order := modNames
	sort.Strings(order)

	var multiHooks types.MultiGovHooks
	for _, modName := range order {
		hook, ok := govHooks[modName]
		if !ok {
			return fmt.Errorf("can't find staking hooks for module %s", modName)
		}
		multiHooks = append(multiHooks, hook)
	}

	keeper.SetHooks(multiHooks)
	return nil
}

// Name returns the gov module's name.
func (AppModule) Name() string {
	return types.ModuleName
}

// RegisterInvariants registers module invariants
func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
	keeper.RegisterInvariants(ir, am.keeper, am.bankKeeper)
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	msgServer := keeper.NewMsgServerImpl(am.keeper)
	v1beta1.RegisterMsgServer(cfg.MsgServer(), keeper.NewLegacyMsgServerImpl(am.accountKeeper.GetModuleAddress(types.ModuleName).String(), msgServer))
	v1.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))

	legacyQueryServer := keeper.NewLegacyQueryServer(am.keeper)
	v1beta1.RegisterQueryServer(cfg.QueryServer(), legacyQueryServer)
	v1.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.keeper))

	customtypes.RegisterMsgServer(cfg.MsgServer(), keeper.NewCustomMsgServerImpl(am.keeper))
	customtypes.RegisterQueryServer(cfg.QueryServer(), keeper.NewCustomQueryServer(am.keeper))
}

// InitGenesis performs genesis initialization for the gov module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genesisState customtypes.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	InitGenesis(ctx, am.accountKeeper, am.bankKeeper, am.keeper, &genesisState)
}

// ExportGenesis returns the exported genesis state as raw bytes for the gov
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs, err := ExportGenesis(ctx, am.keeper)
	if err != nil {
		panic(err)
	}
	return cdc.MustMarshalJSON(gs)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// EndBlock returns the end blocker for the gov module. It returns no validator
// updates.
func (am AppModule) EndBlock(ctx context.Context) error {
	c := sdk.UnwrapSDKContext(ctx)
	return EndBlocker(c, am.keeper)
}
