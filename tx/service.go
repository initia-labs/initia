package tx

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	txtypes "github.com/initia-labs/initia/v1/tx/types"
)

type txServer struct {
	gpk GasPriceKeeper
}

type GasPriceKeeper interface {
	GasPrices(ctx context.Context) (sdk.DecCoins, error)
	GasPrice(ctx context.Context, denom string) (sdk.DecCoin, error)
}

func NewTxServer(k GasPriceKeeper) txtypes.QueryServer {
	return &txServer{gpk: k}
}

// GasPrices implements QueryServer.
func (t *txServer) GasPrices(ctx context.Context, req *txtypes.QueryGasPricesRequest) (*txtypes.QueryGasPricesResponse, error) {
	prices, err := t.gpk.GasPrices(ctx)
	if err != nil {
		return nil, err
	}

	return &txtypes.QueryGasPricesResponse{GasPrices: prices}, nil
}

// GasPrice implements QueryServer.
func (t *txServer) GasPrice(ctx context.Context, req *txtypes.QueryGasPriceRequest) (*txtypes.QueryGasPriceResponse, error) {
	price, err := t.gpk.GasPrice(ctx, req.Denom)
	if err != nil {
		return nil, err
	}

	return &txtypes.QueryGasPriceResponse{GasPrice: price}, nil
}

// RegisterTxQuery registers the tx query on the gRPC router.
func RegisterTxQuery(qrt gogogrpc.Server, gpk GasPriceKeeper) {
	txtypes.RegisterQueryServer(qrt, NewTxServer(gpk))
}

// RegisterGRPCGatewayRoutes mounts the tx query's GRPC-gateway routes on the given Mux.
func RegisterGRPCGatewayRoutes(clientConn gogogrpc.ClientConn, mux *runtime.ServeMux) {
	_ = txtypes.RegisterQueryHandlerClient(context.Background(), mux, txtypes.NewQueryClient(clientConn))
}
