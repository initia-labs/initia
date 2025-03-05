package move_hooks

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	ibchooks "github.com/initia-labs/initia/v1/x/ibc-hooks"
	"github.com/initia-labs/initia/v1/x/ibc-hooks/types"
	nfttransfertypes "github.com/initia-labs/initia/v1/x/ibc/nft-transfer/types"
	movetypes "github.com/initia-labs/initia/v1/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func (h MoveHooks) onTimeoutIcs20Packet(
	ctx sdk.Context,
	im ibchooks.IBCMiddleware,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	data transfertypes.FungibleTokenPacketData,
) error {
	if err := im.App.OnTimeoutPacket(ctx, packet, relayer); err != nil {
		return err
	}

	isMoveRouted, hookData, err := validateAndParseMemo(data.GetMemo())
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

	// create a new cache context to ignore errors during
	// the execution of the callback
	cacheCtx, write := ctx.CacheContext()

	callback := hookData.AsyncCallback
	if allowed, err := h.checkACL(im, cacheCtx, callback.ModuleAddress); err != nil {
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
	_, err = h.execMsg(cacheCtx, &movetypes.MsgExecute{
		Sender:        data.Sender,
		ModuleAddress: callback.ModuleAddress,
		ModuleName:    callback.ModuleName,
		FunctionName:  functionNameTimeout,
		TypeArgs:      []string{},
		Args:          [][]byte{callbackIdBz},
	})
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

func (h MoveHooks) onTimeoutIcs721Packet(
	ctx sdk.Context,
	im ibchooks.IBCMiddleware,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	data nfttransfertypes.NonFungibleTokenPacketData,
) error {
	if err := im.App.OnTimeoutPacket(ctx, packet, relayer); err != nil {
		return err
	}

	isMoveRouted, hookData, err := validateAndParseMemo(data.GetMemo())
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
	if allowed, err := h.checkACL(im, cacheCtx, callback.ModuleAddress); err != nil {
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
	_, err = h.execMsg(cacheCtx, &movetypes.MsgExecute{
		Sender:        data.Sender,
		ModuleAddress: callback.ModuleAddress,
		ModuleName:    callback.ModuleName,
		FunctionName:  functionNameTimeout,
		TypeArgs:      []string{},
		Args:          [][]byte{callbackIdBz},
	})
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
