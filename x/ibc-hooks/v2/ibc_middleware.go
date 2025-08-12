package v2

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibcapi "github.com/cosmos/ibc-go/v10/modules/core/api"

	"github.com/initia-labs/initia/x/ibc-hooks/keeper"
)

var _ ibcapi.IBCModule = &IBCMiddleware{}

// IBCMiddleware wraps the IBC v1 middleware to be compatible with IBC v2
// It provides a bridge between v1 and v2 IBC modules
type IBCMiddleware struct {
	App             ibcapi.IBCModule
	writeAckWrapper ibcapi.WriteAcknowledgementWrapper
	HooksKeeper     *keeper.Keeper
}

// NewIBCMiddleware creates a new IBC middleware
func NewIBCMiddleware(
	app ibcapi.IBCModule,
	writeAckWrapper ibcapi.WriteAcknowledgementWrapper,
	hooksKeeper *keeper.Keeper,
) IBCMiddleware {
	return IBCMiddleware{
		App:             app,
		writeAckWrapper: writeAckWrapper,
		HooksKeeper:     hooksKeeper,
	}
}

// OnSendPacket implements the IBCModule interface
func (im *IBCMiddleware) OnSendPacket(ctx sdk.Context, sourceClient string, destinationClient string, sequence uint64, payload channeltypesv2.Payload, signer sdk.AccAddress) error {
	err := im.App.OnSendPacket(ctx, sourceClient, destinationClient, sequence, payload, signer)
	return err
}

// OnRecvPacket implements the IBCModule interface
func (im *IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) channeltypesv2.RecvPacketResult {
	return im.App.OnRecvPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
}

// OnTimeoutPacket implements the IBCModule interface
func (im *IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	return im.App.OnTimeoutPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
}

// OnAcknowledgementPacket implements the IBCModule interface
func (im *IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	acknowledgement []byte,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	return im.App.OnAcknowledgementPacket(ctx, sourceClient, destinationClient, sequence, acknowledgement, payload, relayer)
}

func (im *IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	srcClientID string,
	sequence uint64,
	ack channeltypesv2.Acknowledgement,
) error {
	return im.writeAckWrapper.WriteAcknowledgement(ctx, srcClientID, sequence, ack)
}
