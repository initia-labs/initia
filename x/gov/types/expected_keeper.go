package types

import (
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StakingKeeper expected staking keeper (Validator and Delegator sets) (noalias)
type StakingKeeper interface {
	// iterate through bonded validators by operator address, execute func for each validator
	IterateBondedValidatorsByPower(
		sdk.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool),
	)

	IterateDelegations(
		ctx sdk.Context, delegator sdk.AccAddress,
		fn func(index int64, delegation stakingtypes.DelegationI) (stop bool),
	)
	GetVotingPowerWeights(ctx sdk.Context) sdk.DecCoins
}
