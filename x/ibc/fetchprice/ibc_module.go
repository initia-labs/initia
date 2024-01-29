package fetchprice

import (
	"fmt"
	"strings"

	"github.com/initia-labs/initia/x/ibc/fetchprice/keeper"
	"github.com/initia-labs/initia/x/ibc/fetchprice/types"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	icqtypes "github.com/cosmos/ibc-apps/modules/async-icq/v8/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// IBCModule implements the ICS26 interface for interchain query host chains
type IBCModule struct {
	keeper keeper.Keeper
}

// NewIBCModule creates a new IBCModule given the associated keeper
func NewIBCModule(k keeper.Keeper) IBCModule {
	return IBCModule{
		keeper: k,
	}
}

// validateFetchpriceChannelParams does validation of a newly created fetchprice channel. A fetchprice
// channel must be UNORDERED, use the correct port (by default 'fetchprice'), and use the current
// supported version.
func validateFetchpriceChannelParams(
	ctx sdk.Context,
	keeper keeper.Keeper,
	order channeltypes.Order,
	portID string,
	channelID string,
	counterparty channeltypes.Counterparty,
) error {
	// ICQ only supports unordered
	if order != channeltypes.UNORDERED {
		return errorsmod.Wrapf(channeltypes.ErrInvalidChannelOrdering, "expected %s channel, got %s", channeltypes.ORDERED, order)
	}

	if counterparty.PortId != icqtypes.PortID {
		return errorsmod.Wrapf(types.ErrInvalidICQPortID, "expected %s, got %s", icqtypes.PortID, counterparty.PortId)
	}

	// Require portID is the portID fetchprice module is bound to
	boundPort, err := keeper.PortID.Get(ctx)
	if err != nil {
		return err
	}
	if boundPort != portID {
		return errorsmod.Wrapf(porttypes.ErrInvalidPort, "invalid port: %s, expected %s", portID, boundPort)
	}

	return nil
}

// OnChanOpenInit implements the IBCMiddleware interface
//
// Interchain Accounts is implemented to act as middleware for connected authentication modules on
// the controller side. The connected modules may not change the controller side portID or
// version. They will be allowed to perform custom logic without changing
// the parameters stored within a channel struct.
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
	if ok, err := im.keeper.GetFetchEnabled(ctx); err != nil {
		return "", err
	} else if !ok {
		return "", types.ErrFetchDisabled
	}

	if err := validateFetchpriceChannelParams(ctx, im.keeper, order, portID, channelID, counterparty); err != nil {
		return "", err
	}

	if strings.TrimSpace(version) == "" {
		version = types.Version
	}

	if version != types.Version {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "got %s, expected %s", version, types.Version)
	}

	if err := im.keeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", err
	}

	return version, nil
}

// OnChanOpenTry implements the IBCMiddleware interface
func (IBCModule) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	return "", errorsmod.Wrap(types.ErrInvalidChannelFlow, "channel handshake must be initiated by fetchprice chain")
}

// OnChanOpenAck implements the IBCMiddleware interface
//
// Interchain Accounts is implemented to act as middleware for connected authentication modules on
// the controller side. The connected modules may not change the portID or
// version. They will be allowed to perform custom logic without changing
// the parameters stored within a channel struct.
func (im IBCModule) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	if ok, err := im.keeper.GetFetchEnabled(ctx); err != nil {
		return err
	} else if !ok {
		return types.ErrFetchDisabled
	}

	if counterpartyVersion != types.Version {
		return errorsmod.Wrapf(types.ErrInvalidVersion, "invalid counterparty version: %s, expected %s", counterpartyVersion, types.Version)
	}

	return nil
}

// OnChanOpenConfirm implements the IBCMiddleware interface
func (IBCModule) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return errorsmod.Wrap(types.ErrInvalidChannelFlow, "channel handshake must be initiated by fetchprice chain")
}

// OnChanCloseInit implements the IBCMiddleware interface
func (IBCModule) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Disallow user-initiated channel closing for interchain account channels
	return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCMiddleware interface
func (im IBCModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return nil
}

// OnRecvPacket implements the IBCMiddleware interface
func (IBCModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	_ sdk.AccAddress,
) ibcexported.Acknowledgement {
	err := errorsmod.Wrapf(types.ErrInvalidChannelFlow, "cannot receive packet on fetchprice chain")
	ack := channeltypes.NewErrorAcknowledgement(err)
	attributes := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(types.AttributeKeyAckSuccess, fmt.Sprintf("%t", ack.Success())),
		sdk.NewAttribute(types.AttributeKeyAckError, err.Error()),
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePacket,
			attributes...,
		),
	)

	return ack
}

// OnAcknowledgementPacket implements the IBCMiddleware interface
func (im IBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	if ok, err := im.keeper.GetFetchEnabled(ctx); err != nil {
		return err
	} else if !ok {
		return types.ErrFetchDisabled
	}

	var ack channeltypes.Acknowledgement
	if err := im.keeper.Codec().UnmarshalJSON(acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrUnknownRequest, "cannot unmarshal fetchprice packet acknowledgement: %v", err)
	}

	switch resp := ack.Response.(type) {
	case *channeltypes.Acknowledgement_Result:
		var icqAck icqtypes.InterchainQueryPacketAck
		err := im.keeper.Codec().UnmarshalJSON(ack.GetResult(), &icqAck)
		if err != nil {
			return err
		}

		err = im.keeper.OnAcknowledgementPacketSuccess(ctx, packet, icqAck)
		if err != nil {
			return err
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypePacket,
				sdk.NewAttribute(types.AttributeKeyAckSuccess, string(resp.Result)),
			),
		)
	case *channeltypes.Acknowledgement_Error:
		err := im.keeper.OnAcknowledgementPacketError(ctx)
		if err != nil {
			return err
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypePacket,
				sdk.NewAttribute(types.AttributeKeyAckError, resp.Error),
			),
		)
	}

	return nil
}

// OnTimeoutPacket implements the IBCMiddleware interface
func (im IBCModule) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	if ok, err := im.keeper.GetFetchEnabled(ctx); err != nil {
		return err
	} else if !ok {
		return types.ErrFetchDisabled
	}

	if err := im.keeper.OnTimeoutPacket(ctx, packet); err != nil {
		return err
	}

	return nil
}
