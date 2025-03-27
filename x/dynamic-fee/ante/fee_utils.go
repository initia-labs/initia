package ante

import (
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// computeRequiredFees returns required fees
func computeRequiredFees(gas storetypes.Gas, minGasPrices sdk.DecCoins) sdk.Coins {
	// special case: if minGasPrices=[], requiredFees=[]
	requiredFees := make(sdk.Coins, len(minGasPrices))

	// if not all coins are zero, check fee with min_gas_price
	if !minGasPrices.IsZero() {
		// Determine the required fees by multiplying each required minimum gas
		// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
		glDec := math.LegacyNewDec(int64(gas))
		for i, gp := range minGasPrices {
			fee := gp.Amount.Mul(glDec)
			requiredFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
		}
	}

	return requiredFees.Sort()
}
