package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
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
	baseAmount := math.NewInt(1_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := math.NewInt(4_000_000_000_000)

	metadataLP := createDexPool(
		t, ctx, input,
		sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomQuote, quoteAmount),
		math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1),
	)
	// finish dex creation

	// create publish operation proposal
	err := input.MoveKeeper.Whitelist(ctx, types.MsgWhitelist{
		MetadataLP:   metadataLP.String(),
		RewardWeight: math.LegacyOneDec(),
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
	require.Contains(t, input.DistKeeper.GetRewardWeights(ctx), distrtypes.RewardWeight{Denom: denomLP, Weight: math.LegacyOneDec()})

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
	require.NotContains(t, input.DistKeeper.GetRewardWeights(ctx), distrtypes.RewardWeight{Denom: denomLP, Weight: math.LegacyOneDec()})

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
	baseAmount := math.NewInt(1_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := math.NewInt(4_000_000_000_000)

	metadataLP := createDexPool(
		t, ctx, input,
		sdk.NewCoin(denomQuote, quoteAmount), sdk.NewCoin(baseDenom, baseAmount),
		math.LegacyNewDecWithPrec(2, 1), math.LegacyNewDecWithPrec(8, 1),
	)
	// finish dex creation

	// create publish operation proposal
	err := input.MoveKeeper.Whitelist(ctx, types.MsgWhitelist{
		MetadataLP:   metadataLP.String(),
		RewardWeight: math.LegacyOneDec(),
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
	require.Contains(t, input.DistKeeper.GetRewardWeights(ctx), distrtypes.RewardWeight{Denom: denomLP, Weight: math.LegacyOneDec()})

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
	require.NotContains(t, input.DistKeeper.GetRewardWeights(ctx), distrtypes.RewardWeight{Denom: denomLP, Weight: math.LegacyOneDec()})

	// check move dex pair update
	found, err = dexKeeper.HasDexPair(ctx, denomQuote)
	require.NoError(t, err)
	require.False(t, found)
}
