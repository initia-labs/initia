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

	relayers, err := req.RelayerList.GetAccAddr(ms.ac)
	if err != nil {
		println("error here")
		return nil, err
	}

	if err := ms.Keeper.SetPermissionedRelayers(ctx, req.PortId, req.ChannelId, relayers); err != nil {
		return nil, err
	}

	ms.Logger(ctx).Info(
		"IBC permissioned channel relayer",
		"port id", req.PortId,
		"channel id", req.ChannelId,
		"relayers", req.RelayerList.Relayers,
	)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSetPermissionedRelayer,
			sdk.NewAttribute(types.AttributeKeyPortId, req.PortId),
			sdk.NewAttribute(types.AttributeKeyChannelId, req.ChannelId),
			sdk.NewAttribute(types.AttributeKeyRelayer, strings.Join(req.RelayerList.Relayers, ",")),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})

	return &types.MsgSetPermissionedRelayersResponse{}, nil
}

// SetPermissionedRelayer update channel relayer to restrict relaying operation of a channel to specific relayer.
func (ms MsgServer) AddPermissionedRelayers(ctx context.Context, req *types.MsgAddPermissionedRelayers) (*types.MsgAddPermissionedRelayersResponse, error) {
	if err := req.Validate(ms.Keeper.ac); err != nil {
		return nil, err
	}

	if ms.authority != req.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, req.Authority)
	}

	relayers, err := req.RelayerList.GetAccAddr(ms.ac)
	if err != nil {
		return nil, err
	}

	if err := ms.Keeper.AddPermissionedRelayers(ctx, req.PortId, req.ChannelId, relayers); err != nil {
		return nil, err
	}

	ms.Logger(ctx).Info(
		"IBC permissioned channel relayer",
		"port id", req.PortId,
		"channel id", req.ChannelId,
		"relayers", req.RelayerList.Relayers,
	)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeAddPermissionedRelayer,
			sdk.NewAttribute(types.AttributeKeyPortId, req.PortId),
			sdk.NewAttribute(types.AttributeKeyChannelId, req.ChannelId),
			sdk.NewAttribute(types.AttributeKeyRelayer, strings.Join(req.RelayerList.Relayers, ",")),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})

	return &types.MsgAddPermissionedRelayersResponse{}, nil
}
