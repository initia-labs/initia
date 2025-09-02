package keeper_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestGetTransferFunds(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	res, err := input.IBCHooksKeeper.GetTransferFunds(ctx, nil)
	require.NoError(t, err)

	var coin sdk.Coin
	coinbz, err := json.Marshal(coin)
	require.NoError(t, err)

	require.True(t, bytes.Equal(res, coinbz))

	coin.Denom = "init"
	coin.Amount = sdkmath.NewInt(10000)

	err = input.IBCHooksKeeper.SetTransferFunds(ctx, coin)
	require.NoError(t, err)

	res, err = input.IBCHooksKeeper.GetTransferFunds(ctx, nil)
	require.NoError(t, err)

	coinbz, err = json.Marshal(coin)
	require.NoError(t, err)

	require.True(t, bytes.Equal(res, coinbz))

	err = input.IBCHooksKeeper.EmptyTransferFunds(ctx)
	require.NoError(t, err)

	res, err = input.IBCHooksKeeper.GetTransferFunds(ctx, nil)
	require.NoError(t, err)

	coinbz, err = json.Marshal(sdk.Coin{})
	require.NoError(t, err)

	require.True(t, bytes.Equal(res, coinbz))
}
