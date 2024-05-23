package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"
)

type Querier struct {
	*Keeper
}

var _ types.QueryServer = &Querier{}

// NewQuerier return new Querier instance
func NewQuerier(k *Keeper) Querier {
	return Querier{k}
}

func (q Querier) Module(ctx context.Context, req *types.QueryModuleRequest) (*types.QueryModuleResponse, error) {
	addr, err := types.AccAddressFromString(q.ac, req.Address)
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

func (q Querier) Modules(ctx context.Context, req *types.QueryModulesRequest) (*types.QueryModulesResponse, error) {
	moduleAddr, err := types.AccAddressFromString(q.ac, req.Address)
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

func (q Querier) Resource(ctx context.Context, req *types.QueryResourceRequest) (*types.QueryResourceResponse, error) {
	addr, err := types.AccAddressFromString(q.ac, req.Address)
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

func (q Querier) Resources(ctx context.Context, req *types.QueryResourcesRequest) (*types.QueryResourcesResponse, error) {
	addr, err := types.AccAddressFromString(q.ac, req.Address)
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

func (q Querier) TableInfo(ctx context.Context, req *types.QueryTableInfoRequest) (*types.QueryTableInfoResponse, error) {
	address, err := types.AccAddressFromString(q.ac, req.Address)
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

func (q Querier) TableEntry(ctx context.Context, req *types.QueryTableEntryRequest) (*types.QueryTableEntryResponse, error) {
	addr, err := types.AccAddressFromString(q.ac, req.Address)
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

func (q Querier) TableEntries(ctx context.Context, req *types.QueryTableEntriesRequest) (*types.QueryTableEntriesResponse, error) {
	vmStore := types.NewVMStore(ctx, q.VMStore)

	addr, err := types.AccAddressFromString(q.ac, req.Address)
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

func (q Querier) LegacyView(ctx context.Context, req *types.QueryLegacyViewRequest) (*types.QueryLegacyViewResponse, error) {
	res, err := q.View(ctx, &types.QueryViewRequest{
		Address:      req.Address,
		ModuleName:   req.ModuleName,
		FunctionName: req.FunctionName,
		TypeArgs:     req.TypeArgs,
		Args:         req.Args,
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryLegacyViewResponse{
		Data:    res.Data,
		Events:  res.Events,
		GasUsed: res.GasUsed,
	}, nil
}

func (q Querier) View(ctx context.Context, req *types.QueryViewRequest) (res *types.QueryViewResponse, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(types.ErrInvalidRequest, fmt.Sprintf("vm panic: %v", r))
		}
	}()

	moduleAddr, err := types.AccAddressFromString(q.ac, req.Address)
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

	output, err := q.ExecuteViewFunction(
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

	events := make([]types.VMEvent, len(output.Events))
	for i, event := range output.Events {
		events[i].Data = event.EventData
		events[i].TypeTag, err = vmapi.StringifyTypeTag(event.TypeTag)
		if err != nil {
			return
		}
	}

	res = &types.QueryViewResponse{
		Data:    output.Ret,
		Events:  events,
		GasUsed: output.GasUsed,
	}

	return
}

func (q Querier) ViewBatch(ctx context.Context, req *types.QueryViewBatchRequest) (res *types.QueryViewBatchResponse, err error) {
	responses := make([]types.QueryViewResponse, len(req.Requests))
	for i, req := range req.Requests {
		res, err := q.View(ctx, &req)
		if err != nil {
			return nil, err
		}

		responses[i] = *res
	}

	return &types.QueryViewBatchResponse{
		Responses: responses,
	}, nil
}

func (q Querier) ViewJSON(ctx context.Context, req *types.QueryViewJSONRequest) (res *types.QueryViewJSONResponse, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(types.ErrInvalidRequest, fmt.Sprintf("vm panic: %v", r))
		}
	}()

	moduleAddr, err := types.AccAddressFromString(q.ac, req.Address)
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

	output, err := q.ExecuteViewFunctionJSON(
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

	events := make([]types.VMEvent, len(output.Events))
	for i, event := range output.Events {
		events[i].Data = event.EventData
		events[i].TypeTag, err = vmapi.StringifyTypeTag(event.TypeTag)
		if err != nil {
			return
		}
	}

	res = &types.QueryViewJSONResponse{
		Data:    output.Ret,
		Events:  events,
		GasUsed: output.GasUsed,
	}

	return
}

func (q Querier) ViewJSONBatch(ctx context.Context, req *types.QueryViewJSONBatchRequest) (res *types.QueryViewJSONBatchResponse, err error) {
	responses := make([]types.QueryViewJSONResponse, len(req.Requests))
	for i, req := range req.Requests {
		res, err := q.ViewJSON(ctx, &req)
		if err != nil {
			return nil, err
		}

		responses[i] = *res
	}

	return &types.QueryViewJSONBatchResponse{
		Responses: responses,
	}, nil
}

func (q Querier) ScriptABI(ctx context.Context, req *types.QueryScriptABIRequest) (*types.QueryScriptABIResponse, error) {
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

func (q Querier) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := q.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

// Denom implements types.QueryServer.
func (q Querier) Denom(ctx context.Context, req *types.QueryDenomRequest) (*types.QueryDenomResponse, error) {
	metadata, err := types.AccAddressFromString(q.ac, req.Metadata)
	if err != nil {
		return nil, err
	}

	denom, err := types.DenomFromMetadataAddress(ctx, NewMoveBankKeeper(q.Keeper), metadata)
	if err != nil {
		return nil, err
	}

	return &types.QueryDenomResponse{
		Denom: denom,
	}, nil
}

// Metadata implements types.QueryServer.
func (q Querier) Metadata(ctx context.Context, req *types.QueryMetadataRequest) (*types.QueryMetadataResponse, error) {
	metadataAddr, err := types.MetadataAddressFromDenom(req.Denom)
	if err != nil {
		return nil, err
	}

	return &types.QueryMetadataResponse{
		Metadata: metadataAddr.String(),
	}, nil
}
