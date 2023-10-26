package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
)

// GetParams returns the total set of distribution parameters.
func (k Keeper) GetParams(clientCtx sdk.Context) (params customtypes.Params) {
	store := clientCtx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the distribution parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, params customtypes.Params) error {
	if err := params.ValidateBasic(); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)

	return nil
}

// GetCommunityTax returns the current distribution community tax.
func (k Keeper) GetCommunityTax(ctx sdk.Context) math.LegacyDec {
	return k.GetParams(ctx).CommunityTax
}

// GetWithdrawAddrEnabled returns the current distribution withdraw address
// enabled parameter.
func (k Keeper) GetWithdrawAddrEnabled(ctx sdk.Context) (enabled bool) {
	return k.GetParams(ctx).WithdrawAddrEnabled
}

// GetRewardWeights returns the current reward weights
func (k Keeper) GetRewardWeights(ctx sdk.Context) []customtypes.RewardWeight {
	return k.GetParams(ctx).RewardWeights
}

// SetRewardWeights store new reward weights
func (k Keeper) SetRewardWeights(ctx sdk.Context, rewardWeights []customtypes.RewardWeight) {
	params := k.GetParams(ctx)
	params.RewardWeights = rewardWeights
	if err := k.SetParams(ctx, params); err != nil {
		panic(err)
	}
}
