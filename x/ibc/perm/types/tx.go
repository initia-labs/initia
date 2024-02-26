package types

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

var (
	_ sdk.Msg = &MsgSetPermissionedRelayer{}
)

// NewMsgSetPermissionedRelayer creates a new MsgSetPermissionedRelayer instance
func NewMsgSetPermissionedRelayer(
	authority, portID, channelID, relayer string,
) *MsgSetPermissionedRelayer {
	return &MsgSetPermissionedRelayer{
		Authority: authority,
		PortId:    portID,
		ChannelId: channelID,
		Relayer:   relayer,
	}
}

// ValidateBasic performs a basic check of the MsgSetPermissionedRelayer fields.
// NOTE: timeout height or timestamp values can be 0 to disable the timeout.
// NOTE: The recipient addresses format is not validated as the format defined by
// the chain is not known to IBC.
func (msg MsgSetPermissionedRelayer) Validate(ac address.Codec) error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return err
	}

	if err := host.ChannelIdentifierValidator(msg.ChannelId); err != nil {
		return err
	}

	_, err := ac.StringToBytes(msg.Authority)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	_, err = ac.StringToBytes(msg.Relayer)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
}
