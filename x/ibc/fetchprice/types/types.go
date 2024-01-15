package types

import (
	oracletypes "github.com/skip-mev/slinky/x/oracle/types"
)

func (p QuotePrice) ToOracleQuotePrice() oracletypes.QuotePrice {
	return oracletypes.QuotePrice{
		Price:          p.Price,
		BlockTimestamp: p.BlockTimestamp,
		BlockHeight:    p.BlockHeight,
	}
}
