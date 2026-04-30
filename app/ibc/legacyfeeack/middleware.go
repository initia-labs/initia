package legacyfeeack

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var _ porttypes.IBCModule = IBCMiddleware{}

type IBCMiddleware struct {
	porttypes.IBCModule
}

func NewIBCMiddleware(app porttypes.IBCModule) IBCMiddleware {
	return IBCMiddleware{IBCModule: app}
}

// effectiveAppVersion returns the `app_version` the inner app should see. v10
// transfer's packet-data parser rejects the wrapped version envelope, so on
// fee-wrapped channels we hand it the unwrapped inner version (e.g. "ics20-1").
func effectiveAppVersion(channelVersion string) string {
	if v, ok := UnwrapAppVersion(channelVersion); ok {
		return v
	}

	return channelVersion
}

// OnRecvPacket dispatches to the underlying app and, if the channel was opened
// with a 29-fee wrapped version, rewraps the inner ack as an
// IncentivizedAcknowledgement so the v8 counterparty can decode it.
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	wrapped := IsFeeWrappedVersion(channelVersion)
	inner := im.IBCModule.OnRecvPacket(ctx, effectiveAppVersion(channelVersion), packet, relayer)
	if inner == nil || !wrapped {
		return inner
	}

	return wrappedAck{inner: inner}
}

// OnAcknowledgementPacket peels the fee envelope from acks coming back from a
// v8 counterparty before passing them to the inner app.
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	if IsFeeWrappedVersion(channelVersion) {
		if peeled, ok := TryUnwrapFeeAck(acknowledgement); ok {
			acknowledgement = peeled
		}
	}

	return im.IBCModule.OnAcknowledgementPacket(ctx, effectiveAppVersion(channelVersion), packet, acknowledgement, relayer)
}

// OnTimeoutPacket unwraps the channel version so the inner app's packet-data
// parser accepts it. The 29-fee envelope only wraps acks, not timeout payloads.
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	return im.IBCModule.OnTimeoutPacket(ctx, effectiveAppVersion(channelVersion), packet, relayer)
}

// wrappedAck implements the inner Acknowledgement.
type wrappedAck struct {
	inner ibcexported.Acknowledgement
}

func (w wrappedAck) Success() bool { return w.inner.Success() }

func (w wrappedAck) Acknowledgement() []byte {
	return WrapFeeAck(w.inner.Acknowledgement(), w.inner.Success())
}
