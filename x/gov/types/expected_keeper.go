package types

import (
	context "context"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"

	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

// StakingKeeper expected staking keeper (Validator and Delegator sets) (noalias)
type StakingKeeper interface {
	// iterate through bonded validators by operator address, execute func for each validator
	IterateBondedValidatorsByPower(
		context.Context, func(validator stakingtypes.ValidatorI) (stop bool, err error),
	) error

	IterateDelegations(
		ctx context.Context, delegator sdk.AccAddress,
		fn func(delegation stakingtypes.DelegationI) (stop bool, err error),
	) error

	GetVotingPowerWeights(ctx context.Context) (sdk.DecCoins, error)
	ValidatorAddressCodec() address.Codec
}
