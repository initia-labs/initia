package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/move/types"
)

type AnteKeeper struct {
	DexKeeper
	eip1559FeeKeeper EIP1559FeeKeeper
}

var _ types.AnteKeeper = AnteKeeper{}

func NewAnteKeeper(dexKeeper DexKeeper, eip1559FeeKeeper EIP1559FeeKeeper) AnteKeeper {
	return AnteKeeper{
		DexKeeper:        dexKeeper,
		eip1559FeeKeeper: eip1559FeeKeeper,
	}
}

func (k AnteKeeper) GetBaseFee(ctx context.Context) (int64, error) {
	return k.eip1559FeeKeeper.GetBaseFee(ctx)
}
