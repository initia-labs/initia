package types

import (
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"

	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
)

// msg types
const (
	TypeMsgUpdateChannelRelayer = "update_channel_relayer"
)

var (
	_ sdk.Msg = &MsgUpdateChannelRelayer{}

	_ legacytx.LegacyMsg = &MsgUpdateChannelRelayer{}
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

// Route implements sdk.Msg
func (MsgUpdateChannelRelayer) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (MsgUpdateChannelRelayer) Type() string {
	return TypeMsgUpdateChannelRelayer
}

// ValidateBasic performs a basic check of the MsgUpdateChannelRelayer fields.
// NOTE: timeout height or timestamp values can be 0 to disable the timeout.
// NOTE: The recipient addresses format is not validated as the format defined by
// the chain is not known to IBC.
func (msg MsgUpdateChannelRelayer) ValidateBasic() error {
	if err := host.ChannelIdentifierValidator(msg.Channel); err != nil {
		return errors.Wrap(err, "invalid source port ID")
	}

	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Relayer)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
}

// GetSignBytes implements sdk.Msg.
func (msg MsgUpdateChannelRelayer) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(&msg))
}

// GetSigners implements sdk.Msg
func (msg MsgUpdateChannelRelayer) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Relayer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}
