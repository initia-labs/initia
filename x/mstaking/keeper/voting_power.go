package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetVotingPowerWeight return voting power weights
func (k Keeper) GetVotingPowerWeights(ctx sdk.Context) sdk.DecCoins {
	return k.VotingPowerKeeper.GetVotingPowerWeights(ctx, k.BondDenoms(ctx))
}
