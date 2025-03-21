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

	denomLP, err := types.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP)
	require.NoError(t, err)

	// check staking params update
	bondDenoms, err := input.StakingKeeper.BondDenoms(ctx)
	require.NoError(t, err)
	require.Contains(t, bondDenoms, denomLP)

	// check distribution params update
	rewardWeights, err := input.DistKeeper.GetRewardWeights(ctx)
	require.NoError(t, err)
	require.Contains(t, rewardWeights, distrtypes.RewardWeight{Denom: denomLP, Weight: math.LegacyOneDec()})

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
	bondDenoms, err = input.StakingKeeper.BondDenoms(ctx)
	require.NoError(t, err)
	require.NotContains(t, bondDenoms, denomLP)

	// check distribution params update
	rewardWeights, err = input.DistKeeper.GetRewardWeights(ctx)
	require.NoError(t, err)
	require.NotContains(t, rewardWeights, distrtypes.RewardWeight{Denom: denomLP, Weight: math.LegacyOneDec()})

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

	denomLP, err := types.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP)
	require.NoError(t, err)

	// check staking params update
	bondDenoms, err := input.StakingKeeper.BondDenoms(ctx)
	require.NoError(t, err)
	require.Contains(t, bondDenoms, denomLP)

	// check distribution params update
	rewardWeights, err := input.DistKeeper.GetRewardWeights(ctx)
	require.NoError(t, err)
	require.Contains(t, rewardWeights, distrtypes.RewardWeight{Denom: denomLP, Weight: math.LegacyOneDec()})

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
	bondDenoms, err = input.StakingKeeper.BondDenoms(ctx)
	require.NoError(t, err)
	require.NotContains(t, bondDenoms, denomLP)

	// check distribution params update
	rewardWeights, err = input.DistKeeper.GetRewardWeights(ctx)
	require.NoError(t, err)
	require.NotContains(t, rewardWeights, distrtypes.RewardWeight{Denom: denomLP, Weight: math.LegacyOneDec()})

	// check move dex pair update
	found, err = dexKeeper.HasDexPair(ctx, denomQuote)
	require.NoError(t, err)
	require.False(t, found)
}

func TestWhitelistProposal_StableSwapPool(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	dexKeeper := keeper.NewDexKeeper(&input.MoveKeeper)

	// start stable swap creation

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
	// finish stable swap creation

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

	denomLP, err := types.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP)
	require.NoError(t, err)

	// check staking params update
	bondDenoms, err := input.StakingKeeper.BondDenoms(ctx)
	require.NoError(t, err)
	require.Contains(t, bondDenoms, denomLP)

	// check distribution params update
	rewardWeights, err := input.DistKeeper.GetRewardWeights(ctx)
	require.NoError(t, err)
	require.Contains(t, rewardWeights, distrtypes.RewardWeight{Denom: denomLP, Weight: math.LegacyOneDec()})

	// dex pair registration was not performed since it is a stable swap pool
	_, err = dexKeeper.GetMetadataLP(ctx, denomCoinB)
	require.Error(t, err)
	_, err = dexKeeper.GetMetadataLP(ctx, denomCoinC)
	require.Error(t, err)

	// check move staking state
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
	bondDenoms, err = input.StakingKeeper.BondDenoms(ctx)
	require.NoError(t, err)
	require.NotContains(t, bondDenoms, denomLP)

	// check distribution params update
	rewardWeights, err = input.DistKeeper.GetRewardWeights(ctx)
	require.NoError(t, err)
	require.NotContains(t, rewardWeights, distrtypes.RewardWeight{Denom: denomLP, Weight: math.LegacyOneDec()})

	// check dex pair update (currently registration itself is not performed)
	found, err = dexKeeper.HasDexPair(ctx, denomCoinB)
	require.NoError(t, err)
	require.False(t, found)
	found, err = dexKeeper.HasDexPair(ctx, denomCoinC)
	require.NoError(t, err)
	require.False(t, found)
}
