package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Test_GetVotingPowerWeights(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

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
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)
	err = dexKeeper.SetDexPair(ctx, types.DexPair{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
	})
	require.NoError(t, err)

	denomLP, err := types.DenomFromMetadataAddress(ctx, keeper.NewMoveBankKeeper(&input.MoveKeeper), metadataLP)
	require.NoError(t, err)

	votingPowerWeights, err := keeper.NewVotingPowerKeeper(&input.MoveKeeper).GetVotingPowerWeights(ctx, []string{bondDenom, denomLP})
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoins(
		sdk.NewDecCoin(bondDenom, math.NewInt(1)),
		sdk.NewDecCoinFromDec(denomLP, math.LegacyNewDecWithPrec(5, 1))),
		votingPowerWeights)
}
