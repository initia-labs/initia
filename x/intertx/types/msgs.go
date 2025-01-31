package types

import (
	fmt "fmt"
	"strings"

	"cosmossdk.io/core/address"
	errorsmod "cosmossdk.io/errors"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	proto "github.com/cosmos/gogoproto/proto"
)

var (
	_ sdk.Msg = &MsgRegisterAccount{}
	_ sdk.Msg = &MsgSubmitTx{}

	_ codectypes.UnpackInterfacesMessage = MsgSubmitTx{}
)

// NewMsgRegisterAccount creates a new MsgRegisterAccount instance
func NewMsgRegisterAccount(owner, connectionID, version string) *MsgRegisterAccount {
	return &MsgRegisterAccount{
		Owner:        owner,
		ConnectionId: connectionID,
		Version:      version,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgRegisterAccount) Validate(ac address.Codec) error {
	if strings.TrimSpace(msg.Owner) == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "missing sender address")
	}

	if _, err := ac.StringToBytes(msg.Owner); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "failed to parse address: %s", msg.Owner)
	}

	return nil
}

// NewMsgSubmitTx creates and returns a new MsgSubmitTx instance
func NewMsgSubmitTx(sdkMsg sdk.Msg, connectionID, owner string) (*MsgSubmitTx, error) {
	protoAny, err := PackTxMsgAny(sdkMsg)
	if err != nil {
		return nil, err
	}

	return &MsgSubmitTx{
		ConnectionId: connectionID,
		Owner:        owner,
		Msg:          protoAny,
	}, nil
}

// PackTxMsgAny marshals the sdk.Msg payload to a protobuf Any type
func PackTxMsgAny(sdkMsg sdk.Msg) (*codectypes.Any, error) {
	msg, ok := sdkMsg.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("can't proto marshal %T", sdkMsg)
	}

	protoAny, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return nil, err
	}

	return protoAny, nil
}

// UnpackInterfaces implements codectypes.UnpackInterfacesMessage
func (msg MsgSubmitTx) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var sdkMsg sdk.Msg

	return unpacker.UnpackAny(msg.Msg, &sdkMsg)
}

// GetTxMsg fetches the cached any message
func (msg *MsgSubmitTx) GetTxMsg() sdk.Msg {
	sdkMsg, ok := msg.Msg.GetCachedValue().(sdk.Msg)
	if !ok {
		return nil
	}

	return sdkMsg
}

// ValidateBasic implements sdk.Msg
func (msg MsgSubmitTx) Validate(ac address.Codec) error {
	_, err := ac.StringToBytes(msg.Owner)
	if err != nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "invalid owner address")
	}

	return nil
}
