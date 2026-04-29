package keeper_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
)

func createStableSwapPool(
	t *testing.T, ctx sdk.Context, input TestKeepers, coins sdk.Coins,
) (metadataLP vmtypes.AccountAddress) {
	metadata := make([]vmtypes.AccountAddress, len(coins))
	amounts := make([]uint64, len(coins))

	for i, coin := range coins {
		metadataCoin, err := types.MetadataAddressFromDenom(coin.Denom)
		require.NoError(t, err)

		metadata[i] = metadataCoin
		amounts[i] = coin.Amount.Uint64()

		// fund test account for stableswap creation
		input.Faucet.Fund(ctx, types.TestAddr, coin)
	}

	denomLP := "ulp"

	//
	// prepare arguments
	//

	name, err := vmtypes.SerializeString("LP Coin")
	require.NoError(t, err)

	symbol, err := vmtypes.SerializeString(denomLP)
	require.NoError(t, err)

	// 0.003 == 0.3%

	swapFeeBz := decToVmArgument(t, math.LegacyNewDecWithPrec(3, 3))
	metadataBz, err := vmtypes.SerializeAddressVector(metadata)
	require.NoError(t, err)
	amountsBz, err := vmtypes.SerializeUint64Vector(amounts)
	require.NoError(t, err)
	annBz, err := vmtypes.SerializeUint64(3000)
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunction(
		ctx,
		vmtypes.TestAddress,
		vmtypes.StdAddress,
		"stableswap",
		"create_pool_script",
		[]vmtypes.TypeTag{},
		[][]byte{
			name,
			symbol,
			swapFeeBz,
			metadataBz,
			amountsBz,
			annBz,
		},
	)
	require.NoError(t, err)

	return types.NamedObjectAddress(vmtypes.TestAddress, denomLP)
}

func Test_StableSwap_HasPool(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	stableSwapKeeper := keeper.NewStableSwapKeeper(&input.MoveKeeper)

	baseDenom := bondDenom
	baseAmount := math.NewInt(1_000_000_000_000)

	denomCoinB := "milkINIT"
	amountCoinB := math.NewInt(1_000_000_000_001)

	denomCoinC := "ibiINIT"
	amountCoinC := math.NewInt(1_000_000_000_002)

	metadataBase, err := types.MetadataAddressFromDenom(baseDenom)
	require.NoError(t, err)

	metadataCoinB, err := types.MetadataAddressFromDenom(denomCoinB)
	require.NoError(t, err)

	metadataCoinC, err := types.MetadataAddressFromDenom(denomCoinC)
	require.NoError(t, err)

	metadataLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomCoinB, amountCoinB), sdk.NewCoin(denomCoinC, amountCoinC)),
	)

	ok, err := stableSwapKeeper.HasPool(ctx, metadataLP)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = stableSwapKeeper.HasPool(ctx, metadataCoinB)
	require.NoError(t, err)
	require.False(t, ok)

	metadata, err := stableSwapKeeper.GetPoolMetadata(ctx, metadataLP)
	require.NoError(t, err)
	require.Contains(t, metadata, metadataBase)
	require.Contains(t, metadata, metadataCoinB)
	require.Contains(t, metadata, metadataCoinC)
}

func Test_StableSwap_Whitelist(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	stableSwapKeeper := keeper.NewStableSwapKeeper(&input.MoveKeeper)

	baseDenom := bondDenom
	baseAmount := math.NewInt(1_000_000_000_000)

	denomCoinB := "milkINIT"
	amountCoinB := math.NewInt(1_000_000_000_001)

	denomCoinC := "ibiINIT"
	amountCoinC := math.NewInt(1_000_000_000_002)

	metadataLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomCoinB, amountCoinB), sdk.NewCoin(denomCoinC, amountCoinC)),
	)

	ok, err := stableSwapKeeper.WhitelistStaking(ctx, metadataLP)
	require.NoError(t, err)
	require.True(t, ok)
}

func Test_StableSwap_Whitelist_Failed_MissingBase(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	stableSwapKeeper := keeper.NewStableSwapKeeper(&input.MoveKeeper)

	baseDenom := "aINIT"
	baseAmount := math.NewInt(1_000_000_000_000)

	denomCoinB := "milkINIT"
	amountCoinB := math.NewInt(1_000_000_000_001)

	denomCoinC := "ibiINIT"
	amountCoinC := math.NewInt(1_000_000_000_002)

	metadataLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomCoinB, amountCoinB), sdk.NewCoin(denomCoinC, amountCoinC)),
	)

	ok, err := stableSwapKeeper.WhitelistStaking(ctx, metadataLP)
	require.Error(t, err)
	require.False(t, ok)
}

func Test_StableSwap_GetBaseSpotPrice(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	stableSwapKeeper := keeper.NewStableSwapKeeper(&input.MoveKeeper)

	baseDenom := bondDenom
	baseAmount := math.NewInt(1_000_000_000_000)

	denomCoinB := "milkINIT"
	amountCoinB := math.NewInt(1_000_000_000_001)

	denomCoinC := "ibiINIT"
	amountCoinC := math.NewInt(1_000_000_000_002)

	metadataLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(
			sdk.NewCoin(baseDenom, baseAmount),
			sdk.NewCoin(denomCoinB, amountCoinB),
			sdk.NewCoin(denomCoinC, amountCoinC),
		),
	)

	metadataCoinB, err := types.MetadataAddressFromDenom(denomCoinB)
	require.NoError(t, err)

	price, err := stableSwapKeeper.GetBaseSpotPrice(ctx, metadataCoinB, metadataLP)
	require.NoError(t, err)
	require.True(t, price.IsPositive())

	// deterministic output for identical pool state
	priceAgain, err := stableSwapKeeper.GetBaseSpotPrice(ctx, metadataCoinB, metadataLP)
	require.NoError(t, err)
	require.Equal(t, price, priceAgain)
}

func Test_StableSwap_GetBaseSpotPrice_InvalidPair(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	stableSwapKeeper := keeper.NewStableSwapKeeper(&input.MoveKeeper)

	baseDenom := "aINIT"
	baseAmount := math.NewInt(1_000_000_000_000)

	denomCoinB := "milkINIT"
	amountCoinB := math.NewInt(1_000_000_000_001)

	denomCoinC := "ibiINIT"
	amountCoinC := math.NewInt(1_000_000_000_002)

	metadataLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(
			sdk.NewCoin(baseDenom, baseAmount),
			sdk.NewCoin(denomCoinB, amountCoinB),
			sdk.NewCoin(denomCoinC, amountCoinC),
		),
	)

	metadataCoinB, err := types.MetadataAddressFromDenom(denomCoinB)
	require.NoError(t, err)

	price, err := stableSwapKeeper.GetBaseSpotPrice(ctx, metadataCoinB, metadataLP)
	require.Error(t, err)
	require.Equal(t, math.LegacyZeroDec(), price)
}

func Test_StableSwap_SwapToBase(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	stableSwapKeeper := keeper.NewStableSwapKeeper(&input.MoveKeeper)

	baseDenom := bondDenom
	denomCoinB := "milkINIT"
	denomCoinC := "ibiINIT"

	metadataBase, err := types.MetadataAddressFromDenom(baseDenom)
	require.NoError(t, err)
	metadataCoinB, err := types.MetadataAddressFromDenom(denomCoinB)
	require.NoError(t, err)

	metadataLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(
			sdk.NewCoin(baseDenom, math.NewInt(1_000_000_000_000)),
			sdk.NewCoin(denomCoinB, math.NewInt(1_000_000_000_001)),
			sdk.NewCoin(denomCoinC, math.NewInt(1_000_000_000_002)),
		),
	)

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
	expectedOut := mustParseJSONUint64ForStableSwap(t, simRes.Ret)

	quoteOfferCoin := sdk.NewCoin(denomCoinB, math.NewIntFromUint64(offerAmount))
	fundedAddr := input.Faucet.NewFundedAccount(ctx, quoteOfferCoin)
	before := input.BankKeeper.GetAllBalances(ctx, fundedAddr)

	err = stableSwapKeeper.SwapToBase(
		ctx,
		types.ConvertSDKAddressToVMAddress(fundedAddr),
		metadataLP,
		metadataCoinB,
		math.NewIntFromUint64(offerAmount),
	)
	require.NoError(t, err)

	after := input.BankKeeper.GetAllBalances(ctx, fundedAddr)
	require.True(t, before.AmountOf(denomCoinB).Sub(math.NewIntFromUint64(offerAmount)).Equal(after.AmountOf(denomCoinB)))
	require.True(t, before.AmountOf(baseDenom).Add(math.NewIntFromUint64(expectedOut)).Equal(after.AmountOf(baseDenom)))
}

func Test_StableSwap_SwapToBase_BlockedRecipient(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	stableSwapKeeper := keeper.NewStableSwapKeeper(&input.MoveKeeper)

	baseDenom := bondDenom
	denomCoinB := "milkINIT"
	denomCoinC := "ibiINIT"

	metadataCoinB, err := types.MetadataAddressFromDenom(denomCoinB)
	require.NoError(t, err)

	metadataLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(
			sdk.NewCoin(baseDenom, math.NewInt(1_000_000_000_000)),
			sdk.NewCoin(denomCoinB, math.NewInt(1_000_000_000_001)),
			sdk.NewCoin(denomCoinC, math.NewInt(1_000_000_000_002)),
		),
	)

	quoteOfferCoin := sdk.NewInt64Coin(denomCoinB, 1_000)
	feeCollectorAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	input.Faucet.Fund(ctx, feeCollectorAddr, quoteOfferCoin)

	err = stableSwapKeeper.SwapToBase(
		ctx,
		types.ConvertSDKAddressToVMAddress(feeCollectorAddr),
		metadataLP,
		metadataCoinB,
		quoteOfferCoin.Amount,
	)
	require.Error(t, err)
}

func mustParseJSONUint64ForStableSwap(t *testing.T, raw string) uint64 {
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
