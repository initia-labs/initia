package types

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
)

/* MsgPublish */

// Validate performs basic MsgPublish message validation.
func (msg MsgPublish) Validate(ac address.Codec) error {
	if _, err := ac.StringToBytes(msg.Sender); err != nil {
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

	if msg.UpgradePolicy == UpgradePolicy_UNSPECIFIED {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"upgrade policy not specified",
		)
	}

	return nil
}

// NewMsgExecute creates a new MsgExecute instance.
//
//nolint:interfacer
func NewMsgExecute(
	sender string,
	moduleAddress string,
	moduleName string,
	functionName string,
	typeArgs []string,
	args [][]byte,
) *MsgExecute {
	return &MsgExecute{
		Sender:        sender,
		ModuleAddress: moduleAddress,
		ModuleName:    moduleName,
		FunctionName:  functionName,
		TypeArgs:      typeArgs,
		Args:          args,
	}
}

/* MsgExecute */

// Validate performs basic MsgExecute message validation.
func (msg MsgExecute) Validate(ac address.Codec) error {
	if _, err := ac.StringToBytes(msg.Sender); err != nil {
		return err
	}

	if _, err := AccAddressFromString(ac, msg.ModuleAddress); err != nil {
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

// NewMsgScript creates a new MsgScript instance.
//
//nolint:interfacer
func NewMsgScript(
	sender string,
	codeBytes []byte,
	typeArgs []string,
	args [][]byte,
) *MsgScript {
	return &MsgScript{
		Sender:    sender,
		CodeBytes: codeBytes,
		TypeArgs:  typeArgs,
		Args:      args,
	}
}

/* MsgScript */

// Validate performs basic MsgScript message validation.
func (msg MsgScript) Validate(ac address.Codec) error {
	if _, err := ac.StringToBytes(msg.Sender); err != nil {
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

/* MsgGovPublish */

// Validate performs basic MsgGovPublish message validation.
func (msg MsgGovPublish) Validate(ac address.Codec) error {
	if _, err := ac.StringToBytes(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if _, err := ac.StringToBytes(msg.Sender); err != nil {
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

/* MsgGovExecute */

// Validate performs basic MsgGovExecute message validation.
func (msg MsgGovExecute) Validate(ac address.Codec) error {
	if _, err := ac.StringToBytes(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if _, err := ac.StringToBytes(msg.Sender); err != nil {
		return err
	}

	if _, err := AccAddressFromString(ac, msg.ModuleAddress); err != nil {
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

/* MsgGovScript */

// Validate performs basic MsgGovScript message validation.
func (msg MsgGovScript) Validate(ac address.Codec) error {
	if _, err := ac.StringToBytes(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if _, err := ac.StringToBytes(msg.Sender); err != nil {
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

/* MsgWhitelist */

// Validate performs basic MsgWhitelist message validation.
func (msg MsgWhitelist) Validate(ac address.Codec) error {
	if _, err := ac.StringToBytes(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if _, err := AccAddressFromString(ac, msg.MetadataLP); err != nil {
		return err
	}

	if msg.RewardWeight.IsNegative() || msg.RewardWeight.GT(math.LegacyOneDec()) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "reward weight must be [0, 1]")
	}

	return nil
}

/* MsgDelist */

// Validate performs basic MsgDelist message validation.
func (msg MsgDelist) Validate(ac address.Codec) error {
	if _, err := ac.StringToBytes(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if _, err := AccAddressFromString(ac, msg.MetadataLP); err != nil {
		return err
	}

	return nil
}

/* MsgUpdateParams */

// Validate performs basic MsgUpdateParams message validation.
func (msg MsgUpdateParams) Validate(ac address.Codec) error {
	if _, err := ac.StringToBytes(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	return msg.Params.Validate(ac)
}
