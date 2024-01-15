package keeper

import (
	"context"
	"fmt"

	providertypes "github.com/initia-labs/initia/x/ibc/fetchprice/provider/types"
)

// InitGenesis initializes the ibc fetchprice provider state and binds to PortID.
func (k Keeper) InitGenesis(ctx context.Context, state providertypes.GenesisState) {
	if err := k.PortID.Set(ctx, state.PortId); err != nil {
		panic(err)
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

// ExportGenesis exports ibc fetchprice provider module's portID into its genesis state.
func (k Keeper) ExportGenesis(ctx context.Context) providertypes.GenesisState {
	portID, err := k.PortID.Get(ctx)
	if err != nil {
		panic(err)
	}

	return providertypes.GenesisState{
		PortId: portID,
	}
}
