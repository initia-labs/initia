package keeper

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/move/types"
)

type CustomQueryWhiteList map[string]CustomQuery

func DefaultCustomQueryWhiteList() CustomQueryWhiteList {
	res := make(CustomQueryWhiteList)
	res["to_sdk_address"] = ToSDKAddress
	res["from_sdk_address"] = FromSDKAddress
	return res
}

func EmptyCustomQueryWhiteList() CustomQueryWhiteList {
	return make(CustomQueryWhiteList)
}

type CustomQuery func(sdk.Context, []byte, *Keeper) ([]byte, error)

type ToSDKAddressRequest struct {
	VMAddr string `json:"vm_addr"`
}

type ToSDKAddressResponse struct {
	SDKAddr string `json:"sdk_addr"`
}

func ToSDKAddress(ctx sdk.Context, req []byte, keeper *Keeper) ([]byte, error) {
	tc := ToSDKAddressRequest{}
	err := json.Unmarshal(req, &tc)
	if err != nil {
		return nil, err
	}

	accAddr, err := types.AccAddressFromString(keeper.ac, tc.VMAddr)
	if err != nil {
		return nil, err
	}

	sdkAddr := types.ConvertVMAddressToSDKAddress(accAddr)
	res := &ToSDKAddressResponse{
		SDKAddr: sdkAddr.String(),
	}
	return json.Marshal(res)
}

type FromSDKAddressRequest struct {
	SDKAddr string `json:"sdk_addr"`
}

type FromSDKAddressResponse struct {
	VMAddr string `json:"vm_addr"`
}

func FromSDKAddress(ctx sdk.Context, req []byte, keeper *Keeper) ([]byte, error) {
	fs := FromSDKAddressRequest{}
	err := json.Unmarshal(req, &fs)
	if err != nil {
		return nil, err
	}

	accAddr, err := sdk.AccAddressFromBech32(fs.SDKAddr)
	if err != nil {
		return nil, err
	}
	vmAddr := types.ConvertSDKAddressToVMAddress(accAddr)

	res := &FromSDKAddressResponse{
		VMAddr: vmAddr.CanonicalString(),
	}
	return json.Marshal(res)
}
