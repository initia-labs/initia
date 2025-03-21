package dynamicfee_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/dynamic-fee/types"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
)

func Test_EndBlocker(t *testing.T) {
	app := createApp(t)

	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)

	ctx := app.BaseApp.NewUncachedContext(false, tmproto.Header{})
	err = app.DynamicFeeKeeper.SetParams(ctx, types.Params{
		BaseGasPrice:    math.LegacyNewDecWithPrec(15, 3),
		MinBaseGasPrice: math.LegacyNewDecWithPrec(1, 3),
		MaxBaseGasPrice: math.LegacyNewDec(10),
		TargetGas:       1_000_000,
		MaxChangeRate:   math.LegacyNewDecWithPrec(1, 1),
	})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)

	// initialize staking for secondBondDenom
	ctx = app.BaseApp.NewUncachedContext(false, tmproto.Header{})
	err = app.MoveKeeper.InitializeStaking(ctx, secondBondDenom)
	require.NoError(t, err)

	// fund addr2
	app.BankKeeper.SendCoins(ctx, movetypes.StdAddr, addr2, sdk.NewCoins(secondBondCoin))

	_, err = app.Commit()
	require.NoError(t, err)

	ctx = app.BaseApp.NewUncachedContext(false, tmproto.Header{})
	lessBaseGasPrice, err := app.DynamicFeeKeeper.BaseGasPrice(ctx)
	require.NoError(t, err)
	require.True(t, lessBaseGasPrice.LT(types.DefaultBaseGasPrice))

	msgs := []sdk.Msg{}
	for i := 0; i < 100; i++ {
		msgs = append(msgs, &banktypes.MsgSend{
			FromAddress: addr2.String(),
			ToAddress:   addr1.String(),
			Amount:      sdk.NewCoins(sdk.NewInt64Coin(secondBondDenom, 10)),
		})
	}

	_, err = executeMsgsWithGasInfo(t, app, msgs, []uint64{1}, []uint64{0}, priv2)
	require.NoError(t, err)

	ctx = app.BaseApp.NewUncachedContext(false, tmproto.Header{})
	baseGasPrice, err := app.DynamicFeeKeeper.BaseGasPrice(ctx)
	require.NoError(t, err)
	require.True(t, baseGasPrice.GT(lessBaseGasPrice))
}
