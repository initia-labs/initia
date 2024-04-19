package lanes_test

import (
	"testing"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/stretchr/testify/require"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	lanes "github.com/initia-labs/initia/app/lanes"
)

func Test_SystemLaneMatchHandler(t *testing.T) {
	ctx := sdk.NewContext(nil, types.Header{}, false, log.NewNopLogger())

	handler := lanes.SystemLaneMatchHandler()

	// 1 system message
	require.True(t, handler(ctx, MockTx{
		msgs: []sdk.Msg{
			&opchildtypes.MsgUpdateOracle{},
		},
	}))

	// 2 system messages
	require.False(t, handler(ctx, MockTx{
		msgs: []sdk.Msg{
			&opchildtypes.MsgUpdateOracle{},
			&opchildtypes.MsgUpdateOracle{},
		},
	}))

	// 1 non-system message
	require.False(t, handler(ctx, MockTx{
		msgs: []sdk.Msg{
			&banktypes.MsgSend{},
		},
	}))
}

func Test_RejectMatchHandler(t *testing.T) {
	ctx := sdk.NewContext(nil, types.Header{}, false, log.NewNopLogger())

	handler := lanes.RejectMatchHandler()
	require.False(t, handler(ctx, MockTx{
		msgs: []sdk.Msg{
			&banktypes.MsgSend{},
		},
	}))

	require.False(t, handler(ctx, MockTx{
		msgs: []sdk.Msg{
			&opchildtypes.MsgUpdateOracle{},
		},
	}))

}
