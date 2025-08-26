package upgrade

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// Interface assertions to ensure IBCMiddleware implements required interfaces
var _ porttypes.Middleware = &IBCMiddleware{}
var _ porttypes.UpgradableModule = &IBCMiddleware{}

// IBCMiddleware wraps an underlying IBC module and provides channel upgrade functionality by delegating upgrade callbacks to the rootModule.
// The app field handles normal IBC callbacks while rootModule specifically handles upgrade-related callbacks.
// The ics4Wrapper provides packet sending/receiving capabilities.
//
// This middleware is necessary because many custom ibc middlewares did not implement porttypes.UpgradableModule.
// It acts as a compatibility layer that ensures upgrade functionality is available even when the underlying
// IBC module doesn't support it directly.
type IBCMiddleware struct {
	// app is the underlying IBC module that handles standard IBC operations
	app porttypes.IBCModule
	// ics4Wrapper provides packet sending/receiving capabilities for the middleware
	ics4Wrapper porttypes.ICS4Wrapper
	// rootModule is the top-level IBC module that handles upgrade-related callbacks
	rootModule porttypes.IBCModule
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application
//
// Parameters:
//   - app: The underlying IBC module that handles standard IBC operations
//   - ics4Wrapper: Provides packet sending/receiving capabilities
//   - rootModule: The top-level IBC module that handles upgrade-related callbacks
//
// Returns:
//   - IBCMiddleware: A configured middleware instance
func NewIBCMiddleware(
	app porttypes.IBCModule,
	ics4Wrapper porttypes.ICS4Wrapper,
	rootModule porttypes.IBCModule,
) IBCMiddleware {
	return IBCMiddleware{
		app:         app,
		ics4Wrapper: ics4Wrapper,
		rootModule:  rootModule,
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

// OnAcknowledgementPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

// OnTimeoutPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	return im.app.OnTimeoutPacket(ctx, packet, relayer)
}

// OnRecvPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	return im.app.OnRecvPacket(ctx, packet, relayer)
}

// SendPacket implements the ICS4 Wrapper interface
// Rate-limited SendPacket found in RateLimit Keeper
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (sequence uint64, err error) {
	return im.ics4Wrapper.SendPacket(
		ctx,
		chanCap,
		sourcePort,
		sourceChannel,
		timeoutHeight,
		timeoutTimestamp,
		data,
	)
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

// GetAppVersion returns the application version of the underlying application
func (i IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return i.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// OnChanUpgradeInit implements types.UpgradableModule.
func (im IBCMiddleware) OnChanUpgradeInit(ctx sdk.Context, portID string, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) (string, error) {
	cbs, ok := im.rootModule.(porttypes.UpgradableModule)
	if !ok {
		return proposedVersion, errorsmod.Wrap(porttypes.ErrInvalidRoute, "upgrade route not found to module in application callstack")
	}

	return cbs.OnChanUpgradeInit(ctx, portID, channelID, proposedOrder, proposedConnectionHops, proposedVersion)
}

// OnChanUpgradeTry implements types.UpgradableModule.
func (im IBCMiddleware) OnChanUpgradeTry(ctx sdk.Context, portID string, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, counterpartyVersion string) (string, error) {
	cbs, ok := im.rootModule.(porttypes.UpgradableModule)
	if !ok {
		return counterpartyVersion, errorsmod.Wrap(porttypes.ErrInvalidRoute, "upgrade route not found to module in application callstack")
	}

	return cbs.OnChanUpgradeTry(ctx, portID, channelID, proposedOrder, proposedConnectionHops, counterpartyVersion)
}

// OnChanUpgradeAck implements types.UpgradableModule.
func (im IBCMiddleware) OnChanUpgradeAck(ctx sdk.Context, portID string, channelID string, counterpartyVersion string) error {
	cbs, ok := im.rootModule.(porttypes.UpgradableModule)
	if !ok {
		return errorsmod.Wrap(porttypes.ErrInvalidRoute, "upgrade route not found to module in application callstack")
	}

	return cbs.OnChanUpgradeAck(ctx, portID, channelID, counterpartyVersion)
}

// OnChanUpgradeOpen implements types.UpgradableModule.
func (im IBCMiddleware) OnChanUpgradeOpen(ctx sdk.Context, portID string, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) {
	cbs, ok := im.rootModule.(porttypes.UpgradableModule)
	if !ok {
		panic(errorsmod.Wrap(porttypes.ErrInvalidRoute, "upgrade route not found to module in application callstack"))
	}

	cbs.OnChanUpgradeOpen(ctx, portID, channelID, proposedOrder, proposedConnectionHops, proposedVersion)
}
