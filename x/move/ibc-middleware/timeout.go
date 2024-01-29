package ibc_middleware

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

func (im IBCMiddleware) onTimeoutIcs20Packet(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	data transfertypes.FungibleTokenPacketData,
) error {
	if err := im.app.OnTimeoutPacket(ctx, packet, relayer); err != nil {
		return err
	}

	isMoveRouted, hookData, err := validateAndParseMemo(data.GetMemo())
	needAsyncCallback := isMoveRouted && hookData.AsyncCallback != nil

	if !needAsyncCallback {
		return nil
	} else if err != nil {
		return err
	}

	callback := hookData.AsyncCallback
	callbackIdBz, err := vmtypes.SerializeUint64(callback.Id)
	if err != nil {
		return err
	}

	_, err = im.execMsg(ctx, &movetypes.MsgExecute{
		Sender:        data.Sender,
		ModuleAddress: callback.ModuleAddress,
		ModuleName:    callback.ModuleName,
		FunctionName:  functionNameTimeout,
		TypeArgs:      []string{},
		Args:          [][]byte{callbackIdBz},
	})
	if err != nil {
		return err
	}

	return nil
}

func (im IBCMiddleware) onTimeoutIcs721Packet(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	data nfttransfertypes.NonFungibleTokenPacketData,
) error {
	if err := im.app.OnTimeoutPacket(ctx, packet, relayer); err != nil {
		return err
	}

	isMoveRouted, hookData, err := validateAndParseMemo(data.GetMemo())
	needAsyncCallback := isMoveRouted && hookData.AsyncCallback != nil

	if !needAsyncCallback {
		return nil
	} else if err != nil {
		return err
	}

	callback := hookData.AsyncCallback
	callbackIdBz, err := vmtypes.SerializeUint64(callback.Id)
	if err != nil {
		return err
	}

	_, err = im.execMsg(ctx, &movetypes.MsgExecute{
		Sender:        data.Sender,
		ModuleAddress: callback.ModuleAddress,
		ModuleName:    callback.ModuleName,
		FunctionName:  functionNameTimeout,
		TypeArgs:      []string{},
		Args:          [][]byte{callbackIdBz},
	})
	if err != nil {
		return err
	}

	return nil
}
