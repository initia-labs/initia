package move_hooks_test

import (
	"encoding/json"
	"testing"

	movehooks "github.com/initia-labs/initia/x/ibc-hooks/move-hooks"
	"github.com/stretchr/testify/require"
)

func Test_Unmarshal_AsyncCallback(t *testing.T) {
	var callback movehooks.AsyncCallback
	err := json.Unmarshal([]byte(`{
		"id": 99,
		"module_address": "0x1",
		"module_name": "Counter"
	}`), &callback)
	require.NoError(t, err)
	require.Equal(t, movehooks.AsyncCallback{
		Id:            99,
		ModuleAddress: "0x1",
		ModuleName:    "Counter",
	}, callback)

	var callbackStringID movehooks.AsyncCallback
	err = json.Unmarshal([]byte(`{
		"id": "99",
		"module_address": "0x1",
		"module_name": "Counter"
	}`), &callbackStringID)
	require.NoError(t, err)
	require.Equal(t, callback, callbackStringID)
}
