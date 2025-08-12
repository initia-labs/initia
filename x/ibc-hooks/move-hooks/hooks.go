package move_hooks

import (
	"cosmossdk.io/core/address"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"
)

var (
	_ ibchooks.OnRecvPacketOverrideHooks            = MoveHooks{}
	_ ibchooks.OnAcknowledgementPacketOverrideHooks = MoveHooks{}
	_ ibchooks.OnTimeoutPacketOverrideHooks         = MoveHooks{}
)

type MoveHooks struct {
	codec      codec.Codec
	ac         address.Codec
	moveKeeper *movekeeper.Keeper
}

func NewMoveHooks(codec codec.Codec, ac address.Codec, moveKeeper *movekeeper.Keeper) *MoveHooks {
	return &MoveHooks{
		codec:      codec,
		ac:         ac,
		moveKeeper: moveKeeper,
	}
}

func (h MoveHooks) OnRecvPacketOverride(im ibchooks.IBCMiddleware, ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	if isIcs20, ics20Data := IsIcs20Packet(packet.GetData(), channelVersion, ""); isIcs20 {
		return h.onRecvIcs20Packet(ctx, im, channelVersion, packet, relayer, ics20Data)
	}

	if isIcs721, ics721Data := IsIcs721Packet(packet.Data, channelVersion, ""); isIcs721 {
		return h.onRecvIcs721Packet(ctx, im, channelVersion, packet, relayer, ics721Data)
	}

	return im.App.OnRecvPacket(ctx, channelVersion, packet, relayer)
}

func (h MoveHooks) OnAcknowledgementPacketOverride(im ibchooks.IBCMiddleware, ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	if isIcs20, ics20Data := IsIcs20Packet(packet.GetData(), channelVersion, ""); isIcs20 {
		return h.onAckIcs20Packet(ctx, im, channelVersion, packet, acknowledgement, relayer, ics20Data)
	}

	if isIcs721, ics721Data := IsIcs721Packet(packet.Data, channelVersion, ""); isIcs721 {
		return h.onAckIcs721Packet(ctx, im, channelVersion, packet, acknowledgement, relayer, ics721Data)
	}

	return im.App.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

func (h MoveHooks) OnTimeoutPacketOverride(im ibchooks.IBCMiddleware, ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	if isIcs20, ics20Data := IsIcs20Packet(packet.GetData(), channelVersion, ""); isIcs20 {
		return h.onTimeoutIcs20Packet(ctx, im, channelVersion, packet, relayer, ics20Data)
	}

	if isIcs721, ics721Data := IsIcs721Packet(packet.Data, channelVersion, ""); isIcs721 {
		return h.onTimeoutIcs721Packet(ctx, im, channelVersion, packet, relayer, ics721Data)
	}

	return im.App.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
}

