package keeper_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

func decToVmArgument(t *testing.T, val math.LegacyDec) []byte {
	bz := val.BigInt().Bytes()
	diff := 16 - len(bz)
	require.True(t, diff >= 0)
	if diff > 0 {
		bz = append(bytes.Repeat([]byte{0}, diff), bz...)
	}

	high := binary.BigEndian.Uint64(bz[:8])
	low := binary.BigEndian.Uint64(bz[8:16])

	// serialize to uint128
	bz, err := vmtypes.SerializeUint128(high, low)
	require.NoError(t, err)

	return bz
}

func createDexPool(
	t *testing.T, ctx sdk.Context, input TestKeepers,
	baseCoin sdk.Coin, quoteCoin sdk.Coin,
	weightBase sdk.Dec, weightQuote sdk.Dec,
) (metadataLP vmtypes.AccountAddress) {
	metadataBase, err := types.MetadataAddressFromDenom(baseCoin.Denom)
	require.NoError(t, err)

	metadataQuote, err := types.MetadataAddressFromDenom(quoteCoin.Denom)
	require.NoError(t, err)

	// fund test account for dex creation
	input.Faucet.Fund(ctx, types.TestAddr, baseCoin, quoteCoin)

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
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)
	moveBankKeeper := keeper.NewMoveBankKeeper(&input.MoveKeeper)

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
	balanceBase, balanceQuote, err := dexKeeper.GetPoolBalances(ctx, denomQuote)
	require.NoError(t, err)
	require.Equal(t, baseAmount, balanceBase)
	require.Equal(t, quoteAmount, balanceQuote)

	// check share balance
	totalShare, err := moveBankKeeper.GetSupplyWithMetadata(ctx, metadataLP)
	require.NoError(t, err)
	require.Equal(t, sdk.MaxInt(baseAmount, quoteAmount), totalShare)
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

	weightBase, weightQuote, err := dexKeeper.GetPoolWeights(ctx, denomQuote)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecWithPrec(8, 1), weightBase)
	require.Equal(t, math.LegacyNewDecWithPrec(2, 1), weightQuote)
}

func Test_GetPoolSpotPrice(t *testing.T) {
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

	quotePrice, err := dexKeeper.GetPoolSpotPrice(ctx, denomQuote)
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
