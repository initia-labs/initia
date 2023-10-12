package types

import (
	"encoding/hex"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	vmtypes "github.com/initia-labs/initiavm/types"
)

var (
	ClassTraceClassIdPrefixIBC  = "ibc/"
	ClassTraceClassIdPrefixMove = "move/"
)

const (
	MaxNftCollectionNameLength   = 256
	MaxNftCollectionSymbolLength = 256
	MaxSftCollectionNameLength   = 256
	MaxSftCollectionSymbolLength = 256
)

type CollectionKeeper interface {
	CollectionInfo(sdk.Context, vmtypes.AccountAddress) (
		creator vmtypes.AccountAddress,
		name string,
		uri string,
		data string,
		err error,
	)
}

func CollectionAddressFromClassId(classId string) (vmtypes.AccountAddress, error) {
	if strings.HasPrefix(classId, ClassTraceClassIdPrefixMove) {
		addrBz, err := hex.DecodeString(strings.TrimPrefix(classId, DenomTraceDenomPrefixMove))
		if err != nil {
			return vmtypes.AccountAddress{}, err
		}

		return vmtypes.NewAccountAddressFromBytes(addrBz)
	}

	// currently no other case exists, non move coins are generated
	// from 0x1.
	return NamedObjectAddress(vmtypes.StdAddress, classId), nil
}

func ClassIdFromCollectionAddress(ctx sdk.Context, k CollectionKeeper, collection vmtypes.AccountAddress) (string, error) {
	creator, name, _, _, err := k.CollectionInfo(ctx, collection)
	if err != nil {
		return "", err
	}

	// If a nft is not ibc, add `move/` prefix
	if creator != vmtypes.StdAddress {
		return DenomTraceDenomPrefixMove + collection.CanonicalString(), nil
	}

	// Else name == classId
	return name, err
}

func IsMoveClassId(classId string) bool {
	return strings.HasPrefix(classId, ClassTraceClassIdPrefixMove)
}

func TokenAddressFromTokenId(collectionCreator vmtypes.AccountAddress, collectionName, tokenId string) (vmtypes.AccountAddress, error) {
	// If a nft is not ibc, then token id == token address
	if collectionCreator != vmtypes.StdAddress {
		return vmtypes.NewAccountAddress(tokenId)
	}

	// Else, collection name + "::" + token id is a seed for object address
	return NamedObjectAddress(vmtypes.StdAddress, collectionName+"::"+tokenId), nil
}
