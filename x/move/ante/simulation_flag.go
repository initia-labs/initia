package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SimulationFlagDecorator ante decorator to set simulation flag to a context
type SimulationFlagDecorator struct{}

// NewSimulationFlagDecorator constructor of the SimulationFlagDecorator
func NewSimulationFlagDecorator() *SimulationFlagDecorator {
	return &SimulationFlagDecorator{}
}

// AnteHandle that set simulation flag to a context to let the move keeper know tx mode.
func (d SimulationFlagDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	ctx = ctx.WithValue(SimulationFlagContextKey, simulate)

	if next != nil {
		return next(ctx, tx, simulate)
	}

	return ctx, nil
}
