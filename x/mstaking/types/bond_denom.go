package types

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IsAllBondDenoms returns true if the given coins are subset of bondDenoms
func IsAllBondDenoms(coins sdk.Coins, bondDenoms []string) bool {
	bondCoins := sdk.NewCoins()
	for _, bondDenom := range bondDenoms {
		bondCoins = bondCoins.Add(sdk.NewCoin(bondDenom, math.OneInt()))
	}

	return coins.DenomsSubsetOf(bondCoins)
}
