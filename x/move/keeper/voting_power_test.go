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

	metadataLP := createDexPool(
		t, ctx, input,
		sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomQuote, quoteAmount),
		math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1),
	)

	denomLP, err := types.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP)
	require.NoError(t, err)

	votingPowerWeights, err := keeper.NewVotingPowerKeeper(&input.MoveKeeper).GetVotingPowerWeights(ctx, []string{bondDenom, denomLP})
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoins(
		sdk.NewDecCoin(bondDenom, math.NewInt(1)),
		// only locked base coin amount is considered
		sdk.NewDecCoinFromDec(denomLP, math.LegacyNewDecWithPrec(4, 1))),
		votingPowerWeights)
}

func Test_GetVotingPowerWeights_StableSwap(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// start stable swap creation
	baseDenom := bondDenom
	baseAmount := math.NewInt(1_000_000_000_000)

	denomCoinB := "milkINIT"
	amountCoinB := math.NewInt(1_000_000_000_000)

	denomCoinC := "ibiINIT"
	amountCoinC := math.NewInt(1_000_000_000_000)

	metadataLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomCoinB, amountCoinB), sdk.NewCoin(denomCoinC, amountCoinC)),
	)
	// finish stable swap creation

	denomLP, err := types.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP)
	require.NoError(t, err)

	votingPowerWeights, err := keeper.NewVotingPowerKeeper(&input.MoveKeeper).GetVotingPowerWeights(ctx, []string{bondDenom, denomLP})
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoins(
		sdk.NewDecCoin(bondDenom, math.NewInt(1)),
		// only locked base coin amount is considered
		sdk.NewDecCoinFromDec(denomLP, math.LegacyOneDec().QuoInt64(3))),
		votingPowerWeights)
}
