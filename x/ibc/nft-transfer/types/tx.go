package types

import (
	"strings"

	"cosmossdk.io/core/address"
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

var (
	_ sdk.Msg = &MsgTransfer{}
	_ sdk.Msg = &MsgUpdateParams{}
)

// NewMsgTransfer creates a new MsgTransfer instance
func NewMsgTransfer(
	sourcePort, sourceChannel string,
	classId string, tokenIds []string,
	sender, receiver string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64,
	memo string,
) *MsgTransfer {
	return &MsgTransfer{
		SourcePort:       sourcePort,
		SourceChannel:    sourceChannel,
		ClassId:          classId,
		TokenIds:         tokenIds,
		Sender:           sender,
		Receiver:         receiver,
		TimeoutHeight:    timeoutHeight,
		TimeoutTimestamp: timeoutTimestamp,
		Memo:             memo,
	}
}

// Validate performs a basic check of the MsgTransfer fields.
// NOTE: timeout height or timestamp values can be 0 to disable the timeout.
// NOTE: The recipient addresses format is not validated as the format defined by
// the chain is not known to IBC.
func (msg MsgTransfer) Validate(ac address.Codec) error {
	if err := host.PortIdentifierValidator(msg.SourcePort); err != nil {
		return errors.Wrap(err, "invalid source port ID")
	}
	if err := host.ChannelIdentifierValidator(msg.SourceChannel); err != nil {
		return errors.Wrap(err, "invalid source channel ID")
	}
	if strings.TrimSpace(msg.ClassId) == "" {
		return errors.Wrap(ErrInvalidClassId, "invalid zero length class id")
	}
	if len(msg.TokenIds) == 0 {
		return errors.Wrap(ErrInvalidTokenIds, "invalid zero length token ids")
	}
	for _, tokenId := range msg.TokenIds {
		if strings.TrimSpace(tokenId) == "" {
			return errors.Wrap(ErrInvalidTokenIds, "tokenId cannot be blank")
		}
	}

	// NOTE: sender format must be validated as it is required by the GetSigners function.
	_, err := ac.StringToBytes(msg.Sender)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	if strings.TrimSpace(msg.Receiver) == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "missing recipient address")
	}
	return ValidateIBCClassId(msg.ClassId)
}

/* MsgUpdateParams */

// Validate performs basic MsgUpdateParams message validation.
func (msg MsgUpdateParams) Validate(ac address.Codec) error {
	if _, err := ac.StringToBytes(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	return msg.Params.Validate()
}
