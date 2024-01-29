package keeper

import (
	"context"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"

	"github.com/initia-labs/initia/x/ibc/fetchprice/types"
)

// to bypass, authority check
var IsTesting = false

type MsgServer struct {
	*Keeper
}

var _ types.MsgServer = MsgServer{}

// NewMsgServerImpl return MsgServer instance
func NewMsgServerImpl(k *Keeper) MsgServer {
	return MsgServer{k}
}

// Activate implements types.MsgServer.
func (ms MsgServer) Activate(ctx context.Context, msg *types.MsgActivate) (*types.MsgActivateResponse, error) {
	if !IsTesting {
		if ms.authority != msg.Authority {
			return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, msg.Authority)
		}
	}

	params, err := ms.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	if params.FetchActivated {
		return nil, types.ErrFetchAlreadyActivated
	}
	if msg.TimeoutDuration == 0 {
		return nil, types.ErrInvalidPacketTimeout.Wrap("timeout duration cannot be zero")
	}

	params.FetchActivated = true
	params.TimeoutDuration = msg.TimeoutDuration
	err = ms.Params.Set(ctx, params)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	_, err = ms.sendICQ(
		sdkCtx,
		msg.SourcePort,
		msg.SourceChannel,
		clienttypes.ZeroHeight(),
		uint64(sdkCtx.BlockTime().Add(params.TimeoutDuration).UnixNano()),
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgActivateResponse{}, nil
}

// Deactivate implements types.MsgServer.
func (ms MsgServer) Deactivate(ctx context.Context, msg *types.MsgDeactivate) (*types.MsgDeactivateResponse, error) {
	if !IsTesting {
		if ms.authority != msg.Authority {
			return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, msg.Authority)
		}
	}

	params, err := ms.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	params.FetchActivated = false
	err = ms.Params.Set(ctx, params)
	if err != nil {
		return nil, err
	}

	return &types.MsgDeactivateResponse{}, nil
}

// UpdateParams implements types.MsgServer.
func (ms MsgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	if err := ms.Params.Set(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
