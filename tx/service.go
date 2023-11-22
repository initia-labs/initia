package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	gogogrpc "github.com/cosmos/gogoproto/grpc"
)

var _ txtypes.ServiceServer = txServer{}

type txServer struct {
	clientCtx client.Context
	txtypes.ServiceServer
}

func NewTxServer(clientCtx client.Context, originTxServer txtypes.ServiceServer) txtypes.ServiceServer {
	return txServer{
		clientCtx:     clientCtx,
		ServiceServer: originTxServer,
	}
}

// protoTxProvider is a type which can provide a proto transaction. It is a
// workaround to get access to the wrapper TxBuilder's method GetProtoTx().
// ref: https://github.com/cosmos/cosmos-sdk/issues/10347
type protoTxProvider interface {
	GetProtoTx() *txtypes.Tx
}

// Override GetBlockWithTxs to ignore first tx in a block.
// - https://github.com/skip-mev/block-sdk/issues/215
func (s txServer) GetBlockWithTxs(ctx context.Context, req *txtypes.GetBlockWithTxsRequest) (*txtypes.GetBlockWithTxsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	if req.Height < 1 || req.Height > currentHeight {
		return nil, sdkerrors.ErrInvalidHeight.Wrapf("requested height %d but height must not be less than 1 "+
			"or greater than the current height %d", req.Height, currentHeight)
	}

	blockID, block, err := tmservice.GetProtoBlock(ctx, s.clientCtx, &req.Height)
	if err != nil {
		return nil, err
	}

	var offset, limit uint64
	if req.Pagination != nil {
		offset = req.Pagination.Offset
		limit = req.Pagination.Limit
	} else {
		offset = 0
		limit = query.DefaultLimit
	}

	// ignore first tx
	block.Data.Txs = block.Data.Txs[1:]

	blockTxs := block.Data.Txs
	blockTxsLn := uint64(len(blockTxs))
	txs := make([]*txtypes.Tx, 0, limit)
	if offset >= blockTxsLn && blockTxsLn != 0 {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("out of range: cannot paginate %d txs with offset %d and limit %d", blockTxsLn, offset, limit)
	}
	decodeTxAt := func(i uint64) error {
		tx := blockTxs[i]
		txb, err := s.clientCtx.TxConfig.TxDecoder()(tx)
		if err != nil {
			return err
		}
		p, ok := txb.(protoTxProvider)
		if !ok {
			return sdkerrors.ErrTxDecode.Wrapf("could not cast %T to %T", txb, txtypes.Tx{})
		}
		txs = append(txs, p.GetProtoTx())
		return nil
	}
	if req.Pagination != nil && req.Pagination.Reverse {
		for i, count := offset, uint64(0); i > 0 && count != limit; i, count = i-1, count+1 {
			if err = decodeTxAt(i); err != nil {
				return nil, err
			}
		}
	} else {
		for i, count := offset, uint64(0); i < blockTxsLn && count != limit; i, count = i+1, count+1 {
			if err = decodeTxAt(i); err != nil {
				return nil, err
			}
		}
	}

	return &txtypes.GetBlockWithTxsResponse{
		Txs:     txs,
		BlockId: &blockID,
		Block:   block,
		Pagination: &query.PageResponse{
			Total: blockTxsLn,
		},
	}, nil
}

// RegisterTxService registers the tx service on the gRPC router.
func RegisterTxService(
	qrt gogogrpc.Server,
	clientCtx client.Context,
	originTxServer txtypes.ServiceServer,
) {
	txtypes.RegisterServiceServer(qrt, NewTxServer(clientCtx, originTxServer))
}
