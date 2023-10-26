package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/reward/types"
)

// InitGenesis new mint genesis
func (k Keeper) InitGenesis(ctx sdk.Context, data *types.GenesisState) {

	if err := k.SetParams(ctx, data.Params); err != nil {
		panic(err)
	}
	k.SetLastReleaseTimestamp(ctx, data.LastReleaseTimestamp)
	k.SetLastDilutionTimestamp(ctx, data.LastDilutionTimestamp)

	k.accKeeper.GetModuleAccount(ctx, types.ModuleName)
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	lastMintTimestamp := k.GetLastReleaseTimestamp(ctx)
	lastDilutionTimestamp := k.GetLastDilutionTimestamp(ctx)
	return types.NewGenesisState(params, lastMintTimestamp, lastDilutionTimestamp)
}
