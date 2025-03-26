package ante

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// custom block gas meter to accumulate gas limit not consumed
type BlockGasMeter interface {
	AccumulateGas(ctx context.Context, gas uint64) error
}

// GasPricesDecorator ante decorator to set gas prices to a context
// and accumulate gas used in the block
type GasPricesDecorator struct {
	blockGasMeter BlockGasMeter
}

// NewGasPricesDecorator constructor of the GasPricesDecorator
func NewGasPricesDecorator(blockGasMeter BlockGasMeter) *GasPricesDecorator {
	return &GasPricesDecorator{
		blockGasMeter: blockGasMeter,
	}
}

// AnteHandle that store gas prices to a context and accumulate gas used in the block
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
		ctx = ctx.WithValue(GasPricesContextKey, sdk.NewDecCoinsFromCoins(feeCoins...).QuoDecTruncate(math.LegacyNewDec(int64(gas))))
	}

	// record the gas amount to the block gas meter
	// NOTE: use infinite gas meter to avoid gas charge for chain operation
	if !simulate && !ctx.IsCheckTx() {
		if err := d.blockGasMeter.AccumulateGas(ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()), gas); err != nil {
			return ctx, err
		}
	}

	if next != nil {
		return next(ctx, tx, simulate)
	}

	return ctx, nil
}
