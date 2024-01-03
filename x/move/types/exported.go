package types

import (
	context "context"

	"cosmossdk.io/math"
)

type AnteKeeper interface {
	HasDexPair(ctx context.Context, denom string) (bool, error)
	GetPoolSpotPrice(ctx context.Context, denomQuote string) (math.LegacyDec, error)
	BaseDenom(ctx context.Context) (string, error)
	BaseMinGasPrice(ctx context.Context) (math.LegacyDec, error)
}
