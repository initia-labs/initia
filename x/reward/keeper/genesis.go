package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/reward/types"
)

// InitGenesis new mint genesis
func (k Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) {

	if err := k.SetParams(ctx, data.Params); err != nil {
		panic(err)
	}

	if err := k.SetLastReleaseTimestamp(ctx, data.LastReleaseTimestamp); err != nil {
		panic(err)
	}

	if err := k.SetLastDilutionTimestamp(ctx, data.LastDilutionTimestamp); err != nil {
		panic(err)
	}

	k.accKeeper.GetModuleAccount(ctx, types.ModuleName)
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}

	lastMintTimestamp, err := k.GetLastReleaseTimestamp(ctx)
	if err != nil {
		panic(err)
	}

	lastDilutionTimestamp, err := k.GetLastDilutionTimestamp(ctx)
	if err != nil {
		panic(err)
	}

	return types.NewGenesisState(params, lastMintTimestamp, lastDilutionTimestamp)
}
