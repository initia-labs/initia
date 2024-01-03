package types

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

var (
	_ sdk.Msg = &MsgUpdateChannelRelayer{}
)

// NewMsgUpdateChannelRelayer creates a new MsgUpdateChannelRelayer instance
func NewMsgUpdateChannelRelayer(
	authority, channel, relayer string,
) *MsgUpdateChannelRelayer {
	return &MsgUpdateChannelRelayer{
		Authority: authority,
		Channel:   channel,
		Relayer:   relayer,
	}
}

// ValidateBasic performs a basic check of the MsgUpdateChannelRelayer fields.
// NOTE: timeout height or timestamp values can be 0 to disable the timeout.
// NOTE: The recipient addresses format is not validated as the format defined by
// the chain is not known to IBC.
func (msg MsgUpdateChannelRelayer) Validate(ac address.Codec) error {
	if err := host.ChannelIdentifierValidator(msg.Channel); err != nil {
		return errors.Wrap(err, "invalid source port ID")
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
