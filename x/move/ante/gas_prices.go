package ante

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	movetypes "github.com/initia-labs/initia/x/move/types"
)

// GasPricesDecorator ante decorator to set gas prices to a context
type GasPricesDecorator struct{}

// NewGasPricesDecorator constructor of the GasPricesDecorator
func NewGasPricesDecorator() *GasPricesDecorator {
	return &GasPricesDecorator{}
}

// AnteHandle that set gas prices to a context
func (d GasPricesDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	feeCoins := feeTx.GetFee()
	gas := feeTx.GetGas()

	if !simulate {
		if gas == 0 {
			return ctx, errors.Wrap(sdkerrors.ErrOutOfGas, "Transaction gas cannot be zero.")
		}

		// store a tx gas prices
		ctx = ctx.WithValue(movetypes.GasPricesContextKey, sdk.NewDecCoinsFromCoins(feeCoins...).QuoDecTruncate(math.LegacyNewDec(int64(gas)))) //nolint: gosec
	}

	if next != nil {
		return next(ctx, tx, simulate)
	}

	return ctx, nil
}
