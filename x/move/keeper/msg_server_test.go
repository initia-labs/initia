package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"

	vmtypes "github.com/initia-labs/movevm/types"
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

	argBz, err := vmtypes.SerializeUint64(200)
	require.NoError(t, err)

	// invalid type args
	_, err = msgServer.Script(ctx, &types.MsgScript{
		Sender:    moduleAddr.String(),
		CodeBytes: basicCoinMintScript,
		TypeArgs:  []string{},
		Args:      [][]byte{argBz},
	})
	require.Error(t, err)

	// correct args
	_, err = msgServer.Script(ctx, &types.MsgScript{
		Sender:    moduleAddr.String(),
		CodeBytes: basicCoinMintScript,
		TypeArgs:  []string{"0x1::BasicCoin::Initia", "bool"},
		Args:      [][]byte{argBz},
	})
	require.NoError(t, err)

	// json args
	_, err = msgServer.ScriptJSON(ctx, &types.MsgScriptJSON{
		Sender:    moduleAddr.String(),
		CodeBytes: basicCoinMintScript,
		TypeArgs:  []string{"0x1::BasicCoin::Initia", "bool"},
		Args:      []string{"\"200\""},
	})
	require.NoError(t, err)
}

func Test_ScriptDisabled(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.MoveKeeper.GetParams(ctx)
	require.NoError(t, err)

	params.ScriptEnabled = false
	err = input.MoveKeeper.SetParams(ctx, params)
	require.NoError(t, err)

	msgServer := keeper.NewMsgServerImpl(&input.MoveKeeper)
	_, err = msgServer.Script(ctx, nil)
	require.ErrorIs(t, err, types.ErrScriptDisabled)
}

func Test_ExecuteMsg(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	ms := keeper.NewMsgServerImpl(&input.MoveKeeper)

	argBz, err := vmtypes.SerializeUint64(100)
	require.NoError(t, err)
	_, err = ms.Execute(ctx, &types.MsgExecute{
		Sender:        types.TestAddr.String(),
		ModuleAddress: vmtypes.StdAddress.String(),
		ModuleName:    "BasicCoin",
		FunctionName:  "mint",
		TypeArgs:      []string{"0x1::BasicCoin::Initia"},
		Args:          [][]byte{argBz},
	})
	require.NoError(t, err)

	events := ctx.EventManager().Events()
	event := events[len(events)-1]

	require.Equal(t, types.EventTypeMove, event.Type)

	require.Len(t, event.Attributes, 5)

	attr, ok := event.GetAttribute(types.AttributeKeyTypeTag)
	require.True(t, ok)
	require.Equal(t, "0x1::BasicCoin::MintEvent", attr.Value)

	attr, ok = event.GetAttribute("account")
	require.True(t, ok)
	require.Equal(t, "0x2", attr.Value)

	attr, ok = event.GetAttribute("amount")
	require.True(t, ok)
	require.Equal(t, "100", attr.Value)

	attr, ok = event.GetAttribute("coin_type")
	require.True(t, ok)
	require.Equal(t, "0x1::BasicCoin::Initia", attr.Value)

	// cleanup events
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	// json args
	_, err = ms.ExecuteJSON(ctx, &types.MsgExecuteJSON{
		Sender:        types.TestAddr.String(),
		ModuleAddress: vmtypes.StdAddress.String(),
		ModuleName:    "BasicCoin",
		FunctionName:  "mint",
		TypeArgs:      []string{"0x1::BasicCoin::Initia"},
		Args:          []string{"\"200\""},
	})
	require.NoError(t, err)

	events = ctx.EventManager().Events()
	event = events[len(events)-1]

	require.Equal(t, types.EventTypeMove, event.Type)

	require.Len(t, event.Attributes, 5)

	attr, ok = event.GetAttribute(types.AttributeKeyTypeTag)
	require.True(t, ok)
	require.Equal(t, "0x1::BasicCoin::MintEvent", attr.Value)

	attr, ok = event.GetAttribute("account")
	require.True(t, ok)
	require.Equal(t, "0x2", attr.Value)

	attr, ok = event.GetAttribute("amount")
	require.True(t, ok)
	require.Equal(t, "200", attr.Value)

	attr, ok = event.GetAttribute("coin_type")
	require.True(t, ok)
	require.Equal(t, "0x1::BasicCoin::Initia", attr.Value)
}

func Test_GovExecuteMsg(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	ms := keeper.NewMsgServerImpl(&input.MoveKeeper)

	argBz, err := vmtypes.SerializeUint64(100)
	require.NoError(t, err)

	_, err = ms.GovExecute(ctx, &types.MsgGovExecute{
		Authority:     input.MoveKeeper.GetAuthority(),
		Sender:        types.TestAddr.String(),
		ModuleAddress: vmtypes.StdAddress.String(),
		ModuleName:    "BasicCoin",
		FunctionName:  "mint",
		TypeArgs:      []string{"0x1::BasicCoin::Initia"},
		Args:          [][]byte{argBz},
	})
	require.NoError(t, err)

	events := ctx.EventManager().Events()
	event := events[len(events)-1]

	require.Equal(t, types.EventTypeMove, event.Type)

	require.Len(t, event.Attributes, 5)

	attr, ok := event.GetAttribute(types.AttributeKeyTypeTag)
	require.True(t, ok)
	require.Equal(t, "0x1::BasicCoin::MintEvent", attr.Value)

	attr, ok = event.GetAttribute("account")
	require.True(t, ok)
	require.Equal(t, "0x2", attr.Value)

	attr, ok = event.GetAttribute("amount")
	require.True(t, ok)
	require.Equal(t, "100", attr.Value)

	attr, ok = event.GetAttribute("coin_type")
	require.True(t, ok)
	require.Equal(t, "0x1::BasicCoin::Initia", attr.Value)

	// cleanup events
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	// json args
	_, err = ms.GovExecuteJSON(ctx, &types.MsgGovExecuteJSON{
		Authority:     input.MoveKeeper.GetAuthority(),
		Sender:        types.TestAddr.String(),
		ModuleAddress: vmtypes.StdAddress.String(),
		ModuleName:    "BasicCoin",
		FunctionName:  "mint",
		TypeArgs:      []string{"0x1::BasicCoin::Initia"},
		Args:          []string{"\"100\""},
	})
	require.NoError(t, err)

	events = ctx.EventManager().Events()
	event = events[len(events)-1]

	require.Equal(t, types.EventTypeMove, event.Type)

	require.Len(t, event.Attributes, 5)

	attr, ok = event.GetAttribute(types.AttributeKeyTypeTag)
	require.True(t, ok)
	require.Equal(t, "0x1::BasicCoin::MintEvent", attr.Value)

	attr, ok = event.GetAttribute("account")
	require.True(t, ok)
	require.Equal(t, "0x2", attr.Value)

	attr, ok = event.GetAttribute("amount")
	require.True(t, ok)
	require.Equal(t, "100", attr.Value)

	attr, ok = event.GetAttribute("coin_type")
	require.True(t, ok)
	require.Equal(t, "0x1::BasicCoin::Initia", attr.Value)
}

func Test_GovScriptMsg(t *testing.T) {
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
	_, err = msgServer.GovScript(ctx, &types.MsgGovScript{
		Authority: input.MoveKeeper.GetAuthority(),
		Sender:    wrongAddr.String(),
		CodeBytes: basicCoinMintScript,
		TypeArgs:  []string{"0x1::BasicCoin::Initia", "bool"},
		Args:      [][]byte{},
	})
	require.Error(t, err)

	argBz, err := vmtypes.SerializeUint64(200)
	require.NoError(t, err)

	// invalid type args
	_, err = msgServer.GovScript(ctx, &types.MsgGovScript{
		Authority: input.MoveKeeper.GetAuthority(),
		Sender:    moduleAddr.String(),
		CodeBytes: basicCoinMintScript,
		TypeArgs:  []string{},
		Args:      [][]byte{argBz},
	})
	require.Error(t, err)

	// correct args
	_, err = msgServer.GovScript(ctx, &types.MsgGovScript{
		Authority: input.MoveKeeper.GetAuthority(),
		Sender:    moduleAddr.String(),
		CodeBytes: basicCoinMintScript,
		TypeArgs:  []string{"0x1::BasicCoin::Initia", "bool"},
		Args:      [][]byte{argBz},
	})
	require.NoError(t, err)

	// json args
	_, err = msgServer.GovScriptJSON(ctx, &types.MsgGovScriptJSON{
		Authority: input.MoveKeeper.GetAuthority(),
		Sender:    moduleAddr.String(),
		CodeBytes: basicCoinMintScript,
		TypeArgs:  []string{"0x1::BasicCoin::Initia", "bool"},
		Args:      []string{"\"200\""},
	})
	require.NoError(t, err)
}
