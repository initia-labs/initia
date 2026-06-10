package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	bankkeeper "github.com/initia-labs/initia/x/bank/keeper"
)

func TestInputOutputCoinsAppliesSendRestriction(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	bk, ok := input.BankKeeper.(bankkeeper.BaseKeeper)
	require.True(t, ok)

	denom := testDenoms[0]
	coins := sdk.NewCoins(sdk.NewInt64Coin(denom, 1_000))

	from := input.Faucet.NewFundedAccount(ctx, sdk.NewInt64Coin(denom, 1_000_000))
	to := input.Faucet.NewFundedAccount(ctx, sdk.NewInt64Coin(denom, 1))

	// register a restriction that records every invocation
	var calls int
	bk.AppendSendRestriction(func(_ context.Context, _, toAddr sdk.AccAddress, _ sdk.Coins) (sdk.AccAddress, error) {
		calls++
		return toAddr, nil
	})

	// SendCoins (MsgSend) must apply the restriction
	calls = 0
	require.NoError(t, bk.SendCoins(ctx, from, to, coins))
	require.Equal(t, 1, calls)

	// InputOutputCoins (MsgMultiSend) must apply the restriction once per output
	calls = 0
	in := banktypes.Input{Address: from.String(), Coins: coins}
	outputs := []banktypes.Output{{Address: to.String(), Coins: coins}}
	require.NoError(t, bk.InputOutputCoins(ctx, in, outputs))
	require.Equal(t, 1, calls)
}
