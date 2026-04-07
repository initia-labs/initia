package derivable

import (
	"bytes"
	"fmt"

	"github.com/cometbft/cometbft/crypto"
	"golang.org/x/crypto/sha3"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	vmtypes "github.com/initia-labs/movevm/types"
)

const (
	// KeyType is the string constant for the Secp256k1 algorithm
	KeyType = "derivable"
)

// amino encoding names
const (
	PubKeyName = "initia/PubKeyDerivable"
)

// DerivableAbstractionDerivedScheme identifies the scheme used for derivable
// account abstraction public keys. This value must exactly match the constant
// defined in 0x1::account_abstraction on the Move side.
const DerivableAbstractionDerivedScheme = byte(0x5)

var (
	_ cryptotypes.PubKey = &PubKey{}
)

// NewPubKey creates a new PubKey object from the given module address, module name, function name, and abstract public key.
func NewPubKey(moduleAddress string, moduleName string, functionName string, abstractPublicKey []byte) *PubKey {
	return &PubKey{
		ModuleAddress:     moduleAddress,
		ModuleName:        moduleName,
		FunctionName:      functionName,
		AbstractPublicKey: abstractPublicKey,
	}
}

// Address returns the address of the derived public key.
// This function implementation should align with `0x1::account_abstraction::derive_account_address` in MoveVM.
//
// Format:
// sha3_256(
//
//	bcs(module_address),
//	bcs(module_name),
//	bcs(function_name),
//	bcs(abstract_public_key),
//	DerivableAbstractionDerivedScheme
//
// )
func (pubKey PubKey) Address() crypto.Address {
	bytes := pubKey.Bytes()

	hasher := sha3.New256()
	hasher.Write(bytes)
	hash := hasher.Sum(nil)

	return crypto.Address(hash)
}

// Bytes returns the bytes of the derived public key.
//
// Format:
// bcs(module_address) | bcs(module_name) | bcs(function_name) | bcs(abstract_public_key) | DerivableAbstractionDerivedScheme
func (pubKey PubKey) Bytes() []byte {
	fInfo, err := vmtypes.NewFunctionInfo(pubKey.ModuleAddress, pubKey.ModuleName, pubKey.FunctionName)
	if err != nil {
		panic(fmt.Sprintf("failed to create function info: %v", err))
	}

	bytes, err := fInfo.BcsSerialize()
	if err != nil {
		panic(fmt.Sprintf("failed to serialize function info: %v", err))
	}

	pubkeyBytes, err := vmtypes.SerializeBytes(pubKey.AbstractPublicKey)
	if err != nil {
		panic(fmt.Sprintf("failed to serialize abstract public key: %v", err))
	}

	bytes = append(append(bytes, pubkeyBytes...), DerivableAbstractionDerivedScheme)

	return bytes
}

func (pubKey PubKey) String() string {
	return fmt.Sprintf(`DerivablePubKey{
	ModuleAddress: %s,
	ModuleName: %s,
	FunctionName: %s,
	AbstractPublicKey: 0x%x,
}`, pubKey.ModuleAddress, pubKey.ModuleName, pubKey.FunctionName, pubKey.AbstractPublicKey)
}

func (pubKey PubKey) Type() string {
	return KeyType
}

// Equals returns true if the pubkey type is the same and their bytes are deeply equal.
func (pubKey PubKey) Equals(other cryptotypes.PubKey) bool {
	return pubKey.Type() == other.Type() && bytes.Equal(pubKey.Bytes(), other.Bytes())
}

// VerifySignature always return false to indicate that the verify signature function is not supposed to be called for this pubkey type.
func (pubKey PubKey) VerifySignature(msg, sig []byte) bool {
	return false
}
