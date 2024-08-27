package types

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

var (
	_ sdk.Msg = &MsgSetPermissionedRelayers{}
)

// NewMsgSetPermissionedRelayer creates a new MsgSetPermissionedRelayer instance
func NewMsgSetPermissionedRelayers(
	authority, portID, channelID string, relayers []string,
) *MsgSetPermissionedRelayers {
	return &MsgSetPermissionedRelayers{
		Authority: authority,
		ChannelId: channelID,
		PortId:    portID,
		Relayers:  relayers,
	}
}

// ValidateBasic performs a basic check of the MsgSetPermissionedRelayer fields.
func (msg MsgSetPermissionedRelayers) Validate(ac address.Codec) error {
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

func NewMsgHaltChannel(
	authority, portID, channelID string,
) *MsgHaltChannel {
	return &MsgHaltChannel{
		Authority: authority,
		ChannelId: channelID,
		PortId:    portID,
	}
}

// ValidateBasic performs a basic check of the MsgHaltChannel fields.
func (msg MsgHaltChannel) Validate(ac address.Codec) error {
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

	return nil
}

func NewMsgResumeChannel(
	authority, portID, channelID string,
) *MsgResumeChannel {
	return &MsgResumeChannel{
		Authority: authority,
		ChannelId: channelID,
		PortId:    portID,
	}
}

// ValidateBasic performs a basic check of the MsgResumeChannel fields.
func (msg MsgResumeChannel) Validate(ac address.Codec) error {
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

	return nil
}
