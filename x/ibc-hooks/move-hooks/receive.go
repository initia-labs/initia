package move_hooks

import (
	"fmt"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

func (h MoveHooks) onRecvIcs20Packet(
	ctx sdk.Context,
	im ibchooks.IBCMiddleware,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	data transfertypes.InternalTransferRepresentation,
) ibcexported.Acknowledgement {
	isMoveRouted, hookData, err := ValidateAndParseMemo(data.Memo)
	if !isMoveRouted || (err == nil && hookData.Message == nil) {
		return im.App.OnRecvPacket(ctx, channelVersion, packet, relayer)
	} else if err != nil {
		return newEmitErrorAcknowledgement(err)
	}

	msg := hookData.Message
	if allowed, err := CheckACL(ctx, h.ac, im.HooksKeeper, msg.ModuleAddress); err != nil {
		return newEmitErrorAcknowledgement(err)
	} else if !allowed {
		return newEmitErrorAcknowledgement(fmt.Errorf("modules deployed by `%s` are not allowed to be used in ibchooks", msg.ModuleAddress))
	}

	// Validate whether the receiver is correctly specified or not.
	if err := ValidateReceiver(msg, data.Receiver, h.ac); err != nil {
		return newEmitErrorAcknowledgement(err)
	}

	// Calculate the receiver / contract caller based on the packet's channel and sender
	intermediateSender := DeriveIntermediateSender(packet.GetDestChannel(), data.Sender)

	// The funds sent on this packet need to be transferred to the intermediary account for the sender.
	// For this, we override the ICS20 packet's Receiver (essentially hijacking the funds to this new address)
	// and execute the underlying OnRecvPacket() call (which should eventually land on the transfer app's
	// relay.go and send the funds to the intermediary account.
	//
	// If that succeeds, we make the contract call
	data.Receiver = intermediateSender

	packet.Data, err = transfertypes.MarshalPacketData(transfertypes.FungibleTokenPacketData{
		Denom:    data.Token.Denom.Path(),
		Amount:   data.Token.Amount,
		Sender:   data.Sender,
		Receiver: data.Receiver,
		Memo:     data.Memo,
	}, transfertypes.V1, transfertypes.EncodingJSON)
	if err != nil {
		return newEmitErrorAcknowledgement(err)
	}
	ack := im.App.OnRecvPacket(ctx, channelVersion, packet, relayer)
	if !ack.Success() {
		return ack
	}

	msg.Sender = intermediateSender
	if _, err := ExecMsg(ctx, msg, h.moveKeeper, h.ac); err != nil {
		return newEmitErrorAcknowledgement(err)
	}

	return ack
}

func (h MoveHooks) onRecvIcs721Packet(
	ctx sdk.Context,
	im ibchooks.IBCMiddleware,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
	data nfttransfertypes.NonFungibleTokenPacketData,
) ibcexported.Acknowledgement {
	isMoveRouted, hookData, err := ValidateAndParseMemo(data.Memo)
	if !isMoveRouted || (err == nil && hookData.Message == nil) {
		return im.App.OnRecvPacket(ctx, channelVersion, packet, relayer)
	} else if err != nil {
		return newEmitErrorAcknowledgement(err)
	}

	msg := hookData.Message
	if allowed, err := CheckACL(ctx, h.ac, im.HooksKeeper, msg.ModuleAddress); err != nil {
		return newEmitErrorAcknowledgement(err)
	} else if !allowed {
		return newEmitErrorAcknowledgement(fmt.Errorf("modules deployed by `%s` are not allowed to be used in ibchooks", msg.ModuleAddress))
	}

	// Validate whether the receiver is correctly specified or not.
	if err := ValidateReceiver(msg, data.Receiver, h.ac); err != nil {
		return newEmitErrorAcknowledgement(err)
	}

	// Calculate the receiver / contract caller based on the packet's channel and sender
	intermediateSender := DeriveIntermediateSender(packet.GetDestChannel(), data.Sender)

	// The funds sent on this packet need to be transferred to the intermediary account for the sender.
	// For this, we override the ICS721 packet's Receiver (essentially hijacking the funds to this new address)
	// and execute the underlying OnRecvPacket() call (which should eventually land on the transfer app's
	// relay.go and send the funds to the intermediary account.
	//
	// If that succeeds, we make the contract call
	data.Receiver = intermediateSender
	
	packet.Data, err = nfttransfertypes.MarshalPacketData(nfttransfertypes.NonFungibleTokenPacketData{
		ClassId:   data.ClassId,
		ClassUri:  data.ClassUri,
		ClassData: data.ClassData,
		TokenIds:  data.TokenIds,
		TokenUris: data.TokenUris,
		TokenData: data.TokenData,
		Sender:    data.Sender,
		Receiver:  data.Receiver,
		Memo:      data.Memo,
	}, nfttransfertypes.V1, nfttransfertypes.EncodingJSON)
	if err != nil {
		return newEmitErrorAcknowledgement(err)
	}

	ack := im.App.OnRecvPacket(ctx, channelVersion, packet, relayer)
	if !ack.Success() {
		return ack
	}

	msg.Sender = intermediateSender
	if _, err := ExecMsg(ctx, msg, h.moveKeeper, h.ac); err != nil {
		return newEmitErrorAcknowledgement(err)
	}

	return ack
}

// newEmitErrorAcknowledgement creates a new error acknowledgement for v1
func newEmitErrorAcknowledgement(err error) channeltypes.Acknowledgement {
	return channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{
			Error: fmt.Sprintf("ibc move hook error: %s", err.Error()),
		},
	}
}
