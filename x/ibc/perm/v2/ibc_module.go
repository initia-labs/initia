package v2

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibcapi "github.com/cosmos/ibc-go/v10/modules/core/api"

	"github.com/initia-labs/initia/x/ibc/perm/keeper"
)

var _ ibcapi.IBCModule = &IBCMiddleware{}

// IBCMiddleware implements the IBC v2 callbacks for the perm middleware given the
// perm keeper and the underlying application.
type IBCMiddleware struct {
	app    ibcapi.IBCModule
	keeper keeper.Keeper
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application
func NewIBCMiddleware(
	app ibcapi.IBCModule,
	k keeper.Keeper,
) IBCMiddleware {
	return IBCMiddleware{
		app:    app,
		keeper: k,
	}
}

// OnSendPacket implements the IBCModule interface
func (im IBCMiddleware) OnSendPacket(
	ctx sdk.Context,
	sourceChannel string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	signer sdk.AccAddress,
) error {
	return im.app.OnSendPacket(ctx, sourceChannel, destinationClient, sequence, payload, signer)
}

// OnRecvPacket implements the IBCModule interface
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	sourceChannel string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) channeltypesv2.RecvPacketResult {
	if ok, err := im.keeper.HasRelayerPermission(ctx, payload.DestinationPort, destinationClient, relayer); err != nil {
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: newEmitErrorAcknowledgement(ctx, err).Acknowledgement(),
		}
	} else if !ok {
		// Raise panic if relayer does not have permission to relay packets on the channel
		// to prevent packets from being relayed without permission.
		panic(fmt.Sprintf("relayer %s does not have permission to relay packets on channel %s", relayer, destinationClient))
	}

	return im.app.OnRecvPacket(ctx, sourceChannel, destinationClient, sequence, payload, relayer)
}

// OnAcknowledgementPacket implements the IBCModule interface
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	sourceChannel string,
	destinationClient string,
	sequence uint64,
	acknowledgement []byte,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	return im.app.OnAcknowledgementPacket(ctx, sourceChannel, destinationClient, sequence, acknowledgement, payload, relayer)
}

// OnTimeoutPacket implements the IBCModule interface
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	sourceChannel string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	return im.app.OnTimeoutPacket(ctx, sourceChannel, destinationClient, sequence, payload, relayer)
}

// newEmitErrorAcknowledgement creates a new error acknowledgement after having emitted an event with the
// details of the error.
func newEmitErrorAcknowledgement(ctx sdk.Context, err error, errorContexts ...string) channeltypesv2.Acknowledgement {
	attributes := make([]sdk.Attribute, len(errorContexts)+1)
	attributes[0] = sdk.NewAttribute("error", err.Error())
	for i, s := range errorContexts {
		attributes[i+1] = sdk.NewAttribute("error-context", s)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"ibc-acknowledgement-error",
			attributes...,
		),
	})

	return channeltypesv2.NewAcknowledgement([]byte(fmt.Sprintf("error: %s", err.Error())))
}
