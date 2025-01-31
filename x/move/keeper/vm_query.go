package keeper

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func (k Keeper) HandleVMQuery(ctx sdk.Context, req *vmtypes.QueryRequest) (res []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in HandleVMQuery: %v", r)
		}

		err = types.ErrVMQueryFailed.Wrap(err.Error())
	}()

	switch {
	case req.Custom != nil:
		return k.queryCustom(ctx, req.Custom)
	case req.Stargate != nil:
		return k.queryStargate(ctx, req.Stargate)
	}

	return nil, types.ErrInvalidQueryRequest
}

func (k Keeper) queryCustom(ctx sdk.Context, req *vmtypes.CustomQuery) ([]byte, error) {
	customQuery, exists := k.vmQueryWhiteList.Custom[req.Name]
	if !exists {
		return nil, types.ErrNotSupportedCustomQuery
	}

	// create cache context for query
	ctx, _ = ctx.CacheContext()
	return customQuery(ctx, req.Data)
}

func (k Keeper) queryStargate(ctx sdk.Context, req *vmtypes.StargateQuery) ([]byte, error) {
	protoSet, exists := k.vmQueryWhiteList.Stargate[req.Path]
	if !exists {
		return nil, types.ErrNotSupportedStargateQuery
	}

	route := k.grpcRouter.Route(req.Path)
	if route == nil {
		return nil, types.ErrNotSupportedStargateQuery
	}

	reqData, err := types.ConvertJSONMarshalToProto(k.cdc, protoSet.Request, req.Data)
	if err != nil {
		return nil, err
	}

	// create cache context for query
	ctx, _ = ctx.CacheContext()
	res, err := route(ctx, &abci.RequestQuery{
		Data: reqData,
		Path: req.Path,
	})
	if err != nil {
		return nil, err
	}

	return types.ConvertProtoToJSONMarshal(k.cdc, protoSet.Response, res.Value)
}
