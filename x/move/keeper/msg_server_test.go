package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
)

func TestScriptMsg(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	moduleAddr := sdk.AccAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	wrongAddr := sdk.AccAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1})
	msgServer := keeper.NewMsgServerImpl(&input.MoveKeeper)
	_, err := msgServer.Publish(ctx, &types.MsgPublish{
		Sender:        moduleAddr.String(),
		CodeBytes:     [][]byte{basicCoinModule},
		UpgradePolicy: types.UpgradePolicy_COMPATIBLE,
	})
	require.NoError(t, err)

	// wrong addr
	_, err = msgServer.Script(ctx, &types.MsgScript{
		Sender:    wrongAddr.String(),
		CodeBytes: basicCoinMintScript,
		TypeArgs:  []string{"0x1::BasicCoin::Initia", "bool"},
		Args:      [][]byte{},
	})
	require.Error(t, err)

	// invalid type args
	_, err = msgServer.Script(ctx, &types.MsgScript{
		Sender:    moduleAddr.String(),
		CodeBytes: basicCoinMintScript,
		TypeArgs:  []string{},
		Args:      [][]byte{},
	})
	require.Error(t, err)

	// correct args
	_, err = msgServer.Script(ctx, &types.MsgScript{
		Sender:    moduleAddr.String(),
		CodeBytes: basicCoinMintScript,
		TypeArgs:  []string{"0x1::BasicCoin::Initia", "bool"},
		Args:      [][]byte{},
	})
	require.NoError(t, err)
}
