package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	"github.com/stretchr/testify/require"
)

func Test_ReadPool(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := input.MoveKeeper.DexKeeper()
	moveBankKeeper := input.MoveKeeper.MoveBankKeeper()

	baseDenom := bondDenom
	baseAmount := math.NewInt(1_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := math.NewInt(2_500_000_000_000)

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	metadataLP := createDexPool(
		t, ctx, input,
		sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomQuote, quoteAmount),
		math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1),
	)

	// store dex pair for queries
	err = dexKeeper.SetDexPair(ctx, types.DexPair{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
	})
	require.NoError(t, err)

	// check pool balance
	balances, err := dexKeeper.PoolBalances(ctx, denomQuote)
	require.NoError(t, err)
	require.Equal(t, baseAmount, balances[0])
	require.Equal(t, quoteAmount, balances[1])

	// check share balance
	totalShare, err := moveBankKeeper.GetSupplyWithMetadata(ctx, metadataLP)
	require.NoError(t, err)
	require.Equal(t, math.MaxInt(baseAmount, quoteAmount), totalShare)
}

func Test_ReadWeights(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	baseDenom := bondDenom
	baseAmount := math.NewInt(1_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := math.NewInt(4_000_000_000_000)

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	metadataLP := createDexPool(
		t, ctx, input,
		sdk.NewCoin(denomQuote, quoteAmount), sdk.NewCoin(baseDenom, baseAmount),
		math.LegacyNewDecWithPrec(2, 1), math.LegacyNewDecWithPrec(8, 1),
	)

	// store dex pair for queries
	err = dexKeeper.SetDexPair(ctx, types.DexPair{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
	})
	require.NoError(t, err)

	weights, err := dexKeeper.PoolWeights(ctx, denomQuote)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecWithPrec(8, 1), weights[0])
	require.Equal(t, math.LegacyNewDecWithPrec(2, 1), weights[1])
}

func Test_GetBaseSpotPrice(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	baseDenom := bondDenom
	baseAmount := math.NewInt(4_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := math.NewInt(1_000_000_000_000)

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	metadataLP := createDexPool(
		t, ctx, input,
		sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomQuote, quoteAmount),
		math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1),
	)

	// store dex pair for queries
	err = dexKeeper.SetDexPair(ctx, types.DexPair{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
	})
	require.NoError(t, err)

	quotePrice, err := dexKeeper.GetBaseSpotPrice(ctx, denomQuote)
	require.NoError(t, err)
	require.Equal(t, math.LegacyOneDec(), quotePrice)
}

func Test_SwapToBase(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	baseDenom := bondDenom
	baseAmount := math.NewInt(4_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := math.NewInt(1_000_000_000_000)

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	metadataLP := createDexPool(
		t, ctx, input,
		sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomQuote, quoteAmount),
		math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1),
	)

	// store dex pair for queries
	err = dexKeeper.SetDexPair(ctx, types.DexPair{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
	})
	require.NoError(t, err)

	// create quote coin funded account
	quoteOfferCoin := sdk.NewInt64Coin(denomQuote, 1_000)
	fundedAddr := input.Faucet.NewFundedAccount(ctx, quoteOfferCoin)

	err = dexKeeper.SwapToBase(ctx, fundedAddr, quoteOfferCoin)
	require.NoError(t, err)

	coins := input.BankKeeper.GetAllBalances(ctx, fundedAddr)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(baseDenom, 997 /* swap fee deducted */)), coins)
}
