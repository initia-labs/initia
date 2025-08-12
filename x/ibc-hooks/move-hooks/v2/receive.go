package v2

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"

	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"

	ibchooksv2 "github.com/initia-labs/initia/x/ibc-hooks/v2"
	movehooks "github.com/initia-labs/initia/x/ibc-hooks/move-hooks"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

// onRecvIcs20Packet handles ICS20 packet reception with move hooks
func (h MoveHooks) onRecvIcs20Packet(
	ctx sdk.Context,
	im ibchooksv2.IBCMiddleware,
	sourceChannel string,
	destinationChannel string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
	data transfertypes.InternalTransferRepresentation,
) channeltypesv2.RecvPacketResult {
	isMoveRouted, hookData, err := movehooks.ValidateAndParseMemo(data.Memo)
	if !isMoveRouted || (err == nil && hookData.Message == nil) {
		return im.App.OnRecvPacket(ctx, sourceChannel, destinationChannel, sequence, payload, relayer)
	} else if err != nil {
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: newEmitErrorAcknowledgement(err).Acknowledgement(),
		}
	}

	msg := hookData.Message
	if allowed, err := movehooks.CheckACL(ctx, h.ac, h.hooksKeeper, msg.ModuleAddress); err != nil {
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: newEmitErrorAcknowledgement(err).Acknowledgement(),
		}
	} else if !allowed {
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: newEmitErrorAcknowledgement(fmt.Errorf("modules deployed by `%s` are not allowed to be used in ibchooks", msg.ModuleAddress)).Acknowledgement(),
		}
	}

	// Validate whether the receiver is correctly specified or not.
	if err := movehooks.ValidateReceiver(msg, data.Receiver, h.ac); err != nil {
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: newEmitErrorAcknowledgement(err).Acknowledgement(),
		}
	}

	// Calculate the receiver / contract caller based on the packet's channel and sender
	intermediateSender := movehooks.DeriveIntermediateSender(destinationChannel, data.Sender)

	// The funds sent on this packet need to be transferred to the intermediary account for the sender.
	// For this, we override the ICS20 packet's Receiver (essentially hijacking the funds to this new address)
	// and execute the underlying OnRecvPacket() call (which should eventually land on the transfer app's
	// relay.go and send the funds to the intermediary account.
	//
	// If that succeeds, we make the contract call
	newData := data
	newData.Receiver = intermediateSender
	newData.Memo = ""

	// Create new payload with modified data
	newPayload := payload
	// Marshal the InternalTransferRepresentation back to payload value
	newPayload.Encoding = transfertypes.EncodingJSON
	newPayload.Value, err = json.Marshal(newData)
	if err != nil {
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: newEmitErrorAcknowledgement(err).Acknowledgement(),
		}
	}

	ack := im.App.OnRecvPacket(ctx, sourceChannel, destinationChannel, sequence, newPayload, relayer)
	if ack.Status != channeltypesv2.PacketStatus_Success {
		return ack
	}

	msg.Sender = intermediateSender
	if _, err := movehooks.ExecMsg(ctx, msg, h.moveKeeper, h.ac); err != nil {
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: newEmitErrorAcknowledgement(err).Acknowledgement(),
		}
	}

	return ack
}

func (h MoveHooks) onRecvIcs721Packet(
	ctx sdk.Context,
	im ibchooksv2.IBCMiddleware,
	sourceChannel string,
	destinationChannel string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
	data nfttransfertypes.NonFungibleTokenPacketData,
) channeltypesv2.RecvPacketResult {
	isMoveRouted, hookData, err := movehooks.ValidateAndParseMemo(data.Memo)
	if !isMoveRouted || (err == nil && hookData.Message == nil) {
		return im.App.OnRecvPacket(ctx, sourceChannel, destinationChannel, sequence, payload, relayer)
	} else if err != nil {
		return channeltypesv2.RecvPacketResult{
			Acknowledgement: newEmitErrorAcknowledgement(err).Acknowledgement(),
		}
	}

	msg := hookData.Message
	if allowed, err := movehooks.CheckACL(ctx, h.ac, h.hooksKeeper, msg.ModuleAddress); err != nil {
		return channeltypesv2.RecvPacketResult{
			Acknowledgement: newEmitErrorAcknowledgement(err).Acknowledgement(),
		}
	} else if !allowed {
		return channeltypesv2.RecvPacketResult{
			Acknowledgement: newEmitErrorAcknowledgement(fmt.Errorf("modules deployed by `%s` are not allowed to be used in ibchooks", msg.ModuleAddress)).Acknowledgement(),
		}
	}

	// Validate whether the receiver is correctly specified or not.
	if err := movehooks.ValidateReceiver(msg, data.Receiver, h.ac); err != nil {
		return channeltypesv2.RecvPacketResult{
			Acknowledgement: newEmitErrorAcknowledgement(err).Acknowledgement(),
		}
	}

	// Calculate the receiver / contract caller based on the packet's channel and sender
	intermediateSender := movehooks.DeriveIntermediateSender(destinationChannel, data.Sender)

	// The NFTs sent on this packet need to be transferred to the intermediary account for the sender.
	// For this, we override the ICS721 packet's Receiver (essentially hijacking the NFTs to this new address)
	// and execute the underlying OnRecvPacket() call
	newData := data
	newData.Receiver = intermediateSender
	newData.Memo = ""

	// Update the payload with modified data
	newPayload := payload
	newPayload.Encoding = transfertypes.EncodingJSON
	newPayload.Value, err = json.Marshal(newData)
	if err != nil {
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: newEmitErrorAcknowledgement(err).Acknowledgement(),
		}
	}

	ack := im.App.OnRecvPacket(ctx, sourceChannel, destinationChannel, sequence, newPayload, relayer)
	if ack.Status != channeltypesv2.PacketStatus_Success {
		return ack
	}

	msg.Sender = intermediateSender
	if _, err := movehooks.ExecMsg(ctx, msg, h.moveKeeper, h.ac); err != nil {
		return channeltypesv2.RecvPacketResult{
			Acknowledgement: newEmitErrorAcknowledgement(err).Acknowledgement(),
		}
	}

	return ack
}

func newEmitErrorAcknowledgement(err error) channeltypesv2.Acknowledgement {
	return channeltypesv2.NewAcknowledgement([]byte(fmt.Sprintf("ibc move hook error: %s", err.Error())))
}
