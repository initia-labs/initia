package types

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ sdk.Msg                            = &MsgCreateValidator{}
	_ codectypes.UnpackInterfacesMessage = (*MsgCreateValidator)(nil)
	_ sdk.Msg                            = &MsgEditValidator{}
	_ sdk.Msg                            = &MsgDelegate{}
	_ sdk.Msg                            = &MsgUndelegate{}
	_ sdk.Msg                            = &MsgBeginRedelegate{}
	_ sdk.Msg                            = &MsgCancelUnbondingDelegation{}
	_ sdk.Msg                            = &MsgUpdateParams{}
	_ sdk.Msg                            = &MsgRegisterMigration{}
	_ sdk.Msg                            = &MsgMigrateDelegation{}
)

// NewMsgCreateValidator creates a new MsgCreateValidator instance.
// Delegator address and validator address are the same.
func NewMsgCreateValidator(
	valAddr string, pubKey cryptotypes.PubKey, //nolint:interfacer
	selfDelegation sdk.Coins, description Description, commission CommissionRates,
) (*MsgCreateValidator, error) {
	var pkAny *codectypes.Any
	if pubKey != nil {
		var err error
		if pkAny, err = codectypes.NewAnyWithValue(pubKey); err != nil {
			return nil, err
		}
	}
	return &MsgCreateValidator{
		Description:      description,
		ValidatorAddress: valAddr,
		Pubkey:           pkAny,
		Amount:           selfDelegation,
		Commission:       commission,
	}, nil
}

// Validate implements the sdk.Msg interface.
func (msg MsgCreateValidator) Validate(valAddrCodec address.Codec) error {
	if msg.ValidatorAddress == "" {
		return ErrEmptyValidatorAddr
	}

	valAddr, err := valAddrCodec.StringToBytes(msg.ValidatorAddress)
	if err != nil {
		return err
	}
	if len(valAddr) == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "empty validator address")
	}

	if msg.Pubkey == nil {
		return ErrEmptyValidatorPubKey
	}

	if !msg.Amount.IsValid() || msg.Amount.Empty() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "invalid delegation amount")
	}

	if msg.Description == (Description{}) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "empty description")
	}

	if msg.Commission == (CommissionRates{}) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "empty commission")
	}

	if err := msg.Commission.Validate(); err != nil {
		return err
	}

	return nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (msg MsgCreateValidator) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pubKey cryptotypes.PubKey
	return unpacker.UnpackAny(msg.Pubkey, &pubKey)
}

// NewMsgEditValidator creates a new MsgEditValidator instance
//
//nolint:interfacer
func NewMsgEditValidator(valAddr string, description Description, newRate *math.LegacyDec) *MsgEditValidator {
	return &MsgEditValidator{
		Description:      description,
		CommissionRate:   newRate,
		ValidatorAddress: valAddr,
	}
}

// Validate implements the sdk.Msg interface.
func (msg MsgEditValidator) Validate(valAddrCodec address.Codec) error {
	if valAddr, err := valAddrCodec.StringToBytes(msg.ValidatorAddress); err != nil {
		return err
	} else if len(valAddr) == 0 {
		return ErrEmptyValidatorAddr
	}

	if msg.Description == (Description{}) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "empty description")
	}

	if msg.CommissionRate != nil {
		if msg.CommissionRate.GT(math.LegacyOneDec()) || msg.CommissionRate.IsNegative() {
			return errors.Wrap(sdkerrors.ErrInvalidRequest, "commission rate must be between 0 and 1 (inclusive)")
		}
	}

	return nil
}

// NewMsgDelegate creates a new MsgDelegate instance.
//
//nolint:interfacer
func NewMsgDelegate(delAddr, valAddr string, amount sdk.Coins) *MsgDelegate {
	return &MsgDelegate{
		DelegatorAddress: delAddr,
		ValidatorAddress: valAddr,
		Amount:           amount,
	}
}

// Validate implements the sdk.Msg interface.
func (msg MsgDelegate) Validate(accAddrCodec address.Codec, valAddrCodec address.Codec) error {
	if addr, err := accAddrCodec.StringToBytes(msg.DelegatorAddress); err != nil {
		return err
	} else if len(addr) == 0 {
		return ErrEmptyDelegatorAddr
	}

	if addr, err := valAddrCodec.StringToBytes(msg.ValidatorAddress); err != nil {
		return err
	} else if len(addr) == 0 {
		return ErrEmptyValidatorAddr
	}

	if !msg.Amount.IsValid() || msg.Amount.Empty() {
		return errors.Wrap(
			sdkerrors.ErrInvalidRequest,
			"invalid delegation amount",
		)
	}

	return nil
}

// NewMsgBeginRedelegate creates a new MsgBeginRedelegate instance.
//
//nolint:interfacer
func NewMsgBeginRedelegate(
	delAddr, valSrcAddr, valDstAddr string, amount sdk.Coins,
) *MsgBeginRedelegate {
	return &MsgBeginRedelegate{
		DelegatorAddress:    delAddr,
		ValidatorSrcAddress: valSrcAddr,
		ValidatorDstAddress: valDstAddr,
		Amount:              amount,
	}
}

// Validate implements the sdk.Msg interface.
func (msg MsgBeginRedelegate) Validate(accAddrCodec address.Codec, valAddrCodec address.Codec) error {
	if addr, err := accAddrCodec.StringToBytes(msg.DelegatorAddress); err != nil {
		return err
	} else if len(addr) == 0 {
		return ErrEmptyDelegatorAddr
	}

	if addr, err := valAddrCodec.StringToBytes(msg.ValidatorSrcAddress); err != nil {
		return err
	} else if len(addr) == 0 {
		return ErrEmptyValidatorAddr
	}

	if addr, err := valAddrCodec.StringToBytes(msg.ValidatorDstAddress); err != nil {
		return err
	} else if len(addr) == 0 {
		return ErrEmptyValidatorAddr
	}

	if !msg.Amount.IsValid() || msg.Amount.Empty() {
		return errors.Wrap(
			sdkerrors.ErrInvalidRequest,
			"invalid shares amount",
		)
	}

	return nil
}

// NewMsgUndelegate creates a new MsgUndelegate instance.
//
//nolint:interfacer
func NewMsgUndelegate(delAddr, valAddr string, amount sdk.Coins) *MsgUndelegate {
	return &MsgUndelegate{
		DelegatorAddress: delAddr,
		ValidatorAddress: valAddr,
		Amount:           amount,
	}
}

// Validate implements the sdk.Msg interface.
func (msg MsgUndelegate) Validate(accAddrCodec address.Codec, valAddrCodec address.Codec) error {
	if addr, err := accAddrCodec.StringToBytes(msg.DelegatorAddress); err != nil {
		return err
	} else if len(addr) == 0 {
		return ErrEmptyDelegatorAddr
	}

	if addr, err := valAddrCodec.StringToBytes(msg.ValidatorAddress); err != nil {
		return err
	} else if len(addr) == 0 {
		return ErrEmptyValidatorAddr
	}

	if !msg.Amount.IsValid() || msg.Amount.Empty() {
		return errors.Wrap(
			sdkerrors.ErrInvalidRequest,
			"invalid shares amount",
		)
	}

	return nil
}

// NewMsgCancelUnbondingDelegation creates a new MsgCancelUnbondingDelegation instance.
//
//nolint:interfacer
func NewMsgCancelUnbondingDelegation(delAddr, valAddr string, creationHeight int64, amount sdk.Coins) *MsgCancelUnbondingDelegation {
	return &MsgCancelUnbondingDelegation{
		DelegatorAddress: delAddr,
		ValidatorAddress: valAddr,
		Amount:           amount,
		CreationHeight:   creationHeight,
	}
}

// Validate implements the sdk.Msg interface.
func (msg MsgCancelUnbondingDelegation) Validate(accAddrCodec address.Codec, valAddrCodec address.Codec) error {
	if addr, err := accAddrCodec.StringToBytes(msg.DelegatorAddress); err != nil {
		return err
	} else if len(addr) == 0 {
		return ErrEmptyDelegatorAddr
	}

	if addr, err := valAddrCodec.StringToBytes(msg.ValidatorAddress); err != nil {
		return err
	} else if len(addr) == 0 {
		return ErrEmptyValidatorAddr
	}

	if !msg.Amount.IsValid() || msg.Amount.Empty() {
		return errors.Wrap(
			sdkerrors.ErrInvalidRequest,
			"invalid amount",
		)
	}

	if msg.CreationHeight <= 0 {
		return errors.Wrap(
			sdkerrors.ErrInvalidRequest,
			"invalid height",
		)
	}

	return nil
}

/* MsgRegisterMigration */

func (msg MsgRegisterMigration) Validate(accAddrCodec address.Codec, valAddrCodec address.Codec) error {
	if addr, err := accAddrCodec.StringToBytes(msg.Authority); err != nil {
		return err
	} else if len(addr) == 0 {
		return errors.Wrap(err, "invalid authority address")
	}

	if msg.DenomLpFrom == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "lp denom from is empty")
	}

	if msg.DenomLpTo == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "lp denom to is empty")
	}

	if msg.ModuleAddress == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "module address is empty")
	}

	if msg.ModuleName == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "module name is empty")
	}

	if msg.DenomLpFrom == msg.DenomLpTo {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "lp denom from and to must differ")
	}

	return nil
}

/* MsgMigrateDelegation */

func (msg MsgMigrateDelegation) Validate(accAddrCodec address.Codec, valAddrCodec address.Codec) error {
	if addr, err := accAddrCodec.StringToBytes(msg.DelegatorAddress); err != nil {
		return err
	} else if len(addr) == 0 {
		return ErrEmptyDelegatorAddr
	}

	if addr, err := valAddrCodec.StringToBytes(msg.ValidatorAddress); err != nil {
		return err
	} else if len(addr) == 0 {
		return ErrEmptyValidatorAddr
	}

	if msg.DenomLpFrom == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "lp denom from is empty")
	}
	if msg.DenomLpTo == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "lp denom to is empty")
	}
	if msg.DenomLpFrom == msg.DenomLpTo {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "lp denom from and to must differ")
	}

	// this is optional, so we only validate if it's provided
	if msg.NewDelegatorAddress != "" {
		if _, err := accAddrCodec.StringToBytes(msg.NewDelegatorAddress); err != nil {
			return err
		}
	}

	return nil
}

/* MsgUpdateParams */

// Validate executes sanity validation on the provided data
func (msg MsgUpdateParams) Validate(accAddrCodec address.Codec) error {
	if addr, err := accAddrCodec.StringToBytes(msg.Authority); err != nil {
		return err
	} else if len(addr) == 0 {
		return errors.Wrap(err, "invalid authority address")
	}

	return msg.Params.Validate()
}
