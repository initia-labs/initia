package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	consumertypes "github.com/initia-labs/initia/x/ibc/fetchprice/consumer/types"
	"github.com/initia-labs/initia/x/ibc/fetchprice/types"

	oracletypes "github.com/skip-mev/slinky/x/oracle/types"
)

const queryHardLimit = 100

var _ consumertypes.QueryServer = &QueryServer{}

type QueryServer struct {
	*Keeper
}

func NewQueryServerImpl(k *Keeper) consumertypes.QueryServer {
	return &QueryServer{k}
}

// AllPrices implements types.QueryServer.
func (qs *QueryServer) AllPrices(ctx context.Context, req *consumertypes.QueryAllPricesRequest) (*consumertypes.QueryAllPricesResponse, error) {
	prices, pageRes, err := query.CollectionPaginate(ctx, qs.Keeper.Prices, req.Pagination, func(currencyId string, quotePrice types.QuotePrice) (types.CurrencyPrice, error) {
		return types.CurrencyPrice{
			CurrencyId: currencyId,
			QuotePrice: quotePrice,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	return &consumertypes.QueryAllPricesResponse{
		Prices:     prices,
		Pagination: pageRes,
	}, nil
}

// Price implements types.QueryServer.
func (qs *QueryServer) Price(ctx context.Context, req *consumertypes.QueryPriceRequest) (*consumertypes.QueryPriceResponse, error) {
	if _, err := oracletypes.CurrencyPairFromString(req.CurrencyId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	price, err := qs.Keeper.Prices.Get(ctx, req.CurrencyId)
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return nil, status.Error(codes.NotFound, err.Error())
	} else if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &consumertypes.QueryPriceResponse{
		Price: price,
	}, nil
}

// Prices implements types.QueryServer.
func (qs *QueryServer) Prices(ctx context.Context, req *consumertypes.QueryPricesRequest) (*consumertypes.QueryPricesResponse, error) {
	if len(req.CurrencyIds) > queryHardLimit {
		return nil, status.Errorf(codes.OutOfRange, "num of max query entries is %d", queryHardLimit)
	}

	prices := make([]types.CurrencyPrice, 0, len(req.CurrencyIds))
	for _, currencyId := range req.CurrencyIds {
		if res, err := qs.Price(ctx, &consumertypes.QueryPriceRequest{
			CurrencyId: currencyId,
		}); err != nil {
			return nil, err
		} else {
			prices = append(prices, types.CurrencyPrice{
				CurrencyId: currencyId,
				QuotePrice: res.Price,
			})
		}
	}

	return &consumertypes.QueryPricesResponse{
		Prices: prices,
	}, nil
}
