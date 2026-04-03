package keeper_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func TestDexPair(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	denom := "foo"
	denomLP := "lp"
	found, err := dexKeeper.HasDexPair(ctx, denom)
	require.NoError(t, err)
	require.False(t, found)

	metadataQuote, err := types.MetadataAddressFromDenom(denom)
	require.NoError(t, err)
	metadataLP, err := types.MetadataAddressFromDenom(denomLP)
	require.NoError(t, err)

	// invalid metadata
	dexPair := types.DexPair{
		MetadataQuote: "quote",
		MetadataLP:    "lp",
	}

	// store dex pair
	err = dexKeeper.SetDexPair(ctx, dexPair)
	require.Error(t, err)

	dexPair = types.DexPair{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
	}

	err = dexKeeper.SetDexPair(ctx, dexPair)
	require.NoError(t, err)

	found, err = dexKeeper.HasDexPair(ctx, denom)
	require.NoError(t, err)
	require.True(t, found)

	res, err := dexKeeper.GetMetadataLP(ctx, denom)
	require.NoError(t, err)
	require.Equal(t, metadataLP, res)
}

func TestDex_GetBaseSpotPrice_CLAMM(t *testing.T) {
	testCases := []struct {
		name          string
		quoteDenom    string
		sqrtPriceHigh uint64
		sqrtPriceLow  uint64
	}{
		{
			name:          "unity_price",
			quoteDenom:    "uusdc",
			sqrtPriceHigh: 1, // 2^64
			sqrtPriceLow:  0,
		},
		{
			name:          "very_small_sqrt_price",
			quoteDenom:    "uusdy",
			sqrtPriceHigh: 0,
			sqrtPriceLow:  1,
		},
		{
			name:          "very_large_sqrt_price",
			quoteDenom:    "uusdz",
			sqrtPriceHigh: ^uint64(0),
			sqrtPriceLow:  ^uint64(0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, input := createDefaultTestInput(t)
			dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

			params, err := input.MoveKeeper.GetParams(ctx)
			require.NoError(t, err)
			params.ClammModuleAddress = cafeAddr.String()
			require.NoError(t, input.MoveKeeper.SetParams(ctx, params))

			metadataQuote, err := types.MetadataAddressFromDenom(tc.quoteDenom)
			require.NoError(t, err)

			metadataLP := createCLAMMPool(t, ctx, input, bondDenom, tc.quoteDenom, tc.sqrtPriceHigh, tc.sqrtPriceLow)

			require.NoError(t, dexKeeper.SetDexPair(ctx, types.DexPair{
				MetadataQuote: metadataQuote.String(),
				MetadataLP:    metadataLP.String(),
			}))

			metadataBase, err := types.MetadataAddressFromDenom(bondDenom)
			require.NoError(t, err)

			metadata0, metadata1, err := keeper.NewCLAMMKeeper(&input.MoveKeeper, cafeAddr).GetPoolMetadata(ctx, metadataLP)
			require.NoError(t, err)
			require.Contains(t, []vmtypes.AccountAddress{metadata0, metadata1}, metadataBase)

			sqrtPriceBz, err := vmtypes.SerializeUint128(tc.sqrtPriceHigh, tc.sqrtPriceLow)
			require.NoError(t, err)
			sqrtPrice, err := types.DeserializeUint128(sqrtPriceBz)
			require.NoError(t, err)

			expectedPrice, err := types.CLAMMBaseSpotPrice(sqrtPrice, metadataBase == metadata0)
			require.NoError(t, err)

			price, err := dexKeeper.GetBaseSpotPrice(ctx, tc.quoteDenom)
			require.NoError(t, err)
			require.Equal(t, expectedPrice, price)

			// deterministic output for identical pool state
			priceAgain, err := dexKeeper.GetBaseSpotPrice(ctx, tc.quoteDenom)
			require.NoError(t, err)
			require.Equal(t, price, priceAgain)
		})
	}
}

func TestDex_GetBaseSpotPrice_CLAMM_ZeroSqrtPrice(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	quoteDenom := "uusdc"

	params, err := input.MoveKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.ClammModuleAddress = cafeAddr.String()
	require.NoError(t, input.MoveKeeper.SetParams(ctx, params))

	metadataQuote, err := types.MetadataAddressFromDenom(quoteDenom)
	require.NoError(t, err)

	metadataLP := createCLAMMPool(t, ctx, input, bondDenom, quoteDenom, 0, 0)
	require.NoError(t, dexKeeper.SetDexPair(ctx, types.DexPair{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
	}))

	price, err := dexKeeper.GetBaseSpotPrice(ctx, quoteDenom)
	require.Error(t, err)
	require.ErrorContains(t, err, "sqrt_price is zero")
	require.Equal(t, math.LegacyZeroDec(), price)
}

func TestDex_GetBaseSpotPrice_Balancer(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	baseAmount := math.NewInt(4_000_000_000_000)
	quoteDenom := "uusdc"
	quoteAmount := math.NewInt(1_000_000_000_000)
	weightBase := math.LegacyNewDecWithPrec(8, 1)
	weightQuote := math.LegacyNewDecWithPrec(2, 1)

	metadataQuote, err := types.MetadataAddressFromDenom(quoteDenom)
	require.NoError(t, err)

	metadataLP := createBalancerPool(
		t, ctx, input,
		sdk.NewCoin(bondDenom, baseAmount), sdk.NewCoin(quoteDenom, quoteAmount),
		weightBase, weightQuote,
	)
	require.NoError(t, dexKeeper.SetDexPair(ctx, types.DexPair{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
	}))

	expectedPrice := types.GetBaseSpotPrice(baseAmount, quoteAmount, weightBase, weightQuote)

	price, err := dexKeeper.GetBaseSpotPrice(ctx, quoteDenom)
	require.NoError(t, err)
	require.Equal(t, expectedPrice, price)

	// deterministic output for identical pool state
	priceAgain, err := dexKeeper.GetBaseSpotPrice(ctx, quoteDenom)
	require.NoError(t, err)
	require.Equal(t, price, priceAgain)
}

func TestDex_GetBaseSpotPrice_StableSwap(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	denomCoinB := "milkINIT"
	denomCoinC := "ibiINIT"

	metadataCoinB, err := types.MetadataAddressFromDenom(denomCoinB)
	require.NoError(t, err)

	metadataLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(
			sdk.NewCoin(bondDenom, math.NewInt(1_000_000_000_000)),
			sdk.NewCoin(denomCoinB, math.NewInt(1_000_000_000_001)),
			sdk.NewCoin(denomCoinC, math.NewInt(1_000_000_000_002)),
		),
	)
	require.NoError(t, dexKeeper.SetDexPair(ctx, types.DexPair{
		MetadataQuote: metadataCoinB.String(),
		MetadataLP:    metadataLP.String(),
	}))

	price, err := dexKeeper.GetBaseSpotPrice(ctx, denomCoinB)
	require.NoError(t, err)
	require.True(t, price.IsPositive())

	// deterministic output for identical pool state
	priceAgain, err := dexKeeper.GetBaseSpotPrice(ctx, denomCoinB)
	require.NoError(t, err)
	require.Equal(t, price, priceAgain)
}

func TestDex_GetBaseSpotPrice_StableSwap_InvalidQuote(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	denomCoinB := "milkINIT"
	denomCoinC := "ibiINIT"
	denomNotInPool := "ufoo"

	metadataNotInPool, err := types.MetadataAddressFromDenom(denomNotInPool)
	require.NoError(t, err)

	metadataLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(
			sdk.NewCoin(bondDenom, math.NewInt(1_000_000_000_000)),
			sdk.NewCoin(denomCoinB, math.NewInt(1_000_000_000_001)),
			sdk.NewCoin(denomCoinC, math.NewInt(1_000_000_000_002)),
		),
	)
	require.NoError(t, dexKeeper.SetDexPair(ctx, types.DexPair{
		MetadataQuote: metadataNotInPool.String(),
		MetadataLP:    metadataLP.String(),
	}))

	price, err := dexKeeper.GetBaseSpotPrice(ctx, denomNotInPool)
	require.Error(t, err)
	require.Equal(t, math.LegacyZeroDec(), price)
}

func TestDex_SwapToBase_StableSwap(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	denomCoinB := "milkINIT"
	denomCoinC := "ibiINIT"
	metadataCoinB, err := types.MetadataAddressFromDenom(denomCoinB)
	require.NoError(t, err)

	metadataLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(
			sdk.NewCoin(bondDenom, math.NewInt(1_000_000_000_000)),
			sdk.NewCoin(denomCoinB, math.NewInt(1_000_000_000_001)),
			sdk.NewCoin(denomCoinC, math.NewInt(1_000_000_000_002)),
		),
	)
	require.NoError(t, dexKeeper.SetDexPair(ctx, types.DexPair{
		MetadataQuote: metadataCoinB.String(),
		MetadataLP:    metadataLP.String(),
	}))

	metadataBase, err := types.MetadataAddressFromDenom(bondDenom)
	require.NoError(t, err)

	const offerAmount uint64 = 1_000
	simRes, _, err := input.MoveKeeper.ExecuteViewFunctionJSON(
		ctx,
		vmtypes.StdAddress,
		types.MoveModuleNameStableSwap,
		"get_swap_simulation",
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", metadataLP),
			fmt.Sprintf("\"%s\"", metadataCoinB),
			fmt.Sprintf("\"%s\"", metadataBase),
			fmt.Sprintf("\"%d\"", offerAmount),
		},
	)
	require.NoError(t, err)

	expectedOut := mustParseJSONUint64(t, simRes.Ret)
	quoteOfferCoin := sdk.NewCoin(denomCoinB, math.NewIntFromUint64(offerAmount))
	fundedAddr := input.Faucet.NewFundedAccount(ctx, quoteOfferCoin)
	before := input.BankKeeper.GetAllBalances(ctx, fundedAddr)

	require.NoError(t, dexKeeper.SwapToBase(ctx, fundedAddr, quoteOfferCoin))

	after := input.BankKeeper.GetAllBalances(ctx, fundedAddr)
	require.True(t, before.AmountOf(denomCoinB).Sub(math.NewIntFromUint64(offerAmount)).Equal(after.AmountOf(denomCoinB)))
	require.True(t, before.AmountOf(bondDenom).Add(math.NewIntFromUint64(expectedOut)).Equal(after.AmountOf(bondDenom)))
}

func TestDex_SwapToBase_UnsupportedPool(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	denomQuote := "ufoo"
	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)
	metadataLP, err := types.MetadataAddressFromDenom("ulppool")
	require.NoError(t, err)

	require.NoError(t, dexKeeper.SetDexPair(ctx, types.DexPair{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
	}))

	addr := input.Faucet.NewFundedAccount(ctx, sdk.NewInt64Coin(denomQuote, 100))
	err = dexKeeper.SwapToBase(ctx, addr, sdk.NewInt64Coin(denomQuote, 100))
	require.Error(t, err)
	require.ErrorContains(t, err, "not a supported DEX pool")
}

func TestDex_SwapToBase_CLAMM(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	quoteDenom := "uusdc"

	params, err := input.MoveKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.ClammModuleAddress = cafeAddr.String()
	require.NoError(t, input.MoveKeeper.SetParams(ctx, params))

	metadataQuote, err := types.MetadataAddressFromDenom(quoteDenom)
	require.NoError(t, err)

	metadataLP := createCLAMMPool(t, ctx, input, bondDenom, quoteDenom, 1, 0)
	require.NoError(t, dexKeeper.SetDexPair(ctx, types.DexPair{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
	}))

	quoteOfferCoin := sdk.NewInt64Coin(quoteDenom, 1_000)
	feeCollectorAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	input.Faucet.Fund(ctx, feeCollectorAddr, quoteOfferCoin)
	before := input.BankKeeper.GetAllBalances(ctx, feeCollectorAddr)
	stdBaseBefore := input.BankKeeper.GetBalance(ctx, types.StdAddr, bondDenom).Amount

	require.NoError(t, dexKeeper.SwapToBase(ctx, feeCollectorAddr, quoteOfferCoin))

	after := input.BankKeeper.GetAllBalances(ctx, feeCollectorAddr)
	stdBaseAfter := input.BankKeeper.GetBalance(ctx, types.StdAddr, bondDenom).Amount
	require.NotNil(t, after)
	require.True(t, before.AmountOf(quoteDenom).Sub(quoteOfferCoin.Amount).Equal(after.AmountOf(quoteDenom)))
	require.True(t, before.AmountOf(bondDenom).Add(math.NewInt(1_000)).Equal(after.AmountOf(bondDenom)))
	require.True(t, stdBaseBefore.Equal(stdBaseAfter))
}

func mustParseJSONUint64(t *testing.T, raw string) uint64 {
	t.Helper()

	var s string
	if err := json.Unmarshal([]byte(raw), &s); err == nil {
		v, err := strconv.ParseUint(s, 10, 64)
		require.NoError(t, err)
		return v
	}

	var v uint64
	require.NoError(t, json.Unmarshal([]byte(raw), &v))
	return v
}
