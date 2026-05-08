package legacyfeeack_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/app/ibc/legacyfeeack"
)

func TestUnwrapAppVersion(t *testing.T) {
	cases := map[string]struct {
		version     string
		wantInner   string
		wantWrapped bool
	}{
		"plain ics20":      {"ics20-1", "", false},
		"plain ics721":     {"ics721-1", "", false},
		"empty":            {"", "", false},
		"fee-wrapped 20":   {`{"fee_version":"ics29-1","app_version":"ics20-1"}`, "ics20-1", true},
		"fee-wrapped 721":  {`{"fee_version":"ics29-1","app_version":"ics721-1"}`, "ics721-1", true},
		"json but not fee": {`{"foo":"bar"}`, "", false},
		"malformed":        {`{`, "", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			inner, wrapped := legacyfeeack.UnwrapAppVersion(tc.version)
			require.Equal(t, tc.wantInner, inner)
			require.Equal(t, tc.wantWrapped, wrapped)
			require.Equal(t, tc.wantWrapped, legacyfeeack.IsFeeWrappedVersion(tc.version))
		})
	}
}

func TestWrapAndUnwrapRoundtrip(t *testing.T) {
	inner := []byte(`{"result":"AQ=="}`)

	wrapped := legacyfeeack.WrapFeeAck(inner, true)

	var shape struct {
		AppAck  []byte `json:"app_acknowledgement"`
		Success bool   `json:"underlying_app_success"`
	}
	require.NoError(t, json.Unmarshal(wrapped, &shape))
	require.Equal(t, inner, shape.AppAck)
	require.True(t, shape.Success)

	peeled, ok := legacyfeeack.TryUnwrapFeeAck(wrapped)
	require.True(t, ok)
	require.Equal(t, inner, peeled)
}

func TestTryUnwrapFeeAckPlainAck(t *testing.T) {
	plain := []byte(`{"result":"AQ=="}`)

	_, ok := legacyfeeack.TryUnwrapFeeAck(plain)
	require.False(t, ok)
}

func TestTryUnwrapFeeAckGarbage(t *testing.T) {
	for _, in := range [][]byte{nil, {}, []byte("not json"), []byte("{")} {
		_, ok := legacyfeeack.TryUnwrapFeeAck(in)
		require.False(t, ok)
	}
}
