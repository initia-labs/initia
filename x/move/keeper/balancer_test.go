package keeper_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func decToVmArgument(t *testing.T, val math.LegacyDec) []byte {
	// big-endian bytes (bytes are cloned)
	bz := val.BigInt().Bytes()

	// reverse bytes to little-endian
	slices.Reverse(bz)

	// serialize bytes
	bz, err := vmtypes.SerializeBytes(bz)
	require.NoError(t, err)

	return bz
}

func createBalancerPool(
	t *testing.T, ctx sdk.Context, input TestKeepers,
	baseCoin sdk.Coin, quoteCoin sdk.Coin,
	weightBase math.LegacyDec, weightQuote math.LegacyDec,
) (metadataLP vmtypes.AccountAddress) {
	metadataBase, err := types.MetadataAddressFromDenom(baseCoin.Denom)
	require.NoError(t, err)

	metadataQuote, err := types.MetadataAddressFromDenom(quoteCoin.Denom)
	require.NoError(t, err)

	// fund test account for dex creation
	input.Faucet.Fund(ctx, types.TestAddr, baseCoin, quoteCoin)

	denomLP := "ulp" + baseCoin.Denom + quoteCoin.Denom

	//
	// prepare arguments
	//

	name, err := vmtypes.SerializeString("LP Coin")
	require.NoError(t, err)

	symbol, err := vmtypes.SerializeString(denomLP)
	require.NoError(t, err)

	// 0.003 == 0.3%
	swapFeeBz := decToVmArgument(t, math.LegacyNewDecWithPrec(3, 3))
	weightBaseBz := decToVmArgument(t, weightBase)
	weightQuoteBz := decToVmArgument(t, weightQuote)

	baseAmount, err := vmtypes.SerializeUint64(baseCoin.Amount.Uint64())
	require.NoError(t, err)

	quoteAmount, err := vmtypes.SerializeUint64(quoteCoin.Amount.Uint64())
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunction(
		ctx,
		vmtypes.TestAddress,
		vmtypes.StdAddress,
		"dex",
		"create_pair_script",
		[]vmtypes.TypeTag{},
		[][]byte{
			name,
			symbol,
			swapFeeBz,
			weightBaseBz,
			weightQuoteBz,
			metadataBase[:],
			metadataQuote[:],
			baseAmount,
			quoteAmount,
		},
	)
	require.NoError(t, err)

	return types.NamedObjectAddress(vmtypes.TestAddress, denomLP)
}

func Test_ReadPool(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := input.MoveKeeper.DexKeeper()
	balancerKeeper := input.MoveKeeper.BalancerKeeper()
	moveBankKeeper := input.MoveKeeper.MoveBankKeeper()

	baseDenom := bondDenom
	baseAmount := math.NewInt(1_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := math.NewInt(2_500_000_000_000)

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	metadataLP := createBalancerPool(
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
	balances, _, err := balancerKeeper.GetPoolInfo(ctx, metadataLP)
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
	dexKeeper := input.MoveKeeper.DexKeeper()
	balancerKeeper := input.MoveKeeper.BalancerKeeper()

	baseDenom := bondDenom
	baseAmount := math.NewInt(1_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := math.NewInt(4_000_000_000_000)

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	metadataLP := createBalancerPool(
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

	_, weights, err := balancerKeeper.GetPoolInfo(ctx, metadataLP)
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

	metadataLP := createBalancerPool(
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

	metadataLP := createBalancerPool(
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

	// transfer to fee collector'
	feeCollectorAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	err = input.BankKeeper.SendCoins(ctx, fundedAddr, feeCollectorAddr, sdk.NewCoins(quoteOfferCoin))
	require.NoError(t, err)

	err = dexKeeper.SwapToBase(ctx, feeCollectorAddr, quoteOfferCoin)
	require.NoError(t, err)

	coins := input.BankKeeper.GetAllBalances(ctx, feeCollectorAddr)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(baseDenom, 997 /* swap fee deducted */)), coins)
}
