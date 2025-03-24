package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
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

	ok, err := stableSwapKeeper.Whitelist(ctx, metadataLP)
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

	ok, err := stableSwapKeeper.Whitelist(ctx, metadataLP)
	require.Error(t, err)
	require.False(t, ok)
}
