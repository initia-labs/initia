package types

import (
	context "context"

	"cosmossdk.io/core/address"

	sdk "github.com/cosmos/cosmos-sdk/types"

	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

// StakingKeeper expected staking keeper (noalias)
type StakingKeeper interface {
	// iterate through validators by operator address, execute func for each validator
	IterateValidators(ctx context.Context, cb func(validator stakingtypes.ValidatorI) (stop bool, err error)) error

	Validator(ctx context.Context, address sdk.ValAddress) (stakingtypes.ValidatorI, error)
	ValidatorByConsAddr(ctx context.Context, addr sdk.ConsAddress) (stakingtypes.ValidatorI, error)

	// Delegation allows for getting a particular delegation for a given validator
	// and delegator outside the scope of the staking module.
	Delegation(context.Context, sdk.AccAddress, sdk.ValAddress) (stakingtypes.DelegationI, error)

	IterateDelegations(ctx context.Context, delegator sdk.AccAddress,
		fn func(delegation stakingtypes.DelegationI) (stop bool, err error)) error

	GetAllSDKDelegations(ctx context.Context) ([]stakingtypes.Delegation, error)

	ValidatorAddressCodec() address.Codec
}

// DexKeeper expected dex keeper
type DexKeeper interface {
	SwapToBase(ctx context.Context, addr sdk.AccAddress, quoteCoin sdk.Coin) error
}
