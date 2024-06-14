package move_hooks

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
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
	if err := im.App.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer); err != nil {
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
		return nil
	} else if !allowed {
		h.moveKeeper.Logger(cacheCtx).Error("failed to check ACL", "not allowed")
		return nil
	}
	callbackIdBz, err := vmtypes.SerializeUint64(callback.Id)
	if err != nil {
		return nil
	}
	successBz, err := vmtypes.SerializeBool(!isAckError(h.codec, acknowledgement))
	if err != nil {
		return nil
	}
	_, err = h.execMsg(cacheCtx, &movetypes.MsgExecute{
		Sender:        data.Sender,
		ModuleAddress: callback.ModuleAddress,
		ModuleName:    callback.ModuleName,
		FunctionName:  functionNameAck,
		TypeArgs:      []string{},
		Args:          [][]byte{callbackIdBz, successBz},
	})
	if err != nil {
		h.moveKeeper.Logger(cacheCtx).Error("failed to execute callback", "error", err)
		return nil
	}

	// write the cache context only if the callback execution was successful
	write()

	return nil
}

func (h MoveHooks) onAckIcs721Packet(
	ctx sdk.Context,
	im ibchooks.IBCMiddleware,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
	data nfttransfertypes.NonFungibleTokenPacketData,
) error {
	if err := im.App.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer); err != nil {
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
		return nil
	} else if !allowed {
		h.moveKeeper.Logger(cacheCtx).Error("failed to check ACL", "not allowed")
		return nil
	}
	callbackIdBz, err := vmtypes.SerializeUint64(callback.Id)
	if err != nil {
		return nil
	}
	successBz, err := vmtypes.SerializeBool(!isAckError(h.codec, acknowledgement))
	if err != nil {
		return nil
	}
	_, err = h.execMsg(cacheCtx, &movetypes.MsgExecute{
		Sender:        data.Sender,
		ModuleAddress: callback.ModuleAddress,
		ModuleName:    callback.ModuleName,
		FunctionName:  functionNameAck,
		TypeArgs:      []string{},
		Args:          [][]byte{callbackIdBz, successBz},
	})
	if err != nil {
		h.moveKeeper.Logger(cacheCtx).Error("failed to execute callback", "error", err)
		return nil
	}

	// write the cache context only if the callback execution was successful
	write()

	return nil
}
