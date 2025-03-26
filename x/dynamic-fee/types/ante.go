package types

import (
	context "context"

	"cosmossdk.io/math"
)

type AnteKeeper interface {
	GetBaseSpotPrice(ctx context.Context, denomQuote string) (math.LegacyDec, error)
	BaseDenom(ctx context.Context) (string, error)
	BaseGasPrice(ctx context.Context) (math.LegacyDec, error)
	AccumulateGas(ctx context.Context, gas uint64) error
}
