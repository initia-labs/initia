package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetVotingPowerWeight return voting power weights
func (k Keeper) GetVotingPowerWeights(ctx context.Context) (sdk.DecCoins, error) {
	bondDenoms, err := k.BondDenoms(ctx)
	if err != nil {
		return nil, err
	}

	return k.VotingPowerKeeper.GetVotingPowerWeights(ctx, bondDenoms)
}
