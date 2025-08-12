package types

import (
	"encoding/json"
	"strings"
	"time"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec/unknownproto"
	"github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

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

	_ ibcexported.PacketData         = (*NonFungibleTokenPacketData)(nil)
	_ ibcexported.PacketDataProvider = (*NonFungibleTokenPacketData)(nil)
)

const (
	EncodingJSON     = "application/json"
	EncodingProtobuf = "application/x-protobuf"
	EncodingABI      = "application/x-solidity-abi"
)

// NewNonFungibleTokenPacketData constructs a new NonFungibleTokenPacketData instance
func NewNonFungibleTokenPacketData(
	classId, classUri, classData string,
	tokenIds, tokenUris, tokenData []string,
	sender, receiver, memo string,
) NonFungibleTokenPacketData {
	return NonFungibleTokenPacketData{
		ClassId:   classId,
		ClassUri:  classUri,
		ClassData: classData,
		TokenIds:  tokenIds,
		TokenUris: tokenUris,
		TokenData: tokenData,
		Sender:    sender,
		Receiver:  receiver,
		Memo:      memo,
	}
}

// ValidateBasic is used for validating the token transfer.
// NOTE: The addresses formats are not validated as the sender and recipient can have different
// formats defined by their corresponding chains that are not known to IBC.
func (nftpd NonFungibleTokenPacketData) ValidateBasic() error {
	if strings.TrimSpace(nftpd.Sender) == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "sender address cannot be blank")
	}
	if strings.TrimSpace(nftpd.Receiver) == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "receiver address cannot be blank")
	}
	if strings.TrimSpace(nftpd.ClassId) == "" {
		return errors.Wrap(ErrInvalidClassId, "invalid zero length class id")
	}
	if len(nftpd.TokenIds) == 0 {
		return errors.Wrap(ErrInvalidTokenIds, "invalid zero length token ids")
	}
	if len(nftpd.TokenIds) != len(nftpd.TokenUris) {
		return errors.Wrap(ErrInvalidPacket, "the length of tokenUri must be 0 or the same as the length of TokenIds")
	}
	if len(nftpd.TokenIds) != len(nftpd.TokenData) {
		return errors.Wrap(ErrInvalidPacket, "the length of tokenData must be 0 or the same as the length of TokenIds")
	}
	seenTokens := make(map[string]struct{})
	for _, tokenId := range nftpd.TokenIds {
		if strings.TrimSpace(tokenId) == "" {
			return errors.Wrap(ErrInvalidTokenIds, "invalid zero length token id")
		}
		// check duplicate
		if _, exists := seenTokens[tokenId]; exists {
			return errors.Wrapf(ErrInvalidTokenIds, "duplicate token id: %s", tokenId)
		}
		seenTokens[tokenId] = struct{}{}
	}
	return ValidatePrefixedClassId(nftpd.ClassId)
}

// GetBytes is a helper for serializing
func (nftpd NonFungibleTokenPacketData) GetBytes() []byte {
	// Format will reshape tokenUris and tokenData in NonFungibleTokenPacketData:
	// 1. if tokenUris/tokenData is ["","",""] or [], then set it to nil.
	// 2. if tokenUris/tokenData is ["a","b","c"] or ["a", "", "c"], then keep it.
	// NOTE: Only use this before sending pkg.
	if requireShape(nftpd.TokenUris) {
		nftpd.TokenUris = nil
	}

	if requireShape(nftpd.TokenData) {
		nftpd.TokenData = nil
	}

	return sdk.MustSortJSON(mustProtoMarshalJSON(&nftpd))
}

// requireShape checks if TokenUris/TokenData needs to be set as nil
func requireShape(contents []string) bool {
	if contents == nil {
		return false
	}

	// empty slice of string
	if len(contents) == 0 {
		return true
	}

	emptyStringCount := 0
	for _, v := range contents {
		if len(v) == 0 {
			emptyStringCount++
		}
	}
	return emptyStringCount == len(contents)
}

// GetCustomPacketData interprets the memo field of the packet data as a JSON object
// and returns the value associated with the given key.
// If the key is missing or the memo is not properly formatted, then nil is returned.
func (nftpd NonFungibleTokenPacketData) GetCustomPacketData(key string) any {
	if len(nftpd.Memo) == 0 {
		return nil
	}

	jsonObject := make(map[string]any)
	err := json.Unmarshal([]byte(nftpd.Memo), &jsonObject)
	if err != nil {
		return nil
	}

	memoData, found := jsonObject[key]
	if !found {
		return nil
	}

	return memoData
}

// decode packet data to NonFungibleTokenPacketData
func DecodePacketData(packetData []byte, channelVersion string) (NonFungibleTokenPacketData, error) {
	if channelVersion != V1 {
		return NonFungibleTokenPacketData{}, errors.Wrapf(ErrInvalidVersion, "invalid channel version: expected %s, got %s", V1, channelVersion)
	}

	var data NonFungibleTokenPacketData
	if err := unmarshalProtoJSON(packetData, &data); err != nil {
		return NonFungibleTokenPacketData{}, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}

	// reshape tokenUris and tokenData from nil to ["","",""]
	if len(data.TokenUris) == 0 {
		data.TokenUris = make([]string, len(data.TokenIds))
	}
	if len(data.TokenData) == 0 {
		data.TokenData = make([]string, len(data.TokenIds))
	}

	return data, nil
}

// MarshalPacketData attempts to marshal the provided NonFungibleTokenPacketData into bytes with the provided encoding.
func MarshalPacketData(data NonFungibleTokenPacketData, ics721Version string, encoding string) ([]byte, error) {
	if ics721Version != V1 {
		return nil, errors.Wrapf(ErrInvalidVersion, "unsupported ics721 version: %s", ics721Version)
	}

	switch encoding {
	case EncodingJSON:
		return data.GetBytes(), nil
	case EncodingProtobuf:
		return proto.Marshal(&data)
	case EncodingABI:
		return EncodeABINonFungibleTokenPacketData(&data)
	default:
		return nil, errors.Wrapf(ibcerrors.ErrInvalidType, "invalid encoding provided, must be either empty or one of [%q, %q, %q], got %s", EncodingJSON, EncodingProtobuf, EncodingABI, encoding)
	}
}

// UnmarshalPacketData attempts to unmarshal the provided packet data bytes into a NonFungibleTokenPacketData.
func UnmarshalPacketData(bz []byte, ics721Version string, encoding string) (NonFungibleTokenPacketData, error) {
	const failedUnmarshalingErrorMsg = "cannot unmarshal %s transfer packet data: %s"

	var data proto.Message
	switch ics721Version {
	case V1:
		if encoding == "" {
			encoding = EncodingJSON
		}
		data = &NonFungibleTokenPacketData{}
	default:
		return NonFungibleTokenPacketData{}, errors.Wrapf(ErrInvalidVersion, "unsupported ics721 version: %s", ics721Version)
	}

	errorMsgVersion := "ICS721-V1"

	switch encoding {
	case EncodingJSON:
		if err := unmarshalProtoJSON(bz, data); err != nil {
			return NonFungibleTokenPacketData{}, errors.Wrapf(ibcerrors.ErrInvalidType, failedUnmarshalingErrorMsg, errorMsgVersion, err.Error())
		}
	case EncodingProtobuf:
		if err := unknownproto.RejectUnknownFieldsStrict(bz, data, unknownproto.DefaultAnyResolver{}); err != nil {
			return NonFungibleTokenPacketData{}, errors.Wrapf(ibcerrors.ErrInvalidType, failedUnmarshalingErrorMsg, errorMsgVersion, err.Error())
		}

		if err := proto.Unmarshal(bz, data); err != nil {
			return NonFungibleTokenPacketData{}, errors.Wrapf(ibcerrors.ErrInvalidType, failedUnmarshalingErrorMsg, errorMsgVersion, err.Error())
		}
	case EncodingABI:
		if ics721Version != V1 {
			return NonFungibleTokenPacketData{}, errors.Wrapf(ibcerrors.ErrInvalidType, "encoding %s is only supported for ICS721-V1", EncodingABI)
		}
		var err error
		data, err = DecodeABINonFungibleTokenPacketData(bz)
		if err != nil {
			return NonFungibleTokenPacketData{}, errors.Wrapf(ibcerrors.ErrInvalidType, failedUnmarshalingErrorMsg, errorMsgVersion, err.Error())
		}
	default:
		return NonFungibleTokenPacketData{}, errors.Wrapf(ibcerrors.ErrInvalidType, "invalid encoding provided, must be either empty or one of [%q, %q, %q], got %s", EncodingJSON, EncodingProtobuf, EncodingABI, encoding)
	}

	datav1, ok := data.(*NonFungibleTokenPacketData)
	if !ok {
		return NonFungibleTokenPacketData{}, errors.Wrapf(ibcerrors.ErrInvalidType, "cannot convert proto message into NonFungibleTokenPacketData")
	}

	// reshape tokenUris and tokenData from nil to ["","",""]
	if len(datav1.TokenUris) == 0 {
		datav1.TokenUris = make([]string, len(datav1.TokenIds))
	}
	if len(datav1.TokenData) == 0 {
		datav1.TokenData = make([]string, len(datav1.TokenIds))
	}

	// Validate the packet data
	if err := datav1.ValidateBasic(); err != nil {
		return NonFungibleTokenPacketData{}, errors.Wrapf(err, "invalid packet data")
	}

	return *datav1, nil
}
