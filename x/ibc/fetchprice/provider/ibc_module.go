package provider

import (
	"fmt"
	"math"
	"strings"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	providerkeeper "github.com/initia-labs/initia/x/ibc/fetchprice/provider/keeper"
	providertypes "github.com/initia-labs/initia/x/ibc/fetchprice/provider/types"
	"github.com/initia-labs/initia/x/ibc/fetchprice/types"
)

// IBCModule implements oracle price provider given the oracle keeper.
type IBCModule struct {
	cdc codec.Codec
	pk  providerkeeper.Keeper
}

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModule(
	cdc codec.Codec,
	pk providerkeeper.Keeper,
) IBCModule {
	return IBCModule{
		cdc: cdc,
		pk:  pk,
	}
}

// validateFetchPriceProviderChannelParams does validation of a newly created fetchprice channel. A fetchprice
// channel must be UNORDERED, use the correct port (by default 'fetchprice-provide' and 'fetchprice-consumer'),
// and use the current supported version. Only 2^32 channels are allowed to be created.
func validateFetchPriceProviderChannelParams(
	ctx sdk.Context,
	pk providerkeeper.Keeper,
	order channeltypes.Order,
	portID string,
	counterpartyPortID string,
	channelID string,
) error {
	// NOTE: for escrow address security only 2^32 channels are allowed to be created
	// Issue: https://github.com/cosmos/cosmos-sdk/issues/7737
	channelSequence, err := channeltypes.ParseChannelSequence(channelID)
	if err != nil {
		return err
	}

	if channelSequence > uint64(math.MaxUint32) {
		return errorsmod.Wrapf(types.ErrMaxTransferChannels, "channel sequence %d is greater than max allowed fetchprice channels %d", channelSequence, uint64(math.MaxUint32))
	}

	if order != channeltypes.UNORDERED {
		return errorsmod.Wrapf(channeltypes.ErrInvalidChannelOrdering, "expected %s channel, got %s ", channeltypes.UNORDERED, order)
	}

	if counterpartyPortID != types.ConsumerPortID {
		return errorsmod.Wrapf(types.ErrInvalidConsumerPort, "expected %s, got %s", types.ConsumerPortID, counterpartyPortID)
	}

	// Require portID is the portID fetchprice provider module is bound to
	boundPort, err := pk.PortID.Get(ctx)
	if err != nil {
		return err
	}
	if portID != boundPort {
		return errorsmod.Wrapf(types.ErrInvalidProviderPort, "expected %s, got %s", boundPort, portID)
	}

	return nil
}

// OnChanOpenInit implements the IBCModule interface
func (im IBCModule) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return "", errorsmod.Wrap(types.ErrInvalidChannelFlow, "channel handshake must be initiated by consumer chain")
}

// OnChanOpenTry implements the IBCModule interface.
func (im IBCModule) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	if err := validateFetchPriceProviderChannelParams(ctx, im.pk, order, portID, counterparty.PortId, channelID); err != nil {
		return "", err
	}

	if counterpartyVersion != types.Version {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "invalid counterparty version: got: %s, expected %s", counterpartyVersion, types.Version)
	}

	// OpenTry must claim the channelCapability that IBC passes into the callback
	if err := im.pk.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", err
	}

	return types.Version, nil
}

// OnChanOpenAck implements the IBCModule interface
func (im IBCModule) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	_ string,
	counterpartyVersion string,
) error {
	return errorsmod.Wrap(types.ErrInvalidChannelFlow, "channel handshake must be initiated by consumer chain")
}

// OnChanOpenConfirm implements the IBCModule interface
func (im IBCModule) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return nil
}

// OnChanCloseInit implements the IBCModule interface
func (im IBCModule) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Disallow user-initiated channel closing for fetchprice channels
	return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCModule interface
func (im IBCModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return nil
}

// OnRecvPacket implements the IBCModule interface. A successful acknowledgement
// is returned if the packet data is successfully decoded and the receive application
// logic returns without error.
func (im IBCModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

	var data types.FetchPricePacketData
	var ackErr error
	if err := im.cdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		ackErr = errorsmod.Wrapf(sdkerrors.ErrInvalidType, "cannot unmarshal %s packet data", types.Version)
		ack = channeltypes.NewErrorAcknowledgement(ackErr)
	}

	// call application logic
	if ackErr == nil {
		if ackData, err := im.pk.OnRecvPacket(ctx, data); err != nil {
			ackErr = err
			ack = channeltypes.NewErrorAcknowledgement(ackErr)
		} else {
			if ackBz, err := im.cdc.MarshalJSON(ackData); err != nil {
				ackErr = errorsmod.Wrapf(types.ErrFailedToFetchPrice, err.Error())
				ack = channeltypes.NewErrorAcknowledgement(ackErr)
			} else {
				ack = channeltypes.NewResultAcknowledgement(ackBz)
			}
		}
	}

	eventAttributes := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyModule, providertypes.SubModuleName),
		sdk.NewAttribute(types.AttributeKeyCurrencyIds, strings.Join(data.CurrencyIds, ",")),
		sdk.NewAttribute(types.AttributeKeyMemo, data.Memo),
		sdk.NewAttribute(types.AttributeKeyAckSuccess, fmt.Sprintf("%t", ack.Success())),
	}

	if ackErr != nil {
		eventAttributes = append(eventAttributes, sdk.NewAttribute(types.AttributeKeyAckError, ackErr.Error()))
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePacket,
			eventAttributes...,
		),
	)

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return ack
}

// OnAcknowledgementPacket implements the IBCModule interface
func (im IBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	return errorsmod.Wrap(types.ErrInvalidChannelFlow, "channel handshake must be initiated by consumer chain")
}

// OnTimeoutPacket implements the IBCModule interface
func (im IBCModule) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	return nil
}
