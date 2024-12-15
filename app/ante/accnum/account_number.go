package accnum

import (
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmosante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
)

// AccountNumberDecorator is a custom AnteHandler that increments the account number
// to avoid conflicts when running concurrent Simulate, CheckTx, and Finalize operations.
type AccountNumberDecorator struct {
	accountKeeper cosmosante.AccountKeeper
}

// NewAccountNumberDecorator creates a new AccountNumberDecorator.
func NewAccountNumberDecorator(accountKeeper cosmosante.AccountKeeper) AccountNumberDecorator {
	return AccountNumberDecorator{accountKeeper: accountKeeper}
}

// AnteHandle implements the AnteHandler interface.
// It increments the account number depending on the execution mode (Simulate, CheckTx).
func (and AccountNumberDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Skip account number modification for FinalizeTx or DeliverTx.
	if !ctx.IsCheckTx() && !ctx.IsReCheckTx() && !simulate {
		return next(ctx, tx, simulate)
	}

	// Safely cast to the concrete implementation of AccountKeeper.
	authKeeper, ok := and.accountKeeper.(*authkeeper.AccountKeeper)
	if !ok {
		return ctx, sdk.ErrInvalidRequest.Wrap("invalid AccountKeeper type")
	}

	// Create a gas-free context to interact with the account number storage.
	gasFreeCtx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	// Peek at the current account number.
	currentAccountNum, err := authKeeper.AccountNumber.Peek(gasFreeCtx)
	if err != nil {
		return ctx, sdk.ErrInternal.Wrapf("failed to peek account number: %v", err)
	}

	// Determine the increment value based on the execution mode.
	accountNumIncrement := uint64(1_000_000)
	if simulate {
		accountNumIncrement += 1_000_000
	}

	// Increment and set the account number.
	newAccountNum := currentAccountNum + accountNumIncrement
	if err := authKeeper.AccountNumber.Set(gasFreeCtx, newAccountNum); err != nil {
		return ctx, sdk.ErrInternal.Wrapf("failed to set account number: %v", err)
	}

	// Proceed to the next AnteHandler.
	return next(ctx, tx, simulate)
}
