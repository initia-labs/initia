package types

import (
	context "context"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/sha3"

	sdk "github.com/cosmos/cosmos-sdk/types"

	vmtypes "github.com/initia-labs/initiavm/types"
)

type FungibleAssetKeeper interface {
	Issuer(context.Context, vmtypes.AccountAddress) (vmtypes.AccountAddress, error)
	Symbol(context.Context, vmtypes.AccountAddress) (string, error)
}

const (
	DenomTraceDenomPrefixMove = "move/"
)

// Generate named object address from the seed (address + name + 0xFE)
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

// Extract metadata address from a denom
func MetadataAddressFromDenom(denom string) (vmtypes.AccountAddress, error) {
	if strings.HasPrefix(denom, DenomTraceDenomPrefixMove) {
		addrBz, err := hex.DecodeString(strings.TrimPrefix(denom, DenomTraceDenomPrefixMove))
		if err != nil {
			return vmtypes.AccountAddress{}, err
		}

		return vmtypes.NewAccountAddressFromBytes(addrBz)
	}

	// non move coins are generated from 0x1.
	return NamedObjectAddress(vmtypes.StdAddress, denom), nil
}

// Return denom of a metadata
func DenomFromMetadataAddress(ctx context.Context, k FungibleAssetKeeper, metadata vmtypes.AccountAddress) (string, error) {
	symbol, err := k.Symbol(ctx, metadata)
	if err != nil {
		return "", err
	}

	// If a coin is issued from move side, add `move/` prefix
	if NamedObjectAddress(vmtypes.StdAddress, symbol) == metadata {
		return symbol, err
	}

	return DenomTraceDenomPrefixMove + metadata.CanonicalString(), nil
}

// Return the flag whether the coin denom contains `move/` prefix or not.
func IsMoveCoin(coin sdk.Coin) bool {
	return IsMoveDenom(coin.Denom)
}

// Return the flag whether the denom contains `move/` prefix or not.
func IsMoveDenom(denom string) bool {
	return strings.HasPrefix(denom, DenomTraceDenomPrefixMove)
}
