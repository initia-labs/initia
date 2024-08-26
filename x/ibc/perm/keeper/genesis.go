package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

// InitGenesis initializes the ibc-perm state.
func (k Keeper) InitGenesis(ctx sdk.Context, genesisState types.GenesisState) {
	for _, channelState := range genesisState.ChannelStates {
		err := k.SetChannelState(ctx, channelState)
		if err != nil {
			panic(err)
		}
	}
}

// ExportGenesis exports ibc-perm module's channel relayers.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	channelStates := []types.ChannelState{}
	err := k.IterateChannelStates(ctx, func(channelRelayer types.ChannelState) (bool, error) {
		channelStates = append(channelStates, channelRelayer)
		return false, nil
	})

	if err != nil {
		panic(err)
	}

	return &types.GenesisState{
		ChannelStates: channelStates,
	}
}
