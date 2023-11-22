package service

import (
	"context"

	gogogrpc "github.com/cosmos/gogoproto/grpc"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
)

type queryServer struct {
	tmservice.ServiceServer
}

// NewQueryServer creates a new tendermint query server.
func NewQueryServer(
	originServer tmservice.ServiceServer,
) tmservice.ServiceServer {
	return queryServer{originServer}
}

// Override GetLatestBlock to ignore first tx in a block.
// - https://github.com/skip-mev/block-sdk/issues/215
func (s queryServer) GetLatestBlock(ctx context.Context, req *tmservice.GetLatestBlockRequest) (*tmservice.GetLatestBlockResponse, error) {
	res, err := s.ServiceServer.GetLatestBlock(ctx, req)
	if err != nil {
		return nil, err
	}

	if res.Block != nil && len(res.Block.Data.Txs) > 0 {
		res.Block.Data.Txs = res.Block.Data.Txs[1:]
	}

	if res.SdkBlock != nil && len(res.SdkBlock.Data.Txs) > 0 {
		res.SdkBlock.Data.Txs = res.SdkBlock.Data.Txs[1:]
	}

	return res, err
}

// Override GetBlockByHeight to ignore first tx in a block.
// - https://github.com/skip-mev/block-sdk/issues/215
func (s queryServer) GetBlockByHeight(ctx context.Context, req *tmservice.GetBlockByHeightRequest) (*tmservice.GetBlockByHeightResponse, error) {
	res, err := s.ServiceServer.GetBlockByHeight(ctx, req)
	if err != nil {
		return nil, err
	}

	if res.Block != nil && len(res.Block.Data.Txs) > 0 {
		res.Block.Data.Txs = res.Block.Data.Txs[1:]
	}

	if res.SdkBlock != nil && len(res.SdkBlock.Data.Txs) > 0 {
		res.SdkBlock.Data.Txs = res.SdkBlock.Data.Txs[1:]
	}

	return res, err
}

// RegisterTendermintService registers the tendermint queries on the gRPC router.
func RegisterTendermintService(
	server gogogrpc.Server,
	originService tmservice.ServiceServer,
) {
	tmservice.RegisterServiceServer(server, NewQueryServer(originService))
}
