package v2

import (
	"cosmossdk.io/core/address"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"

	ibchookskeeper "github.com/initia-labs/initia/x/ibc-hooks/keeper"
	ibchooksv2 "github.com/initia-labs/initia/x/ibc-hooks/v2"
	movehooks "github.com/initia-labs/initia/x/ibc-hooks/move-hooks"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"
)

// v2

var (
	_ ibchooksv2.OnRecvPacketOverrideHooks            = MoveHooks{}
	_ ibchooksv2.OnAcknowledgementPacketOverrideHooks = MoveHooks{}
	_ ibchooksv2.OnTimeoutPacketOverrideHooks         = MoveHooks{}
)

type MoveHooks struct {
	codec       codec.Codec
	ac          address.Codec
	moveKeeper  *movekeeper.Keeper
	hooksKeeper *ibchookskeeper.Keeper
}

func NewMoveHooks(
	codec codec.Codec,
	ac address.Codec,
	moveKeeper *movekeeper.Keeper,
	hooksKeeper *ibchookskeeper.Keeper,
) *MoveHooks {
	return &MoveHooks{
		codec:       codec,
		ac:          ac,
		moveKeeper:  moveKeeper,
		hooksKeeper: hooksKeeper,
	}
}

// OnRecvPacketOverride implements OnRecvPacketOverrideHooksV2.
func (h MoveHooks) OnRecvPacketOverride(
	im ibchooksv2.IBCMiddleware,
	ctx sdk.Context,
	sourceChannel string,
	destinationChannel string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) channeltypesv2.RecvPacketResult {
	// Parse as ICS20 v2 payload
	if isIcs20, internalData := movehooks.IsIcs20Packet(payload.Value, payload.Version, payload.Encoding); isIcs20 {
		return h.onRecvIcs20Packet(ctx, im, sourceChannel, destinationChannel, sequence, payload, relayer, internalData)
	}

	if isIcs721, nftData := movehooks.IsIcs721Packet(payload.Value, payload.Version, payload.Encoding); isIcs721 {
		return h.onRecvIcs721Packet(ctx, im, sourceChannel, destinationChannel, sequence, payload, relayer, nftData)
	}

	return im.App.OnRecvPacket(ctx, sourceChannel, destinationChannel, sequence, payload, relayer)
}

// OnAcknowledgementPacketOverride implements OnAcknowledgementPacketOverrideHooksV2.
func (h MoveHooks) OnAcknowledgementPacketOverride(
	im ibchooksv2.IBCMiddleware,
	ctx sdk.Context,
	sourceChannel string,
	destinationChannel string,
	sequence uint64,
	acknowledgement []byte,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	// Parse as ICS20 v2 payload
	if isIcs20, internalData := movehooks.IsIcs20Packet(payload.Value, payload.Version, payload.Encoding); isIcs20 {
		return h.onAckIcs20Packet(ctx, im, sourceChannel, destinationChannel, sequence, acknowledgement, payload, relayer, internalData)
	}

	if isIcs721, nftData := movehooks.IsIcs721Packet(payload.Value, payload.Version, payload.Encoding); isIcs721 {
		return h.onAckIcs721Packet(ctx, im, sourceChannel, destinationChannel, sequence, acknowledgement, payload, relayer, nftData)
	}

	return im.App.OnAcknowledgementPacket(ctx, sourceChannel, destinationChannel, sequence, acknowledgement, payload, relayer)
}

// OnTimeoutPacketOverride implements OnTimeoutPacketOverrideHooksV2.
func (h MoveHooks) OnTimeoutPacketOverride(
	im ibchooksv2.IBCMiddleware,
	ctx sdk.Context,
	sourceChannel string,
	destinationChannel string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	// Parse as ICS20 v2 payload
	if isIcs20, internalData := movehooks.IsIcs20Packet(payload.Value, payload.Version, payload.Encoding); isIcs20 {
		return h.onTimeoutIcs20Packet(ctx, im, sourceChannel, destinationChannel, sequence, payload, relayer, internalData)
	}

	if isIcs721, nftData := movehooks.IsIcs721Packet(payload.Value, payload.Version, payload.Encoding); isIcs721 {
		return h.onTimeoutIcs721Packet(ctx, im, sourceChannel, destinationChannel, sequence, payload, relayer, nftData)
	}

	return im.App.OnTimeoutPacket(ctx, sourceChannel, destinationChannel, sequence, payload, relayer)
}

