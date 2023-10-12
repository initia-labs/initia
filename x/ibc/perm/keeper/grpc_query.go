package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

var _ types.QueryServer = Keeper{}

// ChannelRelayer implements the Query/ChannelRelayer gRPC method
func (q Keeper) ChannelRelayer(c context.Context, req *types.QueryChannelRelayerRequest) (*types.QueryChannelRelayerResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	return &types.QueryChannelRelayerResponse{
		ChannelRelayer: q.GetChannelRelayer(ctx, req.GetChannel()),
	}, nil
}
