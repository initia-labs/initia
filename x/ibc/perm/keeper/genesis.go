package keeper

import (
	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

// InitGenesis initializes the ibc-perm state.
func (k Keeper) InitGenesis(ctx sdk.Context, genesisState types.GenesisState) {
	for _, channelRelayerSet := range genesisState.PermissionedRelayerSets {
		var channelRelayers []string
		for _, channelRelayer := range channelRelayerSet.RelayerList.Relayers {
			_, err := k.ac.StringToBytes(channelRelayer)
			if err != nil {
				panic(err)
			}
		}
		if err := k.PermissionedRelayers.Set(ctx, collections.Join(channelRelayerSet.PortId, channelRelayerSet.ChannelId), types.PermissionedRelayerList{
			Relayers: append(channelRelayerSet.RelayerList.Relayers, channelRelayers...),
		}); err != nil {
			panic(err)
		}

	}
}

// ExportGenesis exports ibc-perm module's channel relayers.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	channelRelayerSets := []types.PermissionedRelayersSet{}
	err := k.IteratePermissionedRelayers(ctx, func(channelRelayer types.PermissionedRelayersSet) (bool, error) {
		channelRelayerSets = append(channelRelayerSets, channelRelayer)
		return false, nil
	})
	if err != nil {
		panic(err)
	}

	return &types.GenesisState{
		PermissionedRelayerSets: channelRelayerSets,
	}
}
