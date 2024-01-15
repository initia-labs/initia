package fetchprice

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/initia-labs/initia/x/ibc/fetchprice/client/cli"
	consumerkeeper "github.com/initia-labs/initia/x/ibc/fetchprice/consumer/keeper"
	consumertypes "github.com/initia-labs/initia/x/ibc/fetchprice/consumer/types"
	genesistypes "github.com/initia-labs/initia/x/ibc/fetchprice/genesis/types"
	"github.com/initia-labs/initia/x/ibc/fetchprice/provider"
	providerkeeper "github.com/initia-labs/initia/x/ibc/fetchprice/provider/keeper"
	providertypes "github.com/initia-labs/initia/x/ibc/fetchprice/provider/types"
	"github.com/initia-labs/initia/x/ibc/fetchprice/types"

	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
)

var (
	_ module.AppModule           = (*AppModule)(nil)
	_ module.AppModuleBasic      = (*AppModuleBasic)(nil)
	_ module.HasGenesis          = (*AppModule)(nil)
	_ module.HasName             = (*AppModule)(nil)
	_ module.HasConsensusVersion = (*AppModule)(nil)
	_ module.HasServices         = (*AppModule)(nil)
	_ appmodule.AppModule        = (*AppModule)(nil)

	_ porttypes.IBCModule = (*provider.IBCModule)(nil)
)

// AppModuleBasic is the IBC interchain accounts AppModuleBasic
type AppModuleBasic struct {
	cdc codec.Codec
}

// Name implements AppModuleBasic interface
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// RegisterLegacyAminoCodec implements AppModuleBasic.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

// RegisterInterfaces registers module concrete types into protobuf Any
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	consumertypes.RegisterInterfaces(registry)
}

// DefaultGenesis returns default genesis state as raw bytes for the IBC
// interchain accounts module
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(genesistypes.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the IBC interchain acounts module
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var gs genesistypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return gs.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the interchain accounts module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	err := consumertypes.RegisterQueryHandlerClient(context.Background(), mux, consumertypes.NewQueryClient(clientCtx))
	if err != nil {
		panic(err)
	}
}

// GetTxCmd implements AppModuleBasic interface
func (ab AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd(ab.cdc.InterfaceRegistry().SigningContext().AddressCodec())
}

// GetQueryCmd implements AppModuleBasic interface
func (ab AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.NewQueryCmd(ab.cdc.InterfaceRegistry().SigningContext().AddressCodec())
}

// AppModule is the application module for the IBC interchain accounts module
type AppModule struct {
	AppModuleBasic
	consumerKeeper *consumerkeeper.Keeper
	providerKeeper *providerkeeper.Keeper
}

// NewAppModule creates a new IBC interchain accounts module
func NewAppModule(cdc codec.Codec, consumerKeeper *consumerkeeper.Keeper, providerKeeper *providerkeeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc},
		consumerKeeper: consumerKeeper,
		providerKeeper: providerKeeper,
	}
}

// RegisterServices registers module services
func (am AppModule) RegisterServices(cfg module.Configurator) {
	if am.consumerKeeper != nil {
		consumertypes.RegisterMsgServer(cfg.MsgServer(), consumerkeeper.NewMsgServerImpl(am.consumerKeeper))
		consumertypes.RegisterQueryServer(cfg.QueryServer(), consumerkeeper.NewQueryServerImpl(am.consumerKeeper))
	}
}

// InitGenesis performs genesis initialization for the interchain accounts module.
// It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genesisState genesistypes.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)

	if am.consumerKeeper != nil {
		am.consumerKeeper.InitGenesis(ctx, genesisState.ConsumerGenesisState)
	}

	if am.providerKeeper != nil {
		am.providerKeeper.InitGenesis(ctx, genesisState.ProviderGenesisState)
	}
}

// ExportGenesis returns the exported genesis state as raw bytes for the interchain accounts module
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	var (
		consumerGenesisState = consumertypes.DefaultGenesisState()
		providerGenesisState = providertypes.DefaultGenesisState()
	)

	if am.consumerKeeper != nil {
		consumerGenesisState = am.consumerKeeper.ExportGenesis(ctx)
	}

	if am.providerKeeper != nil {
		providerGenesisState = am.providerKeeper.ExportGenesis(ctx)
	}

	gs := genesistypes.NewGenesisState(consumerGenesisState, providerGenesisState)
	return cdc.MustMarshalJSON(gs)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 3 }
