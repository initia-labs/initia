package ante

import (
	"context"

	"cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// custom block gas meter to accumulate gas limit not consumed
type BlockGasMeter interface {
	AccumulateGas(ctx context.Context, gas uint64) error
}

// BlockGasDecorator ante decorator to accumulate gas used in the block
type BlockGasDecorator struct {
	blockGasMeter BlockGasMeter
}

// NewBlockGasDecorator constructor of the BlockGasDecorator
func NewBlockGasDecorator(blockGasMeter BlockGasMeter) *BlockGasDecorator {
	return &BlockGasDecorator{
		blockGasMeter: blockGasMeter,
	}
}

// AnteHandle that accumulate gas used in the block
func (d BlockGasDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	gas := feeTx.GetGas()

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
