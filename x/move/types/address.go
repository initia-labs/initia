package types

import (
	"bytes"
	"strings"

	"cosmossdk.io/core/address"
	vmtypes "github.com/initia-labs/movevm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccAddressFromString return sdk.AccAddress from the bech32 encoded string address
// or hex encoded string address
func AccAddressFromString(ac address.Codec, addrStr string) (vmtypes.AccountAddress, error) {
	if strings.HasPrefix(addrStr, "0x") {
		addrStr = strings.TrimPrefix(addrStr, "0x")
		if len(addrStr)%2 == 1 {
			addrStr = "0" + addrStr
		}

		return vmtypes.NewAccountAddress(addrStr)
	} else if addr, err := ac.StringToBytes(addrStr); err != nil {
		return vmtypes.AccountAddress{}, err
	} else {
		return ConvertSDKAddressToVMAddress(addr), nil
	}
}

// ConvertSDKAddressToVMAddress returns address conversion add `0` prefix padding to the given address
// until 32 bytes size filled.
func ConvertSDKAddressToVMAddress(addr sdk.AccAddress) vmtypes.AccountAddress {
	vmAddr, err := vmtypes.NewAccountAddressFromBytes(addr)
	if err != nil {
		panic(err)
	}

	return vmAddr
}

// ConvertVMAddressToSDKAddress converts vm address to sdk.AccAddress by removing 0s prefix
func ConvertVMAddressToSDKAddress(addr vmtypes.AccountAddress) sdk.AccAddress {
	return bytes.TrimPrefix(addr[:], bytes.Repeat([]byte{0}, AddressBytesLength-20))
}
