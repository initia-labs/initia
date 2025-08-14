package v2

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"

	ibchooksv2 "github.com/initia-labs/initia/x/ibc-hooks/v2"
	movehooks "github.com/initia-labs/initia/x/ibc-hooks/move-hooks"
	"github.com/initia-labs/initia/x/ibc-hooks/types"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

// onAckIcs20Packet handles ICS20 packet acknowledgement with move hooks
func (h MoveHooks) onAckIcs20Packet(
	ctx sdk.Context,
	im ibchooksv2.IBCMiddleware,
	sourceChannel string,
	destinationChannel string,
	sequence uint64,
	acknowledgement []byte,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
	internalData transfertypes.InternalTransferRepresentation,
) error {
	// First, let the next middleware process the acknowledgementㄴ
	if err := im.App.OnAcknowledgementPacket(ctx, sourceChannel, destinationChannel, sequence, acknowledgement, payload, relayer); err != nil {
		return err
	}

	// Check if the packet has move hook in memo
	isMoveRouted, hookData, err := movehooks.ValidateAndParseMemo(internalData.Memo)
	if !isMoveRouted || hookData.AsyncCallback == nil {
		return nil
	} else if err != nil {
		h.moveKeeper.Logger(ctx).Error("failed to parse memo", "error", err)
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeHookFailed,
			sdk.NewAttribute(types.AttributeKeyReason, "failed to parse memo"),
			sdk.NewAttribute(types.AttributeKeyError, err.Error()),
		))
		return nil
	}

	// Create a new cache context to ignore errors during
	// the execution of the callback
	cacheCtx, write := ctx.CacheContext()

	callback := hookData.AsyncCallback
	if allowed, err := movehooks.CheckACL(ctx, h.ac, h.hooksKeeper, callback.ModuleAddress); err != nil {
		h.moveKeeper.Logger(cacheCtx).Error("failed to check ACL", "error", err)
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeHookFailed,
			sdk.NewAttribute(types.AttributeKeyReason, "failed to check ACL"),
			sdk.NewAttribute(types.AttributeKeyError, err.Error()),
		))
		return nil
	} else if !allowed {
		h.moveKeeper.Logger(cacheCtx).Error("failed to check ACL", "not allowed")
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeHookFailed,
			sdk.NewAttribute(types.AttributeKeyReason, "failed to check ACL"),
			sdk.NewAttribute(types.AttributeKeyError, "not allowed"),
		))
		return nil
	}

	callbackIdBz, err := vmtypes.SerializeUint64(callback.Id)
	if err != nil {
		return nil
	}
	successBz, err := vmtypes.SerializeBool(!movehooks.IsAckError(h.codec, acknowledgement))
	if err != nil {
		return nil
	}
	_, err = movehooks.ExecMsg(cacheCtx, &movetypes.MsgExecute{
		Sender:        internalData.Sender,
		ModuleAddress: callback.ModuleAddress,
		ModuleName:    callback.ModuleName,
		FunctionName:  movehooks.FunctionNameAck,
		TypeArgs:      []string{},
		Args:          [][]byte{callbackIdBz, successBz},
	}, h.moveKeeper, h.ac)
	if err != nil {
		h.moveKeeper.Logger(cacheCtx).Error("failed to execute callback", "error", err)
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

func (h MoveHooks) onAckIcs721Packet(
	ctx sdk.Context,
	im ibchooksv2.IBCMiddleware,
	sourceChannel string,
	destinationChannel string,
	sequence uint64,
	acknowledgement []byte,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
	data nfttransfertypes.NonFungibleTokenPacketData,
) error {
	if err := im.App.OnAcknowledgementPacket(ctx, sourceChannel, destinationChannel, sequence, acknowledgement, payload, relayer); err != nil {
		return err
	}

	isMoveRouted, hookData, err := movehooks.ValidateAndParseMemo(data.Memo)
	if !isMoveRouted || hookData.AsyncCallback == nil {
		return nil
	} else if err != nil {
		h.moveKeeper.Logger(ctx).Error("failed to parse memo", "error", err)
		return nil
	}

	// create a new cache context to ignore errors during
	// the execution of the callback
	cacheCtx, write := ctx.CacheContext()

	callback := hookData.AsyncCallback
	if allowed, err := movehooks.CheckACL(cacheCtx, h.ac, h.hooksKeeper, callback.ModuleAddress); err != nil {
		h.moveKeeper.Logger(cacheCtx).Error("failed to check ACL", "error", err)
		cacheCtx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeHookFailed,
			sdk.NewAttribute(types.AttributeKeyReason, "failed to check ACL"),
			sdk.NewAttribute(types.AttributeKeyError, err.Error()),
		))

		return nil
	} else if !allowed {
		h.moveKeeper.Logger(cacheCtx).Error("failed to check ACL", "not allowed")
		cacheCtx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeHookFailed,
			sdk.NewAttribute(types.AttributeKeyReason, "failed to check ACL"),
			sdk.NewAttribute(types.AttributeKeyError, "not allowed"),
		))

		return nil
	}

	callbackIdBz, err := vmtypes.SerializeUint64(callback.Id)
	if err != nil {
		return nil
	}
	successBz, err := vmtypes.SerializeBool(!movehooks.IsAckError(h.codec, acknowledgement))
	if err != nil {
		return nil
	}

	_, err = movehooks.ExecMsg(cacheCtx, &movetypes.MsgExecute{
		Sender:        data.Sender,
		ModuleAddress: callback.ModuleAddress,
		ModuleName:    callback.ModuleName,
		FunctionName:  movehooks.FunctionNameAck,
		TypeArgs:      []string{},
		Args:          [][]byte{callbackIdBz, successBz},
	}, h.moveKeeper, h.ac)
	if err != nil {
		h.moveKeeper.Logger(cacheCtx).Error("failed to execute callback", "error", err)
		cacheCtx.EventManager().EmitEvent(sdk.NewEvent(
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
