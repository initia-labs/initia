package types

import (
	"cosmossdk.io/core/address"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ sdk.Msg = &MsgUpdateParams{}
)

const (
	TypeMsgUpdateParams = "update_params"
)

/* MsgUpdateParams */

// NewMsgUpdateParams returns a new MsgUpdateParams instance
func NewMsgUpdateParams(authority string, params Params) *MsgUpdateParams {
	return &MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}
}

// Validate performs basic MsgUpdateParams message validation.
func (msg MsgUpdateParams) Validate(accCodec address.Codec) error {
	if _, err := accCodec.StringToBytes(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	return msg.Params.Validate()
}

/* MsgFundCommunityPool */
func NewMsgFundCommunityPool(authority string, amount sdk.Coins) *MsgFundCommunityPool {
	return &MsgFundCommunityPool{
		Authority: authority,
		Amount:    amount,
	}
}

// Validate performs basic MsgFundCommunityPool message validation.
func (msg MsgFundCommunityPool) Validate(accCodec address.Codec) error {
	if _, err := accCodec.StringToBytes(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if err := msg.Amount.Validate(); err != nil {
		return sdkerrors.ErrInvalidCoins.Wrapf("amount must be positive: %s", err)
	}

	return nil
}
