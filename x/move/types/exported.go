package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AnteKeeper interface {
	HasDexPair(ctx sdk.Context, denom string) (bool, error)
	GetPoolSpotPrice(ctx sdk.Context, denomQuote string) (math.LegacyDec, error)
	BaseDenom(ctx sdk.Context) (res string)
	BaseMinGasPrice(ctx sdk.Context) math.LegacyDec
}
