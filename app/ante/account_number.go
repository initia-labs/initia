package ante

import (
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmosante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
)

// AccountNumberDecorator is a custom ante handler that increments the account number depending on
// the execution mode (Simulate, CheckTx, Finalize).
//
// This is to avoid account number conflicts when running concurrent Simulate, CheckTx, and Finalize.
type AccountNumberDecorator struct {
	ak cosmosante.AccountKeeper
}

// NewAccountNumberDecorator creates a new instance of AccountNumberDecorator.
func NewAccountNumberDecorator(ak cosmosante.AccountKeeper) AccountNumberDecorator {
	return AccountNumberDecorator{ak}
}

// AnteHandle is the AnteHandler implementation for AccountNumberDecorator.
func (and AccountNumberDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !ctx.IsCheckTx() && !ctx.IsReCheckTx() && !simulate {
		return next(ctx, tx, simulate)
	}

	ak := and.ak.(*authkeeper.AccountKeeper)

	gasFreeCtx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	num, err := ak.AccountNumber.Peek(gasFreeCtx)
	if err != nil {
		return ctx, err
	}

	accountNumAddition := uint64(1_000_000)
	if simulate {
		accountNumAddition += 1_000_000
	}

	if err := ak.AccountNumber.Set(gasFreeCtx, num+accountNumAddition); err != nil {
		return ctx, err
	}

	return next(ctx, tx, simulate)
}
