package types

import (
	"encoding/json"
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
	if len(nftpd.ClassId) == 0 {
		return errors.Wrap(ErrInvalidClassId, "invalid zero length class id")
	}
	if len(nftpd.TokenIds) == 0 {
		return errors.Wrap(ErrInvalidTokenIds, "invalid zero length token ids")
	}
	if len(nftpd.TokenIds) != len(nftpd.TokenData) || len(nftpd.TokenIds) != len(nftpd.TokenUris) {
		return errors.Wrap(ErrInvalidPacket, "all token infos should have same length")
	}
	for _, tokenId := range nftpd.TokenIds {
		if len(tokenId) == 0 {
			return errors.Wrap(ErrInvalidTokenIds, "invalid zero length token id")
		}
	}
	return ValidatePrefixedClassId(nftpd.ClassId)
}

// GetBytes is a helper for serializing
func (nftpd NonFungibleTokenPacketData) GetBytes(counterPartyPort string) []byte {
	var bz []byte
	var err error
	if isWasmPacket(counterPartyPort) {
		bz, err = json.Marshal(nftpd.ToWasmData())
	} else {
		bz, err = json.Marshal(nftpd)
	}
	if err != nil {
		panic(err)
	}

	return sdk.MustSortJSON(bz)
}

// DecodePacketData decodes packet data to NonFungibleTokenPacketData
func DecodePacketData(packetData []byte, counterPartyPort string) (NonFungibleTokenPacketData, error) {
	decoder := json.NewDecoder(strings.NewReader(string(packetData)))
	decoder.DisallowUnknownFields()

	if isWasmPacket(counterPartyPort) {
		var wasmData NonFungibleTokenPacketDataWasm
		if err := decoder.Decode(&wasmData); err != nil {
			return NonFungibleTokenPacketData{}, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
		}
		return wasmData.ToPacketData(), nil
	}

	var data NonFungibleTokenPacketData
	if err := decoder.Decode(&data); err != nil {
		return NonFungibleTokenPacketData{}, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}

	return data, nil
}

func (wasmData *NonFungibleTokenPacketDataWasm) ToPacketData() NonFungibleTokenPacketData {
	data := NonFungibleTokenPacketData{
		ClassId:   wasmData.ClassId,
		ClassUri:  wasmData.ClassUri,
		ClassData: wasmData.ClassData,
		TokenIds:  wasmData.TokenIds,
		TokenUris: wasmData.TokenUris,
		TokenData: wasmData.TokenData,
		Sender:    wasmData.Sender,
		Receiver:  wasmData.Receiver,
		Memo:      wasmData.Memo,
	}

	return data
}

func (data *NonFungibleTokenPacketData) ToWasmData() NonFungibleTokenPacketDataWasm {
	return NonFungibleTokenPacketDataWasm{
		ClassId:   data.ClassId,
		ClassUri:  data.ClassUri,
		ClassData: data.ClassData,
		TokenIds:  data.TokenIds,
		TokenUris: data.TokenUris,
		TokenData: data.TokenData,
		Sender:    data.Sender,
		Receiver:  data.Receiver,
		Memo:      data.Memo,
	}
}

const wasmPortPrefix = "wasm."

func isWasmPacket(port string) bool {
	return strings.HasPrefix(port, wasmPortPrefix)
}
