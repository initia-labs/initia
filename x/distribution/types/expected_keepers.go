package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

// StakingKeeper expected staking keeper (noalias)
type StakingKeeper interface {
	// iterate through validators by operator address, execute func for each validator
	IterateValidators(sdk.Context,
		func(index int64, validator stakingtypes.ValidatorI) (stop bool))

	Validator(sdk.Context, sdk.ValAddress) stakingtypes.ValidatorI            // get a particular validator by operator address
	ValidatorByConsAddr(sdk.Context, sdk.ConsAddress) stakingtypes.ValidatorI // get a particular validator by consensus address

	// Delegation allows for getting a particular delegation for a given validator
	// and delegator outside the scope of the staking module.
	Delegation(sdk.Context, sdk.AccAddress, sdk.ValAddress) stakingtypes.DelegationI

	IterateDelegations(ctx sdk.Context, delegator sdk.AccAddress,
		fn func(index int64, delegation stakingtypes.DelegationI) (stop bool))

	GetAllSDKDelegations(ctx sdk.Context) []stakingtypes.Delegation
}

// DexKeeper expected dex keeper
type DexKeeper interface {
	SwapToBase(ctx sdk.Context, addr sdk.AccAddress, quoteCoin sdk.Coin) error
}
