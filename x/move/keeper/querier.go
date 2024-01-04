package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/initiavm/api"
	vmtypes "github.com/initia-labs/initiavm/types"
)

type Querier struct {
	*Keeper
}

var _ types.QueryServer = &Querier{}

// NewQuerier return new Querier instance
func NewQuerier(k *Keeper) Querier {
	return Querier{k}
}

func (q Querier) Module(context context.Context, req *types.QueryModuleRequest) (*types.QueryModuleResponse, error) {
	ctx := sdk.UnwrapSDKContext(context)

	addr, err := types.AccAddressFromString(q.authKeeper.AddressCodec(), req.Address)
	if err != nil {
		return nil, err
	}

	module, err := q.GetModule(
		ctx,
		addr,
		req.ModuleName,
	)

	return &types.QueryModuleResponse{
		Module: module,
	}, err
}

func (q Querier) Modules(context context.Context, req *types.QueryModulesRequest) (*types.QueryModulesResponse, error) {
	ctx := sdk.UnwrapSDKContext(context)

	moduleAddr, err := types.AccAddressFromString(q.authKeeper.AddressCodec(), req.Address)
	if err != nil {
		return nil, err
	}

	prefixBytes := types.GetModulePrefix(moduleAddr)
	modules, pageRes, err := query.CollectionPaginate(ctx, q.Keeper.VMStore, req.Pagination, func(key []byte, rawBytes []byte) (types.Module, error) {
		bz, err := q.DecodeModuleBytes(rawBytes)
		if err != nil {
			return types.Module{}, err
		}

		key = key[len(prefixBytes):]
		moduleName, err := vmtypes.BcsDeserializeIdentifier(key)
		if err != nil {
			return types.Module{}, err
		}

		policy, err := NewCodeKeeper(q.Keeper).GetUpgradePolicy(ctx, moduleAddr, string(moduleName))
		if err != nil {
			return types.Module{}, err
		}

		return types.Module{
			Address:       moduleAddr.String(),
			ModuleName:    string(moduleName),
			Abi:           string(bz),
			RawBytes:      rawBytes,
			UpgradePolicy: policy,
		}, nil
	}, func(opt *query.CollectionsPaginateOptions[[]byte]) {
		opt.Prefix = &prefixBytes
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryModulesResponse{
		Modules:    modules,
		Pagination: pageRes,
	}, nil
}

func (q Querier) Resource(context context.Context, req *types.QueryResourceRequest) (*types.QueryResourceResponse, error) {
	ctx := sdk.UnwrapSDKContext(context)

	addr, err := types.AccAddressFromString(q.authKeeper.AddressCodec(), req.Address)
	if err != nil {
		return nil, err
	}

	structTag, err := vmapi.ParseStructTag(req.StructTag)
	if err != nil {
		return nil, err
	}

	resource, err := q.GetResource(
		ctx,
		addr,
		structTag,
	)

	return &types.QueryResourceResponse{
		Resource: resource,
	}, err
}

func (q Querier) Resources(context context.Context, req *types.QueryResourcesRequest) (*types.QueryResourcesResponse, error) {
	ctx := sdk.UnwrapSDKContext(context)

	addr, err := types.AccAddressFromString(q.authKeeper.AddressCodec(), req.Address)
	if err != nil {
		return nil, err
	}

	prefixBytes := types.GetResourcePrefix(addr)
	resources, pageRes, err := query.CollectionPaginate(ctx, q.VMStore, req.Pagination, func(key []byte, rawBytes []byte) (types.Resource, error) {
		key = key[len(prefixBytes):]
		structTag, err := vmtypes.BcsDeserializeStructTag(key)
		if err != nil {
			return types.Resource{}, err
		}

		structTagStr, err := vmapi.StringifyStructTag(structTag)
		if err != nil {
			return types.Resource{}, err
		}

		bz, err := q.DecodeMoveResource(ctx, structTag, rawBytes)
		if err != nil {
			bz = []byte(`""`)
		}

		return types.Resource{
			Address:      addr.String(),
			StructTag:    structTagStr,
			MoveResource: string(bz),
			RawBytes:     rawBytes,
		}, nil
	}, func(opt *query.CollectionsPaginateOptions[[]byte]) {
		opt.Prefix = &prefixBytes
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryResourcesResponse{
		Resources:  resources,
		Pagination: pageRes,
	}, nil
}

func (q Querier) TableInfo(context context.Context, req *types.QueryTableInfoRequest) (*types.QueryTableInfoResponse, error) {
	ctx := sdk.UnwrapSDKContext(context)

	address, err := types.AccAddressFromString(q.authKeeper.AddressCodec(), req.Address)
	if err != nil {
		return nil, err
	}

	tableInfo, err := q.GetTableInfo(
		ctx,
		address,
	)

	return &types.QueryTableInfoResponse{
		TableInfo: tableInfo,
	}, err
}

func (q Querier) TableEntry(context context.Context, req *types.QueryTableEntryRequest) (*types.QueryTableEntryResponse, error) {
	ctx := sdk.UnwrapSDKContext(context)

	addr, err := types.AccAddressFromString(q.authKeeper.AddressCodec(), req.Address)
	if err != nil {
		return nil, err
	}

	if len(req.KeyBytes) == 0 {
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "empty key bytes")
	}

	tableEntry, err := q.GetTableEntry(
		ctx,
		addr,
		req.KeyBytes,
	)

	return &types.QueryTableEntryResponse{
		TableEntry: tableEntry,
	}, err
}

func (q Querier) TableEntries(context context.Context, req *types.QueryTableEntriesRequest) (*types.QueryTableEntriesResponse, error) {
	ctx := sdk.UnwrapSDKContext(context)
	vmStore := types.NewVMStore(ctx, q.VMStore)

	addr, err := types.AccAddressFromString(q.authKeeper.AddressCodec(), req.Address)
	if err != nil {
		return nil, err
	}

	info, err := q.GetTableInfo(ctx, addr)
	if err != nil {
		return nil, err
	}

	keyTypeTag, err := vmapi.TypeTagFromString(info.KeyType)
	if err != nil {
		return nil, err
	}

	valueTypeTag, err := vmapi.TypeTagFromString(info.ValueType)
	if err != nil {
		return nil, err
	}

	prefixBytes := types.GetTableEntryPrefix(addr)
	entries, pageRes, err := query.CollectionPaginate(ctx, q.VMStore, req.Pagination, func(key []byte, value []byte) (types.TableEntry, error) {
		key = key[len(prefixBytes):]
		keyStr, err := vmapi.DecodeMoveValue(vmStore, keyTypeTag, key)
		if err != nil {
			return types.TableEntry{}, err
		}

		valueStr, err := vmapi.DecodeMoveValue(vmStore, valueTypeTag, value)
		if err != nil {
			return types.TableEntry{}, err
		}

		return types.TableEntry{
			Address:    addr.String(),
			Key:        string(keyStr),
			Value:      string(valueStr),
			KeyBytes:   key,
			ValueBytes: value,
		}, nil
	}, func(opt *query.CollectionsPaginateOptions[[]byte]) {
		opt.Prefix = &prefixBytes
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryTableEntriesResponse{
		TableEntries: entries,
		Pagination:   pageRes,
	}, nil
}

func (q Querier) ViewFunction(context context.Context, req *types.QueryViewFunctionRequest) (res *types.QueryViewFunctionResponse, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(types.ErrInvalidRequest, fmt.Sprintf("vm panic: %v", r))
		}
	}()

	ctx := sdk.UnwrapSDKContext(context)

	moduleAddr, err := types.AccAddressFromString(q.authKeeper.AddressCodec(), req.Address)
	if err != nil {
		return
	}

	typeTags, err := types.TypeTagsFromTypeArgs(req.TypeArgs)
	if err != nil {
		return
	}

	if len(req.ModuleName) == 0 {
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "empty module name")
	}

	if len(req.FunctionName) == 0 {
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "empty function name")
	}

	data, err := q.ExecuteViewFunction(
		ctx,
		moduleAddr,
		req.ModuleName,
		req.FunctionName,
		typeTags,
		req.Args,
	)
	if err != nil {
		return
	}

	res = &types.QueryViewFunctionResponse{
		Data: data,
	}
	return
}

func (q Querier) ScriptABI(context context.Context, req *types.QueryScriptABIRequest) (*types.QueryScriptABIResponse, error) {
	if len(req.CodeBytes) == 0 {
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "empty code bytes")
	}

	res, err := q.DecodeScriptBytes(
		req.CodeBytes,
	)

	if err != nil {
		return nil, err
	}

	return &types.QueryScriptABIResponse{
		Abi: res,
	}, nil
}

func (q Querier) Params(context context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(context)
	params, err := q.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}
