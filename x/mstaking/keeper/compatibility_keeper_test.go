package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	stakingkeeper "github.com/initia-labs/initia/x/mstaking/keeper"
	"github.com/stretchr/testify/require"
)

func Test_CompatibleValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	for i := 1; i <= 10; i++ {
		valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, i)

		compatilibityKeeper := stakingkeeper.NewCompatibilityKeeper(&input.StakingKeeper)
		comValI, err := compatilibityKeeper.Validator(ctx, valAddr)
		require.NoError(t, err)
		comVal := comValI.(cosmostypes.Validator)

		val, err := input.StakingKeeper.GetValidator(ctx, valAddr)
		require.NoError(t, err)

		require.Equal(t, val.VotingPower, comVal.Tokens)
		require.Equal(t, val.OperatorAddress, comVal.OperatorAddress)
		require.Equal(t, val.ConsensusPubkey, comVal.ConsensusPubkey)
		require.Equal(t, val.Jailed, comVal.Jailed)
		require.Equal(t, int32(val.Status), int32(comVal.Status))
		require.Equal(t, cosmostypes.Description(val.Description), comVal.Description)
		require.Equal(t, val.UnbondingHeight, comVal.UnbondingHeight)
		require.Equal(t, val.UnbondingTime, comVal.UnbondingTime)
		require.Equal(t, val.Commission.Rate, comVal.Commission.Rate)
		require.Equal(t, val.Commission.MaxRate, comVal.Commission.MaxRate)
		require.Equal(t, val.Commission.MaxChangeRate, comVal.Commission.MaxChangeRate)

		consAddr, err := val.GetConsAddr()
		require.NoError(t, err)

		comValI2, err := compatilibityKeeper.ValidatorByConsAddr(ctx, consAddr)
		require.Equal(t, comVal, comValI2.(cosmostypes.Validator))
	}
}

func Test_CompatibleTotalBondedTokens(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	sum := math.ZeroInt()
	compatilibityKeeper := stakingkeeper.NewCompatibilityKeeper(&input.StakingKeeper)

	for i := 1; i <= 10; i++ {
		valAddr := createValidatorWithBalance(ctx, input, 100_000_000, int64(1_000_000*i), i)

		comValI, err := compatilibityKeeper.Validator(ctx, valAddr)
		require.NoError(t, err)

		sum = sum.Add(comValI.GetBondedTokens())
	}

	comTotalBondedTokens, err := compatilibityKeeper.TotalBondedTokens(ctx)
	require.NoError(t, err)
	require.Equal(t, comTotalBondedTokens, sum)
}

func Test_CompatibleGetPubKeyByConsAddr(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	compatilibityKeeper := stakingkeeper.NewCompatibilityKeeper(&input.StakingKeeper)

	val, err := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)

	consAddr, err := val.GetConsAddr()
	require.NoError(t, err)

	pubKey, err := compatilibityKeeper.GetPubKeyByConsAddr(ctx, consAddr)
	require.NoError(t, err)

	valPubKey, err := val.CmtConsPublicKey()
	require.NoError(t, err)
	require.Equal(t, valPubKey, pubKey)
}
