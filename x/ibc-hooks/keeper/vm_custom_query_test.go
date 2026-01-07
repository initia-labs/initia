package keeper_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/initia-labs/initia/x/ibc-hooks/types"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestQueryTransferFunds(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	res, err := input.IBCHooksKeeper.QueryTransferFunds(ctx, nil)
	require.NoError(t, err)

	nullBz, err := json.Marshal(nil)
	require.NoError(t, err)
	require.True(t, bytes.Equal(res, nullBz))

	var coin sdk.Coin
	coin.Denom = "init"
	coin.Amount = sdkmath.NewInt(10000)

	expected := types.TransferFunds{
		AmountInPacket: coin,
		BalanceChange:  coin.Sub(coin),
	}
	err = input.IBCHooksKeeper.SetTransferFunds(ctx, expected)
	require.NoError(t, err)

	res, err = input.IBCHooksKeeper.QueryTransferFunds(ctx, nil)
	require.NoError(t, err)

	expectedBz, err := json.Marshal(expected)
	require.NoError(t, err)

	require.True(t, bytes.Equal(res, expectedBz))

	err = input.IBCHooksKeeper.RemoveTransferFunds(ctx)
	require.NoError(t, err)

	res, err = input.IBCHooksKeeper.QueryTransferFunds(ctx, nil)
	require.NoError(t, err)

	require.True(t, bytes.Equal(res, nullBz))
}
