package ante

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	dynamicfeetypes "github.com/initia-labs/initia/x/dynamic-fee/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// MempoolFeeChecker will check if the transaction's fee is at least as large
// as the local validator's minimum gasFee (defined in validator config).
// If fee is too low, decorator returns error and tx is rejected from mempool.
// Note this only applies when ctx.CheckTx = true
// If fee is high enough or not CheckTx, then call next AnteHandler
// CONTRACT: Tx must implement FeeTx to use MempoolFeeChecker
type MempoolFeeChecker struct {
	keeper dynamicfeetypes.AnteKeeper
}

// NewMempoolFeeChecker create MempoolFeeChecker instance
func NewMempoolFeeChecker(
	keeper dynamicfeetypes.AnteKeeper,
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

			for _, coin := range feeTx.GetFee() {
				basePrice, err := fc.keeper.GetBaseSpotPrice(ctx, coin.Denom)
				if err != nil {
					return nil, 0, err
				}

				quoteAmount := coin.Amount
				baseAmount := basePrice.MulInt(quoteAmount).TruncateInt()
				totalFeeBaseAmount = totalFeeBaseAmount.Add(baseAmount)
			}
			if totalFeeBaseAmount.GT(math.OneInt()) {
				priority = totalFeeBaseAmount.Int64()
			}

			baseGasPrice, err := fc.keeper.BaseGasPrice(ctx)
			if err != nil {
				return nil, 0, err
			}

			gasPriceFromTotalFee := math.LegacyNewDecFromInt(totalFeeBaseAmount).Quo(math.LegacyNewDec(int64(gas)))

			if gasPriceFromTotalFee.LT(baseGasPrice) {
				return nil, 0, errors.Wrapf(
					sdkerrors.ErrInsufficientFee,
					"insufficient gas price; got: %s (sum %s), base gas price required: %s",
					feeCoins,
					gasPriceFromTotalFee.String(),
					baseGasPrice.String(),
				)
			}
		}

		if !minGasPrices.IsZero() {
			requiredFees := computeRequiredFees(gas, minGasPrices)

			if !feeCoins.IsAnyGTE(requiredFees) {
				// convert baseDenom min gas prices to quote denom prices
				// and check the paid fee is enough or not.
				isSufficient := false

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
						totalFeeBaseAmount,
						requiredFees,
					)
				}
			}
		}
	}

	return feeCoins, priority, nil
}
