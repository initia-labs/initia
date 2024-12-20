package move_hooks_test

import (
	"encoding/json"
	"testing"

	movehooks "github.com/initia-labs/initia/x/ibc-hooks/move-hooks"
	"github.com/stretchr/testify/require"
)

func Test_Unmarshal_AsyncCallback(t *testing.T) {
	t.Run("valid numeric id", func(t *testing.T) {
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
	})

	t.Run("valid string id", func(t *testing.T) {
		var callbackStringID movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": "99",
			"module_address": "0x1",
			"module_name": "Counter"
		}`), &callbackStringID)
		require.NoError(t, err)
		require.Equal(t, movehooks.AsyncCallback{
			Id:            99,
			ModuleAddress: "0x1",
			ModuleName:    "Counter",
		}, callbackStringID)
	})

	t.Run("empty module address", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": 99,
			"module_address": "",
			"module_name": "Counter"
		}`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "module_address cannot be empty")
	})

	t.Run("empty module name", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": 99,
			"module_address": "0x1",
			"module_name": ""
		}`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "module_name cannot be empty")
	})

	t.Run("invalid module address format", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": 99,
			"module_address": "invalid",
			"module_name": "Counter"
		}`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid module_address format")
	})

	t.Run("invalid id type", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": true,
			"module_address": "0x1",
			"module_name": "Counter"
		}`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid id type")
	})

	t.Run("invalid id string format", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": "not_a_number",
			"module_address": "0x1",
			"module_name": "Counter"
		}`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid id format")
	})

	t.Run("malformed json", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{malformed`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid character")
	})
}
