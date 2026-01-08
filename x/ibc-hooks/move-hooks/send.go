package move_hooks

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"

	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
	ibchookstypes "github.com/initia-labs/initia/x/ibc-hooks/types"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

func (h MoveHooks) sendIcs20Packet(
	ctx sdk.Context,
	im ibchooks.ICS4Middleware,
	chanCap *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	ics20Data transfertypes.FungibleTokenPacketData,
) (uint64, error) {
	return h.handleSendPacket(ctx, im, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, ibchookstypes.ICSData{
		ICS20Data: &ics20Data,
	})
}

func (h MoveHooks) sendIcs721Packet(
	ctx sdk.Context,
	im ibchooks.ICS4Middleware,
	chanCap *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	ics721Data nfttransfertypes.NonFungibleTokenPacketData,
) (uint64, error) {
	return h.handleSendPacket(ctx, im, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, ibchookstypes.ICSData{
		ICS721Data: &ics721Data,
	})
}

func (h MoveHooks) handleSendPacket(
	ctx sdk.Context,
	im ibchooks.ICS4Middleware,
	chanCap *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	icsData ibchookstypes.ICSData,
) (uint64, error) {
	hookData, isMoveRouted, err := parseHookData(icsData.GetMemo())
	if err != nil {
		return 0, err
	}
	if !isMoveRouted || hookData == nil || hookData.AsyncCallback == nil {
		return im.ICS4Wrapper.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, icsData.GetBytes())
	}

	asyncCallback := hookData.AsyncCallback

	var memoMap map[string]any
	// ignore error, it is already checked in unmarshalMemo
	_ = json.Unmarshal([]byte(icsData.GetMemo()), &memoMap)
	if hookData.Message == nil && hookData.MessageJSON == nil {
		delete(memoMap, moveHookMemoKey)
	} else {
		hookData.AsyncCallback = nil
		moveMemo := MoveMemo{
			MoveHook: hookData,
		}
		bz, err := json.Marshal(moveMemo)
		if err != nil {
			return 0, err
		}
		memoMap[moveHookMemoKey] = json.RawMessage(bz)
	}
	bz, err := json.Marshal(memoMap)
	if err != nil {
		return 0, err
	}
	icsData.SetMemo(string(bz))

	sequence, err := im.ICS4Wrapper.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, icsData.GetBytes())
	if err != nil {
		return sequence, err
	}

	asyncCallbackBz, err := json.Marshal(asyncCallback)
	if err != nil {
		return sequence, err
	}
	if err := im.HooksKeeper.SetAsyncCallback(ctx, sourcePort, sourceChannel, sequence, asyncCallbackBz); err != nil {
		return sequence, err
	}

	return sequence, nil
}
