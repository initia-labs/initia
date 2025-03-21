package types

import (
	"context"
	"encoding/json"

	"cosmossdk.io/core/address"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The whitelist of custom queries.
type CustomQueryWhiteList map[string]CustomQuery

// Default CustomQueryWhiteList
func DefaultCustomQueryWhiteList(ac address.Codec) CustomQueryWhiteList {
	res := make(CustomQueryWhiteList)
	res["to_sdk_address"] = ToSDKAddress(ac)
	res["from_sdk_address"] = FromSDKAddress(ac)
	return res
}

// CustomQuery interface for registration
type CustomQuery func(context.Context, []byte) ([]byte, error)

///////////////////////////////////////
// move address => SDK string address

type ToSDKAddressRequest struct {
	VMAddr string `json:"vm_addr"`
}

type ToSDKAddressResponse struct {
	SDKAddr string `json:"sdk_addr"`
}

func ToSDKAddress(ac address.Codec) func(ctx context.Context, req []byte) ([]byte, error) {
	return func(_ context.Context, req []byte) ([]byte, error) {
		tc := ToSDKAddressRequest{}
		err := json.Unmarshal(req, &tc)
		if err != nil {
			return nil, err
		}

		accAddr, err := AccAddressFromString(ac, tc.VMAddr)
		if err != nil {
			return nil, err
		}

		sdkAddr := ConvertVMAddressToSDKAddress(accAddr)
		res := &ToSDKAddressResponse{
			SDKAddr: sdkAddr.String(),
		}
		return json.Marshal(res)
	}
}

///////////////////////////////////////
// SDK string address => move address

type FromSDKAddressRequest struct {
	SDKAddr string `json:"sdk_addr"`
}

type FromSDKAddressResponse struct {
	VMAddr string `json:"vm_addr"`
}

func FromSDKAddress(ac address.Codec) func(ctx context.Context, req []byte) ([]byte, error) {
	return func(_ context.Context, req []byte) ([]byte, error) {
		fs := FromSDKAddressRequest{}
		err := json.Unmarshal(req, &fs)
		if err != nil {
			return nil, err
		}

		accAddr, err := sdk.AccAddressFromBech32(fs.SDKAddr)
		if err != nil {
			return nil, err
		}
		vmAddr := ConvertSDKAddressToVMAddress(accAddr)

		res := &FromSDKAddressResponse{
			VMAddr: vmAddr.CanonicalString(),
		}
		return json.Marshal(res)
	}
}
