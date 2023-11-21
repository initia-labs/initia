package ante

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	movetypes "github.com/initia-labs/initia/x/move/types"
)

// MempoolFeeChecker will check if the transaction's fee is at least as large
// as the local validator's minimum gasFee (defined in validator config).
// If fee is too low, decorator returns error and tx is rejected from mempool.
// Note this only applies when ctx.CheckTx = true
// If fee is high enough or not CheckTx, then call next AnteHandler
// CONTRACT: Tx must implement FeeTx to use MempoolFeeChecker
type MempoolFeeChecker struct {
	keeper movetypes.AnteKeeper
}

// NewMempoolFeeChecker create MempoolFeeChecker instance
func NewMempoolFeeChecker(
	keeper movetypes.AnteKeeper,
) MempoolFeeChecker {
	return MempoolFeeChecker{
		keeper,
	}
}
func (fc MempoolFeeChecker) CheckTxFeeWithMinGasPrices(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil, 0, errors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	feeCoins := feeTx.GetFee()
	gas := feeTx.GetGas()

	if ctx.IsCheckTx() {
		minGasPrices := ctx.MinGasPrices()
		if fc.keeper != nil {
			baseDenom := fc.keeper.BaseDenom(ctx)
			baseMinGasPrice := fc.keeper.BaseMinGasPrice(ctx)

			minGasPrices = combinedMinGasPrices(baseDenom, baseMinGasPrice, minGasPrices)
		}

		if !minGasPrices.IsZero() {
			requiredFees := computeRequiredFees(gas, minGasPrices)

			if !feeCoins.IsAnyGTE(requiredFees) {
				// convert baseDenom min gas prices to quote denom prices
				// and check the paid fee is enough or not.
				isSufficient := false
				sumInBaseUnit := math.ZeroInt()

				if fc.keeper != nil {
					baseDenom := fc.keeper.BaseDenom(ctx)
					requiredBaseAmount := requiredFees.AmountOfNoDenomValidation(baseDenom)

					// If the requiredBaseAmount is zero, it means the operator
					// do not want to receive base denom fee but want to get other
					// denom fee.
					if !requiredBaseAmount.IsZero() {
						for _, coin := range feeCoins {
							quotePrice, skip, err := fc.fetchOrSkipPrice(ctx, baseDenom, coin.Denom)
							if err != nil {
								return nil, 0, err
							}
							if skip {
								continue
							}

							// sum the converted fee values
							quoteValueInBaseUnit := quotePrice.MulInt(coin.Amount).TruncateInt()
							sumInBaseUnit = sumInBaseUnit.Add(quoteValueInBaseUnit)

							// check the sum is greater than the required.
							if sumInBaseUnit.GTE(requiredBaseAmount) {
								isSufficient = true
								break
							}
						}
					}
				}

				if !isSufficient {
					return nil, 0, errors.Wrapf(
						sdkerrors.ErrInsufficientFee,
						"insufficient fees; got: %s (sum %s), required: %s",
						feeCoins,
						sumInBaseUnit,
						requiredFees,
					)
				}
			}
		}
	}

	// TODO - if we want to use ethereum like priority system,
	// then we need to compute all dex prices of all fee coins
	return feeCoins, 1 /* FIFO */, nil
}

func (fc MempoolFeeChecker) fetchOrSkipPrice(ctx sdk.Context, baseDenom, quoteDenom string) (price sdk.Dec, skip bool, err error) {
	if quoteDenom == baseDenom {
		return sdk.OneDec(), false, nil
	}

	if found, err := fc.keeper.HasDexPair(ctx, quoteDenom); err != nil {
		return sdk.ZeroDec(), false, err
	} else if !found {
		return sdk.ZeroDec(), true, nil
	}

	if quotePrice, err := fc.keeper.GetPoolSpotPrice(ctx, quoteDenom); err != nil {
		return sdk.ZeroDec(), false, err
	} else if quotePrice.IsZero() {
		return sdk.ZeroDec(), true, nil
	} else {
		return quotePrice, false, nil
	}
}
