package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// combinedMinGasPrices will combine the on-chain fee and min_gas_prices.
func combinedMinGasPrices(baseDenom string, baseMinGasPrice sdk.Dec, minGasPrices sdk.DecCoins) sdk.DecCoins {
	// empty min_gas_price
	if len(minGasPrices) == 0 {
		return sdk.DecCoins{sdk.NewDecCoinFromDec(baseDenom, baseMinGasPrice)}
	}

	baseMinGasPriceFromConfig := minGasPrices.AmountOf(baseDenom)

	// if the configured value is bigger than
	// on chain baseMinGasPrice, return origin minGasPrices
	if baseMinGasPriceFromConfig.GTE(baseMinGasPrice) {
		return minGasPrices
	}

	// else, change min gas price of base denom to on chain value
	diff := baseMinGasPrice.Sub(baseMinGasPriceFromConfig)
	return minGasPrices.Add(sdk.NewDecCoinFromDec(baseDenom, diff))
}

// computeRequiredFees returns required fees
func computeRequiredFees(gas sdk.Gas, minGasPrices sdk.DecCoins) sdk.Coins {
	// special case: if minGasPrices=[], requiredFees=[]
	requiredFees := make(sdk.Coins, len(minGasPrices))

	// if not all coins are zero, check fee with min_gas_price
	if !minGasPrices.IsZero() {
		// Determine the required fees by multiplying each required minimum gas
		// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
		glDec := sdk.NewDec(int64(gas))
		for i, gp := range minGasPrices {
			fee := gp.Amount.Mul(glDec)
			requiredFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
		}
	}

	return requiredFees.Sort()
}
