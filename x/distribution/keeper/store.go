package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/distribution/types"
)

// GetDelegatorWithdrawAddr returns the delegator withdraw address, defaulting to the delegator address
func (k Keeper) GetDelegatorWithdrawAddr(ctx context.Context, delAddr sdk.AccAddress) (sdk.AccAddress, error) {
	bz, err := k.DelegatorWithdrawAddrs.Get(ctx, delAddr)
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return delAddr, nil
	} else if err != nil {
		return sdk.AccAddress{}, err
	}

	return sdk.AccAddress(bz), nil
}

// GetValidatorAccumulatedCommission returns accumulated commission for a validator
func (k Keeper) GetValidatorAccumulatedCommission(ctx context.Context, val sdk.ValAddress) (commission types.ValidatorAccumulatedCommission, err error) {
	commission, err = k.ValidatorAccumulatedCommissions.Get(ctx, val)
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return types.ValidatorAccumulatedCommission{}, nil
	} else if err != nil {
		return types.ValidatorAccumulatedCommission{}, err
	}

	return
}

// GetValidatorSlashEvent returns slash event for height
func (k Keeper) GetValidatorSlashEvent(ctx context.Context, val sdk.ValAddress, height, period uint64) (event types.ValidatorSlashEvent, found bool, err error) {
	event, err = k.ValidatorSlashEvents.Get(ctx, collections.Join3(val.Bytes(), height, period))
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return types.ValidatorSlashEvent{}, false, nil
	} else if err != nil {
		return types.ValidatorSlashEvent{}, false, err
	}

	return event, true, nil
}

// GetValidatorOutstandingRewards returns validator outstanding rewards
// ValidatorOutstandingRewards should be initialized at initValidator()
func (k Keeper) GetValidatorOutstandingRewards(ctx context.Context, val sdk.ValAddress) (rewards types.ValidatorOutstandingRewards, err error) {
	rewards, err = k.ValidatorOutstandingRewards.Get(ctx, val)
	if err != nil {
		return types.ValidatorOutstandingRewards{}, err
	}

	return
}

// GetValidatorCurrentRewards returns current rewards for a validator
// ValidatorCurrentRewards rewards should be initialized at initValidator()
func (k Keeper) GetValidatorCurrentRewards(ctx context.Context, val sdk.ValAddress) (rewards types.ValidatorCurrentRewards, err error) {
	rewards, err = k.ValidatorCurrentRewards.Get(ctx, val)
	if err != nil {
		return types.ValidatorCurrentRewards{}, err
	}

	return
}
