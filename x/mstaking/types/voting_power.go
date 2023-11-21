package types

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func CalculateVotingPower(tokens sdk.Coins, weights sdk.DecCoins) (math.Int, sdk.Coins) {
	totalVotingPower := sdk.ZeroInt()
	votingPowers := make(sdk.Coins, 0, len(weights))
	for _, weight := range weights {
		votingPower := weight.Amount.MulInt(tokens.AmountOf(weight.Denom)).TruncateInt()

		if votingPower.IsPositive() {
			votingPowers = append(votingPowers, sdk.NewCoin(weight.Denom, votingPower))
			totalVotingPower = totalVotingPower.Add(votingPower)
		}
	}

	return totalVotingPower, votingPowers
}
