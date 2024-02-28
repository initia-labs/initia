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
		return err
	}

	callback := hookData.AsyncCallback
	if allowed, err := h.checkACL(im, ctx, callback.ModuleAddress); err != nil {
		return err
	} else if !allowed {
		return nil
	}

	callbackIdBz, err := vmtypes.SerializeUint64(callback.Id)
	if err != nil {
		return err
	}
	successBz, err := vmtypes.SerializeBool(!isAckError(acknowledgement))
	if err != nil {
		return err
	}

	_, err = h.execMsg(ctx, &movetypes.MsgExecute{
		Sender:        data.Sender,
		ModuleAddress: callback.ModuleAddress,
		ModuleName:    callback.ModuleName,
		FunctionName:  functionNameAck,
		TypeArgs:      []string{},
		Args:          [][]byte{callbackIdBz, successBz},
	})
	if err != nil {
		return err
	}

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
		return err
	}

	callback := hookData.AsyncCallback
	if allowed, err := h.checkACL(im, ctx, callback.ModuleAddress); err != nil {
		return err
	} else if !allowed {
		return nil
	}

	callbackIdBz, err := vmtypes.SerializeUint64(callback.Id)
	if err != nil {
		return err
	}
	successBz, err := vmtypes.SerializeBool(!isAckError(acknowledgement))
	if err != nil {
		return err
	}

	_, err = h.execMsg(ctx, &movetypes.MsgExecute{
		Sender:        data.Sender,
		ModuleAddress: callback.ModuleAddress,
		ModuleName:    callback.ModuleName,
		FunctionName:  functionNameAck,
		TypeArgs:      []string{},
		Args:          [][]byte{callbackIdBz, successBz},
	})
	if err != nil {
		return err
	}

	return nil
}
