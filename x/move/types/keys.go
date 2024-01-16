package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	vmtypes "github.com/initia-labs/initiavm/types"
)

const (
	// ModuleName is the name of the move module
	ModuleName = "move"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// TStoreKey is the string transient store representation
	TStoreKey = "transient_" + ModuleName

	// QuerierRoute is the querier route for the move module
	QuerierRoute = ModuleName

	// RouterKey is the msg router key for the move module
	RouterKey = ModuleName
)

var (
	// AddressBytesLength address bytes length
	AddressBytesLength = len(vmtypes.AccountAddress{})
)

// MoveStakingModuleName is special purpose module name to store coins for staking feature
// staking address: initia185vd23fjl3pw2ecfppfdkm4jr7jj372jryhptt
const MoveStakingModuleName = "move_staking"

// MoveStakingModuleAddress is a staking module address for staking feature implementation
// staking address: initia185vd23fjl3pw2ecfppfdkm4jr7jj372jryhptt
var MoveStakingModuleAddress = authtypes.NewModuleAddress(MoveStakingModuleName)

// GetDelegatorModuleName return a unique delegator module name for a validator
func GetDelegatorModuleName(val sdk.ValAddress) string {
	return fmt.Sprintf("%s/%s", ModuleName, val.String())
}

// GetDelegatorModuleAddress return a unique delegator module address for a validator
func GetDelegatorModuleAddress(val sdk.ValAddress) sdk.AccAddress {
	moduleName := GetDelegatorModuleName(val)
	return authtypes.NewModuleAddress(moduleName)
}

// 0x1 address
var StdAddr = sdk.AccAddress(vmtypes.StdAddress[:])

var TestAddr = sdk.AccAddress(vmtypes.TestAddress[:])

// Keys for move store
// Items are stored with the following key: values
var (
	ExecutionCounterKey = []byte{0x11}
	DexPairPrefix       = []byte{0x12} // prefix for dex pairs
	VMStorePrefix       = []byte{0x21} // prefix for vm

	ParamsKey = []byte{0x31} // prefix for parameters for module x/move

	ModuleSeparator     = byte(0)
	ChecksumSeparator   = byte(1)
	ResourceSeparator   = byte(2)
	TableEntrySeparator = byte(3)
	TableInfoSeparator  = byte(4)
)

// GetModulePrefix returns the prefix key of an account module store
func GetModulePrefix(addr vmtypes.AccountAddress) []byte {
	return append(addr.Bytes(), ModuleSeparator)
}

// GetModuleKey returns the key of the published move module
func GetModuleKey(addr vmtypes.AccountAddress, moduleName string) ([]byte, error) {
	identifier := vmtypes.Identifier(moduleName)
	bz, err := identifier.BcsSerialize()
	if err != nil {
		return nil, err
	}

	return append(append(addr.Bytes(), ModuleSeparator), bz...), nil
}

// GetChecksumPrefix returns the prefix key of an account module checksum store
func GetChecksumPrefix(addr vmtypes.AccountAddress) []byte {
	return append(addr.Bytes(), ChecksumSeparator)
}

// GetChecksumKey returns the key of the published move module checksum
func GetChecksumKey(addr vmtypes.AccountAddress, moduleName string) ([]byte, error) {
	identifier := vmtypes.Identifier(moduleName)
	bz, err := identifier.BcsSerialize()
	if err != nil {
		return nil, err
	}

	return append(append(addr.Bytes(), ChecksumSeparator), bz...), nil
}

// GetResourcePrefix returns the prefix key of an account resource store
func GetResourcePrefix(addr vmtypes.AccountAddress) []byte {
	return append(addr.Bytes(), ResourceSeparator)
}

// GetResourceKey returns the store key of the Move resource
func GetResourceKey(addr vmtypes.AccountAddress, structTag vmtypes.StructTag) ([]byte, error) {
	bz, err := structTag.BcsSerialize()
	if err != nil {
		return nil, err
	}

	return append(append(addr.Bytes(), ResourceSeparator), bz...), nil
}

// GetTableInfoKey returns the store key of the table info
func GetTableInfoKey(tableAddr vmtypes.AccountAddress) []byte {
	return append(tableAddr.Bytes(), TableInfoSeparator)
}

// GetTableEntryPrefix returns the prefix key of an table store
func GetTableEntryPrefix(tableAddr vmtypes.AccountAddress) []byte {
	return append(tableAddr.Bytes(), TableEntrySeparator)
}

// GetTableEntryKey returns the store key of the Move resource
func GetTableEntryKey(tableAddr vmtypes.AccountAddress, key []byte) []byte {
	return append(append(tableAddr.Bytes(), TableEntrySeparator), key...)
}
