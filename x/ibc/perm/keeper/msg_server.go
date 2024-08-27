package keeper

import (
	"context"
	"strings"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

type MsgServer struct {
	*Keeper
}

var _ types.MsgServer = MsgServer{}

// NewMsgServerImpl return MsgServer instance
func NewMsgServerImpl(k *Keeper) MsgServer {
	return MsgServer{k}
}

// SetPermissionedRelayer update channel relayer to restrict relaying operation of a channel to specific relayer.
func (ms MsgServer) SetPermissionedRelayers(ctx context.Context, req *types.MsgSetPermissionedRelayers) (*types.MsgSetPermissionedRelayersResponse, error) {
	if err := req.Validate(ms.Keeper.ac); err != nil {
		return nil, err
	}

	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	cs, err := ms.Keeper.GetChannelState(ctx, req.PortId, req.ChannelId)
	if err != nil {
		return nil, err
	}

	cs.Relayers = req.Relayers
	err = ms.Keeper.SetChannelState(ctx, cs)
	if err != nil {
		return nil, err
	}

	ms.Logger(ctx).Info(
		"IBC permissioned channel relayer",
		"port id", req.PortId,
		"channel id", req.ChannelId,
		"relayers", req.Relayers,
	)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSetPermissionedRelayers,
			sdk.NewAttribute(types.AttributeKeyPortId, req.PortId),
			sdk.NewAttribute(types.AttributeKeyChannelId, req.ChannelId),
			sdk.NewAttribute(types.AttributeKeyRelayers, strings.Join(req.Relayers, ",")),
		),
	})

	return &types.MsgSetPermissionedRelayersResponse{}, nil
}

// HaltChannel implements types.MsgServer.
func (ms MsgServer) HaltChannel(ctx context.Context, req *types.MsgHaltChannel) (*types.MsgHaltChannelResponse, error) {
	if err := req.Validate(ms.Keeper.ac); err != nil {
		return nil, err
	}

	cs, err := ms.Keeper.GetChannelState(ctx, req.PortId, req.ChannelId)
	if err != nil {
		return nil, err
	}

	if cs.HaltState.Halted {
		return nil, errors.Wrap(types.ErrInvalidHaltState, "channel is already halted")
	}

	if ms.authority != req.Authority && !cs.HasRelayer(req.Authority) {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s or one of permissinoed-relayers, got %s", ms.authority, req.Authority)
	}

	cs.HaltState.Halted = true
	cs.HaltState.HaltedBy = req.Authority
	err = ms.Keeper.SetChannelState(ctx, cs)
	if err != nil {
		return nil, err
	}

	ms.Logger(ctx).Info(
		"IBC permissioned channel halted",
		"port id", req.PortId,
		"channel id", req.ChannelId,
		"halted by", req.Authority,
	)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeHaltChannel,
			sdk.NewAttribute(types.AttributeKeyPortId, req.PortId),
			sdk.NewAttribute(types.AttributeKeyChannelId, req.ChannelId),
			sdk.NewAttribute(types.AttributeKeyHaltedBy, req.Authority),
		),
	})

	return &types.MsgHaltChannelResponse{}, nil
}

// ResumeChannel implements types.MsgServer.
func (ms MsgServer) ResumeChannel(ctx context.Context, req *types.MsgResumeChannel) (*types.MsgResumeChannelResponse, error) {
	if err := req.Validate(ms.Keeper.ac); err != nil {
		return nil, err
	}

	cs, err := ms.Keeper.GetChannelState(ctx, req.PortId, req.ChannelId)
	if err != nil {
		return nil, err
	}

	if !cs.HaltState.Halted {
		return nil, errors.Wrap(types.ErrInvalidHaltState, "channel is not halted")
	}

	if ms.authority != req.Authority && req.Authority != cs.HaltState.HaltedBy {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s or %s, got %s", ms.authority, cs.HaltState.HaltedBy, req.Authority)
	}

	cs.HaltState.Halted = false
	cs.HaltState.HaltedBy = ""
	err = ms.Keeper.SetChannelState(ctx, cs)
	if err != nil {
		return nil, err
	}

	ms.Logger(ctx).Info(
		"IBC permissioned channel resumed",
		"port id", req.PortId,
		"channel id", req.ChannelId,
		"resumed by", req.Authority,
	)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeHaltChannel,
			sdk.NewAttribute(types.AttributeKeyPortId, req.PortId),
			sdk.NewAttribute(types.AttributeKeyChannelId, req.ChannelId),
			sdk.NewAttribute(types.AttributeKeyResumedBy, req.Authority),
		),
	})

	return &types.MsgResumeChannelResponse{}, nil
}
