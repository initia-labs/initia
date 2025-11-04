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

// UpdateAdmin update channel relayer to restrict relaying operation of a channel to specific relayer.
func (ms MsgServer) UpdateAdmin(ctx context.Context, req *types.MsgUpdateAdmin) (*types.MsgUpdateAdminResponse, error) {
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	cs, err := ms.GetChannelState(ctx, req.PortId, req.ChannelId)
	if err != nil {
		return nil, err
	}

	// gov or admin can update relayers
	if ms.authority != req.Authority && cs.Admin != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s or %s, got %s", ms.authority, cs.Admin, req.Authority)
	}

	cs.Admin = req.Admin
	err = ms.SetChannelState(ctx, cs)
	if err != nil {
		return nil, err
	}

	ms.Logger(ctx).Info(
		"IBC channel admin is updated",
		"port id", req.PortId,
		"channel id", req.ChannelId,
		"admin", req.Admin,
	)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpdateAdmin,
			sdk.NewAttribute(types.AttributeKeyPortId, req.PortId),
			sdk.NewAttribute(types.AttributeKeyChannelId, req.ChannelId),
			sdk.NewAttribute(types.AttributeKeyAdmin, req.Admin),
		),
	})

	return &types.MsgUpdateAdminResponse{}, nil
}

// UpdatePermissionedRelayers update channel relayer to restrict relaying operation of a channel to specific relayer.
func (ms MsgServer) UpdatePermissionedRelayers(ctx context.Context, req *types.MsgUpdatePermissionedRelayers) (*types.MsgUpdatePermissionedRelayersResponse, error) {
	if err := req.Validate(ms.ac); err != nil {
		return nil, err
	}

	cs, err := ms.GetChannelState(ctx, req.PortId, req.ChannelId)
	if err != nil {
		return nil, err
	}

	// gov or admin can update relayers
	if ms.authority != req.Authority && cs.Admin != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s or %s, got %s", ms.authority, cs.Admin, req.Authority)
	}

	cs.Relayers = req.Relayers
	err = ms.SetChannelState(ctx, cs)
	if err != nil {
		return nil, err
	}

	ms.Logger(ctx).Info(
		"IBC channel relayers are updated",
		"port id", req.PortId,
		"channel id", req.ChannelId,
		"relayers", req.Relayers,
	)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpdatePermissionedRelayers,
			sdk.NewAttribute(types.AttributeKeyPortId, req.PortId),
			sdk.NewAttribute(types.AttributeKeyChannelId, req.ChannelId),
			sdk.NewAttribute(types.AttributeKeyRelayers, strings.Join(req.Relayers, ",")),
		),
	})

	return &types.MsgUpdatePermissionedRelayersResponse{}, nil
}
