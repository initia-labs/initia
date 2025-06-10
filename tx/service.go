package tx

import (
	"context"

	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktxtypes "github.com/cosmos/cosmos-sdk/types/tx"

	txtypes "github.com/initia-labs/initia/tx/types"

	txcli "github.com/initia-labs/initia/tx/cli"
)

type txQueryServer struct {
	clientCtx client.Context
	gpk       GasPriceKeeper
}

type GasPriceKeeper interface {
	GasPrices(ctx context.Context) (sdk.DecCoins, error)
	GasPrice(ctx context.Context, denom string) (sdk.DecCoin, error)
}

func NewTxQueryServer(clientCtx client.Context, k GasPriceKeeper) txtypes.QueryServer {
	return &txQueryServer{clientCtx: clientCtx, gpk: k}
}

// GasPrices implements QueryServer.
func (t *txQueryServer) GasPrices(ctx context.Context, req *txtypes.QueryGasPricesRequest) (*txtypes.QueryGasPricesResponse, error) {
	prices, err := t.gpk.GasPrices(ctx)
	if err != nil {
		return nil, err
	}

	return &txtypes.QueryGasPricesResponse{GasPrices: prices}, nil
}

// GasPrice implements QueryServer.
func (t *txQueryServer) GasPrice(ctx context.Context, req *txtypes.QueryGasPriceRequest) (*txtypes.QueryGasPriceResponse, error) {
	price, err := t.gpk.GasPrice(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	return &txtypes.QueryGasPriceResponse{GasPrice: price}, nil
}

// TxsByEvents implements QueryServer.
func (t *txQueryServer) TxsByEvents(ctx context.Context, req *txtypes.TxsByEventsRequest) (*txtypes.TxsByEventsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	result, err := txcli.TxSearchV2(t.clientCtx, int(req.Page), int(req.Limit), req.Query)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	txsList := make([]*sdktxtypes.Tx, len(result.Txs))
	for i, tx := range result.Txs {
		protoTx, ok := tx.Tx.GetCachedValue().(*sdktxtypes.Tx)
		if !ok {
			return nil, status.Errorf(codes.Internal, "expected %T, got %T", sdktxtypes.Tx{}, tx.Tx.GetCachedValue())
		}

		txsList[i] = protoTx
	}

	return &txtypes.TxsByEventsResponse{
		Txs:         txsList,
		TxResponses: result.Txs,
		Total:       result.TotalCount,
	}, nil
}

// RegisterTxQuery registers the tx query on the gRPC router.
func RegisterQueryService(qrt gogogrpc.Server, clientCtx client.Context, gpk GasPriceKeeper) {
	txtypes.RegisterQueryServer(qrt, NewTxQueryServer(clientCtx, gpk))
}

// RegisterGRPCGatewayRoutes mounts the tx query's GRPC-gateway routes on the given Mux.
func RegisterGRPCGatewayRoutes(clientConn gogogrpc.ClientConn, mux *runtime.ServeMux) {
	_ = txtypes.RegisterQueryHandlerClient(context.Background(), mux, txtypes.NewQueryClient(clientConn))
}
