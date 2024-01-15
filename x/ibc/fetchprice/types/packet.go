package types

import (
	"bytes"
	time "time"

	errorsmod "cosmossdk.io/errors"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/jsonpb"
	proto "github.com/cosmos/gogoproto/proto"

	oracletypes "github.com/skip-mev/slinky/x/oracle/types"
)

// MaxMemoCharLength defines the maximum length for the InterchainAccountPacketData memo field
const MaxMemoCharLength = 256

var (
	// DefaultRelativePacketTimeoutHeight is the default packet timeout height (in blocks) relative
	// to the current block height of the counterparty chain provided by the client state. The
	// timeout is disabled when set to 0.
	DefaultRelativePacketTimeoutHeight = "0-1000"

	// DefaultRelativePacketTimeoutTimestamp is the default packet timeout timestamp (in nanoseconds)
	// relative to the current block timestamp of the counterparty chain provided by the client
	// state. The timeout is disabled when set to 0. The default is currently set to a 10 minute
	// timeout.
	DefaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())
)

// GetBytes is a helper for serializing
func (packet FetchPricePacketData) GetBytes() []byte {
	return sdk.MustSortJSON(mustProtoMarshalJSON(&packet))
}

// ValidateBasic performs basic validation of the interchain account packet data.
// The memo may be empty.
func (packet FetchPricePacketData) ValidateBasic() error {
	if len(packet.CurrencyIds) == 0 {
		return errorsmod.Wrap(ErrInvalidOutgoingData, "packet currency ids cannot be empty")
	}

	for _, currencyId := range packet.CurrencyIds {
		if _, err := oracletypes.CurrencyPairFromString(currencyId); err != nil {
			return errorsmod.Wrap(ErrInvalidOutgoingData, err.Error())
		}
	}

	if len(packet.Memo) > MaxMemoCharLength {
		return errorsmod.Wrapf(ErrInvalidOutgoingData, "packet data memo cannot be greater than %d characters", MaxMemoCharLength)
	}

	return nil
}

// mustProtoMarshalJSON provides an auxiliary function to return Proto3 JSON encoded
// bytes of a message.
// NOTE: Copied from https://github.com/cosmos/cosmos-sdk/blob/971c542453e0972ef1dfc5a80159ad5049c7211c/codec/json.go
// and modified in order to allow `EmitDefaults` to be set to false for ics20 packet marshalling.
// This allows for the introduction of the memo field to be backwards compatible.
func mustProtoMarshalJSON(msg proto.Message) []byte {
	anyResolver := codectypes.NewInterfaceRegistry()

	// EmitDefaults is set to false to prevent marshalling of unpopulated fields (memo)
	// OrigName and the anyResovler match the fields the original SDK function would expect
	// in order to minimize changes.

	// OrigName is true since there is no particular reason to use camel case
	// The any resolver is empty, but provided anyways.
	jm := &jsonpb.Marshaler{OrigName: true, EmitDefaults: false, AnyResolver: anyResolver}

	err := codectypes.UnpackInterfaces(msg, codectypes.ProtoJSONPacker{JSONPBMarshaler: jm})
	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	if err := jm.Marshal(buf, msg); err != nil {
		panic(err)
	}

	return buf.Bytes()
}
