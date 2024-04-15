package keepers

import "context"

// BondDenomKeeper is a keeper that holds the bond denom
// for lz-cosmos.
type BondDenomKeeper struct {
	denom string
}

func NewBondDenomKeeper(denom string) *BondDenomKeeper {
	return &BondDenomKeeper{
		denom: denom,
	}
}

func (k BondDenomKeeper) BondDenom(ctx context.Context) (string, error) {
	return k.denom, nil
}
