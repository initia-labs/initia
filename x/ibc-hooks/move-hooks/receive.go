package move_hooks

import (
	"errors"
	"fmt"

	"cosmossdk.io/math"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
	ibchookstypes "github.com/initia-labs/initia/x/ibc-hooks/types"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"
	movetypes "github.com/initia-labs/initia/x/move/types"
)

func (h MoveHooks) onRecvIcs20Packet(
	ctx sdk.Context,
	im ibchooks.IBCMiddleware,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	data transfertypes.FungibleTokenPacketData,
) ibcexported.Acknowledgement {
	var beforeBalance math.Int
	return h.handleOnReceive(ctx, im, packet, relayer, data.GetSender(), ibchookstypes.ICSData{
		ICS20Data: &data,
	}, func(intermediateAddr sdk.AccAddress) error {
		denom := ibchookstypes.GetReceivedTokenDenom(packet, data)
		var err error
		beforeBalance, err = h.moveKeeper.MoveBankKeeper().GetBalance(ctx, intermediateAddr, denom)
		if err != nil {
			return err
		}
		return nil
	}, func(intermediateAddr sdk.AccAddress) error {
		denom := ibchookstypes.GetReceivedTokenDenom(packet, data)
		// get balance after underlying OnRecvPacket() call
		afterBalance, err := h.moveKeeper.MoveBankKeeper().GetBalance(ctx, intermediateAddr, denom)
		if err != nil {
			return err
		}

		// compute amount in packet
		amountInPacket, ok := math.NewIntFromString(data.Amount)
		if !ok {
			return errors.New("invalid amount for transfer")
		}

		// compute balance change
		balanceChange := math.ZeroInt()
		if afterBalance.GT(beforeBalance) {
			balanceChange = afterBalance.Sub(beforeBalance)
		}

		// store transfer funds to be used in contract call
		if err := im.HooksKeeper.SetTransferFunds(ctx, ibchookstypes.TransferFunds{
			BalanceChange:  sdk.NewCoin(denom, balanceChange),
			AmountInPacket: sdk.NewCoin(denom, amountInPacket),
		}); err != nil {
			return err
		}
		return nil
	}, func() error {
		// clean up transfer funds after contract call
		return im.HooksKeeper.RemoveTransferFunds(ctx)
	})
}

func (h MoveHooks) onRecvIcs721Packet(
	ctx sdk.Context,
	im ibchooks.IBCMiddleware,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	data nfttransfertypes.NonFungibleTokenPacketData,
) ibcexported.Acknowledgement {
	return h.handleOnReceive(ctx, im, packet, relayer, data.GetSender(), ibchookstypes.ICSData{
		ICS721Data: &data,
	}, nil, nil, nil)
}

func (h MoveHooks) execMsg(ctx sdk.Context, msg *movetypes.MsgExecute) (*movetypes.MsgExecuteResponse, error) {
	if err := msg.Validate(h.ac); err != nil {
		return nil, err
	}

	moveMsgServer := movekeeper.NewMsgServerImpl(h.moveKeeper)
	res, err := moveMsgServer.Execute(ctx, msg)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (h MoveHooks) execMsgJSON(ctx sdk.Context, msg *movetypes.MsgExecuteJSON) (*movetypes.MsgExecuteJSONResponse, error) {
	if err := msg.Validate(h.ac); err != nil {
		return nil, err
	}

	moveMsgServer := movekeeper.NewMsgServerImpl(h.moveKeeper)
	res, err := moveMsgServer.ExecuteJSON(ctx, msg)
	if err != nil {
		return nil, err
	}

	return res, nil
}

type hookMessage struct {
	moduleAddress      string
	functionIdentifier string
	exec               func(ctx sdk.Context, sender string) error
}

func (h MoveHooks) prepareHookMessage(hookData HookData) (hookMessage, error) {
	if hookData.Message != nil && hookData.MessageJSON != nil {
		return hookMessage{}, errors.New("only one of message or message_json can be set")
	}

	switch {
	case hookData.Message != nil:
		return hookMessage{
			moduleAddress:      hookData.Message.ModuleAddress,
			functionIdentifier: fmt.Sprintf("%s::%s::%s", hookData.Message.ModuleAddress, hookData.Message.ModuleName, hookData.Message.FunctionName),
			exec: func(ctx sdk.Context, sender string) error {
				hookData.Message.Sender = sender
				_, err := h.execMsg(ctx, hookData.Message)
				return err
			},
		}, nil
	case hookData.MessageJSON != nil:
		return hookMessage{
			moduleAddress:      hookData.MessageJSON.ModuleAddress,
			functionIdentifier: fmt.Sprintf("%s::%s::%s", hookData.MessageJSON.ModuleAddress, hookData.MessageJSON.ModuleName, hookData.MessageJSON.FunctionName),
			exec: func(ctx sdk.Context, sender string) error {
				hookData.MessageJSON.Sender = sender
				_, err := h.execMsgJSON(ctx, hookData.MessageJSON)
				return err
			},
		}, nil
	default:
		return hookMessage{}, nil
	}
}

func (h MoveHooks) handleOnReceive(
	ctx sdk.Context,
	im ibchooks.IBCMiddleware,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	sender string,
	data ibchookstypes.ICSData,
	beforeAppExecuted func(intermediateAddr sdk.AccAddress) error,
	beforeHookExecuted func(intermediateAddr sdk.AccAddress) error,
	afterHookExecuted func() error,
) ibcexported.Acknowledgement {
	hookData := unmarshalMemo(data.GetMemo())
	if hookData == nil {
		return im.App.OnRecvPacket(ctx, packet, relayer)
	}

	hookMsg, err := h.prepareHookMessage(*hookData)
	if err != nil {
		return newEmitErrorAcknowledgement(err)
	}
	if hookMsg.exec == nil {
		return im.App.OnRecvPacket(ctx, packet, relayer)
	}

	if allowed, err := h.checkACL(im, ctx, hookMsg.moduleAddress); err != nil {
		return newEmitErrorAcknowledgement(err)
	} else if !allowed {
		return newEmitErrorAcknowledgement(fmt.Errorf("modules deployed by `%s` are not allowed to be used in ibchooks", hookMsg.moduleAddress))
	}

	// Validate whether the receiver is correctly specified or not.
	if err := validateReceiver(hookMsg.functionIdentifier, data.GetReceiver(), h.ac); err != nil {
		return newEmitErrorAcknowledgement(err)
	}

	// Calculate the receiver / contract caller based on the packet's channel and sender
	intermediateSender := DeriveIntermediateSender(packet.GetDestChannel(), data.GetSender())
	intermediateAddr, err := h.ac.StringToBytes(intermediateSender)
	if err != nil {
		return newEmitErrorAcknowledgement(err)
	}

	// The funds sent on this packet need to be transferred to the intermediary account for the sender.
	// For this, we override the ICS721 packet's Receiver (essentially hijacking the funds to this new address)
	// and execute the underlying OnRecvPacket() call (which should eventually land on the transfer app's
	// relay.go and send the funds to the intermediary account.
	//
	// If that succeeds, we make the contract call
	data.SetReceiver(intermediateSender)
	packet.Data = data.GetBytes()

	// execute any beforeAppExecuted logic
	if beforeAppExecuted != nil {
		if err := beforeAppExecuted(intermediateAddr); err != nil {
			return newEmitErrorAcknowledgement(err)
		}
	}

	ack := im.App.OnRecvPacket(ctx, packet, relayer)
	if !ack.Success() {
		return ack
	}

	// execute any beforeHookExecuted logic
	if beforeHookExecuted != nil {
		if err := beforeHookExecuted(intermediateAddr); err != nil {
			return newEmitErrorAcknowledgement(err)
		}
	}

	// execute contract call
	if err := hookMsg.exec(ctx, intermediateSender); err != nil {
		return newEmitErrorAcknowledgement(err)
	}

	// execute any afterHookExecuted logic
	if afterHookExecuted != nil {
		if err := afterHookExecuted(); err != nil {
			return newEmitErrorAcknowledgement(err)
		}
	}

	return ack
}
