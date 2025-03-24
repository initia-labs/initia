package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/dynamic-fee/types"
)

// InitGenesis sets supply information for genesis.
func (k Keeper) InitGenesis(ctx context.Context, moduleNames []string, genState types.GenesisState) error {
	params := genState.GetParams()
	if err := k.SetParams(ctx, params); err != nil {
		return err
	}
	return nil
}

// ExportGenesis export genesis state
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	var genState types.GenesisState

	var err error
	genState.Params, err = k.GetParams(ctx)
	if err != nil {
		panic(err)
	}
	return &genState
}
