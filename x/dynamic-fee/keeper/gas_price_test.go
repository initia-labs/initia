package keeper_test

import (
	"slices"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/dynamic-fee/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
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

func createDexPool(
	t *testing.T, ctx sdk.Context, input TestKeepers,
	baseCoin sdk.Coin, quoteCoin sdk.Coin,
	weightBase math.LegacyDec, weightQuote math.LegacyDec,
) (metadataLP vmtypes.AccountAddress) {
	metadataBase, err := movetypes.MetadataAddressFromDenom(baseCoin.Denom)
	require.NoError(t, err)

	metadataQuote, err := movetypes.MetadataAddressFromDenom(quoteCoin.Denom)
	require.NoError(t, err)

	// fund test account for dex creation
	input.Faucet.Fund(ctx, movetypes.TestAddr, baseCoin, quoteCoin)

	denomLP := "ulp" + baseCoin.Denom + quoteCoin.Denom

	//
	// prepare arguments
	//

	name, err := vmtypes.SerializeString("LP Coin" + baseCoin.Denom + quoteCoin.Denom)
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

	return movetypes.NamedObjectAddress(vmtypes.TestAddress, denomLP)
}

func registerDexPool(t *testing.T, ctx sdk.Context, input TestKeepers, basePrice math.LegacyDec) ([]string, []math.LegacyDec) {
	err := input.DynamicFeeKeeper.SetParams(ctx, types.Params{
		MinBaseGasPrice: basePrice,
		MaxBaseGasPrice: basePrice,
		BaseGasPrice:    basePrice,
	})
	require.NoError(t, err)

	dexKeeper := input.MoveKeeper.DexKeeper()

	baseDenom := bondDenom
	baseAmount := math.NewInt(40)

	denomQuote := "uusdc"
	quoteAmount := math.NewInt(10)

	metadataQuote, err := movetypes.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	metadataLP := createDexPool(
		t, ctx, input,
		sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomQuote, quoteAmount),
		math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1),
	)

	// store dex pair for queries
	err = dexKeeper.SetDexPair(ctx, movetypes.DexPair{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
	})
	require.NoError(t, err)

	quotePrice, err := dexKeeper.GetBaseSpotPrice(ctx, denomQuote)
	require.NoError(t, err)
	require.Equal(t, math.LegacyOneDec(), quotePrice)

	denomQuote2 := "utia"
	quoteAmount = math.NewInt(20)

	metadataQuote2, err := movetypes.MetadataAddressFromDenom(denomQuote2)
	require.NoError(t, err)

	metadataLP2 := createDexPool(
		t, ctx, input,
		sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomQuote2, quoteAmount),
		math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1),
	)

	// store dex pair for queries
	err = dexKeeper.SetDexPair(ctx, movetypes.DexPair{
		MetadataQuote: metadataQuote2.String(),
		MetadataLP:    metadataLP2.String(),
	})
	require.NoError(t, err)

	quotePrice2, err := dexKeeper.GetBaseSpotPrice(ctx, denomQuote2)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(2), quotePrice2)

	return []string{denomQuote, denomQuote2}, []math.LegacyDec{quotePrice, quotePrice2}
}

func TestGasPrices(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	basePrice := math.LegacyNewDecWithPrec(1, 2) // 0.01
	denoms, prices := registerDexPool(t, ctx, input, basePrice)

	gasPrices, err := input.DynamicFeeKeeper.GasPrices(ctx)
	require.NoError(t, err)

	require.Equal(t, basePrice.Quo(gasPrices.AmountOf(denoms[0])), prices[0])
	require.Equal(t, basePrice.Quo(gasPrices.AmountOf(denoms[1])), prices[1])
}

func TestGasPrice(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	basePrice := math.LegacyNewDecWithPrec(1, 2) // 0.01
	denoms, prices := registerDexPool(t, ctx, input, basePrice)

	gasPrice, err := input.DynamicFeeKeeper.GasPrice(ctx, denoms[0])
	require.NoError(t, err)
	require.Equal(t, gasPrice.Denom, denoms[0])
	require.Equal(t, basePrice.Quo(gasPrice.Amount), prices[0])

	gasPrice, err = input.DynamicFeeKeeper.GasPrice(ctx, denoms[1])
	require.NoError(t, err)
	require.Equal(t, gasPrice.Denom, denoms[1])
	require.Equal(t, basePrice.Quo(gasPrice.Amount), prices[1])
}
