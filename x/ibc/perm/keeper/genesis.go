package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

// InitGenesis initializes the ibc-perm state.
func (k Keeper) InitGenesis(ctx sdk.Context, genesisState types.GenesisState) {
	for _, channelRelayer := range genesisState.ChannelRelayers {
		addr, err := sdk.AccAddressFromBech32(channelRelayer.Relayer)
		if err != nil {
			panic(err)
		}

		k.SetChannelRelayer(ctx, channelRelayer.Channel, addr)
	}
}

// ExportGenesis exports ibc-perm module's channel relayers.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	channelRelayers := []types.ChannelRelayer{}
	k.IterateChannelRelayer(ctx, func(channelRelayer types.ChannelRelayer) bool {
		channelRelayers = append(channelRelayers, channelRelayer)
		return false
	})

	return &types.GenesisState{
		ChannelRelayers: channelRelayers,
	}
}
