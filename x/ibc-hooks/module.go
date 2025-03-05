package ibc_hooks

import (
	"context"
	"encoding/json"

	"cosmossdk.io/core/appmodule"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/initia-labs/initia/v1/x/ibc-hooks/keeper"
	"github.com/initia-labs/initia/v1/x/ibc-hooks/types"
)

const ConsensusVersion = 1

var (
	_ module.AppModuleBasic      = AppModule{}
	_ module.HasGenesis          = AppModule{}
	_ module.HasServices         = AppModule{}
	_ module.HasConsensusVersion = AppModule{}
	_ module.HasName             = AppModule{}

	_ appmodule.AppModule = AppModule{}
)

// AppModuleBasic defines the basic application module used by the move module.
type AppModuleBasic struct {
	cdc codec.Codec
}

func (b AppModuleBasic) RegisterLegacyAminoCodec(amino *codec.LegacyAmino) { //nolint:staticcheck
	types.RegisterLegacyAminoCodec(amino)
}

func (b AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, serveMux *runtime.ServeMux) {
	err := types.RegisterQueryHandlerClient(context.Background(), serveMux, types.NewQueryClient(clientCtx))
	if err != nil {
		panic(err)
	}
}

// Name returns the move module's name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// DefaultGenesis returns default genesis state as raw bytes for the move
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the move module.
func (b AppModuleBasic) ValidateGenesis(marshaler codec.JSONCodec, config client.TxEncodingConfig, message json.RawMessage) error {
	var genState types.GenesisState
	err := marshaler.UnmarshalJSON(message, &genState)
	if err != nil {
		return err
	}

	return genState.ValidateGenesis()
}

// GetTxCmd returns the root tx command for the move module.
func (b AppModuleBasic) GetTxCmd() *cobra.Command {
	return &cobra.Command{}
}

// RegisterInterfaces implements InterfaceModule
func (b AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// ____________________________________________________________________________

// AppModule implements an application module for the move module.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// ConsensusVersion is a sequence number for state-breaking change of the
// module. It should be incremented on each consensus-breaking change
// introduced by the module. To avoid wrong/empty versions, the initial version
// should be set to 1.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// NewAppModule creates a new AppModule object
func NewAppModule(
	cdc codec.Codec,
	k keeper.Keeper,
) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc},
		keeper:         k,
	}
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(&am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServerImpl(&am.keeper))
}

// RegisterInvariants registers the move module invariants.
func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {}

// InitGenesis performs genesis initialization for the move module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genesisState types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	am.keeper.InitGenesis(ctx, &genesisState)
}

// ExportGenesis returns the exported genesis state as raw bytes for the move
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(gs)
}
