package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"github.com/initia-labs/initia/v1/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// return a validator that a delegator is bonded to
func (k Keeper) GetDelegatorValidator(
	ctx context.Context, delegatorAddr sdk.AccAddress, validatorAddr sdk.ValAddress,
) (validator types.Validator, err error) {
	delegation, err := k.GetDelegation(ctx, delegatorAddr, validatorAddr)
	if err != nil {
		return validator, err
	}

	valAddr, err := k.validatorAddressCodec.StringToBytes(delegation.GetValidatorAddr())
	if err != nil {
		return validator, err
	}

	return k.Validators.Get(ctx, valAddr)
}

// return all delegations for a delegator
func (k Keeper) GetAllDelegatorDelegations(ctx context.Context, delegator sdk.AccAddress) ([]types.Delegation, error) {
	delegations := []types.Delegation{}
	err := k.Delegations.Walk(ctx, collections.NewPrefixedPairRange[[]byte, []byte](delegator), func(key collections.Pair[[]byte, []byte], delegation types.Delegation) (stop bool, err error) {
		delegations = append(delegations, delegation)
		return false, nil
	})

	return delegations, err
}

// return all unbonding-delegations for a delegator
func (k Keeper) GetAllUnbondingDelegations(ctx context.Context, delegator sdk.AccAddress) ([]types.UnbondingDelegation, error) {
	unbondingDelegations := []types.UnbondingDelegation{}
	err := k.UnbondingDelegations.Walk(ctx, collections.NewPrefixedPairRange[[]byte, []byte](delegator), func(key collections.Pair[[]byte, []byte], unbondingDelegation types.UnbondingDelegation) (stop bool, err error) {
		unbondingDelegations = append(unbondingDelegations, unbondingDelegation)
		return false, nil
	})

	return unbondingDelegations, err
}

// return all redelegations for a delegator
func (k Keeper) GetAllRedelegations(
	ctx context.Context, delegator sdk.AccAddress, srcValAddress, dstValAddress sdk.ValAddress,
) ([]types.Redelegation, error) {

	srcValFilter := !(srcValAddress.Empty())
	dstValFilter := !(dstValAddress.Empty())

	redelegations := []types.Redelegation{}

	err := k.Redelegations.Walk(ctx, collections.NewPrefixedTripleRange[[]byte, []byte, []byte](delegator), func(key collections.Triple[[]byte, []byte, []byte], redelegation types.Redelegation) (stop bool, err error) {

		valSrcAddr, err := k.validatorAddressCodec.StringToBytes(redelegation.ValidatorSrcAddress)
		if err != nil {
			return true, err
		}
		valDstAddr, err := k.validatorAddressCodec.StringToBytes(redelegation.ValidatorDstAddress)
		if err != nil {
			return true, err
		}

		if srcValFilter && !(srcValAddress.Equals(sdk.ValAddress(valSrcAddr))) {
			return false, nil
		}

		if dstValFilter && !(dstValAddress.Equals(sdk.ValAddress(valDstAddr))) {
			return false, nil
		}

		redelegations = append(redelegations, redelegation)

		return false, nil
	})

	return redelegations, err
}
