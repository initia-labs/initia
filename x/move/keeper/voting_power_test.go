package keeper_test

import (
	"testing"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Test_GetVotingPowerWeights(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	baseDenom := bondDenom
	baseAmount := sdk.NewInt(1_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := sdk.NewInt(2_500_000_000_000)

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	metadataLP := createDexPool(
		t, ctx, input,
		sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomQuote, quoteAmount),
		sdk.NewDecWithPrec(8, 1), sdk.NewDecWithPrec(2, 1),
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

	require.Equal(t, sdk.NewDecCoins(
		sdk.NewDecCoin(bondDenom, sdk.NewInt(1)),
		sdk.NewDecCoinFromDec(denomLP, sdk.NewDecWithPrec(5, 1))),
		keeper.NewVotingPowerKeeper(&input.MoveKeeper).GetVotingPowerWeights(ctx, []string{bondDenom, denomLP}))
}
