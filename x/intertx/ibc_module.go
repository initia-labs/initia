package intertx

import (
	proto "github.com/gogo/protobuf/proto"

	errorsmod "cosmossdk.io/errors"
	"github.com/initia-labs/initia/x/intertx/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ porttypes.IBCModule = IBCModule{}

// IBCModule implements the ICS26 interface for interchain accounts controller chains
type IBCModule struct {
	keeper keeper.Keeper
}

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModule(k keeper.Keeper) IBCModule {
	return IBCModule{
		keeper: k,
	}
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
	return version, nil
}

// OnChanOpenTry implements the IBCModule interface
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
	return "", nil
}

// OnChanOpenAck implements the IBCModule interface
func (im IBCModule) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	return nil
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
	return nil
}

// OnChanCloseConfirm implements the IBCModule interface
func (im IBCModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return nil
}

// OnRecvPacket implements the IBCModule interface
func (im IBCModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	return channeltypes.NewErrorAcknowledgement(errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "cannot receive packet via interchain accounts authentication module"))
}

// OnAcknowledgementPacket implements the IBCModule interface
func (im IBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	var ack channeltypes.Acknowledgement
	if err := channeltypes.SubModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "cannot unmarshal ICS-27 packet acknowledgement: %v", err)
	}

	var txMsgData sdk.TxMsgData
	if err := proto.Unmarshal(ack.GetResult(), &txMsgData); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "cannot unmarshal ICS-27 tx message data: %v", err)
	}

	switch len(txMsgData.Data) {
	case 0:
		for _, msgResp := range txMsgData.GetMsgResponses() {
			im.keeper.Logger(ctx).Info("msg response in ICS-27 packet", "response", msgResp.GoString(), "typeURL", msgResp.GetTypeUrl())
		}
		return nil
	default:
		for _, msgData := range txMsgData.Data {
			response, err := handleMsgData(ctx, msgData)
			if err != nil {
				return err
			}

			im.keeper.Logger(ctx).Info("message response in ICS-27 packet response", "response", response)
		}
		return nil
	}
}

// OnTimeoutPacket implements the IBCModule interface.
func (im IBCModule) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	return nil
}

func handleMsgData(ctx sdk.Context, msgData *sdk.MsgData) (string, error) { //nolint:staticcheck // SA1019: sdk.MsgData is deprecated: Do not use.
	switch msgData.MsgType {
	case sdk.MsgTypeURL(&banktypes.MsgSend{}):
		msgResponse := &banktypes.MsgSendResponse{}
		if err := proto.Unmarshal(msgData.Data, msgResponse); err != nil {
			return "", errorsmod.Wrapf(sdkerrors.ErrJSONUnmarshal, "cannot unmarshal send response message: %s", err.Error())
		}

		return msgResponse.String(), nil
	case sdk.MsgTypeURL(&stakingtypes.MsgDelegate{}):
		msgResponse := &stakingtypes.MsgDelegateResponse{}
		if err := proto.Unmarshal(msgData.Data, msgResponse); err != nil {
			return "", errorsmod.Wrapf(sdkerrors.ErrJSONUnmarshal, "cannot unmarshal delegate response message: %s", err.Error())
		}

		return msgResponse.String(), nil
	default:
		return "", nil
	}
}
