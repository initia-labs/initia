package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/initiavm/api"
	vmtypes "github.com/initia-labs/initiavm/types"
)

type Querier struct {
	Keeper
}

var _ types.QueryServer = &Querier{}

// NewQuerier return new Querier instance
func NewQuerier(k Keeper) Querier {
	return Querier{k}
}

func (q Querier) Module(context context.Context, req *types.QueryModuleRequest) (*types.QueryModuleResponse, error) {
	ctx := sdk.UnwrapSDKContext(context)

	address, err := types.AccAddressFromString(req.Address)
	if err != nil {
		return nil, err
	}

	module, err := q.GetModule(
		ctx,
		address,
		req.ModuleName,
	)

	return &types.QueryModuleResponse{
		Module: module,
	}, err
}

func (q Querier) Modules(context context.Context, req *types.QueryModulesRequest) (*types.QueryModulesResponse, error) {
	ctx := sdk.UnwrapSDKContext(context)

	address, err := types.AccAddressFromString(req.Address)
	if err != nil {
		return nil, err
	}

	modules := make([]types.Module, 0)
	prefixStore := prefix.NewStore(prefix.NewStore(ctx.KVStore(q.storeKey), types.PrefixKeyVMStore), types.GetModulePrefix(address))
	pageRes, err := query.FilteredPaginate(prefixStore, req.Pagination, func(key []byte, rawBytes []byte, accumulate bool) (bool, error) {
		bz, err := q.DecodeModuleBytes(rawBytes)
		if err != nil {
			return false, err
		}

		moduleName, err := vmtypes.BcsDeserializeIdentifier(key)
		if err != nil {
			return false, err
		}

		if accumulate {
			policy, err := NewCodeKeeper(&q.Keeper).GetUpgradePolicy(ctx, address, string(moduleName))
			if err != nil {
				return false, err
			}

			modules = append(modules, types.Module{
				Address:       address.String(),
				ModuleName:    string(moduleName),
				Abi:           string(bz),
				RawBytes:      rawBytes,
				UpgradePolicy: policy,
			})
		}

		return true, nil
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

	address, err := types.AccAddressFromString(req.Address)
	if err != nil {
		return nil, err
	}

	structTag, err := vmapi.ParseStructTag(req.StructTag)
	if err != nil {
		return nil, err
	}

	resource, err := q.GetResource(
		ctx,
		address,
		structTag,
	)

	return &types.QueryResourceResponse{
		Resource: resource,
	}, err
}

func (q Querier) Resources(context context.Context, req *types.QueryResourcesRequest) (*types.QueryResourcesResponse, error) {
	ctx := sdk.UnwrapSDKContext(context)

	address, err := types.AccAddressFromString(req.Address)
	if err != nil {
		return nil, err
	}

	resources := make([]types.Resource, 0)
	prefixStore := prefix.NewStore(prefix.NewStore(ctx.KVStore(q.storeKey), types.PrefixKeyVMStore), types.GetResourcePrefix(address))
	pageRes, err := query.FilteredPaginate(prefixStore, req.Pagination, func(key []byte, rawBytes []byte, accumulate bool) (bool, error) {
		structTag, err := vmtypes.BcsDeserializeStructTag(key)
		if err != nil {
			return false, err
		}

		structTagStr, err := vmapi.StringifyStructTag(structTag)
		if err != nil {
			return false, err
		}

		bz, err := q.DecodeMoveResource(ctx, structTag, rawBytes)
		if err != nil {
			bz = []byte(`""`)
		}

		if accumulate {
			resources = append(resources, types.Resource{
				Address:      address.String(),
				StructTag:    structTagStr,
				MoveResource: string(bz),
				RawBytes:     rawBytes,
			})
		}

		return true, nil
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

	address, err := types.AccAddressFromString(req.Address)
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

	address, err := types.AccAddressFromString(req.Address)
	if err != nil {
		return nil, err
	}

	if len(req.KeyBytes) == 0 {
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "empty key bytes")
	}

	tableEntry, err := q.GetTableEntry(
		ctx,
		address,
		req.KeyBytes,
	)

	return &types.QueryTableEntryResponse{
		TableEntry: tableEntry,
	}, err
}

func (q Querier) TableEntries(context context.Context, req *types.QueryTableEntriesRequest) (*types.QueryTableEntriesResponse, error) {
	ctx := sdk.UnwrapSDKContext(context)
	store := prefix.NewStore(ctx.KVStore(q.storeKey), types.PrefixKeyVMStore)

	address, err := types.AccAddressFromString(req.Address)
	if err != nil {
		return nil, err
	}

	info, err := q.GetTableInfo(ctx, address)
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

	entries := make([]types.TableEntry, 0)
	prefixStore := prefix.NewStore(store, types.GetTableEntryPrefix(address))
	pageRes, err := query.FilteredPaginate(prefixStore, req.Pagination, func(keyBz []byte, valueBz []byte, accumulate bool) (bool, error) {
		if accumulate {
			keyStr, err := vmapi.DecodeMoveValue(store, keyTypeTag, keyBz)
			if err != nil {
				return false, err
			}

			valueStr, err := vmapi.DecodeMoveValue(store, valueTypeTag, valueBz)
			if err != nil {
				return false, err
			}

			entries = append(entries, types.TableEntry{
				Address:    address.String(),
				Key:        string(keyStr),
				Value:      string(valueStr),
				KeyBytes:   keyBz,
				ValueBytes: valueBz,
			})
		}

		return true, nil
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

	moduleAddr, err := types.AccAddressFromString(req.Address)
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
	params := q.GetParams(ctx)

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}
