// noalias
// DONTCOVER
package types

import (
	"cosmossdk.io/math"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StakingKeeper expected staking keeper
type StakingKeeper interface {
	// iterate through validators by operator address, execute func for each validator
	IterateValidators(sdk.Context,
		func(index int64, validator stakingtypes.ValidatorI) (stop bool))

	Validator(sdk.Context, sdk.ValAddress) stakingtypes.ValidatorI            // get a particular validator by operator address
	ValidatorByConsAddr(sdk.Context, sdk.ConsAddress) stakingtypes.ValidatorI // get a particular validator by consensus address

	// slash the validator and delegators of the validator, specifying offense height, and slash fraction
	Slash(sdk.Context, sdk.ConsAddress, int64, sdk.Dec) sdk.Coins
	SlashWithInfractionReason(sdk.Context, sdk.ConsAddress, int64, sdk.Dec, stakingtypes.Infraction) sdk.Coins
	Jail(sdk.Context, sdk.ConsAddress)   // jail a validator
	Unjail(sdk.Context, sdk.ConsAddress) // unjail a validator

	// Delegation allows for getting a particular delegation for a given validator
	// and delegator outside the scope of the staking module.
	Delegation(sdk.Context, sdk.AccAddress, sdk.ValAddress) stakingtypes.DelegationI

	// MaxValidators returns the maximum amount of bonded validators
	MaxValidators(sdk.Context) uint32

	// VotingPower converts delegated tokens to voting power
	VotingPower(ctx sdk.Context, tokens sdk.Coins) math.Int
	// VotingPowerToConsensusPower converts voting power to consensus power
	VotingPowerToConsensusPower(ctx sdk.Context, votingPower math.Int) int64

	// IsValidatorJailed returns if the validator is jailed.
	IsValidatorJailed(ctx sdk.Context, addr sdk.ConsAddress) bool
}
