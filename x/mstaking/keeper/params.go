package keeper

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/mstaking/types"
)

// MinVotingPower - minimum voting power to get into power update whitelist
func (k Keeper) MinVotingPower(ctx sdk.Context) math.Int {
	return sdk.NewIntFromUint64(k.GetParams(ctx).MinVotingPower)
}

// UnbondingTime
func (k Keeper) UnbondingTime(ctx sdk.Context) time.Duration {
	return k.GetParams(ctx).UnbondingTime
}

// MaxValidators - Maximum number of validators
func (k Keeper) MaxValidators(ctx sdk.Context) uint32 {
	return k.GetParams(ctx).MaxValidators
}

// MaxEntries - Maximum number of simultaneous unbonding
// delegations or redelegations (per pair/trio)
func (k Keeper) MaxEntries(ctx sdk.Context) uint32 {
	return k.GetParams(ctx).MaxEntries
}

// HistoricalEntries = number of historical info entries
// to persist in store
func (k Keeper) HistoricalEntries(ctx sdk.Context) uint32 {
	return k.GetParams(ctx).HistoricalEntries
}

// BondDenoms - Bondable coin denominations
func (k Keeper) BondDenoms(ctx sdk.Context) []string {
	return k.GetParams(ctx).BondDenoms
}

// SetBondDenoms - store bondable coin denominations
func (k Keeper) SetBondDenoms(ctx sdk.Context, bondDenoms []string) error {
	params := k.GetParams(ctx)
	params.BondDenoms = bondDenoms
	return k.SetParams(ctx, params)
}

// PowerReduction - is the amount of staking tokens required for 1 unit of consensus-engine power.
// Currently, this returns a global variable that the app developer can tweak.
// TODO: we might turn this into an on-chain param:
// https://github.com/cosmos/cosmos-sdk/issues/8365
func (k Keeper) PowerReduction(ctx sdk.Context) math.Int {
	return sdk.DefaultPowerReduction
}

// MinCommissionRate - Minimum validator commission rate
func (k Keeper) MinCommissionRate(ctx sdk.Context) math.LegacyDec {
	return k.GetParams(ctx).MinCommissionRate
}

// SetParams sets the x/staking module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
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

// GetParams sets the x/staking module parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}
