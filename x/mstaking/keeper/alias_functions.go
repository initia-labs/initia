package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/initia-labs/initia/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Validator Set

// iterate through the validator set and perform the provided function
func (k Keeper) IterateValidators(ctx context.Context, cb func(validator types.ValidatorI) (stop bool, err error)) error {
	return k.Validators.Walk(ctx, nil, func(key []byte, val types.Validator) (stop bool, err error) {
		return cb(val)
	})
}

// iterate through the bonded validator set and perform the provided function
func (k Keeper) IterateBondedValidatorsByPower(ctx context.Context, cb func(validator types.ValidatorI) (stop bool, err error)) error {
	maxValidators, err := k.MaxValidators(ctx)
	if err != nil {
		return err
	}

	counter := 0
	return k.ValidatorsByConsPowerIndex.Walk(ctx, new(collections.PairRange[int64, []byte]).Descending(), func(key collections.Pair[int64, []byte], value bool) (stop bool, err error) {
		val, err := k.Validators.Get(ctx, key.K2())
		if err != nil {
			return true, err
		}

		if val.IsBonded() {
			if stop, err := cb(val); err != nil || stop {
				return stop, err
			}

			counter++
		}

		return counter == int(maxValidators), nil
	})
}

// iterate through the active validator set and perform the provided function
func (k Keeper) IterateLastValidators(ctx context.Context, cb func(validator types.ValidatorI) (stop bool, err error)) error {
	return k.LastValidatorConsPowers.Walk(ctx, nil, func(valAddr []byte, power int64) (stop bool, err error) {
		val, err := k.Validators.Get(ctx, valAddr)
		if err != nil {
			return true, err
		}

		return cb(val)
	})
}

// Validator gets the Validator interface for a particular address
func (k Keeper) Validator(ctx context.Context, address sdk.ValAddress) (types.ValidatorI, error) {
	return k.Validators.Get(ctx, address)
}

// ValidatorByConsAddr gets the validator interface for a particular pubkey
func (k Keeper) ValidatorByConsAddr(ctx context.Context, addr sdk.ConsAddress) (types.ValidatorI, error) {
	return k.GetValidatorByConsAddr(ctx, addr)
}

// Delegation Set

// Returns self as it is both a validatorset and delegationset
func (k Keeper) GetValidatorSet() types.ValidatorSet {
	return k
}

// Delegation get the delegation interface for a particular set of delegator and validator addresses
func (k Keeper) Delegation(ctx context.Context, addrDel sdk.AccAddress, addrVal sdk.ValAddress) (types.DelegationI, error) {
	return k.GetDelegation(ctx, addrDel, addrVal)
}

// iterate through all of the delegations from a delegator
func (k Keeper) IterateDelegations(
	ctx context.Context,
	delAddr sdk.AccAddress,
	cb func(del types.DelegationI) (stop bool, err error),
) error {
	return k.Delegations.Walk(ctx, collections.NewPrefixedPairRange[[]byte, []byte](delAddr), func(key collections.Pair[[]byte, []byte], del types.Delegation) (stop bool, err error) {
		return cb(del)
	})
}

// return all delegations used during genesis dump
// TODO: remove this func, change all usage for iterate functionality
func (k Keeper) GetAllSDKDelegations(ctx context.Context) (delegations []types.Delegation, err error) {
	err = k.Delegations.Walk(ctx, nil, func(key collections.Pair[[]byte, []byte], del types.Delegation) (stop bool, err error) {
		delegations = append(delegations, del)
		return false, nil
	})

	return
}

// VotingPower convert staking tokens to voting power
func (k Keeper) VotingPower(ctx context.Context, tokens sdk.Coins) (math.Int, error) {
	weights, err := k.GetVotingPowerWeights(ctx)
	if err != nil {
		return math.ZeroInt(), err
	}

	power, _ := types.CalculateVotingPower(tokens, weights)
	return power, nil
}
