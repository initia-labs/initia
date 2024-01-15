package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/ibc/fetchprice/types"
)

func (k Keeper) GetAllCurrencyPrices(ctx context.Context) (pairs []types.CurrencyPrice, err error) {
	err = k.Prices.Walk(ctx, nil, func(currencyId string, quotePrice types.QuotePrice) (stop bool, err error) {
		pairs = append(pairs, types.CurrencyPrice{
			CurrencyId: currencyId,
			QuotePrice: quotePrice,
		})

		return false, nil
	})

	return
}
