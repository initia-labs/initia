package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/gov/keeper"
	"github.com/initia-labs/initia/x/gov/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
)

func Test_isLowThresholdProposal(t *testing.T) {
	params := types.DefaultParams()

	messages := []sdk.Msg{
		&movetypes.MsgExecute{
			ModuleAddress: "0x1",
			ModuleName:    "vip",
			FunctionName:  "register_snapshot",
		},
		&movetypes.MsgExecuteJSON{
			ModuleAddress: "0x1",
			ModuleName:    "vip",
			FunctionName:  "register_snapshot",
		},
	}
	proposal, err := types.NewProposal(messages, 1, time.Now().UTC(), time.Now().UTC().Add(time.Hour), "", "", "", addrs[0], true)
	require.NoError(t, err)
	require.True(t, keeper.IsLowThresholdProposal(params, proposal))

	messages = []sdk.Msg{
		&movetypes.MsgExecute{
			ModuleAddress: "0x1",
			ModuleName:    "vip",
			FunctionName:  "register_snapshot",
		},
		&movetypes.MsgScript{},
	}
	proposal, err = types.NewProposal(messages, 1, time.Now().UTC(), time.Now().UTC().Add(time.Hour), "", "", "", addrs[0], true)
	require.NoError(t, err)
	require.False(t, keeper.IsLowThresholdProposal(params, proposal))
}
