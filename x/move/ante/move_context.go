package ante

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	movetypes "github.com/initia-labs/initia/x/move/types"

	vmtypes "github.com/initia-labs/movevm/types"
)

// MoveContextDecorator sets Move-related values on the context.
type MoveContextDecorator struct{}

// NewMoveContextDecorator returns a MoveContextDecorator.
func NewMoveContextDecorator() *MoveContextDecorator {
	return &MoveContextDecorator{}
}

// GasPricesDecorator sets Move-related values on the context.
//
// Deprecated: use MoveContextDecorator.
type GasPricesDecorator = MoveContextDecorator

// NewGasPricesDecorator returns a MoveContextDecorator.
//
// Deprecated: use NewMoveContextDecorator.
func NewGasPricesDecorator() *MoveContextDecorator {
	return NewMoveContextDecorator()
}

// AnteHandle sets Move-related values on the context.
func (d MoveContextDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if !simulate {
		feeCoins := feeTx.GetFee()
		gas := feeTx.GetGas()
		if gas == 0 {
			return ctx, errors.Wrap(sdkerrors.ErrOutOfGas, "Transaction gas cannot be zero.")
		}

		// store a tx gas prices
		ctx = ctx.WithValue(movetypes.GasPricesContextKey, sdk.NewDecCoinsFromCoins(feeCoins...).QuoDecTruncate(math.LegacyNewDec(int64(gas)))) //nolint: gosec
	}

	feePayer := feeTx.FeePayer()
	feePayerVMAddr, err := vmtypes.NewAccountAddressFromBytes(feePayer)
	if err == nil {
		ctx = ctx.WithValue(movetypes.FeePayerContextKey, &feePayerVMAddr)
	}

	if next != nil {
		return next(ctx, tx, simulate)
	}

	return ctx, nil
}
