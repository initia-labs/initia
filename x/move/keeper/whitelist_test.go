package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	distrtypes "github.com/initia-labs/initia/x/distribution/types"
	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"

	"github.com/stretchr/testify/require"
)

func TestWhitelistProposal(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	// start dex creation
	baseDenom := bondDenom
	baseAmount := sdk.NewInt(1_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := sdk.NewInt(4_000_000_000_000)

	metadataLP := createDexPool(
		t, ctx, input,
		sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomQuote, quoteAmount),
		sdk.NewDecWithPrec(8, 1), sdk.NewDecWithPrec(2, 1),
	)
	// finish dex creation

	// create publish operation proposal
	err := input.MoveKeeper.Whitelist(ctx, types.MsgWhitelist{
		MetadataLP:   metadataLP.String(),
		RewardWeight: sdk.OneDec(),
		Authority:    input.MoveKeeper.GetAuthority(),
	})
	require.NoError(t, err)

	//
	// check after whitelist
	//

	denomLP, err := types.DenomFromMetadataAddress(ctx, keeper.NewMoveBankKeeper(&input.MoveKeeper), metadataLP)
	require.NoError(t, err)

	// check staking params update
	require.Contains(t, input.StakingKeeper.BondDenoms(ctx), denomLP)

	// check distribution params update
	require.Contains(t, input.DistKeeper.GetRewardWeights(ctx), distrtypes.RewardWeight{Denom: denomLP, Weight: sdk.OneDec()})

	// dex pair registration
	_metadataLP, err := dexKeeper.GetMetadataLP(ctx, denomQuote)
	require.NoError(t, err)
	require.Equal(t, metadataLP, _metadataLP)

	found, err := input.MoveKeeper.HasStakingState(ctx, metadataLP)
	require.NoError(t, err)
	require.True(t, found)

	//
	// delist
	//

	err = input.MoveKeeper.Delist(ctx, types.MsgDelist{
		MetadataLP: metadataLP.String(),
		Authority:  input.MoveKeeper.GetAuthority(),
	})
	require.NoError(t, err)

	//
	// check after delist
	//

	// check staking params update
	require.NotContains(t, input.StakingKeeper.GetParams(ctx).BondDenoms, denomLP)

	// check distribution params update
	require.NotContains(t, input.DistKeeper.GetRewardWeights(ctx), distrtypes.RewardWeight{Denom: denomLP, Weight: sdk.OneDec()})

	// check move dex pair update
	found, err = dexKeeper.HasDexPair(ctx, denomQuote)
	require.NoError(t, err)
	require.False(t, found)
}

func TestWhitelistProposalReverse(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	// start dex creation
	baseDenom := bondDenom
	baseAmount := sdk.NewInt(1_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := sdk.NewInt(4_000_000_000_000)

	metadataLP := createDexPool(
		t, ctx, input,
		sdk.NewCoin(denomQuote, quoteAmount), sdk.NewCoin(baseDenom, baseAmount),
		sdk.NewDecWithPrec(2, 1), sdk.NewDecWithPrec(8, 1),
	)
	// finish dex creation

	// create publish operation proposal
	err := input.MoveKeeper.Whitelist(ctx, types.MsgWhitelist{
		MetadataLP:   metadataLP.String(),
		RewardWeight: sdk.OneDec(),
		Authority:    input.MoveKeeper.GetAuthority(),
	})
	require.NoError(t, err)

	//
	// check after whitelist
	//

	denomLP, err := types.DenomFromMetadataAddress(ctx, keeper.NewMoveBankKeeper(&input.MoveKeeper), metadataLP)
	require.NoError(t, err)

	// check staking params update
	require.Contains(t, input.StakingKeeper.BondDenoms(ctx), denomLP)

	// check distribution params update
	require.Contains(t, input.DistKeeper.GetRewardWeights(ctx), distrtypes.RewardWeight{Denom: denomLP, Weight: sdk.OneDec()})

	// dex pair registration
	_metadataLP, err := dexKeeper.GetMetadataLP(ctx, denomQuote)
	require.NoError(t, err)
	require.Equal(t, metadataLP, _metadataLP)

	found, err := input.MoveKeeper.HasStakingState(ctx, metadataLP)
	require.NoError(t, err)
	require.True(t, found)

	//
	// delist
	//

	err = input.MoveKeeper.Delist(ctx, types.MsgDelist{
		MetadataLP: metadataLP.String(),
		Authority:  input.MoveKeeper.GetAuthority(),
	})
	require.NoError(t, err)

	//
	// check after delist
	//

	// check staking params update
	require.NotContains(t, input.StakingKeeper.GetParams(ctx).BondDenoms, denomLP)

	// check distribution params update
	require.NotContains(t, input.DistKeeper.GetRewardWeights(ctx), distrtypes.RewardWeight{Denom: denomLP, Weight: sdk.OneDec()})

	// check move dex pair update
	found, err = dexKeeper.HasDexPair(ctx, denomQuote)
	require.NoError(t, err)
	require.False(t, found)
}
