package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

var _ types.QueryServer = QueryServerImpl{}

type QueryServerImpl struct {
	*Keeper
}

func NewQueryServer(k *Keeper) QueryServerImpl {
	return QueryServerImpl{k}
}

// ChannelRelayer implements the Query/ChannelRelayer gRPC method
func (q QueryServerImpl) ChannelRelayer(c context.Context, req *types.QueryChannelRelayerRequest) (*types.QueryChannelRelayerResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	relayer, err := q.ChannelRelayers.Get(ctx, req.GetChannel())
	if err != nil {
		return nil, err
	}

	relayerStr, err := q.ac.BytesToString(relayer)
	if err != nil {
		return nil, err
	}

	return &types.QueryChannelRelayerResponse{
		ChannelRelayer: &types.ChannelRelayer{
			Channel: req.GetChannel(),
			Relayer: relayerStr,
		},
	}, nil
}
