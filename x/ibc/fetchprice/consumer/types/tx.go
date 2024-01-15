package types

import (
	"strings"

	"cosmossdk.io/core/address"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	types "github.com/initia-labs/initia/x/ibc/fetchprice/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"

	oracletypes "github.com/skip-mev/slinky/x/oracle/types"
)

var (
	_ sdk.Msg = &MsgFetchPrice{}
)

func NewMsgFetchPrice(
	sourcePort string,
	sourceChannel string,
	currencyIds []string,
	sender string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64,
	memo string,
) *MsgFetchPrice {
	return &MsgFetchPrice{
		SourcePort:       sourcePort,
		SourceChannel:    sourceChannel,
		CurrencyIds:      currencyIds,
		Sender:           sender,
		TimeoutHeight:    timeoutHeight,
		TimeoutTimestamp: timeoutTimestamp,
		Memo:             memo,
	}
}

// Validate performs a basic check of the MsgTransfer fields.
// NOTE: timeout height or timestamp values can be 0 to disable the timeout.
// NOTE: The recipient addresses format is not validated as the format defined by
// the chain is not known to IBC.
func (msg MsgFetchPrice) Validate(ac address.Codec) error {
	if err := host.PortIdentifierValidator(msg.SourcePort); err != nil {
		return errorsmod.Wrap(err, "invalid source port ID")
	}
	if err := host.ChannelIdentifierValidator(msg.SourceChannel); err != nil {
		return errorsmod.Wrap(err, "invalid source channel ID")
	}

	if len(msg.CurrencyIds) == 0 {
		return errorsmod.Wrap(types.ErrInvalidCurrencyId, "invalid zero length currency ids")
	}
	for _, currencyId := range msg.CurrencyIds {
		if strings.TrimSpace(currencyId) == "" {
			return errorsmod.Wrap(types.ErrInvalidCurrencyId, "currency id cannot be blank")
		}

		_, err := oracletypes.CurrencyPairFromString(currencyId)
		if err != nil {
			return errorsmod.Wrap(types.ErrInvalidCurrencyId, err.Error())
		}
	}

	// NOTE: sender format must be validated as it is required by the GetSigners function.
	_, err := ac.StringToBytes(msg.Sender)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
}
