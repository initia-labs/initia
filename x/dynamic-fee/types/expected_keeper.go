package types

import (
	context "context"

	"cosmossdk.io/math"
)

type TokenPriceKeeper interface {
	GetBaseSpotPrice(ctx context.Context, denom string) (math.LegacyDec, error)
}

type WhitelistKeeper interface {
	GetWhitelistedTokens(ctx context.Context) ([]string, error)
}

type BaseDenomKeeper interface {
	BaseDenom(ctx context.Context) (string, error)
}
