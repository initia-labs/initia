package ibc_middleware

import (
	"encoding/base64"
	"fmt"
	"testing"

	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
	"github.com/stretchr/testify/require"
)

func Test_validateAndParseMemo(t *testing.T) {

	argBz, err := vmtypes.SerializeUint64(100)
	require.NoError(t, err)

	memo := fmt.Sprintf(
		`{
			"move" : {
				"module_address": "0x1",
				"module_name": "dex",
				"function_name": "swap",
				"type_args": ["0x1::native_uinit::Coin", "0x1::native_uusdc::Coin"],
				"args": ["%s"]
			}
		}`, base64.StdEncoding.EncodeToString(argBz))
	isMoveRouted, msg, err := validateAndParseMemo(memo, "0x1::dex::swap")
	require.True(t, isMoveRouted)
	require.NoError(t, err)
	require.Equal(t, movetypes.MsgExecute{
		ModuleAddress: "0x1",
		ModuleName:    "dex",
		FunctionName:  "swap",
		TypeArgs:      []string{"0x1::native_uinit::Coin", "0x1::native_uusdc::Coin"},
		Args:          [][]byte{argBz},
	}, msg)

	isMoveRouted, msg, err = validateAndParseMemo(memo, "0x1::dex::swap")
	require.True(t, isMoveRouted)
	require.NoError(t, err)
	require.Equal(t, movetypes.MsgExecute{
		ModuleAddress: "0x1",
		ModuleName:    "dex",
		FunctionName:  "swap",
		TypeArgs:      []string{"0x1::native_uinit::Coin", "0x1::native_uusdc::Coin"},
		Args:          [][]byte{argBz},
	}, msg)

	// invalid receiver
	isMoveRouted, _, err = validateAndParseMemo(memo, "0x2::dex::swap")
	require.True(t, isMoveRouted)
	require.Error(t, err)

	isMoveRouted, _, err = validateAndParseMemo("hihi", "0x2::dex::swap")
	require.False(t, isMoveRouted)
	require.NoError(t, err)
}
