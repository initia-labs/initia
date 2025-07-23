package integration_tests

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func Test_DispatchableWhitelist(t *testing.T) {
	app, addrs, privs := CreateApp(t)

	denom := dispatchableTokenDenom(t)

	// we should able to send dispatchable token via bank MsgSend
	err := executeMsgs(t, app, []sdk.Msg{
		&banktypes.MsgSend{
			FromAddress: addrs[0].String(),
			ToAddress:   addrs[1].String(),
			Amount:      sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(1))),
		},
	}, []uint64{0}, []uint64{1}, true, true, privs[0])
	require.NoError(t, err)

	// can't send after we reset the context decorator to do not allow dispatchable token
	app.MsgServiceRouter().SetContextDecorator(func(ctx sdk.Context, msg sdk.Msg) sdk.Context {
		return ctx
	})

	err = executeMsgs(t, app, []sdk.Msg{
		&banktypes.MsgSend{
			FromAddress: addrs[0].String(),
			ToAddress:   addrs[1].String(),
			Amount:      sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(1))),
		},
	}, []uint64{0}, []uint64{2}, false, false, privs[0])
	require.ErrorContains(t, err, "dispatchable fungible asset is not allowed in this context")
}
