package types

import (
	context "context"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/crypto/sha3"

	sdk "github.com/cosmos/cosmos-sdk/types"

	vmtypes "github.com/initia-labs/movevm/types"
)

type FungibleAssetKeeper interface {
	Issuer(context.Context, vmtypes.AccountAddress) (vmtypes.AccountAddress, error)
	Symbol(context.Context, vmtypes.AccountAddress) (string, error)
}

const (
	DenomTraceDenomPrefixMove = "move/"
)

// NamedObjectAddress generates named object address from the seed (address + name + 0xFE)
func NamedObjectAddress(source vmtypes.AccountAddress, name string) vmtypes.AccountAddress {
	// 0xFE is the suffix of named object address, which is
	// defined in object.move as `OBJECT_FROM_SEED_ADDRESS_SCHEME`.
	hasher := sha3.New256()
	hasher.Write(append(append(source[:], []byte(name)...), 0xFE))
	bz := hasher.Sum(nil)

	addr, err := vmtypes.NewAccountAddressFromBytes(bz[:])
	if err != nil {
		panic(err)
	}

	return addr
}

func UserDerivedObjectAddress(source vmtypes.AccountAddress, deriveFrom vmtypes.AccountAddress) vmtypes.AccountAddress {
	hasher := sha3.New256()
	hasher.Write(append(append(source[:], deriveFrom[:]...), 0xFC))
	bz := hasher.Sum(nil)

	addr, err := vmtypes.NewAccountAddressFromBytes(bz[:])
	if err != nil {
		panic(err)
	}

	return addr
}

// MetadataAddressFromDenom extracts metadata address from a denom
func MetadataAddressFromDenom(denom string) (vmtypes.AccountAddress, error) {
	if strings.HasPrefix(denom, DenomTraceDenomPrefixMove) {
		hexStr := strings.TrimPrefix(denom, DenomTraceDenomPrefixMove)
		if strings.ToLower(hexStr) != hexStr {
			return vmtypes.AccountAddress{}, errors.New("metadata address should be lowercase")
		}

		addrBz, err := hex.DecodeString(hexStr)
		if err != nil {
			return vmtypes.AccountAddress{}, err
		}

		return vmtypes.NewAccountAddressFromBytes(addrBz)
	}

	// non move coins are generated from 0x1.
	return NamedObjectAddress(vmtypes.StdAddress, denom), nil
}

// DenomFromMetadataAddress returns denom of a metadata
func DenomFromMetadataAddress(ctx context.Context, k FungibleAssetKeeper, metadata vmtypes.AccountAddress) (string, error) {
	symbol, err := k.Symbol(ctx, metadata)
	if err != nil {
		return "", err
	}

	// If the coin is issued by 0x1, then return symbol as denom
	if NamedObjectAddress(vmtypes.StdAddress, symbol) == metadata {
		return symbol, err
	}

	// Else, add `move/` prefix
	return DenomTraceDenomPrefixMove + metadata.CanonicalString(), nil
}

// IsMoveCoin returns the flag whether the coin denom contains `move/` prefix or not.
func IsMoveCoin(coin sdk.Coin) bool {
	return IsMoveDenom(coin.Denom)
}

// IsMoveDenom returns the flag whether the denom contains `move/` prefix or not.
func IsMoveDenom(denom string) bool {
	return strings.HasPrefix(denom, DenomTraceDenomPrefixMove)
}
