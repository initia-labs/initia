package ibc_middleware

import (
	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	movekeeper "github.com/initia-labs/initia/x/move/keeper"
)

var _ porttypes.Middleware = &IBCMiddleware{}

type IBCMiddleware struct {
	app         porttypes.IBCModule
	ics4Wrapper porttypes.ICS4Wrapper
	moveKeeper  *movekeeper.Keeper
	ac          address.Codec
}

func NewIBCMiddleware(
	app porttypes.IBCModule,
	ics4Wrapper porttypes.ICS4Wrapper,
	moveKeeper *movekeeper.Keeper,
	ac address.Codec,
) IBCMiddleware {
	return IBCMiddleware{
		app:         app,
		ics4Wrapper: ics4Wrapper,
		moveKeeper:  moveKeeper,
		ac:          ac,
	}
}

// OnChanOpenInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, version)
}

// OnChanOpenTry implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// SendPacket implements the ICS4 Wrapper interface
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (sequence uint64, err error) {
	return im.ics4Wrapper.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
) error {
	return im.ics4Wrapper.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	isIcs20, ics20Data := isIcs20Packet(packet.GetData())
	if isIcs20 {
		return im.onRecvIcs20Packet(ctx, packet, relayer, ics20Data)
	}

	isIcs721, ics721Data := isIcs721Packet(packet.GetData())
	if isIcs721 {
		return im.onRecvIcs721Packet(ctx, packet, relayer, ics721Data)
	}

	return im.app.OnRecvPacket(ctx, packet, relayer)
}

// OnAcknowledgementPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	isIcs20, ics20Data := isIcs20Packet(packet.GetData())
	if isIcs20 {
		return im.onAckIcs20Packet(ctx, packet, relayer, acknowledgement, ics20Data)
	}

	isIcs721, ics721Data := isIcs721Packet(packet.GetData())
	if isIcs721 {
		return im.onAckIcs721Packet(ctx, packet, relayer, acknowledgement, ics721Data)
	}

	return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

// OnTimeoutPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	isIcs20, ics20Data := isIcs20Packet(packet.GetData())
	if isIcs20 {
		return im.onTimeoutIcs20Packet(ctx, packet, relayer, ics20Data)
	}

	isIcs721, ics721Data := isIcs721Packet(packet.GetData())
	if isIcs721 {
		return im.onTimeoutIcs721Packet(ctx, packet, relayer, ics721Data)
	}

	return im.app.OnTimeoutPacket(ctx, packet, relayer)
}
