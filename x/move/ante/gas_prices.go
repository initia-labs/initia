package ante

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// GasPricesDecorator ante decorator to set simulation flag to a context
type GasPricesDecorator struct{}

// NewGasPricesDecorator constructor of the GasPricesDecorator
func NewGasPricesDecorator() *GasPricesDecorator {
	return &GasPricesDecorator{}
}

// AnteHandle that store gas prices to a context to let the move keeper know tx gas prices.
func (d GasPricesDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	feeCoins := feeTx.GetFee()
	gas := feeTx.GetGas()

	if !simulate {
		if gas == 0 {
			return ctx, errors.Wrap(sdkerrors.ErrOutOfGas, "gas is not provided")
		}

		// CSR: store a tx gas prices
		ctx = ctx.WithValue(GasPricesContextKey, sdk.NewDecCoinsFromCoins(feeCoins...).QuoDec(math.LegacyNewDec(int64(gas))))
	}

	if next != nil {
		return next(ctx, tx, simulate)
	}

	return ctx, nil
}
