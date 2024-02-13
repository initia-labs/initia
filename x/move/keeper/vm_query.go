package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"
	codec "github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

type VMQueryWhiteList struct {
	Custom   CustomQueryWhiteList
	Stargate StargateQueryWhiteList
}

func DefaultVMQueryWhiteList() *VMQueryWhiteList {
	return &VMQueryWhiteList{
		Custom:   DefaultCustomQueryWhiteList(),
		Stargate: DefaultStargateQueryWhiteList(),
	}
}

func (k Keeper) HandleVMQuery(ctx sdk.Context, req *vmtypes.QueryRequest) ([]byte, error) {
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
	return customQuery(ctx, req.Data, &k)
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

	reqData, err := ConvertJSONMarshalToProto(k.cdc, protoSet.Request, req.Data)
	if err != nil {
		return nil, err
	}

	res, err := route(ctx, &abci.RequestQuery{
		Data: reqData,
		Path: req.Path,
	})
	if err != nil {
		return nil, err
	}

	return ConvertProtoToJSONMarshal(k.cdc, protoSet.Response, res.Value)
}

// ConvertProtoToJSONMarshal  unmarshals the given bytes into a proto message and then marshals it to json.
// This is done so that clients calling stargate queries do not need to define their own proto unmarshalers,
// being able to use response directly by json marshaling, which is supported in cosmwasm.
func ConvertProtoToJSONMarshal(cdc codec.Codec, protoResponse proto.Message, bz []byte) ([]byte, error) {
	// unmarshal binary into stargate response data structure
	err := cdc.Unmarshal(bz, protoResponse)
	if err != nil {
		return nil, err
	}

	bz, err = cdc.MarshalJSON(protoResponse)
	if err != nil {
		return nil, err
	}

	protoResponse.Reset()
	return bz, nil
}

func ConvertJSONMarshalToProto(cdc codec.Codec, protoRequest proto.Message, bz []byte) ([]byte, error) {
	// unmarshal binary into stargate response data structure
	err := cdc.UnmarshalJSON(bz, protoRequest)
	if err != nil {
		return nil, err
	}

	bz, err = cdc.Marshal(protoRequest)
	if err != nil {
		return nil, err
	}

	protoRequest.Reset()
	return bz, nil
}
