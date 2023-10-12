package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
)

// staking message types
const (
	TypeMsgPublish      = "publish"
	TypeMsgExecute      = "execute"
	TypeMsgScript       = "script"
	TypeMsgGovPublish   = "gov_publish"
	TypeMsgGovExecute   = "gov_execute"
	TypeMsgGovScript    = "gov_script"
	TypeMsgWhitelist    = "whitelist"
	TypeMsgDelist       = "delist"
	TypeMsgUpdateParams = "update_params"
)

var (
	_ sdk.Msg = &MsgPublish{}
	_ sdk.Msg = &MsgExecute{}
	_ sdk.Msg = &MsgScript{}
	_ sdk.Msg = &MsgGovPublish{}
	_ sdk.Msg = &MsgGovExecute{}
	_ sdk.Msg = &MsgGovScript{}
	_ sdk.Msg = &MsgWhitelist{}
	_ sdk.Msg = &MsgDelist{}
	_ sdk.Msg = &MsgUpdateParams{}

	_ legacytx.LegacyMsg = &MsgPublish{}
	_ legacytx.LegacyMsg = &MsgExecute{}
	_ legacytx.LegacyMsg = &MsgScript{}
	_ legacytx.LegacyMsg = &MsgGovPublish{}
	_ legacytx.LegacyMsg = &MsgGovExecute{}
	_ legacytx.LegacyMsg = &MsgGovScript{}
	_ legacytx.LegacyMsg = &MsgWhitelist{}
	_ legacytx.LegacyMsg = &MsgDelist{}
	_ legacytx.LegacyMsg = &MsgUpdateParams{}
)

/* MsgPublish */

// Route implements the sdk.Msg interface.
func (msg MsgPublish) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg MsgPublish) Type() string {
	return TypeMsgPublish
}

// ValidateBasic performs basic MsgPublish message validation.
func (msg MsgPublish) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}

	for _, module := range msg.CodeBytes {
		if len(module) > ModuleSizeHardLimit {
			return errors.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"module size over hard limit %d",
				ModuleSizeHardLimit,
			)
		}

		if len(module) == 0 {
			return errors.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"empty module bytes",
			)
		}
	}

	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgPublish) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgPublish) GetSigners() []sdk.AccAddress {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil { // should never happen as valid basic rejects invalid addresses
		panic(err.Error())
	}
	return []sdk.AccAddress{senderAddr}
}

/* MsgExecute */

// Route implements the sdk.Msg interface.
func (msg MsgExecute) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg MsgExecute) Type() string {
	return TypeMsgExecute
}

// ValidateBasic performs basic MsgExecute message validation.
func (msg MsgExecute) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}

	if _, err := AccAddressFromString(msg.ModuleAddress); err != nil {
		return err
	}

	if len(msg.ModuleName) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "empty module name")
	}

	if len(msg.ModuleName) > ModuleNameLengthHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"module name length over hard limit %d",
			ModuleNameLengthHardLimit,
		)
	}

	if len(msg.FunctionName) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "empty function name")
	}

	if len(msg.FunctionName) > FunctionNameLengthHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"function name length over hard limit %d",
			FunctionNameLengthHardLimit,
		)
	}

	if len(msg.TypeArgs) > NumArgumentsHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"number of type argument over hard limit %d",
			NumArgumentsHardLimit,
		)
	}

	if len(msg.TypeArgs) > NumArgumentsHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"number of type argument over hard limit %d",
			NumArgumentsHardLimit,
		)
	}

	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgExecute) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgExecute) GetSigners() []sdk.AccAddress {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil { // should never happen as valid basic rejects invalid addresses
		panic(err.Error())
	}
	return []sdk.AccAddress{senderAddr}
}

/* MsgScript */

// Route implements the sdk.Msg interface.
func (msg MsgScript) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg MsgScript) Type() string {
	return TypeMsgScript
}

// ValidateBasic performs basic MsgScript message validation.
func (msg MsgScript) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}

	if len(msg.CodeBytes) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "empty code bytes")
	}

	if len(msg.CodeBytes) > ModuleSizeHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"module size length over hard limit %d",
			ModuleSizeHardLimit,
		)
	}

	if len(msg.TypeArgs) > NumArgumentsHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"number of type argument over hard limit %d",
			NumArgumentsHardLimit,
		)
	}

	if len(msg.TypeArgs) > NumArgumentsHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"number of type argument over hard limit %d",
			NumArgumentsHardLimit,
		)
	}

	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgScript) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgScript) GetSigners() []sdk.AccAddress {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil { // should never happen as valid basic rejects invalid addresses
		panic(err.Error())
	}
	return []sdk.AccAddress{senderAddr}
}

/* MsgGovPublish */

// Route implements the sdk.Msg interface.
func (msg MsgGovPublish) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg MsgGovPublish) Type() string {
	return TypeMsgGovPublish
}

// ValidateBasic performs basic MsgGovPublish message validation.
func (msg MsgGovPublish) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}

	for _, module := range msg.CodeBytes {
		if len(module) > ModuleSizeHardLimit {
			return errors.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"module size over hard limit %d",
				ModuleSizeHardLimit,
			)
		}
	}

	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgGovPublish) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes, which is the authority.
func (msg MsgGovPublish) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

/* MsgGovExecute */

// Route implements the sdk.Msg interface.
func (msg MsgGovExecute) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg MsgGovExecute) Type() string {
	return TypeMsgGovExecute
}

// ValidateBasic performs basic MsgGovExecute message validation.
func (msg MsgGovExecute) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}

	if _, err := AccAddressFromString(msg.ModuleAddress); err != nil {
		return err
	}

	if len(msg.ModuleName) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "empty module name")
	}

	if len(msg.ModuleName) > ModuleNameLengthHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"module name length over hard limit %d",
			ModuleNameLengthHardLimit,
		)
	}

	if len(msg.FunctionName) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "empty function name")
	}

	if len(msg.FunctionName) > FunctionNameLengthHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"function name length over hard limit %d",
			FunctionNameLengthHardLimit,
		)
	}

	if len(msg.TypeArgs) > NumArgumentsHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"number of type argument over hard limit %d",
			NumArgumentsHardLimit,
		)
	}

	if len(msg.TypeArgs) > NumArgumentsHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"number of type argument over hard limit %d",
			NumArgumentsHardLimit,
		)
	}

	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgGovExecute) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes, which is the authority.
func (msg MsgGovExecute) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

/* MsgGovScript */

// Route implements the sdk.Msg interface.
func (msg MsgGovScript) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg MsgGovScript) Type() string {
	return TypeMsgGovScript
}

// ValidateBasic performs basic MsgGovScript message validation.
func (msg MsgGovScript) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}

	if len(msg.CodeBytes) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "empty code bytes")
	}

	if len(msg.CodeBytes) > ModuleSizeHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"module size length over hard limit %d",
			ModuleSizeHardLimit,
		)
	}

	if len(msg.TypeArgs) > NumArgumentsHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"number of type argument over hard limit %d",
			NumArgumentsHardLimit,
		)
	}

	if len(msg.TypeArgs) > NumArgumentsHardLimit {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"number of type argument over hard limit %d",
			NumArgumentsHardLimit,
		)
	}

	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgGovScript) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes, which is the authority.
func (msg MsgGovScript) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

/* MsgWhitelist */

// Route implements the sdk.Msg interface.
func (msg MsgWhitelist) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg MsgWhitelist) Type() string {
	return TypeMsgWhitelist
}

// ValidateBasic performs basic MsgWhitelist message validation.
func (msg MsgWhitelist) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if _, err := AccAddressFromString(msg.MetadataLP); err != nil {
		return err
	}

	if msg.RewardWeight.IsNegative() || msg.RewardWeight.GT(sdk.OneDec()) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "reward weight must be [0, 1]")
	}

	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgWhitelist) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes, which is the authority.
func (msg MsgWhitelist) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

/* MsgDelist */

// Route implements the sdk.Msg interface.
func (msg MsgDelist) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg MsgDelist) Type() string {
	return TypeMsgDelist
}

// ValidateBasic performs basic MsgDelist message validation.
func (msg MsgDelist) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if _, err := AccAddressFromString(msg.MetadataLP); err != nil {
		return err
	}

	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgDelist) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes, which is the authority.
func (msg MsgDelist) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

/* MsgUpdateParams */

// Route implements the sdk.Msg interface.
func (msg MsgUpdateParams) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg MsgUpdateParams) Type() string {
	return TypeMsgUpdateParams
}

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgUpdateParams) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgUpdateParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// ValidateBasic performs basic MsgUpdateParams message validation.
func (msg MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	return msg.Params.Validate()
}
