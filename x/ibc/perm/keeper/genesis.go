package keeper

import (
	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

// InitGenesis initializes the ibc-perm state.
func (k Keeper) InitGenesis(ctx sdk.Context, genesisState types.GenesisState) {
	for _, channelRelayer := range genesisState.PermissionedRelayers {
		addr, err := k.ac.StringToBytes(channelRelayer.Relayer)
		if err != nil {
			panic(err)
		}

		if err := k.PermissionedRelayers.Set(ctx, collections.Join(channelRelayer.PortId, channelRelayer.ChannelId), addr); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis exports ibc-perm module's channel relayers.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	channelRelayers := []types.PermissionedRelayer{}
	err := k.IteratePermissionedRelayers(ctx, func(channelRelayer types.PermissionedRelayer) (bool, error) {
		channelRelayers = append(channelRelayers, channelRelayer)
		return false, nil
	})
	if err != nil {
		panic(err)
	}

	return &types.GenesisState{
		PermissionedRelayers: channelRelayers,
	}
}
