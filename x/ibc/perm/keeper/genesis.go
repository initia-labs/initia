package keeper

import (
	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

// InitGenesis initializes the ibc-perm state.
func (k Keeper) InitGenesis(ctx sdk.Context, genesisState types.GenesisState) {

	for _, relayersByChannel := range genesisState.PermissionedRelayers {
		var channelRelayers []string
		for _, channelRelayer := range relayersByChannel.Relayers {
			_, err := k.ac.StringToBytes(channelRelayer)
			if err != nil {
				panic(err)
			}
			channelRelayers = append(channelRelayers, channelRelayer)
		}
		if err := k.PermissionedRelayers.Set(ctx, collections.Join(relayersByChannel.PortId, relayersByChannel.ChannelId), types.PermissionedRelayersList{
			Relayers: channelRelayers,
		}); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis exports ibc-perm module's channel relayers.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	channelRelayerSets := []types.PermissionedRelayers{}
	err := k.IteratePermissionedRelayers(ctx, func(channelRelayer types.PermissionedRelayers) (bool, error) {
		channelRelayerSets = append(channelRelayerSets, channelRelayer)
		return false, nil
	})
	if err != nil {
		panic(err)
	}

	return &types.GenesisState{
		PermissionedRelayers: channelRelayerSets,
	}
}
