package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

// InitGenesis initializes the ibc-perm state.
func (k Keeper) InitGenesis(ctx sdk.Context, genesisState types.GenesisState) {
	for _, channelRelayer := range genesisState.ChannelRelayers {
		addr, err := k.ac.StringToBytes(channelRelayer.Relayer)
		if err != nil {
			panic(err)
		}

		if err := k.ChannelRelayers.Set(ctx, channelRelayer.Channel, addr); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis exports ibc-perm module's channel relayers.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	channelRelayers := []types.ChannelRelayer{}
	err := k.IterateChannelRelayer(ctx, func(channelRelayer types.ChannelRelayer) (bool, error) {
		channelRelayers = append(channelRelayers, channelRelayer)
		return false, nil
	})
	if err != nil {
		panic(err)
	}

	return &types.GenesisState{
		ChannelRelayers: channelRelayers,
	}
}
