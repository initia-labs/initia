package keeper

import (
	"context"
	"time"

	"cosmossdk.io/math"
	"github.com/initia-labs/initia/v1/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MinVotingPower - minimum voting power to get into power update whitelist
func (k Keeper) MinVotingPower(ctx context.Context) (math.Int, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.ZeroInt(), err
	}

	return math.NewIntFromUint64(params.MinVotingPower), nil
}

// UnbondingTime
func (k Keeper) UnbondingTime(ctx context.Context) (time.Duration, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	return params.UnbondingTime, nil
}

// MaxValidators - Maximum number of validators
func (k Keeper) MaxValidators(ctx context.Context) (uint32, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxValidators, nil
}

// MaxEntries - Maximum number of simultaneous unbonding
// delegations or redelegations (per pair/trio)
func (k Keeper) MaxEntries(ctx context.Context) (uint32, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.MaxEntries, nil
}

// HistoricalEntries = number of historical info entries
// to persist in store
func (k Keeper) HistoricalEntries(ctx context.Context) (uint32, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return params.HistoricalEntries, nil
}

// BondDenoms - Bondable coin denominations
func (k Keeper) BondDenoms(ctx context.Context) ([]string, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	return params.BondDenoms, nil
}

// SetBondDenoms - store bondable coin denominations
func (k Keeper) SetBondDenoms(ctx context.Context, bondDenoms []string) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	params.BondDenoms = bondDenoms
	return k.SetParams(ctx, params)
}

// PowerReduction - is the amount of staking tokens required for 1 unit of consensus-engine power.
// Currently, this returns a global variable that the app developer can tweak.
// TODO: we might turn this into an on-chain param:
// https://github.com/cosmos/cosmos-sdk/issues/8365
func (k Keeper) PowerReduction(ctx context.Context) math.Int {
	return sdk.DefaultPowerReduction
}

// MinCommissionRate - Minimum validator commission rate
func (k Keeper) MinCommissionRate(ctx context.Context) (math.LegacyDec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	return params.MinCommissionRate, nil
}

// SetParams sets the x/staking module parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	return k.Params.Set(ctx, params)
}

// GetParams sets the x/staking module parameters.
func (k Keeper) GetParams(ctx context.Context) (params types.Params, err error) {
	return k.Params.Get(ctx)
}
