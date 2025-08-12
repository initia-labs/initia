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
		// String IDs are not supported anymore with uint64 type
		err := json.Unmarshal([]byte(`{
			"id": "99",
			"module_address": "0x1",
			"module_name": "Counter"
		}`), &callbackStringID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal string into Go struct field AsyncCallback.id of type uint64")
	})

	t.Run("empty module address", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": 99,
			"module_address": "",
			"module_name": "Counter"
		}`), &callback)
		// No validation occurs during unmarshaling anymore
		require.NoError(t, err)
		require.Equal(t, "", callback.ModuleAddress)
	})

	t.Run("empty module name", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": 99,
			"module_address": "0x1",
			"module_name": ""
		}`), &callback)
		// No validation occurs during unmarshaling anymore
		require.NoError(t, err)
		require.Equal(t, "", callback.ModuleName)
	})

	t.Run("invalid module address format", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": 99,
			"module_address": "invalid",
			"module_name": "Counter"
		}`), &callback)
		// No validation occurs during unmarshaling anymore
		require.NoError(t, err)
		require.Equal(t, "invalid", callback.ModuleAddress)
	})

	t.Run("invalid id type", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": true,
			"module_address": "0x1",
			"module_name": "Counter"
		}`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal bool into Go struct field AsyncCallback.id of type uint64")
	})

	t.Run("invalid id string format", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": "not_a_number",
			"module_address": "0x1",
			"module_name": "Counter"
		}`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal string into Go struct field AsyncCallback.id of type uint64")
	})

	t.Run("malformed json", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{malformed`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid character")
	})

	t.Run("id with decimal value", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": 99.5,
			"module_address": "0x1",
			"module_name": "Counter"
		}`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal number 99.5 into Go struct field AsyncCallback.id of type uint64")
	})

	t.Run("id with string decimal value", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": "99.5",
			"module_address": "0x1",
			"module_name": "Counter"
		}`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal string into Go struct field AsyncCallback.id of type uint64")
	})

	t.Run("negative id value", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": -1,
			"module_address": "0x1",
			"module_name": "Counter"
		}`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal number -1 into Go struct field AsyncCallback.id of type uint64")
	})

	t.Run("negative string id value", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": "-1",
			"module_address": "0x1",
			"module_name": "Counter"
		}`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal string into Go struct field AsyncCallback.id of type uint64")
	})

	t.Run("id value exceeding uint64 max", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": 18446744073709551616,
			"module_address": "0x1",
			"module_name": "Counter"
		}`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal number 18446744073709551616 into Go struct field AsyncCallback.id of type uint64")
	})

	t.Run("string id value exceeding uint64 max", func(t *testing.T) {
		var callback movehooks.AsyncCallback
		err := json.Unmarshal([]byte(`{
			"id": "18446744073709551616",
			"module_address": "0x1",
			"module_name": "Counter"
		}`), &callback)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal string into Go struct field AsyncCallback.id of type uint64")
	})
}
