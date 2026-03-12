package keeper_test

import (
	"bytes"
	"testing"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/stretchr/testify/require"
)

// cafeAddr is the mock CLAMM module deployer address.
// It matches `cafe = 0xcafe` in contracts/Move.toml.
var cafeAddr = vmtypes.AccountAddress{30: 0xca, 31: 0xfe}

// createCLAMMPool publishes cafe::pool at cafeAddr, calls create_pool via the
// Move VM, and returns the named object address as metadataLP (the address
// where the Pool resource is stored). The faucet is used to ensure both token Metadata
// objects exist before the entry function is called.
//
// sqrtPriceHigh and sqrtPriceLow are the high/low 64-bit halves of the
// Q64.64 sqrt_price u128: sqrt(token1/token0) * 2^64.
func createCLAMMPool(
	t *testing.T,
	ctx sdk.Context,
	input TestKeepers,
	denomBase, denomQuote string,
	sqrtPriceHigh, sqrtPriceLow uint64,
) vmtypes.AccountAddress {
	t.Helper()

	metadataBase, err := types.MetadataAddressFromDenom(denomBase)
	require.NoError(t, err)

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	// Fund the tokens so their Metadata objects are initialised in the Move VM.
	input.Faucet.Fund(ctx, types.TestAddr,
		sdk.NewCoin(denomBase, math.NewInt(1)),
		sdk.NewCoin(denomQuote, math.NewInt(1)),
	)
	metadataLP := types.NamedObjectAddress(cafeAddr, "mock_pool")
	// Fund mock pool object address to allow output transfers during swap tests.
	input.Faucet.Fund(ctx, types.ConvertVMAddressToSDKAddress(metadataLP),
		sdk.NewCoin(denomBase, math.NewInt(1_000_000)),
		sdk.NewCoin(denomQuote, math.NewInt(1_000_000)),
	)

	// Publish mock CLAMM modules at cafeAddr.
	err = input.MoveKeeper.PublishModuleBundle(
		ctx, cafeAddr,
		vmtypes.NewModuleBundle(
			vmtypes.NewModule(clammPoolModule),
			vmtypes.NewModule(clammScriptsModule),
		),
		types.UpgradePolicy_COMPATIBLE,
	)
	require.NoError(t, err)

	// Sort metadata addresses (the contract requires metadata_0 <= metadata_1).
	var metadata0, metadata1 vmtypes.AccountAddress
	if bytes.Compare(metadataBase[:], metadataQuote[:]) <= 0 {
		metadata0, metadata1 = metadataBase, metadataQuote
	} else {
		metadata0, metadata1 = metadataQuote, metadataBase
	}

	sqrtPriceBz, err := vmtypes.SerializeUint128(sqrtPriceHigh, sqrtPriceLow)
	require.NoError(t, err)

	// Call create_pool; Pool resource is stored at the signer's address (cafeAddr).
	err = input.MoveKeeper.ExecuteEntryFunction(
		ctx,
		cafeAddr,
		cafeAddr,
		"pool",
		"create_pool",
		[]vmtypes.TypeTag{},
		[][]byte{metadata0[:], metadata1[:], sqrtPriceBz},
	)
	require.NoError(t, err)

	// Configure test swap outputs used by mock pool::swap validation.
	zeroForOneOutBz, err := vmtypes.SerializeUint64(1_000)
	require.NoError(t, err)
	oneForZeroOutBz, err := vmtypes.SerializeUint64(1_000)
	require.NoError(t, err)
	err = input.MoveKeeper.ExecuteEntryFunction(
		ctx,
		cafeAddr,
		cafeAddr,
		"pool",
		"set_test_swap_amounts",
		[]vmtypes.TypeTag{},
		[][]byte{metadataLP[:], zeroForOneOutBz, oneForZeroOutBz},
	)
	require.NoError(t, err)

	return metadataLP
}

func Test_CLAMM_WhitelistGasPrice(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	baseDenom := bondDenom
	denomQuote := "uusdc"

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	// sqrt_price = 2^64 → price = 1.0 (base is metadata0)
	metadataLP := createCLAMMPool(t, ctx, input, baseDenom, denomQuote, 1, 0)

	clammKeeper := keeper.NewCLAMMKeeper(&input.MoveKeeper, cafeAddr)

	ok, err := clammKeeper.WhitelistGasPrice(ctx, metadataQuote, metadataLP)
	require.NoError(t, err)
	require.True(t, ok)
}

func Test_CLAMM_DelistGasPrice(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	baseDenom := bondDenom
	denomQuote := "uusdc"

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	metadataLP := createCLAMMPool(t, ctx, input, baseDenom, denomQuote, 1, 0)

	clammKeeper := keeper.NewCLAMMKeeper(&input.MoveKeeper, cafeAddr)

	ok, err := clammKeeper.DelistGasPrice(ctx, metadataQuote, metadataLP)
	require.NoError(t, err)
	require.True(t, ok)
}

func Test_CLAMM_GetBaseSpotPrice(t *testing.T) {
	testCases := []struct {
		name          string
		baseDenom     string
		quoteDenom    string
		sqrtPriceHigh uint64
		sqrtPriceLow  uint64
	}{
		{
			name:          "unity_price",
			baseDenom:     bondDenom,
			quoteDenom:    "uusdc",
			sqrtPriceHigh: 1, // 2^64
			sqrtPriceLow:  0,
		},
		{
			name:          "very_small_sqrt_price",
			baseDenom:     bondDenom,
			quoteDenom:    "uusdy",
			sqrtPriceHigh: 0,
			sqrtPriceLow:  1,
		},
		{
			name:          "very_large_sqrt_price",
			baseDenom:     bondDenom,
			quoteDenom:    "uusdz",
			sqrtPriceHigh: ^uint64(0),
			sqrtPriceLow:  ^uint64(0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, input := createDefaultTestInput(t)

			metadataQuote, err := types.MetadataAddressFromDenom(tc.quoteDenom)
			require.NoError(t, err)

			metadataBase, err := types.MetadataAddressFromDenom(tc.baseDenom)
			require.NoError(t, err)

			metadataLP := createCLAMMPool(
				t, ctx, input, tc.baseDenom, tc.quoteDenom, tc.sqrtPriceHigh, tc.sqrtPriceLow,
			)

			clammKeeper := keeper.NewCLAMMKeeper(&input.MoveKeeper, cafeAddr)

			metadata0, metadata1, err := clammKeeper.GetPoolMetadata(ctx, metadataLP)
			require.NoError(t, err)
			require.Contains(t, []vmtypes.AccountAddress{metadata0, metadata1}, metadataBase)

			sqrtPriceBz, err := vmtypes.SerializeUint128(tc.sqrtPriceHigh, tc.sqrtPriceLow)
			require.NoError(t, err)
			sqrtPrice, err := types.DeserializeUint128(sqrtPriceBz)
			require.NoError(t, err)

			expectedPrice, err := types.CLAMMBaseSpotPrice(sqrtPrice, metadataBase == metadata0)
			require.NoError(t, err)

			price, err := clammKeeper.GetBaseSpotPrice(ctx, metadataQuote, metadataLP)
			require.NoError(t, err)
			require.Equal(t, expectedPrice, price)

			// deterministic output for identical pool state
			priceAgain, err := clammKeeper.GetBaseSpotPrice(ctx, metadataQuote, metadataLP)
			require.NoError(t, err)
			require.Equal(t, price, priceAgain)
		})
	}
}

func Test_CLAMM_GetBaseSpotPrice_ZeroSqrtPrice(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	baseDenom := bondDenom
	quoteDenom := "uusdc"

	metadataQuote, err := types.MetadataAddressFromDenom(quoteDenom)
	require.NoError(t, err)

	metadataLP := createCLAMMPool(t, ctx, input, baseDenom, quoteDenom, 0, 0)

	clammKeeper := keeper.NewCLAMMKeeper(&input.MoveKeeper, cafeAddr)

	price, err := clammKeeper.GetBaseSpotPrice(ctx, metadataQuote, metadataLP)
	require.Error(t, err)
	require.ErrorContains(t, err, "sqrt_price is zero")
	require.Equal(t, math.LegacyZeroDec(), price)
}

func Test_CLAMM_SwapToBase(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	baseDenom := bondDenom
	quoteDenom := "uusdc"
	metadataQuote, err := types.MetadataAddressFromDenom(quoteDenom)
	require.NoError(t, err)
	metadataLP := createCLAMMPool(t, ctx, input, baseDenom, quoteDenom, 1, 0)

	clammKeeper := keeper.NewCLAMMKeeper(&input.MoveKeeper, cafeAddr)

	quoteOfferCoin := sdk.NewInt64Coin(quoteDenom, 1_000)
	fundedAddr := input.Faucet.NewFundedAccount(ctx, quoteOfferCoin)
	before := input.BankKeeper.GetAllBalances(ctx, fundedAddr)

	err = clammKeeper.SwapToBase(
		ctx,
		types.ConvertSDKAddressToVMAddress(fundedAddr),
		metadataLP,
		metadataQuote,
		quoteOfferCoin.Amount,
	)
	require.NoError(t, err)

	after := input.BankKeeper.GetAllBalances(ctx, fundedAddr)
	require.True(t, before.AmountOf(quoteDenom).Sub(quoteOfferCoin.Amount).Equal(after.AmountOf(quoteDenom)))
	require.True(t, before.AmountOf(baseDenom).Add(math.NewInt(1_000)).Equal(after.AmountOf(baseDenom)))
}

func Test_CLAMM_SwapToBase_InvalidQuote(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	baseDenom := bondDenom
	quoteDenom := "uusdc"
	metadataLP := createCLAMMPool(t, ctx, input, baseDenom, quoteDenom, 1, 0)
	invalidQuote, err := types.MetadataAddressFromDenom("ufoo")
	require.NoError(t, err)

	clammKeeper := keeper.NewCLAMMKeeper(&input.MoveKeeper, cafeAddr)
	fundedAddr := input.Faucet.NewFundedAccount(ctx, sdk.NewInt64Coin(quoteDenom, 1_000))

	err = clammKeeper.SwapToBase(
		ctx,
		types.ConvertSDKAddressToVMAddress(fundedAddr),
		metadataLP,
		invalidQuote,
		math.NewInt(1_000),
	)
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid quote metadata")
}
