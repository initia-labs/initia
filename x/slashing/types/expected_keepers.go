// noalias
// DONTCOVER
package types

import (
	"context"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StakingKeeper expected staking keeper
type StakingKeeper interface {
	ValidatorAddressCodec() address.Codec
	ConsensusAddressCodec() address.Codec

	// iterate through validators by operator address, execute func for each validator
	IterateValidators(context.Context, func(validator stakingtypes.ValidatorI) (stop bool, err error)) error

	Validator(context.Context, sdk.ValAddress) (stakingtypes.ValidatorI, error)            // get a particular validator by operator address
	ValidatorByConsAddr(context.Context, sdk.ConsAddress) (stakingtypes.ValidatorI, error) // get a particular validator by consensus address

	// slash the validator and delegators of the validator, specifying offense height, offense power, and slash fraction
	Slash(context.Context, sdk.ConsAddress, int64, math.LegacyDec) (sdk.Coins, error)
	SlashWithInfractionReason(context.Context, sdk.ConsAddress, int64, math.LegacyDec, stakingtypes.Infraction) (sdk.Coins, error)
	Jail(context.Context, sdk.ConsAddress) error   // jail a validator
	Unjail(context.Context, sdk.ConsAddress) error // unjail a validator

	// Delegation allows for getting a particular delegation for a given validator
	// and delegator outside the scope of the staking module.
	Delegation(context.Context, sdk.AccAddress, sdk.ValAddress) (stakingtypes.DelegationI, error)

	// MaxValidators returns the maximum amount of bonded validators
	MaxValidators(context.Context) (uint32, error)

	// IsValidatorJailed returns if the validator is jailed.
	IsValidatorJailed(ctx context.Context, addr sdk.ConsAddress) (bool, error)

	// VotingPower converts delegated tokens to voting power
	VotingPower(ctx context.Context, tokens sdk.Coins) (math.Int, error)
	// VotingPowerToConsensusPower converts voting power to consensus power
	VotingPowerToConsensusPower(ctx context.Context, votingPower math.Int) int64
}
