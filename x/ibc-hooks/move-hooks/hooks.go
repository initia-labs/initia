package move_hooks

import (
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"cosmossdk.io/core/address"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"
	movetypes "github.com/initia-labs/initia/x/move/types"
)

var (
	_ ibchooks.OnRecvPacketOverrideHooks            = MoveHooks{}
	_ ibchooks.OnAcknowledgementPacketOverrideHooks = MoveHooks{}
	_ ibchooks.OnTimeoutPacketOverrideHooks         = MoveHooks{}
	_ ibchooks.SendPacketOverrideHooks              = MoveHooks{}
)

type MoveHooks struct {
	ac     address.Codec
	codec  codec.Codec
	logger log.Logger

	moveKeeper *movekeeper.Keeper
}

func NewMoveHooks(
	ac address.Codec,
	codec codec.Codec,
	logger log.Logger,
	moveKeeper *movekeeper.Keeper,
) *MoveHooks {
	return &MoveHooks{
		codec:      codec,
		ac:         ac,
		logger:     logger.With("module", "ibc-hooks/move-hooks"),
		moveKeeper: moveKeeper,
	}
}

func (h MoveHooks) SendPacketOverride(im ibchooks.ICS4Middleware, ctx sdk.Context, chanCap *capabilitytypes.Capability, sourcePort string, sourceChannel string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (uint64, error) {
	if isIcs20, ics20Data := isIcs20Packet(data); isIcs20 {
		return h.sendIcs20Packet(ctx, im, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, ics20Data)
	}

	if isIcs721, ics721Data := isIcs721Packet(data); isIcs721 {
		return h.sendIcs721Packet(ctx, im, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, ics721Data)
	}

	return im.ICS4Wrapper.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

func (h MoveHooks) OnRecvPacketOverride(im ibchooks.IBCMiddleware, ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	if isIcs20, ics20Data := isIcs20Packet(packet.GetData()); isIcs20 {
		return h.onRecvIcs20Packet(ctx, im, packet, relayer, ics20Data)
	}

	if isIcs721, ics721Data := isIcs721Packet(packet.Data); isIcs721 {
		return h.onRecvIcs721Packet(ctx, im, packet, relayer, ics721Data)
	}

	return im.App.OnRecvPacket(ctx, packet, relayer)
}

func (h MoveHooks) OnAcknowledgementPacketOverride(im ibchooks.IBCMiddleware, ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	if isIcs20, ics20Data := isIcs20Packet(packet.GetData()); isIcs20 {
		return h.onAckIcs20Packet(ctx, im, packet, acknowledgement, relayer, ics20Data)
	}

	if isIcs721, ics721Data := isIcs721Packet(packet.Data); isIcs721 {
		return h.onAckIcs721Packet(ctx, im, packet, acknowledgement, relayer, ics721Data)
	}

	return im.App.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

func (h MoveHooks) OnTimeoutPacketOverride(im ibchooks.IBCMiddleware, ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	if isIcs20, ics20Data := isIcs20Packet(packet.GetData()); isIcs20 {
		return h.onTimeoutIcs20Packet(ctx, im, packet, relayer, ics20Data)
	}

	if isIcs721, ics721Data := isIcs721Packet(packet.Data); isIcs721 {
		return h.onTimeoutIcs721Packet(ctx, im, packet, relayer, ics721Data)
	}

	return im.App.OnTimeoutPacket(ctx, packet, relayer)
}

func (h MoveHooks) checkACL(im ibchooks.IBCMiddleware, ctx sdk.Context, addrStr string) (bool, error) {
	vmAddr, err := movetypes.AccAddressFromString(h.ac, addrStr)
	if err != nil {
		return false, err
	}

	sdkAddr := movetypes.ConvertVMAddressToSDKAddress(vmAddr)
	return im.HooksKeeper.GetAllowed(ctx, sdkAddr)
}
