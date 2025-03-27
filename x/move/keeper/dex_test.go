package keeper_test

import (
	"slices"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

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

func createDexPool(
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
