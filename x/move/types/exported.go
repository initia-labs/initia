package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AnteKeeper interface {
	HasDexPair(ctx sdk.Context, denom string) (bool, error)
	GetPoolSpotPrice(ctx sdk.Context, denomQuote string) (sdk.Dec, error)
	BaseDenom(ctx sdk.Context) (res string)
	BaseMinGasPrice(ctx sdk.Context) sdk.Dec
}
