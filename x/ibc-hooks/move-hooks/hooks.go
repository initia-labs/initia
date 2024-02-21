package move_hooks

import (
	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"
)

var (
	_ ibchooks.OnRecvPacketOverrideHooks            = MoveHooks{}
	_ ibchooks.OnAcknowledgementPacketOverrideHooks = MoveHooks{}
	_ ibchooks.OnTimeoutPacketOverrideHooks         = MoveHooks{}
)

type MoveHooks struct {
	moveKeeper *movekeeper.Keeper
	ac         address.Codec
}

func NewMoveHooks(moveKeeper *movekeeper.Keeper, ac address.Codec) *MoveHooks {
	return &MoveHooks{
		moveKeeper: moveKeeper,
		ac:         ac,
	}
}

func (h MoveHooks) OnRecvPacketOverride(im ibchooks.IBCMiddleware, ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	if isIcs20, ics20Data := isIcs20Packet(packet.GetData()); isIcs20 {
		return h.onRecvIcs20Packet(ctx, im, packet, relayer, ics20Data)
	}

	if isIcs721, ics721Data := isIcs721Packet(packet.GetData()); isIcs721 {
		return h.onRecvIcs721Packet(ctx, im, packet, relayer, ics721Data)
	}

	return im.App.OnRecvPacket(ctx, packet, relayer)
}

func (h MoveHooks) OnAcknowledgementPacketOverride(im ibchooks.IBCMiddleware, ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	if isIcs20, ics20Data := isIcs20Packet(packet.GetData()); isIcs20 {
		return h.onAckIcs20Packet(ctx, im, packet, acknowledgement, relayer, ics20Data)
	}

	if isIcs721, ics721Data := isIcs721Packet(packet.GetData()); isIcs721 {
		return h.onAckIcs721Packet(ctx, im, packet, acknowledgement, relayer, ics721Data)
	}

	return im.App.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

func (h MoveHooks) OnTimeoutPacketOverride(im ibchooks.IBCMiddleware, ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	if isIcs20, ics20Data := isIcs20Packet(packet.GetData()); isIcs20 {
		return h.onTimeoutIcs20Packet(ctx, im, packet, relayer, ics20Data)
	}

	if isIcs721, ics721Data := isIcs721Packet(packet.GetData()); isIcs721 {
		return h.onTimeoutIcs721Packet(ctx, im, packet, relayer, ics721Data)
	}

	return im.App.OnTimeoutPacket(ctx, packet, relayer)
}

func (h MoveHooks) checkACL(im ibchooks.IBCMiddleware, ctx sdk.Context, addrStr string) (bool, error) {
	addr, err := h.ac.StringToBytes(addrStr)
	if err != nil {
		return false, err
	}

	return im.HooksKeeper.GetAllowed(ctx, addr)
}
