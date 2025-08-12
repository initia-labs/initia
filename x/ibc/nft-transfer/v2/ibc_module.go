package v2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/api"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"

	"github.com/initia-labs/initia/x/ibc/nft-transfer/keeper"
	"github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

var _ api.IBCModule = (*IBCModule)(nil)

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModule(k keeper.Keeper) IBCModule {
	return IBCModule{
		keeper: k,
	}
}

type IBCModule struct {
	keeper keeper.Keeper
}

// buildV1Packet builds a v1 packet for keeper compatibility
func buildV1Packet(sequence uint64, payload channeltypesv2.Payload, sourceChannel, destinationChannel string) channeltypes.Packet {
	return channeltypes.NewPacket(
		[]byte{}, // data will be set by keeper
		sequence,
		payload.SourcePort,
		sourceChannel,
		payload.DestinationPort,
		destinationChannel,
		clienttypes.ZeroHeight(),
		0,
	)
}

func (im IBCModule) OnSendPacket(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, signer sdk.AccAddress) error {
	// Enforce that the source and destination portIDs are the same and equal to the nft-transfer portID
	// Enforce that the source and destination clientIDs are also in the clientID format that nft-transfer expects: {clientid}-{sequence}
	// This is necessary for IBC v2 since the portIDs (and thus the application-application connection) is not prenegotiated
	// by the channel handshake
	// This restriction can be removed in a future where the trace hop on receive commits to **both** the source and destination portIDs
	// rather than just the destination port
	if payload.SourcePort != types.PortID || payload.DestinationPort != types.PortID {
		return errorsmod.Wrapf(channeltypesv2.ErrInvalidPacket, "payload port ID is invalid: expected %s, got sourcePort: %s destPort: %s", types.PortID, payload.SourcePort, payload.DestinationPort)
	}
	if !clienttypes.IsValidClientID(sourceChannel) || !clienttypes.IsValidClientID(destinationChannel) {
		return errorsmod.Wrapf(channeltypesv2.ErrInvalidPacket, "client IDs must be in valid format: {string}-{number}")
	}

	data, err := types.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		return err
	}

	sender, err := sdk.AccAddressFromBech32(data.Sender)
	if err != nil {
		return err
	}

	if !signer.Equals(sender) {
		return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "sender %s is different from signer %s", sender, signer)
	}

	// Enforce that the base class does not contain any slashes
	// Since IBC v2 packets will no longer have channel identifiers, we cannot rely
	// on the channel format to easily divide the trace from the base denomination in ICS721 v1 packets
	// The simplest way to prevent any potential issues from arising is to simply disallow any slashes in the base class
	// This prevents such classes from being sent with IBC v2 packets, however we can still support them in IBC v1 packets
	if strings.Contains(data.ClassId, "/") {
		return errorsmod.Wrapf(types.ErrInvalidClassId, "base class %s cannot contain slashes for IBC v2 packet", data.ClassId)
	}

	// Use SendNftTransfer from keeper which handles escrow/burn logic
	// Get timeout from payload if available, otherwise use defaults
	timeoutHeight := clienttypes.ZeroHeight()
	timeoutTimestamp := uint64(0)
	
	if err := im.keeper.SendNftTransfer(
		ctx,
		payload.SourcePort,
		sourceChannel,
		data.ClassId,
		data.TokenIds,
		sender,
		data.Receiver,
		timeoutHeight,
		timeoutTimestamp,
	); err != nil {
		return err
	}

	// Emit transfer event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypePacket,
			sdk.NewAttribute(sdk.AttributeKeySender, data.Sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, data.Receiver),
			sdk.NewAttribute(types.AttributeKeyClassId, data.ClassId),
			sdk.NewAttribute(types.AttributeKeyTokenIds, strings.Join(data.TokenIds, ",")),
			sdk.NewAttribute(types.AttributeKeyMemo, data.Memo),
		),
	})

	return nil
}

func (im IBCModule) OnRecvPacket(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
	// Enforce that the source and destination portIDs are the same and equal to the nft-transfer portID
	// Enforce that the source and destination clientIDs are also in the clientID format that nft-transfer expects: {clientid}-{sequence}
	// This is necessary for IBC v2 since the portIDs (and thus the application-application connection) is not prenegotiated
	// by the channel handshake
	// This restriction can be removed in a future where the trace hop on receive commits to **both** the source and destination portIDs
	// rather than just the destination port
	if payload.SourcePort != types.PortID || payload.DestinationPort != types.PortID {
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
	}
	if !clienttypes.IsValidClientID(sourceChannel) || !clienttypes.IsValidClientID(destinationChannel) {
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
	}

	var (
		ackErr error
		data   types.NonFungibleTokenPacketData
	)

	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	recvResult := channeltypesv2.RecvPacketResult{
		Status:          channeltypesv2.PacketStatus_Success,
		Acknowledgement: ack.Acknowledgement(),
	}
	// we are explicitly wrapping this emit event call in an anonymous function so that
	// the packet data is evaluated after it has been assigned a value.
	defer func() {
		// Emit on recv packet event
		if ackErr == nil {
			ctx.EventManager().EmitEvents(sdk.Events{
				sdk.NewEvent(
					types.EventTypePacket,
					sdk.NewAttribute(sdk.AttributeKeySender, data.Sender),
					sdk.NewAttribute(types.AttributeKeyReceiver, data.Receiver),
					sdk.NewAttribute(types.AttributeKeyClassId, data.ClassId),
					sdk.NewAttribute(types.AttributeKeyTokenIds, strings.Join(data.TokenIds, ",")),
					sdk.NewAttribute(types.AttributeKeyAckSuccess, fmt.Sprintf("%t", ackErr == nil)),
				),
			})
		}
	}()

	data, ackErr = types.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if ackErr != nil {
		im.keeper.Logger(ctx).Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), sequence))
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
	}

	// Build v1 packet for keeper compatibility
	packet := buildV1Packet(sequence, payload, sourceChannel, destinationChannel)

	if ackErr = im.keeper.OnRecvPacket(ctx, packet, data); ackErr != nil {
		im.keeper.Logger(ctx).Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), sequence))
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
	}

	im.keeper.Logger(ctx).Info("successfully handled ICS-721 packet", "sequence", sequence)

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return recvResult
}

func (im IBCModule) OnTimeoutPacket(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	data, err := types.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		return err
	}

	// Build v1 packet for keeper compatibility
	packet := buildV1Packet(sequence, payload, sourceChannel, destinationChannel)

	// refund tokens
	if err := im.keeper.OnTimeoutPacket(ctx, packet, data); err != nil {
		return err
	}

	// Emit timeout event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTimeout,
			sdk.NewAttribute(types.AttributeKeyRefundReceiver, data.Sender),
			sdk.NewAttribute(types.AttributeKeyRefundClassId, data.ClassId),
			sdk.NewAttribute(types.AttributeKeyRefundTokenIds, strings.Join(data.TokenIds, ",")),
		),
	})

	return nil
}

func (im IBCModule) OnAcknowledgementPacket(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, acknowledgement []byte, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	var ack channeltypes.Acknowledgement
	// construct an error acknowledgement if the acknowledgement bytes are the sentinel error acknowledgement so we can use the shared nft-transfer logic
	if bytes.Equal(acknowledgement, channeltypesv2.ErrorAcknowledgement[:]) {
		// the specific error does not matter, use a generic error
		ack = channeltypes.NewErrorAcknowledgement(errorsmod.Wrap(types.ErrInvalidPacket, "acknowledgement failed"))
	} else {
		// Try to unmarshal as JSON acknowledgement
		if err := json.Unmarshal(acknowledgement, &ack); err != nil {
			// If JSON unmarshal fails, treat as opaque successful acknowledgement
			// This maintains compatibility with different acknowledgement formats
			ack = channeltypes.NewResultAcknowledgement(acknowledgement)
		}
		// Only error acknowledgements that aren't the sentinel error are invalid in v2
		if !ack.Success() {
			return errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "cannot pass in a custom error acknowledgement with IBC v2")
		}
	}

	data, err := types.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		return err
	}

	// Build v1 packet for keeper compatibility
	packet := buildV1Packet(sequence, payload, sourceChannel, destinationChannel)

	// Process acknowledgement
	if err := im.keeper.OnAcknowledgementPacket(ctx, packet, data, ack); err != nil {
		return err
	}

	// Emit acknowledgement event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypePacket,
			sdk.NewAttribute(sdk.AttributeKeySender, data.Sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, data.Receiver),
			sdk.NewAttribute(types.AttributeKeyClassId, data.ClassId),
			sdk.NewAttribute(types.AttributeKeyTokenIds, strings.Join(data.TokenIds, ",")),
			sdk.NewAttribute(types.AttributeKeyAckSuccess, fmt.Sprintf("%t", ack.Success())),
		),
	})

	return nil
}

// UnmarshalPacketData unmarshals the ICS721 packet data based on the version and encoding
// it implements the PacketDataUnmarshaler interface
func (IBCModule) UnmarshalPacketData(payload channeltypesv2.Payload) (any, error) {
	return types.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
}