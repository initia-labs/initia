package keeper

import (
	"context"
	"fmt"

	"github.com/initia-labs/initia/x/ibc/fetchprice/consumer/types"
)

// InitGenesis initializes the ibc fetchprice consumer state and binds to PortID.
func (k Keeper) InitGenesis(ctx context.Context, state types.GenesisState) {
	if err := k.PortID.Set(ctx, state.PortId); err != nil {
		panic(err)
	}

	for _, cp := range state.CurrencyPrices {
		if err := k.Prices.Set(ctx, cp.CurrencyId, cp.QuotePrice); err != nil {
			panic(err)
		}
	}

	// Only try to bind to port if it is not already bound, since we may already own
	// port capability from capability InitGenesis
	if !k.IsBound(ctx, state.PortId) {
		// transfer module binds to the transfer port on InitChain
		// and claims the returned capability
		err := k.BindPort(ctx, state.PortId)
		if err != nil {
			panic(fmt.Sprintf("could not claim port capability: %v", err))
		}
	}
}

// ExportGenesis exports ibc fetchprice consumer module's portID and currency pairs into its genesis state.
func (k Keeper) ExportGenesis(ctx context.Context) types.GenesisState {
	portID, err := k.PortID.Get(ctx)
	if err != nil {
		panic(err)
	}

	allCurrencyPrices, err := k.GetAllCurrencyPrices(ctx)
	if err != nil {
		panic(err)
	}

	return types.GenesisState{
		PortId:         portID,
		CurrencyPrices: allCurrencyPrices,
	}
}
