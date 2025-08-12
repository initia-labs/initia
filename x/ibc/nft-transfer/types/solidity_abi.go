package types

import (
	"github.com/ethereum/go-ethereum/accounts/abi"

	errorsmod "cosmossdk.io/errors"
)

// getICS721ABI returns an abi.Arguments slice describing the Solidity types of the struct.
func getICS721ABI() abi.Arguments {
	// Create the ABI types for each field.
	// The Solidity types used are:
	// - string for ClassId, ClassUri, ClassData, Sender, Receiver and Memo
	// - string[] for TokenIds, TokenUris, TokenData
	tupleType, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{
			Name: "classId",
			Type: "string",
		},
		{
			Name: "classUri",
			Type: "string",
		},
		{
			Name: "classData",
			Type: "string",
		},
		{
			Name: "tokenIds",
			Type: "string[]",
		},
		{
			Name: "tokenUris",
			Type: "string[]",
		},
		{
			Name: "tokenData",
			Type: "string[]",
		},
		{
			Name: "sender",
			Type: "string",
		},
		{
			Name: "receiver",
			Type: "string",
		},
		{
			Name: "memo",
			Type: "string",
		},
	})
	if err != nil {
		panic(err)
	}

	// Create an ABI argument representing our struct as a single tuple argument.
	arguments := abi.Arguments{
		{
			Type: tupleType,
		},
	}

	return arguments
}

// DecodeABINonFungibleTokenPacketData decodes a solidity ABI encoded ICS721 packet data
// and converts it into an ibc-go NonFungibleTokenPacketData.
func DecodeABINonFungibleTokenPacketData(data []byte) (*NonFungibleTokenPacketData, error) {
	arguments := getICS721ABI()

	packetDataI, err := arguments.Unpack(data)
	if err != nil {
		return nil, errorsmod.Wrapf(ErrAbiDecoding, "failed to unpack data: %s", err)
	}

	packetData, ok := packetDataI[0].(struct {
		ClassId   string   `json:"classId"`
		ClassUri  string   `json:"classUri"`
		ClassData string   `json:"classData"`
		TokenIds  []string `json:"tokenIds"`
		TokenUris []string `json:"tokenUris"`
		TokenData []string `json:"tokenData"`
		Sender    string   `json:"sender"`
		Receiver  string   `json:"receiver"`
		Memo      string   `json:"memo"`
	})
	if !ok {
		return nil, errorsmod.Wrapf(ErrAbiDecoding, "failed to parse packet data")
	}

	return &NonFungibleTokenPacketData{
		ClassId:   packetData.ClassId,
		ClassUri:  packetData.ClassUri,
		ClassData: packetData.ClassData,
		TokenIds:  packetData.TokenIds,
		TokenUris: packetData.TokenUris,
		TokenData: packetData.TokenData,
		Sender:    packetData.Sender,
		Receiver:  packetData.Receiver,
		Memo:      packetData.Memo,
	}, nil
}

// EncodeABINonFungibleTokenPacketData encodes NonFungibleTokenPacketData into solidity ABI format
func EncodeABINonFungibleTokenPacketData(data *NonFungibleTokenPacketData) ([]byte, error) {
	packetData := struct {
		ClassId   string   `json:"classId"`
		ClassUri  string   `json:"classUri"`
		ClassData string   `json:"classData"`
		TokenIds  []string `json:"tokenIds"`
		TokenUris []string `json:"tokenUris"`
		TokenData []string `json:"tokenData"`
		Sender    string   `json:"sender"`
		Receiver  string   `json:"receiver"`
		Memo      string   `json:"memo"`
	}{
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

	arguments := getICS721ABI()
	// Pack the values in the order defined in the ABI.
	encodedData, err := arguments.Pack(packetData)
	if err != nil {
		return nil, errorsmod.Wrapf(ErrAbiEncoding, "failed to pack data: %s", err)
	}

	return encodedData, nil
}