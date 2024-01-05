package keeper

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// VotingPowerToConsensusPower - convert input tokens to potential consensus-engine power
func (k Keeper) VotingPowerToConsensusPower(ctx context.Context, votingPower math.Int) int64 {
	return sdk.TokensToConsensusPower(votingPower, k.PowerReduction(ctx))
}

// VotingPowerFromConsensusPower - convert input power to tokens
func (k Keeper) VotingPowerFromConsensusPower(ctx context.Context, power int64) math.Int {
	return sdk.TokensFromConsensusPower(power, k.PowerReduction(ctx))
}
