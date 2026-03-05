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

	// start dex creation
	baseDenom := bondDenom
	baseAmount := math.NewInt(1_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := math.NewInt(4_000_000_000_000)

	metadataLP := createBalancerPool(
		t, ctx, input,
		sdk.NewCoin(baseDenom, baseAmount), sdk.NewCoin(denomQuote, quoteAmount),
		math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1),
	)
	// finish dex creation

	// create publish operation proposal
	err := input.MoveKeeper.WhitelistStaking(ctx, types.MsgWhitelistStaking{
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

	found, err := input.MoveKeeper.HasStakingState(ctx, metadataLP)
	require.NoError(t, err)
	require.True(t, found)

	//
	// delist
	//

	err = input.MoveKeeper.DelistStaking(ctx, types.MsgDelistStaking{
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

}

func TestWhitelistProposalReverse(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// start dex creation
	baseDenom := bondDenom
	baseAmount := math.NewInt(1_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := math.NewInt(4_000_000_000_000)

	metadataLP := createBalancerPool(
		t, ctx, input,
		sdk.NewCoin(denomQuote, quoteAmount), sdk.NewCoin(baseDenom, baseAmount),
		math.LegacyNewDecWithPrec(2, 1), math.LegacyNewDecWithPrec(8, 1),
	)
	// finish dex creation

	// create publish operation proposal
	err := input.MoveKeeper.WhitelistStaking(ctx, types.MsgWhitelistStaking{
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

	found, err := input.MoveKeeper.HasStakingState(ctx, metadataLP)
	require.NoError(t, err)
	require.True(t, found)

	//
	// delist
	//

	err = input.MoveKeeper.DelistStaking(ctx, types.MsgDelistStaking{
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

}

func TestWhitelistProposal_StableSwapPool(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

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
	err := input.MoveKeeper.WhitelistStaking(ctx, types.MsgWhitelistStaking{
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

	// check move staking state
	found, err := input.MoveKeeper.HasStakingState(ctx, metadataLP)
	require.NoError(t, err)
	require.True(t, found)

	//
	// delist
	//

	err = input.MoveKeeper.DelistStaking(ctx, types.MsgDelistStaking{
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

}

func TestWhitelistGasPrice_Balancer(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	baseDenom := bondDenom
	denomQuote := "uusdc"

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	metadataLP := createBalancerPool(
		t, ctx, input,
		sdk.NewCoin(baseDenom, math.NewInt(1_000_000_000_000)),
		sdk.NewCoin(denomQuote, math.NewInt(4_000_000_000_000)),
		math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1),
	)

	err = input.MoveKeeper.WhitelistGasPrice(ctx, types.MsgWhitelistGasPrice{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
		Authority:     input.MoveKeeper.GetAuthority(),
	})
	require.NoError(t, err)

	found, err := keeper.NewDexKeeper(&input.MoveKeeper).HasDexPair(ctx, denomQuote)
	require.NoError(t, err)
	require.True(t, found)

	price, err := keeper.NewDexKeeper(&input.MoveKeeper).GetBaseSpotPrice(ctx, denomQuote)
	require.NoError(t, err)
	require.True(t, price.IsPositive())

	err = input.MoveKeeper.DelistGasPrice(ctx, types.MsgDelistGasPrice{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
		Authority:     input.MoveKeeper.GetAuthority(),
	})
	require.NoError(t, err)

	found, err = keeper.NewDexKeeper(&input.MoveKeeper).HasDexPair(ctx, denomQuote)
	require.NoError(t, err)
	require.False(t, found)
}

func TestWhitelistGasPrice_StableSwap(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	baseDenom := bondDenom
	denomCoinB := "milkINIT"
	denomCoinC := "ibiINIT"

	metadataCoinB, err := types.MetadataAddressFromDenom(denomCoinB)
	require.NoError(t, err)

	metadataLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(
			sdk.NewCoin(baseDenom, math.NewInt(1_000_000_000_000)),
			sdk.NewCoin(denomCoinB, math.NewInt(1_000_000_000_001)),
			sdk.NewCoin(denomCoinC, math.NewInt(1_000_000_000_002)),
		),
	)

	err = input.MoveKeeper.WhitelistGasPrice(ctx, types.MsgWhitelistGasPrice{
		MetadataQuote: metadataCoinB.String(),
		MetadataLP:    metadataLP.String(),
		Authority:     input.MoveKeeper.GetAuthority(),
	})
	require.NoError(t, err)

	found, err := keeper.NewDexKeeper(&input.MoveKeeper).HasDexPair(ctx, denomCoinB)
	require.NoError(t, err)
	require.True(t, found)

	price, err := keeper.NewDexKeeper(&input.MoveKeeper).GetBaseSpotPrice(ctx, denomCoinB)
	require.NoError(t, err)
	require.True(t, price.IsPositive())

	err = input.MoveKeeper.DelistGasPrice(ctx, types.MsgDelistGasPrice{
		MetadataQuote: metadataCoinB.String(),
		MetadataLP:    metadataLP.String(),
		Authority:     input.MoveKeeper.GetAuthority(),
	})
	require.NoError(t, err)

	found, err = keeper.NewDexKeeper(&input.MoveKeeper).HasDexPair(ctx, denomCoinB)
	require.NoError(t, err)
	require.False(t, found)
}

func TestWhitelistGasPrice_StableSwap_BaseDenomRejected(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	baseDenom := bondDenom
	denomCoinB := "milkINIT"
	denomCoinC := "ibiINIT"

	metadataBase, err := types.MetadataAddressFromDenom(baseDenom)
	require.NoError(t, err)

	metadataLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(
			sdk.NewCoin(baseDenom, math.NewInt(1_000_000_000_000)),
			sdk.NewCoin(denomCoinB, math.NewInt(1_000_000_000_001)),
			sdk.NewCoin(denomCoinC, math.NewInt(1_000_000_000_002)),
		),
	)

	err = input.MoveKeeper.WhitelistGasPrice(ctx, types.MsgWhitelistGasPrice{
		MetadataQuote: metadataBase.String(),
		MetadataLP:    metadataLP.String(),
		Authority:     input.MoveKeeper.GetAuthority(),
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot be base denom")

	found, err := keeper.NewDexKeeper(&input.MoveKeeper).HasDexPair(ctx, baseDenom)
	require.NoError(t, err)
	require.False(t, found)
}

func TestWhitelistGasPrice_CLAMM(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	baseDenom := bondDenom
	denomQuote := "uusdc"

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	params, err := input.MoveKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.ClammModuleAddress = cafeAddr.String()
	err = input.MoveKeeper.SetParams(ctx, params)
	require.NoError(t, err)

	// sqrt_price = 2^64 → price = 1.0 (base = metadata0)
	metadataLP := createCLAMMPool(t, ctx, input, baseDenom, denomQuote, 1, 0)

	err = input.MoveKeeper.WhitelistGasPrice(ctx, types.MsgWhitelistGasPrice{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
		Authority:     input.MoveKeeper.GetAuthority(),
	})
	require.NoError(t, err)

	found, err := keeper.NewDexKeeper(&input.MoveKeeper).HasDexPair(ctx, denomQuote)
	require.NoError(t, err)
	require.True(t, found)

	price, err := keeper.NewDexKeeper(&input.MoveKeeper).GetBaseSpotPrice(ctx, denomQuote)
	require.NoError(t, err)
	require.True(t, price.IsPositive())

	err = input.MoveKeeper.DelistGasPrice(ctx, types.MsgDelistGasPrice{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
		Authority:     input.MoveKeeper.GetAuthority(),
	})
	require.NoError(t, err)

	found, err = keeper.NewDexKeeper(&input.MoveKeeper).HasDexPair(ctx, denomQuote)
	require.NoError(t, err)
	require.False(t, found)
}

func TestWhitelistGasPrice_CLAMM_NoCLAMMInParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	baseDenom := bondDenom
	denomQuote := "uusdc"

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	metadataLP := createCLAMMPool(t, ctx, input, baseDenom, denomQuote, 1, 0)

	// params.ClammModuleAddress is empty — should fail
	err = input.MoveKeeper.WhitelistGasPrice(ctx, types.MsgWhitelistGasPrice{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
		Authority:     input.MoveKeeper.GetAuthority(),
	})
	require.Error(t, err)
}

func TestWhitelistGasPrice_DuplicateQuoteRejected(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	baseDenom := bondDenom
	denomQuote := "uusdc"

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	firstLP := createBalancerPool(
		t, ctx, input,
		sdk.NewCoin(baseDenom, math.NewInt(1_000_000_000_000)),
		sdk.NewCoin(denomQuote, math.NewInt(4_000_000_000_000)),
		math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1),
	)
	require.NoError(t, input.MoveKeeper.WhitelistGasPrice(ctx, types.MsgWhitelistGasPrice{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    firstLP.String(),
		Authority:     input.MoveKeeper.GetAuthority(),
	}))

	secondLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(
			sdk.NewCoin(baseDenom, math.NewInt(2_000_000_000_000)),
			sdk.NewCoin(denomQuote, math.NewInt(2_000_000_000_000)),
			sdk.NewCoin("milkINIT", math.NewInt(2_000_000_000_000)),
		),
	)
	err = input.MoveKeeper.WhitelistGasPrice(ctx, types.MsgWhitelistGasPrice{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    secondLP.String(),
		Authority:     input.MoveKeeper.GetAuthority(),
	})
	require.Error(t, err)

	registeredLP, err := keeper.NewDexKeeper(&input.MoveKeeper).GetMetadataLP(ctx, denomQuote)
	require.NoError(t, err)
	require.Equal(t, firstLP, registeredLP)
}

func TestDelistGasPrice_MismatchedLPRejected(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	baseDenom := bondDenom
	denomQuote := "uusdc"

	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	require.NoError(t, err)

	registeredLP := createBalancerPool(
		t, ctx, input,
		sdk.NewCoin(baseDenom, math.NewInt(1_000_000_000_000)),
		sdk.NewCoin(denomQuote, math.NewInt(4_000_000_000_000)),
		math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1),
	)
	require.NoError(t, input.MoveKeeper.WhitelistGasPrice(ctx, types.MsgWhitelistGasPrice{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    registeredLP.String(),
		Authority:     input.MoveKeeper.GetAuthority(),
	}))

	otherLP := createStableSwapPool(
		t, ctx, input,
		sdk.NewCoins(
			sdk.NewCoin(baseDenom, math.NewInt(2_000_000_000_000)),
			sdk.NewCoin(denomQuote, math.NewInt(2_000_000_000_000)),
			sdk.NewCoin("ibiINIT", math.NewInt(2_000_000_000_000)),
		),
	)
	err = input.MoveKeeper.DelistGasPrice(ctx, types.MsgDelistGasPrice{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    otherLP.String(),
		Authority:     input.MoveKeeper.GetAuthority(),
	})
	require.Error(t, err)

	registeredAfter, err := keeper.NewDexKeeper(&input.MoveKeeper).GetMetadataLP(ctx, denomQuote)
	require.NoError(t, err)
	require.Equal(t, registeredLP, registeredAfter)
}
