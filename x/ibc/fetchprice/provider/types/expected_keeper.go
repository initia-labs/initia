package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

	oracletypes "github.com/skip-mev/slinky/x/oracle/types"
)

type OracleKeeper interface {
	GetPriceForCurrencyPair(ctx sdk.Context, cp oracletypes.CurrencyPair) (oracletypes.QuotePrice, error)
}

// PortKeeper defines the expected IBC port keeper
type PortKeeper interface {
	BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability
}
