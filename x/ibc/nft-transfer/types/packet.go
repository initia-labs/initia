package types

import (
	"strings"
	"time"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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

// decode packet data to NonFungibleTokenPacketData
func DecodePacketData(packetData []byte) (NonFungibleTokenPacketData, error) {
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
