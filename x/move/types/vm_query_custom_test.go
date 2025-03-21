package types_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/address"
)

func Test_ToSDKAddress(t *testing.T) {
	ac := address.NewBech32Codec("init")

	reqBz, err := json.Marshal(types.ToSDKAddressRequest{
		VMAddr: vmtypes.StdAddress.String(),
	})
	require.NoError(t, err)

	toSDKAddrFn := types.ToSDKAddress(ac)
	resBz, err := toSDKAddrFn(context.Background(), reqBz)
	require.NoError(t, err)

	var res types.ToSDKAddressResponse
	err = json.Unmarshal(resBz, &res)
	require.NoError(t, err)
	require.Equal(t, types.StdAddr[12:].String(), res.SDKAddr)
}

func Test_FromSDKAddress(t *testing.T) {
	ac := address.NewBech32Codec("init")

	reqBz, err := json.Marshal(types.FromSDKAddressRequest{
		SDKAddr: types.StdAddr[12:].String(),
	})
	require.NoError(t, err)

	fromSDKAddressFn := types.FromSDKAddress(ac)
	resBz, err := fromSDKAddressFn(context.Background(), reqBz)
	require.NoError(t, err)

	var res types.FromSDKAddressResponse
	err = json.Unmarshal(resBz, &res)
	require.NoError(t, err)
	require.Equal(t, vmtypes.StdAddress.CanonicalString(), res.VMAddr)
}
