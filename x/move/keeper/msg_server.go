package keeper

import (
	"context"
	"time"

	"cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/initia-labs/initia/x/move/types"
)

type MsgServer struct {
	*Keeper
}

var _ types.MsgServer = MsgServer{}

// NewMsgServerImpl return MsgServer instance
func NewMsgServerImpl(k *Keeper) MsgServer {
	return MsgServer{k}
}

// Publish implements publishing module to move vm
func (ms MsgServer) Publish(context context.Context, req *types.MsgPublish) (*types.MsgPublishResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "publish")
	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	sender, err := types.AccAddressFromString(ms.ac, req.Sender)
	if err != nil {
		return nil, err
	}

	var modules []vmtypes.Module
	for _, module := range req.CodeBytes {
		modules = append(modules, vmtypes.NewModule(module))
	}

	err = ms.PublishModuleBundle(
		ctx,
		sender,
		vmtypes.NewModuleBundle(modules...),
		req.UpgradePolicy,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgPublishResponse{}, nil
}

// Execute implements entry function execution feature
func (ms MsgServer) Execute(context context.Context, req *types.MsgExecute) (*types.MsgExecuteResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "execute")
	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	ac := ms.ac
	sender, err := types.AccAddressFromString(ac, req.Sender)
	if err != nil {
		return nil, err
	}

	moduleAddr, err := types.AccAddressFromString(ac, req.ModuleAddress)
	if err != nil {
		return nil, err
	}

	typeTags, err := types.TypeTagsFromTypeArgs(req.TypeArgs)
	if err != nil {
		return nil, err
	}

	err = ms.ExecuteEntryFunction(
		ctx,
		sender,
		moduleAddr,
		req.ModuleName,
		req.FunctionName,
		typeTags,
		req.Args,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgExecuteResponse{}, nil
}

// ExecuteJSON implements entry function execution feature
func (ms MsgServer) ExecuteJSON(context context.Context, req *types.MsgExecuteJSON) (*types.MsgExecuteJSONResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "execute_json")
	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	ac := ms.ac
	sender, err := types.AccAddressFromString(ac, req.Sender)
	if err != nil {
		return nil, err
	}

	moduleAddr, err := types.AccAddressFromString(ac, req.ModuleAddress)
	if err != nil {
		return nil, err
	}

	typeTags, err := types.TypeTagsFromTypeArgs(req.TypeArgs)
	if err != nil {
		return nil, err
	}

	err = ms.ExecuteEntryFunctionJSON(
		ctx,
		sender,
		moduleAddr,
		req.ModuleName,
		req.FunctionName,
		typeTags,
		req.Args,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgExecuteJSONResponse{}, nil
}

// Script implements script execution
func (ms MsgServer) Script(ctx context.Context, req *types.MsgScript) (*types.MsgScriptResponse, error) {
	if ok, err := ms.ScriptEnabled(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.Wrap(types.ErrScriptDisabled, "script execution is disabled")
	}

	defer telemetry.MeasureSince(time.Now(), "move", "msg", "script")

	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	ac := ms.ac
	sender, err := types.AccAddressFromString(ac, req.Sender)
	if err != nil {
		return nil, err
	}

	typeTags, err := types.TypeTagsFromTypeArgs(req.TypeArgs)
	if err != nil {
		return nil, err
	}

	err = ms.ExecuteScript(
		ctx,
		sender,
		req.CodeBytes,
		typeTags,
		req.Args,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgScriptResponse{}, nil
}

// ScriptJSON implements script execution
func (ms MsgServer) ScriptJSON(context context.Context, req *types.MsgScriptJSON) (*types.MsgScriptJSONResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "script_json")
	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	ac := ms.ac
	sender, err := types.AccAddressFromString(ac, req.Sender)
	if err != nil {
		return nil, err
	}

	typeTags, err := types.TypeTagsFromTypeArgs(req.TypeArgs)
	if err != nil {
		return nil, err
	}

	err = ms.ExecuteScriptJSON(
		ctx,
		sender,
		req.CodeBytes,
		typeTags,
		req.Args,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgScriptJSONResponse{}, nil
}

// GovPublish implements publishing module to move vm via gov proposal
func (ms MsgServer) GovPublish(context context.Context, req *types.MsgGovPublish) (*types.MsgGovPublishResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "gov-publish")
	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	ac := ms.ac
	sender, err := types.AccAddressFromString(ac, req.Sender)
	if err != nil {
		return nil, err
	}

	var modules []vmtypes.Module
	for _, module := range req.CodeBytes {
		modules = append(modules, vmtypes.NewModule(module))
	}

	err = ms.PublishModuleBundle(
		ctx,
		sender,
		vmtypes.NewModuleBundle(modules...),
		req.UpgradePolicy,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgGovPublishResponse{}, nil
}

// GovExecute implements entry function execution feature via gov proposal
func (ms MsgServer) GovExecute(context context.Context, req *types.MsgGovExecute) (*types.MsgGovExecuteResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "gov-execute")
	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	ac := ms.ac
	sender, err := types.AccAddressFromString(ac, req.Sender)
	if err != nil {
		return nil, err
	}

	moduleAddr, err := types.AccAddressFromString(ac, req.ModuleAddress)
	if err != nil {
		return nil, err
	}

	typeTags, err := types.TypeTagsFromTypeArgs(req.TypeArgs)
	if err != nil {
		return nil, err
	}

	err = ms.ExecuteEntryFunction(
		ctx,
		sender,
		moduleAddr,
		req.ModuleName,
		req.FunctionName,
		typeTags,
		req.Args,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgGovExecuteResponse{}, nil
}

// GovExecuteJSON implements entry function execution feature via gov proposal
func (ms MsgServer) GovExecuteJSON(context context.Context, req *types.MsgGovExecuteJSON) (*types.MsgGovExecuteJSONResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "gov-execute")
	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	ac := ms.ac
	sender, err := types.AccAddressFromString(ac, req.Sender)
	if err != nil {
		return nil, err
	}

	moduleAddr, err := types.AccAddressFromString(ac, req.ModuleAddress)
	if err != nil {
		return nil, err
	}

	typeTags, err := types.TypeTagsFromTypeArgs(req.TypeArgs)
	if err != nil {
		return nil, err
	}

	err = ms.ExecuteEntryFunctionJSON(
		ctx,
		sender,
		moduleAddr,
		req.ModuleName,
		req.FunctionName,
		typeTags,
		req.Args,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgGovExecuteJSONResponse{}, nil
}

// GovScript implements script execution via gov proposal
func (ms MsgServer) GovScript(context context.Context, req *types.MsgGovScript) (*types.MsgGovScriptResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "gov-script")
	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	ac := ms.ac
	sender, err := types.AccAddressFromString(ac, req.Sender)
	if err != nil {
		return nil, err
	}

	typeTags, err := types.TypeTagsFromTypeArgs(req.TypeArgs)
	if err != nil {
		return nil, err
	}

	err = ms.ExecuteScript(
		ctx,
		sender,
		req.CodeBytes,
		typeTags,
		req.Args,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgGovScriptResponse{}, nil
}

// GovScriptJSON implements script execution via gov proposal
func (ms MsgServer) GovScriptJSON(context context.Context, req *types.MsgGovScriptJSON) (*types.MsgGovScriptJSONResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "gov-script")
	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	ac := ms.ac
	sender, err := types.AccAddressFromString(ac, req.Sender)
	if err != nil {
		return nil, err
	}

	typeTags, err := types.TypeTagsFromTypeArgs(req.TypeArgs)
	if err != nil {
		return nil, err
	}

	err = ms.ExecuteScriptJSON(
		ctx,
		sender,
		req.CodeBytes,
		typeTags,
		req.Args,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgGovScriptJSONResponse{}, nil
}

func (ms MsgServer) WhitelistStaking(context context.Context, req *types.MsgWhitelistStaking) (*types.MsgWhitelistStakingResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "whitelist")
	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	err := ms.Keeper.WhitelistStaking(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &types.MsgWhitelistStakingResponse{}, nil
}

func (ms MsgServer) WhitelistGasPrice(context context.Context, req *types.MsgWhitelistGasPrice) (*types.MsgWhitelistGasPriceResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "whitelist")
	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	err := ms.Keeper.WhitelistGasPrice(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &types.MsgWhitelistGasPriceResponse{}, nil
}

func (ms MsgServer) DelistStaking(context context.Context, req *types.MsgDelistStaking) (*types.MsgDelistStakingResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "delist")
	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	err := ms.Keeper.DelistStaking(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &types.MsgDelistStakingResponse{}, nil
}

func (ms MsgServer) DelistGasPrice(context context.Context, req *types.MsgDelistGasPrice) (*types.MsgDelistGasPriceResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "delist")
	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	err := ms.Keeper.DelistGasPrice(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &types.MsgDelistGasPriceResponse{}, nil
}

func (ms MsgServer) UpdateParams(context context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	defer telemetry.MeasureSince(time.Now(), "move", "msg", "update-params")
	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(context)
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	if err := ms.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
