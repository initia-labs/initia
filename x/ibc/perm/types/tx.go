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

// NewMsgSetPermissionedRelayers creates a new MsgSetPermissionedRelayer instance
func NewMsgSetPermissionedRelayers(
	authority, portID, channelID string, relayers []string,
) *MsgSetPermissionedRelayers {
	return &MsgSetPermissionedRelayers{
		Authority: authority,
		PortId:    portID,
		ChannelId: channelID,
		Relayers:  relayers,
	}
}

// Validate performs a basic check of the MsgSetPermissionedRelayer fields.
// NOTE: timeout height or timestamp values can be 0 to disable the timeout.
// NOTE: The recipient addresses format is not validated as the format defined by
// the chain is not known to IBC.
func (msg MsgSetPermissionedRelayers) Validate(ac address.Codec) error {
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

	for _, relayer := range msg.Relayers {
		_, err = ac.StringToBytes(relayer)
		if err != nil {
			return errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
		}
	}

	return nil
}
