package ante

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	movetypes "github.com/initia-labs/initia/v1/x/move/types"
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

	priority := int64(1)
	if ctx.IsCheckTx() {
		minGasPrices := ctx.MinGasPrices()
		totalFeeBaseAmount := math.ZeroInt()

		var baseDenom string
		var err error
		if fc.keeper != nil {
			baseDenom, err = fc.keeper.BaseDenom(ctx)
			if err != nil {
				return nil, 0, err
			}

			baseMinGasPrice, err := fc.keeper.BaseMinGasPrice(ctx)
			if err != nil {
				return nil, 0, err
			}

			minGasPrices = combinedMinGasPrices(baseDenom, baseMinGasPrice, minGasPrices)

			for _, coin := range feeTx.GetFee() {
				basePrice, err := fc.fetchPrice(ctx, baseDenom, coin.Denom)
				if err != nil {
					return nil, 1, err
				}

				quoteAmount := coin.Amount
				baseAmount := basePrice.MulInt(quoteAmount).TruncateInt()
				totalFeeBaseAmount = totalFeeBaseAmount.Add(baseAmount)
			}
			if totalFeeBaseAmount.GT(math.OneInt()) {
				priority = totalFeeBaseAmount.Int64()
			}
		}

		if !minGasPrices.IsZero() {
			requiredFees := computeRequiredFees(gas, minGasPrices)

			if !feeCoins.IsAnyGTE(requiredFees) {
				// convert baseDenom min gas prices to quote denom prices
				// and check the paid fee is enough or not.
				isSufficient := false
				sumInBaseUnit := math.ZeroInt()

				if fc.keeper != nil {
					requiredBaseAmount := requiredFees.AmountOfNoDenomValidation(baseDenom)

					// converting to base token only works when the requiredBaseAmount is non-zero.
					isSufficient = !requiredBaseAmount.IsZero() && totalFeeBaseAmount.GTE(requiredBaseAmount)
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

	return feeCoins, priority, nil
}

func (fc MempoolFeeChecker) fetchPrice(ctx sdk.Context, baseDenom, quoteDenom string) (price math.LegacyDec, err error) {
	if quoteDenom == baseDenom {
		return math.LegacyOneDec(), nil
	}

	if found, err := fc.keeper.HasDexPair(ctx, quoteDenom); err != nil {
		return math.LegacyZeroDec(), err
	} else if !found {
		return math.LegacyZeroDec(), nil
	}

	if basePrice, err := fc.keeper.GetBaseSpotPrice(ctx, quoteDenom); err != nil {
		return math.LegacyZeroDec(), err
	} else {
		return basePrice, nil
	}
}
