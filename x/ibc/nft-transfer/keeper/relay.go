package keeper

import (
	"strings"

	metrics "github.com/hashicorp/go-metrics"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	coretypes "github.com/cosmos/ibc-go/v8/modules/core/types"

	"github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

// SendNftTransfer handles transfer sending logic. There are 2 possible cases:
//
// 1. Sender chain is acting as the source zone. The nfts are transferred
// to an escrow address (i.e locked) on the sender chain and then transferred
// to the receiving chain through IBC TAO logic. It is expected that the
// receiving chain will mint vouchers to the receiving address.
//
// 2. Sender chain is acting as the sink zone. The nfts (vouchers) are burned
// on the sender chain and then transferred to the receiving chain though IBC
// TAO logic. It is expected that the receiving chain, which had previously
// sent the original nft, will un-escrow the non fungible token and send
// it to the receiving address.
//
// Note: An IBC Nft Transfer must be initiated using a MsgTransfer via the Transfer rpc handler
//
// A sending chain may be acting as a source or sink zone. When a chain is sending
// tokens across a port and channel which are not equal to the last prefixed port and
// channel pair, it is acting as a source zone. When tokens are sent from a source zone,
// the destination port and channel will be prefixed onto the classId (once the tokens are received)
// adding another hop to the tokens record. When a chain is sending tokens across a port
// and channel which are equal to the last prefixed port and channel pair, it is acting as
// a sink zone. When tokens are sent from a sink zone, the last prefixed port and channel
// pair on the classId is removed (once the tokens are received), undoing the last hop in
// the tokens record.
//
// For example, assume these steps of transfer occur:
//
// A -> B -> C -> A -> C -> B -> A
//
// A(p1,c1) -> (p2,c2)B : A is source zone. classId in B: 'p2/c2/nftClass'
// B(p3,c3) -> (p4,c4)C : B is source zone. classId in C: 'p4/c4/p2/c2/nftClass'
// C(p5,c5) -> (p6,c6)A : C is source zone. classId in A: 'p6/c6/p4/c4/p2/c2/nftClass'
// A(p6,c6) -> (p5,c5)C : A is sink zone. classId in C: 'p4/c4/p2/c2/nftClass'
// C(p4,c4) -> (p3,c3)B : C is sink zone. classId in B: 'p2/c2/nftClass'
// B(p2,c2) -> (p1,c1)A : B is sink zone. classId in A: 'nftClass'
func (k Keeper) SendNftTransfer(
	ctx sdk.Context,
	sourcePort,
	sourceChannel string,
	classId string,
	tokenIds []string,
	sender sdk.AccAddress,
	receiver string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
) error {
	_, err := k.sendNftTransfer(
		ctx,
		sourcePort,
		sourceChannel,
		classId,
		tokenIds,
		sender,
		receiver,
		timeoutHeight,
		timeoutTimestamp,
		"",
	)
	return err
}

// sendTransfer handles nft transfer sending logic.
func (k Keeper) sendNftTransfer(
	ctx sdk.Context,
	sourcePort,
	sourceChannel string,
	classId string,
	tokenIds []string,
	sender sdk.AccAddress,
	receiver string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	memo string,
) (uint64, error) {
	if ok, err := k.GetSendEnabled(ctx); err != nil {
		return 0, err
	} else if !ok {
		return 0, types.ErrSendDisabled
	}

	if len(tokenIds) == 0 {
		return 0, errors.Wrap(types.ErrInvalidTokenIds, "empty token ids")
	}

	sourceChannelEnd, found := k.channelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, errors.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", sourcePort, sourceChannel)
	}

	destinationPort := sourceChannelEnd.GetCounterparty().GetPortID()
	destinationChannel := sourceChannelEnd.GetCounterparty().GetChannelID()

	// get the next sequence
	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, errors.Wrapf(
			channeltypes.ErrSequenceSendNotFound,
			"source port: %s, source channel: %s", sourcePort, sourceChannel,
		)
	}
	// begin createOutgoingPacket logic
	// See spec for this logic: https://github.com/cosmos/ibc/tree/main/spec/app/ics-721-nft-transfer#packet-relay
	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePort, sourceChannel))
	if !ok {
		return 0, errors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	labels := []metrics.Label{
		telemetry.NewLabel(coretypes.LabelDestinationPort, destinationPort),
		telemetry.NewLabel(coretypes.LabelDestinationChannel, destinationChannel),
	}

	// get class info
	className, classUri, classDesc, err := k.nftKeeper.GetClassInfo(ctx, classId)
	if err != nil {
		return 0, err
	}

	// get token info
	tokenUris, tokenDesc, err := k.nftKeeper.GetTokenInfos(ctx, classId, tokenIds)
	if err != nil {
		return 0, err
	}

	// NOTE: class id and hex hash correctness checked during msg.ValidateBasic
	fullClassIdPath := classId

	var classData string
	tokenData := make([]string, len(tokenIds))

	// deconstruct the nft class id into the class id trace info
	// to determine if the sender is the source chain
	if strings.HasPrefix(classId, types.ClassIdPrefix+"/") {
		fullClassIdPath, err = k.ClassIdPathFromHash(ctx, classId)
		if err != nil {
			return 0, err
		}
		// construct the class id trace from the full raw class id
		classTrace := types.ParseClassTrace(fullClassIdPath)
		traceHash := classTrace.Hash()

		// override classData to the data stored, which is relayed from the source chain
		classData, err = k.ClassData.Get(ctx, traceHash)
		if err != nil {
			return 0, err
		}

		// override tokenData to the data stored, which is relayed from the source chain
		for i, tokenId := range tokenIds {
			tokenData[i], err = k.TokenData.Get(ctx, collections.Join(traceHash.Bytes(), tokenId))
			if err != nil {
				return 0, err
			}
		}
	} else {
		classData, err = types.ConvertClassDataToICS721(className, classDesc)
		if err != nil {
			return 0, err
		}

		for i := range tokenDesc {
			tokenData[i], err = types.ConvertTokenDataToICS721(tokenDesc[i])
			if err != nil {
				return 0, err
			}
		}
	}

	// NOTE: SendNftTransfer simply sends the class id as it exists on its own
	// chain inside the packet data. The receiving chain will perform class id
	// prefixing as necessary.
	if types.SenderChainIsSource(sourcePort, sourceChannel, fullClassIdPath) {
		labels = append(labels, telemetry.NewLabel(coretypes.LabelSource, "true"))

		// create the escrow address for the tokens
		escrowAddress := types.GetEscrowAddress(sourcePort, sourceChannel)

		// escrow source tokens. It fails if balance insufficient.
		if err := k.nftKeeper.Transfers(
			ctx, sender, escrowAddress, classId, tokenIds,
		); err != nil {
			return 0, err
		}

	} else {
		labels = append(labels, telemetry.NewLabel(coretypes.LabelSource, "false"))

		if err := k.nftKeeper.Burns(
			ctx, sender, classId, tokenIds,
		); err != nil {
			return 0, err
		}
	}

	packetData := types.NewNonFungibleTokenPacketData(
		fullClassIdPath, classUri, classData,
		tokenIds, tokenUris, tokenData, sender.String(), receiver, memo,
	)
	if _, err := k.ics4Wrapper.SendPacket(
		ctx, channelCap, sourcePort, sourceChannel,
		timeoutHeight, timeoutTimestamp, packetData.GetBytes(),
	); err != nil {
		return 0, err
	}

	defer func() {
		if len(tokenIds) > 0 {
			telemetry.SetGaugeWithLabels(
				[]string{"tx", "msg", "ibc", "nft", "transfer"},
				float32(len(tokenIds)),
				[]metrics.Label{
					telemetry.NewLabel(types.LabelClassId, fullClassIdPath),
					telemetry.NewLabel(types.LabelTokenIds, strings.Join(tokenIds, ",")),
				},
			)
		}

		telemetry.IncrCounterWithLabels(
			[]string{"ibc", types.ModuleName, "send"},
			1,
			labels,
		)
	}()

	return sequence, nil
}

// OnRecvPacket processes a cross chain non fungible token transfer. If the
// sender chain is the source of minted tokens then vouchers will be minted
// and sent to the receiving address. Otherwise if the sender chain is sending
// back tokens this chain originally transferred to it, the tokens are
// un-escrowed and sent to the receiving address.
func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, data types.NonFungibleTokenPacketData) error {
	// validate packet data upon receiving
	if err := data.ValidateBasic(); err != nil {
		return err
	}

	if ok, err := k.GetReceiveEnabled(ctx); err != nil {
		return err
	} else if !ok {
		return types.ErrReceiveDisabled
	}

	// decode the receiver address
	receiver, err := k.authKeeper.AddressCodec().StringToBytes(data.Receiver)
	if err != nil {
		return err
	}

	// create account if receiver does not exist
	if !k.authKeeper.HasAccount(ctx, receiver) {
		k.authKeeper.SetAccount(ctx, k.authKeeper.NewAccountWithAddress(ctx, receiver))
	}

	labels := []metrics.Label{
		telemetry.NewLabel(coretypes.LabelSourcePort, packet.GetSourcePort()),
		telemetry.NewLabel(coretypes.LabelSourceChannel, packet.GetSourceChannel()),
	}

	// This is the prefix that would have been prefixed to the class id
	// on sender chain IF and only if the token originally came from the
	// receiving chain.
	//
	// NOTE: We use SourcePort and SourceChannel here, because the counterpart
	// chain would have prefixed with DestPort and DestChannel when originally
	// receiving this token as seen in the "sender chain is the source" condition.

	if types.ReceiverChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), data.ClassId) {
		// sender chain is not the source, un-escrow tokens

		// remove prefix added by sender chain
		voucherPrefix := types.GetClassIdPrefix(packet.GetSourcePort(), packet.GetSourceChannel())
		unprefixedClassId := data.ClassId[len(voucherPrefix):]

		// token class id used in sending from the escrow address
		classId := unprefixedClassId

		// The class id used to send the coins is either the native classId or the hash of the path
		// if the class id is not native.
		classTrace := types.ParseClassTrace(unprefixedClassId)
		if classTrace.Path != "" {
			classId = classTrace.IBCClassId()
		}

		// un-escrow tokens
		escrowAddress := types.GetEscrowAddress(packet.GetDestPort(), packet.GetDestChannel())
		if err := k.nftKeeper.Transfers(ctx, escrowAddress, receiver, classId, data.TokenIds); err != nil {
			// NOTE: this error is only expected to occur given an unexpected bug or a malicious
			// counterpartymodule. The bug may occur in bank or any part of the code that allows
			// the escrow address to be drained. A malicious counterpartymodule could drain the
			// escrow address by allowing more tokens to be sent back then were escrowed.
			return errors.Wrap(err, "unable to un-escrow tokens, this may be caused by a malicious counterpartymodule or a bug: please open an issue on counterpartymodule")
		}

		defer func() {
			if len(data.TokenIds) > 0 {
				telemetry.SetGaugeWithLabels(
					[]string{"ibc", types.ModuleName, "packet", "receive"},
					float32(len(data.TokenIds)),
					[]metrics.Label{
						telemetry.NewLabel(types.LabelClassId, unprefixedClassId),
						telemetry.NewLabel(types.LabelTokenIds, strings.Join(data.TokenIds, ",")),
					},
				)
			}

			telemetry.IncrCounterWithLabels(
				[]string{"ibc", types.ModuleName, "receive"},
				1,
				append(
					labels, telemetry.NewLabel(coretypes.LabelSource, "true"),
				),
			)
		}()

		return nil
	}

	// sender chain is the source, mint vouchers

	// since SendPacket did not prefix the class id, we must prefix class id here
	sourcePrefix := types.GetClassIdPrefix(packet.GetDestPort(), packet.GetDestChannel())
	// NOTE: sourcePrefix contains the trailing "/"
	prefixedClassId := sourcePrefix + data.ClassId

	// construct the class id trace from the full raw class id
	classTrace := types.ParseClassTrace(prefixedClassId)
	traceHash := classTrace.Hash()

	if ok, err := k.ClassTraces.Has(ctx, traceHash); err != nil {
		return err
	} else if !ok {
		if err := k.ClassTraces.Set(ctx, traceHash, classTrace); err != nil {
			return err
		}
	}

	// store the class data
	if ok, err := k.ClassData.Has(ctx, traceHash); err != nil {
		return err
	} else if !ok {
		err = k.ClassData.Set(ctx, traceHash, data.ClassData)
		if err != nil {
			return err
		}
	}

	// store the token data
	for i, tokenId := range data.TokenIds {
		if ok, err := k.TokenData.Has(ctx, collections.Join(traceHash.Bytes(), tokenId)); err != nil {
			return err
		} else if !ok {
			err = k.TokenData.Set(ctx, collections.Join(traceHash.Bytes(), tokenId), data.TokenData[i])
			if err != nil {
				return err
			}
		}
	}

	voucherClassId := classTrace.IBCClassId()
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeClassTrace,
			sdk.NewAttribute(types.AttributeKeyTraceHash, traceHash.String()),
			sdk.NewAttribute(types.AttributeKeyClassId, voucherClassId),
		),
	)

	// create or update class
	if err := k.nftKeeper.CreateOrUpdateClass(ctx, voucherClassId, data.ClassUri, data.ClassData); err != nil {
		return err
	}

	// mint new tokens if the source of the transfer is the same chain
	if err := k.nftKeeper.Mints(
		ctx, receiver, voucherClassId,
		data.TokenIds, data.TokenUris, data.TokenData,
	); err != nil {
		return err
	}

	defer func() {
		if len(data.TokenIds) > 0 {
			telemetry.SetGaugeWithLabels(
				[]string{"ibc", types.ModuleName, "packet", "receive"},
				float32(len(data.TokenIds)),
				[]metrics.Label{
					telemetry.NewLabel(types.LabelClassId, data.ClassId),
					telemetry.NewLabel(types.LabelTokenIds, strings.Join(data.TokenIds, ",")),
				},
			)
		}

		telemetry.IncrCounterWithLabels(
			[]string{"ibc", types.ModuleName, "receive"},
			1,
			append(
				labels, telemetry.NewLabel(coretypes.LabelSource, "false"),
			),
		)
	}()

	return nil
}

// OnAcknowledgementPacket responds to the success or failure of a packet
// acknowledgement written on the receiving chain. If the acknowledgement
// was a success then nothing occurs. If the acknowledgement failed, then
// the sender is refunded their tokens using the refundPacketToken function.
func (k Keeper) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, data types.NonFungibleTokenPacketData, ack channeltypes.Acknowledgement) error {
	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Error:
		return k.refundPacketToken(ctx, packet, data)
	default:
		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
		return nil
	}
}

// OnTimeoutPacket refunds the sender since the original packet sent was
// never received and has been timed out.
func (k Keeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, data types.NonFungibleTokenPacketData) error {
	return k.refundPacketToken(ctx, packet, data)
}

// refundPacketToken will un-escrow and send back the tokens back to sender
// if the sending chain was the source chain. Otherwise, the sent tokens
// were burnt in the original send so new tokens are minted and sent to
// the sending address.
func (k Keeper) refundPacketToken(ctx sdk.Context, packet channeltypes.Packet, data types.NonFungibleTokenPacketData) error {
	// NOTE: packet data type already checked in handler.go

	// parse the class id from the full classId path
	trace := types.ParseClassTrace(data.ClassId)

	classId := trace.IBCClassId()
	tokenIds := data.TokenIds
	tokenUris := data.TokenUris
	tokenData := data.TokenData

	// decode the sender address
	sender, err := k.authKeeper.AddressCodec().StringToBytes(data.Sender)
	if err != nil {
		return err
	}

	if types.SenderChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), data.ClassId) {
		// un-escrow tokens back to sender
		escrowAddress := types.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())
		if err := k.nftKeeper.Transfers(ctx, escrowAddress, sender, classId, tokenIds); err != nil {
			// NOTE: this error is only expected to occur given an unexpected bug or a malicious
			// counterpartymodule. The bug may occur in bank or any part of the code that allows
			// the escrow address to be drained. A malicious counterpartymodule could drain the
			// escrow address by allowing more tokens to be sent back then were escrowed.
			return errors.Wrap(err, "unable to un-escrow tokens, this may be caused by a malicious counterpartymodule or a bug: please open an issue on counterpartymodule")
		}

		return nil
	}

	// mint vouchers back to sender
	if err := k.nftKeeper.Mints(
		ctx, sender, classId, tokenIds, tokenUris, tokenData,
	); err != nil {
		return err
	}

	return nil
}

// ClassIdPathFromHash returns the full class id path prefix from an ibc class id with a hash
// component.
func (k Keeper) ClassIdPathFromHash(ctx sdk.Context, classId string) (string, error) {
	// trim the class id prefix, by default "ibc/"
	hexHash := classId[len(types.ClassIdPrefix+"/"):]

	hash, err := types.ParseHexHash(hexHash)
	if err != nil {
		return "", errors.Wrap(types.ErrInvalidClassIdForNftTransfer, err.Error())
	}

	classTrace, err := k.ClassTraces.Get(ctx, hash)
	if err != nil {
		return "", err
	}

	fullClassIdPath := classTrace.GetFullClassIdPath()
	return fullClassIdPath, nil
}
