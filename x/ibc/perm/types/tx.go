package types

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

var (
	_ sdk.Msg = &MsgUpdateAdmin{}
	_ sdk.Msg = &MsgUpdatePermissionedRelayers{}
)

// NewMsgUpdateAdmin creates a new MsgUpdateAdmin instance
func NewMsgUpdateAdmin(
	authority, portID, channelID string, admin string,
) *MsgUpdateAdmin {
	return &MsgUpdateAdmin{
		Authority: authority,
		ChannelId: channelID,
		PortId:    portID,
		Admin:     admin,
	}
}

// ValidateBasic performs a basic check of the MsgUpdateAdmin fields.
func (msg MsgUpdateAdmin) Validate(ac address.Codec) error {
	if err := host.ChannelIdentifierValidator(msg.ChannelId); err != nil {
		return err
	}

	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return err
	}

	_, err := ac.StringToBytes(msg.Authority)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	_, err = ac.StringToBytes(msg.Admin)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
}

// NewMsgSetPermissionedRelayer creates a new MsgSetPermissionedRelayer instance
func NewMsgUpdatePermissionedRelayers(
	authority, portID, channelID string, relayers []string,
) *MsgUpdatePermissionedRelayers {
	return &MsgUpdatePermissionedRelayers{
		Authority: authority,
		ChannelId: channelID,
		PortId:    portID,
		Relayers:  relayers,
	}
}

// ValidateBasic performs a basic check of the MsgSetPermissionedRelayer fields.
func (msg MsgUpdatePermissionedRelayers) Validate(ac address.Codec) error {
	if err := host.ChannelIdentifierValidator(msg.ChannelId); err != nil {
		return err
	}

	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return err
	}

	_, err := ac.StringToBytes(msg.Authority)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	for _, relayer := range msg.Relayers {
		_, err = ac.StringToBytes(relayer)
		if err != nil {
			return errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
		}
	}

	return nil
}
