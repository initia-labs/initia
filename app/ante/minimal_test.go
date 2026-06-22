package ante

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestNewRecoverAnteHandler(t *testing.T) {
	t.Run("converts panic into error", func(t *testing.T) {
		panicking := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
			panic("ante handler panicked")
		}

		h := NewRecoverAnteHandler(panicking)

		require.NotPanics(t, func() {
			_, err := h(sdk.Context{}, nil, false)
			require.Error(t, err)
			require.Contains(t, err.Error(), "ante handler panicked")
		})
	})

	t.Run("passes through on success", func(t *testing.T) {
		ok := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
			return ctx.WithTxBytes([]byte("ok")), nil
		}

		h := NewRecoverAnteHandler(ok)

		newCtx, err := h(sdk.Context{}, nil, false)
		require.NoError(t, err)
		require.Equal(t, []byte("ok"), newCtx.TxBytes())
	})
}
