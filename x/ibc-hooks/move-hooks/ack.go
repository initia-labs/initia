package move_hooks

import (
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
	"github.com/initia-labs/initia/x/ibc-hooks/types"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func (h MoveHooks) onAckIcs20Packet(
	ctx sdk.Context,
	im ibchooks.IBCMiddleware,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
	data transfertypes.FungibleTokenPacketData,
) error {
	return h.handleOnAck(ctx, im, packet, acknowledgement, relayer, data.Sender)
}

func (h MoveHooks) onAckIcs721Packet(
	ctx sdk.Context,
	im ibchooks.IBCMiddleware,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
	data nfttransfertypes.NonFungibleTokenPacketData,
) error {
	return h.handleOnAck(ctx, im, packet, acknowledgement, relayer, data.Sender)
}

func (h MoveHooks) handleOnAck(
	ctx sdk.Context,
	im ibchooks.IBCMiddleware,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
	sender string,
) error {
	if err := im.App.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer); err != nil {
		return err
	}

	// if no async callback, return early
	bz, err := im.HooksKeeper.GetAsyncCallback(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	if err != nil {
		return nil
	}

	// ignore error on removal; it should not happen
	_ = im.HooksKeeper.RemoveAsyncCallback(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	var asyncCallback AsyncCallback
	if err := asyncCallback.UnmarshalJSON(bz); err != nil {
		h.logger.Error("failed to unmarshal async callback", "error", err)
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeHookFailed,
			sdk.NewAttribute(types.AttributeKeyReason, "failed to unmarshal async callback"),
			sdk.NewAttribute(types.AttributeKeyError, err.Error()),
		))
		return nil
	}

	// create a new cache context to ignore errors during
	// the execution of the callback
	cacheCtx, write := ctx.CacheContext()

	if allowed, err := h.checkACL(im, cacheCtx, asyncCallback.ModuleAddress); err != nil {
		h.logger.Error("failed to check ACL", "error", err)
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeHookFailed,
			sdk.NewAttribute(types.AttributeKeyReason, "failed to check ACL"),
			sdk.NewAttribute(types.AttributeKeyError, err.Error()),
		))

		return nil
	} else if !allowed {
		h.logger.Error("failed to check ACL", "not allowed")
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeHookFailed,
			sdk.NewAttribute(types.AttributeKeyReason, "failed to check ACL"),
			sdk.NewAttribute(types.AttributeKeyError, "not allowed"),
		))

		return nil
	}
	callbackIdBz, err := vmtypes.SerializeUint64(asyncCallback.Id)
	if err != nil {
		return nil
	}
	successBz, err := vmtypes.SerializeBool(!isAckError(h.codec, acknowledgement))
	if err != nil {
		return nil
	}
	_, err = h.execMsg(cacheCtx, &movetypes.MsgExecute{
		Sender:        sender,
		ModuleAddress: asyncCallback.ModuleAddress,
		ModuleName:    asyncCallback.ModuleName,
		FunctionName:  functionNameAck,
		TypeArgs:      []string{},
		Args:          [][]byte{callbackIdBz, successBz},
	})
	if err != nil {
		h.logger.Error("failed to execute callback", "error", err)
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeHookFailed,
			sdk.NewAttribute(types.AttributeKeyReason, "failed to execute callback"),
			sdk.NewAttribute(types.AttributeKeyError, err.Error()),
		))

		return nil
	}

	// write the cache context only if the callback execution was successful
	write()

	return nil
}
