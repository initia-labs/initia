package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/initia-labs/initia/x/dynamic-fee/types"
)

type AnteKeeper struct {
	*Keeper
}

var _ types.AnteKeeper = AnteKeeper{}

func NewAnteKeeper(k *Keeper) AnteKeeper {
	return AnteKeeper{Keeper: k}
}

func (k AnteKeeper) BaseDenom(ctx context.Context) (string, error) {
	return k.baseDenomKeeper.BaseDenom(ctx)
}

func (k AnteKeeper) GetBaseSpotPrice(ctx context.Context, denom string) (math.LegacyDec, error) {
	baseDenom, err := k.BaseDenom(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	} else if baseDenom == denom {
		return math.LegacyOneDec(), nil
	}
	return k.tokenPriceKeeper.GetBaseSpotPrice(ctx, denom)
}
