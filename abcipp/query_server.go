package abcipp

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"cosmossdk.io/math"
	gogogrpc "github.com/cosmos/gogoproto/grpc"

	"github.com/initia-labs/initia/abcipp/types"
)

// RegisterQueryServer registers the ABCI++ query server.
func RegisterQueryServer(
	server gogogrpc.Server,
	mempool Mempool,
) {
	types.RegisterQueryServer(server, &MempoolQueryServer{mempool: mempool})
}

// RegisterGRPCGatewayRoutes mounts the ABCI++ query's GRPC-gateway routes on the
// given Mux.
func RegisterGRPCGatewayRoutes(clientConn gogogrpc.ClientConn, mux *runtime.ServeMux) {
	_ = types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientConn))
}

// MempoolQueryServer implements the ABCI++ mempool query server.
type MempoolQueryServer struct {
	mempool Mempool
}

// QueryTxDistribution returns the current distribution of transactions in the mempool.
func (p *MempoolQueryServer) QueryTxDistribution(ctx context.Context, req *types.QueryTxDistributionRequest) (*types.QueryTxDistributionResponse, error) {
	dist := p.mempool.GetTxDistribution()
	return &types.QueryTxDistributionResponse{
		Distribution: dist,
	}, nil
}

// QueryTxHash looks up the transaction hash for a given sender and sequence number.
func (p *MempoolQueryServer) QueryTxHash(ctx context.Context, req *types.QueryTxHashRequest) (*types.QueryTxHashResponse, error) {
	addr, err := DecodeAddress(req.Sender)
	if err != nil {
		return nil, err
	}

	sender := addr.String()
	nonce, ok := math.NewIntFromString(req.Sequence)
	if !ok {
		return nil, fmt.Errorf("invalid sequence number: %s", req.Sequence)
	}
	if !nonce.IsUint64() {
		return nil, fmt.Errorf("sequence number out of range: %s", req.Sequence)
	}

	txHash, ok := p.mempool.Lookup(sender, nonce.Uint64())
	if ok {
		return &types.QueryTxHashResponse{
			TxHash: txHash,
		}, nil
	}

	return &types.QueryTxHashResponse{
		TxHash: "",
	}, nil
}
