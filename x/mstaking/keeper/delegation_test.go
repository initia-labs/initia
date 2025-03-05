package keeper_test

import (
	"slices"
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	movekeeper "github.com/initia-labs/initia/v1/x/move/keeper"
	movetypes "github.com/initia-labs/initia/v1/x/move/types"
	"github.com/initia-labs/initia/v1/x/mstaking/types"
	vmtypes "github.com/initia-labs/movevm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

	metadataLP = movetypes.NamedObjectAddress(vmtypes.TestAddress, denomLP)
	movekeeper.NewDexKeeper(&input.MoveKeeper).SetDexPair(ctx, movetypes.DexPair{
		MetadataQuote: metadataQuote.String(),
		MetadataLP:    metadataLP.String(),
	})

	return metadataLP
}

// tests GetDelegation, GetDelegatorDelegations, SetDelegation, RemoveDelegation, GetDelegatorDelegations
func Test_Delegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	_, err := input.StakingKeeper.GetDelegation(ctx, addrs[0], valAddrs[0])
	require.ErrorIs(t, err, collections.ErrNotFound)

	delegation := types.NewDelegation(addrsStr[0], valAddrsStr[0], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(100))))
	delegation2 := types.NewDelegation(addrsStr[0], valAddrsStr[1], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(100))))

	require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation))
	resDelegation, err := input.StakingKeeper.GetDelegation(ctx, addrs[0], valAddrs[0])
	require.NoError(t, err)
	require.Equal(t, delegation, resDelegation)

	require.NoError(t, input.StakingKeeper.RemoveDelegation(ctx, delegation))
	_, err = input.StakingKeeper.GetDelegation(ctx, addrs[0], valAddrs[0])
	require.ErrorIs(t, err, collections.ErrNotFound)

	// set two delegations
	require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation))
	require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation2))

	delegations, err := input.StakingKeeper.GetDelegatorDelegations(ctx, addrs[0], 1)
	require.NoError(t, err)
	require.Len(t, delegations, 1)

	delegations, err = input.StakingKeeper.GetDelegatorDelegations(ctx, addrs[0], 2)
	require.NoError(t, err)
	require.Len(t, delegations, 2)

	for _, resDelegation := range delegations {
		if resDelegation.GetValidatorAddr() == valAddrsStr[0] {
			require.Equal(t, delegation, resDelegation)
		} else {
			require.Equal(t, delegation2, resDelegation)
		}
	}
}

func Test_GetValidatorDelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	delegation1 := types.NewDelegation(addrsStr[0], valAddrsStr[0], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(1))))
	delegation2 := types.NewDelegation(addrsStr[1], valAddrsStr[0], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(2))))
	delegation3 := types.NewDelegation(addrsStr[2], valAddrsStr[0], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(3))))
	delegation4 := types.NewDelegation(addrsStr[0], valAddrsStr[1], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(3))))

	require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation1))
	require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation2))
	require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation3))
	require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation4))

	delegations, err := input.StakingKeeper.GetValidatorDelegations(ctx, valAddrs[0])
	require.NoError(t, err)

	for _, resDelegation := range delegations {
		if resDelegation.GetDelegatorAddr() == addrsStr[0] {
			require.Equal(t, delegation1, resDelegation)
		} else if resDelegation.GetDelegatorAddr() == addrsStr[1] {
			require.Equal(t, delegation2, resDelegation)
		} else {
			require.Equal(t, delegation3, resDelegation)
		}
	}
}

func Test_GetAllDelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	delegation1 := types.NewDelegation(addrsStr[0], valAddrsStr[0], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(1))))
	delegation2 := types.NewDelegation(addrsStr[1], valAddrsStr[0], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(2))))
	delegation3 := types.NewDelegation(addrsStr[0], valAddrsStr[1], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(3))))
	delegation4 := types.NewDelegation(addrsStr[1], valAddrsStr[1], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(3))))

	require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation1))
	require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation2))
	require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation3))
	require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation4))

	delegations, err := input.StakingKeeper.GetValidatorDelegations(ctx, valAddrs[0])
	require.NoError(t, err)

	for _, resDelegation := range delegations {
		if resDelegation.GetDelegatorAddr() == addrsStr[0] {
			if resDelegation.GetValidatorAddr() == valAddrsStr[0] {
				require.Equal(t, delegation1, resDelegation)
			} else {
				require.Equal(t, delegation3, resDelegation)
			}
		} else if resDelegation.GetValidatorAddr() == valAddrsStr[0] {
			require.Equal(t, delegation2, resDelegation)
		} else {
			require.Equal(t, delegation4, resDelegation)
		}
	}

	require.NoError(t, input.StakingKeeper.IterateAllDelegations(ctx, func(resDelegation types.Delegation) (bool, error) {
		if resDelegation.GetDelegatorAddr() == addrsStr[0] {
			if resDelegation.GetValidatorAddr() == valAddrsStr[0] {
				require.Equal(t, delegation1, resDelegation)
			} else {
				require.Equal(t, delegation3, resDelegation)
			}
		} else if resDelegation.GetValidatorAddr() == valAddrsStr[0] {
			require.Equal(t, delegation2, resDelegation)
		} else {
			require.Equal(t, delegation4, resDelegation)
		}

		return false, nil
	}))
}

func Test_GetDelegatorDelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	delegation1 := types.NewDelegation(addrsStr[0], valAddrsStr[0], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(1))))
	delegation2 := types.NewDelegation(addrsStr[1], valAddrsStr[0], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(2))))
	delegation3 := types.NewDelegation(addrsStr[0], valAddrsStr[1], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(3))))
	delegation4 := types.NewDelegation(addrsStr[1], valAddrsStr[1], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(3))))

	input.StakingKeeper.SetDelegation(ctx, delegation1)
	input.StakingKeeper.SetDelegation(ctx, delegation2)
	input.StakingKeeper.SetDelegation(ctx, delegation3)
	input.StakingKeeper.SetDelegation(ctx, delegation4)

	delegations, err := input.StakingKeeper.GetDelegatorDelegations(ctx, addrs[0], 10)
	require.NoError(t, err)
	for _, resDelegation := range delegations {
		if resDelegation.GetValidatorAddr() == valAddrsStr[0] {
			require.Equal(t, delegation1, resDelegation)
		} else {
			require.Equal(t, delegation3, resDelegation)
		}
	}
}

func Test_UnbondingDelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	completeTime := time.Now().UTC()
	unbondingCoins1 := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	unbondingCoins2 := sdk.NewCoins(sdk.NewCoin("foo", math.NewInt(100)))
	unbondingCoins3 := sdk.NewCoins(sdk.NewCoin("bar", math.NewInt(100)))
	unbondingDelegation1 := types.NewUnbondingDelegation(addrsStr[0], valAddrsStr[0], 10, completeTime, unbondingCoins1, 1)
	unbondingDelegation2 := types.NewUnbondingDelegation(addrsStr[1], valAddrsStr[1], 10, completeTime, unbondingCoins2, 2)
	input.StakingKeeper.IncrementUnbondingId(ctx)
	input.StakingKeeper.IncrementUnbondingId(ctx)

	require.NoError(t, input.StakingKeeper.SetUnbondingDelegation(ctx, unbondingDelegation1))
	require.NoError(t, input.StakingKeeper.SetUnbondingDelegation(ctx, unbondingDelegation2))

	resUnbondingDelegation1, err := input.StakingKeeper.GetUnbondingDelegation(ctx, addrs[0], valAddrs[0])
	require.NoError(t, err)
	require.Equal(t, unbondingDelegation1, resUnbondingDelegation1)

	resUnbondingDelegation2, err := input.StakingKeeper.GetUnbondingDelegation(ctx, addrs[1], valAddrs[1])
	require.NoError(t, err)
	require.Equal(t, unbondingDelegation2, resUnbondingDelegation2)

	ubde, err := input.StakingKeeper.SetUnbondingDelegationEntry(ctx, addrs[0], valAddrs[0], 5, completeTime, unbondingCoins3)
	require.NoError(t, err)
	require.Equal(t, types.UnbondingDelegation{
		DelegatorAddress: addrsStr[0],
		ValidatorAddress: valAddrsStr[0],
		Entries: []types.UnbondingDelegationEntry{
			{
				CreationHeight: 10,
				CompletionTime: completeTime,
				InitialBalance: unbondingCoins1,
				Balance:        unbondingCoins1,
				UnbondingId:    1,
			},
			{
				CreationHeight: 5,
				CompletionTime: completeTime,
				InitialBalance: unbondingCoins3,
				Balance:        unbondingCoins3,
				UnbondingId:    3,
			},
		},
	}, ubde)

	require.NoError(t, input.StakingKeeper.RemoveUnbondingDelegation(ctx, unbondingDelegation1))
	_, err = input.StakingKeeper.GetUnbondingDelegation(ctx, addrs[0], valAddrs[0])
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func Test_GetUnbondingDelegationsFromValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	completeTime := time.Now().UTC()
	unbondingCoins1 := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	unbondingCoins2 := sdk.NewCoins(sdk.NewCoin("foo", math.NewInt(100)))
	unbondingDelegation1 := types.NewUnbondingDelegation(addrsStr[0], valAddrsStr[0], 10, completeTime, unbondingCoins1, 1)
	unbondingDelegation2 := types.NewUnbondingDelegation(addrsStr[1], valAddrsStr[0], 10, completeTime, unbondingCoins2, 2)

	require.NoError(t, input.StakingKeeper.SetUnbondingDelegation(ctx, unbondingDelegation1))
	require.NoError(t, input.StakingKeeper.SetUnbondingDelegation(ctx, unbondingDelegation2))

	unbondingDelegations, err := input.StakingKeeper.GetUnbondingDelegationsFromValidator(ctx, valAddrs[0])
	require.NoError(t, err)

	for _, resUnbondingDelegation := range unbondingDelegations {
		if resUnbondingDelegation.DelegatorAddress == addrsStr[0] {
			require.Equal(t, unbondingDelegation1, resUnbondingDelegation)
		} else {
			require.Equal(t, unbondingDelegation2, resUnbondingDelegation)
		}
	}
}

func Test_UBDQueue(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	completeTime := time.Now().UTC()

	dvPairs := []types.DVPair{
		{
			DelegatorAddress: addrsStr[0],
			ValidatorAddress: valAddrsStr[0],
		},
		{
			DelegatorAddress: addrsStr[1],
			ValidatorAddress: valAddrsStr[0],
		},
	}
	require.NoError(t, input.StakingKeeper.SetUBDQueueTimeSlice(ctx, completeTime, dvPairs))
	resDvPairs, err := input.StakingKeeper.GetUBDQueueTimeSlice(ctx, completeTime)
	require.NoError(t, err)
	require.Equal(t, dvPairs, resDvPairs)
	resDvPairs, err = input.StakingKeeper.DequeueAllMatureUBDQueue(ctx, completeTime.Add(time.Second))
	require.NoError(t, err)
	require.Equal(t, dvPairs, resDvPairs)

	completeTime1 := completeTime.Add(time.Second)
	completeTime2 := completeTime.Add(time.Hour)
	unbondingCoins1 := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	unbondingCoins2 := sdk.NewCoins(sdk.NewCoin("foo", math.NewInt(100)))
	unbondingDelegation1 := types.NewUnbondingDelegation(addrsStr[0], valAddrsStr[0], 10, completeTime1, unbondingCoins1, 1)
	unbondingDelegation2 := types.NewUnbondingDelegation(addrsStr[1], valAddrsStr[0], 10, completeTime2, unbondingCoins2, 2)

	require.NoError(t, input.StakingKeeper.InsertUBDQueue(ctx, unbondingDelegation1, completeTime1))
	require.NoError(t, input.StakingKeeper.InsertUBDQueue(ctx, unbondingDelegation2, completeTime1))
	require.NoError(t, input.StakingKeeper.InsertUBDQueue(ctx, unbondingDelegation2, completeTime2))

	resDvPairs, err = input.StakingKeeper.GetUBDQueueTimeSlice(ctx, completeTime1)
	require.NoError(t, err)
	require.Equal(t, dvPairs, resDvPairs)

	resDvPairs, err = input.StakingKeeper.GetUBDQueueTimeSlice(ctx, completeTime2)
	require.NoError(t, err)
	require.Equal(t, []types.DVPair{dvPairs[1]}, resDvPairs)
}

func Test_Redelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	completeTime := time.Now().UTC()
	amounts := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	shares := sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(100)))

	redelegation1 := types.NewRedelegation(addrsStr[0], valAddrsStr[0], valAddrsStr[1], 100, completeTime, amounts, shares, 1)
	redelegation2 := types.NewRedelegation(addrsStr[1], valAddrsStr[1], valAddrsStr[0], 100, completeTime, amounts, shares, 2)

	require.NoError(t, input.StakingKeeper.SetRedelegation(ctx, redelegation1))
	require.NoError(t, input.StakingKeeper.SetRedelegation(ctx, redelegation2))

	resRedelegation1, err1 := input.StakingKeeper.GetRedelegation(ctx, addrs[0], valAddrs[0], valAddrs[1])
	resRedelegation2, err2 := input.StakingKeeper.GetRedelegation(ctx, addrs[1], valAddrs[1], valAddrs[0])
	_, err3 := input.StakingKeeper.GetRedelegation(ctx, addrs[0], valAddrs[1], valAddrs[0])
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.ErrorIs(t, err3, collections.ErrNotFound)
	require.Equal(t, redelegation1, resRedelegation1)
	require.Equal(t, redelegation2, resRedelegation2)

	redelegations, err := input.StakingKeeper.GetRedelegationsFromSrcValidator(ctx, valAddrs[0])
	require.NoError(t, err)
	require.Equal(t, []types.Redelegation{redelegation1}, redelegations)

	redelegations, err = input.StakingKeeper.GetRedelegationsFromSrcValidator(ctx, valAddrs[1])
	require.NoError(t, err)
	require.Equal(t, []types.Redelegation{redelegation2}, redelegations)

	has, err := input.StakingKeeper.HasReceivingRedelegation(ctx, addrs[0], valAddrs[1])
	require.NoError(t, err)
	require.True(t, has)
	has, err = input.StakingKeeper.HasReceivingRedelegation(ctx, addrs[0], valAddrs[0])
	require.NoError(t, err)
	require.False(t, has)

	// max entry
	has, err = input.StakingKeeper.HasMaxRedelegationEntries(ctx, addrs[0], valAddrs[0], valAddrs[1])
	require.NoError(t, err)
	require.False(t, has)

	// set max entry to 1
	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.MaxEntries = 1
	require.NoError(t, input.StakingKeeper.SetParams(ctx, params))
	has, err = input.StakingKeeper.HasMaxRedelegationEntries(ctx, addrs[0], valAddrs[0], valAddrs[1])
	require.NoError(t, err)
	require.True(t, has)

	// back max entry to 7
	// set max entry to 1
	params.MaxEntries = 7
	input.StakingKeeper.SetParams(ctx, params)

	require.NoError(t, input.StakingKeeper.IterateRedelegations(ctx, func(resRedelegation types.Redelegation) (bool, error) {
		if resRedelegation.ValidatorSrcAddress == valAddrsStr[0] {
			require.Equal(t, redelegation1, resRedelegation)
		} else {
			require.Equal(t, redelegation2, resRedelegation)
		}
		return false, nil
	}))

	require.NoError(t, input.StakingKeeper.RemoveRedelegation(ctx, redelegation1))
	_, err = input.StakingKeeper.GetRedelegation(ctx, addrs[0], valAddrs[0], valAddrs[1])
	require.ErrorIs(t, err, collections.ErrNotFound)

	require.NoError(t, input.StakingKeeper.SetRedelegation(ctx, redelegation1))

	completeTime2 := completeTime.Add(time.Hour)
	amounts2 := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1000000)))
	shares2 := sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(1000000)))
	redelegation1.Entries = append(redelegation1.Entries, types.NewRedelegationEntry(
		110, completeTime2, amounts2, shares2, 1,
	))
	resRedelegation, err := input.StakingKeeper.SetRedelegationEntry(ctx, addrs[0], valAddrs[0], valAddrs[1], 110, completeTime2, amounts2, shares2)
	require.NoError(t, err)
	require.Equal(t, redelegation1, resRedelegation)
	resRedelegation, err = input.StakingKeeper.GetRedelegation(ctx, addrs[0], valAddrs[0], valAddrs[1])
	require.NoError(t, err)
	require.Equal(t, redelegation1, resRedelegation)
}

func Test_RedelegationQueue(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	completeTime := time.Now().UTC()

	dvvTriplets := []types.DVVTriplet{
		{
			DelegatorAddress:    addrsStr[0],
			ValidatorSrcAddress: valAddrsStr[0],
			ValidatorDstAddress: valAddrsStr[1],
		},
		{
			DelegatorAddress:    addrsStr[1],
			ValidatorSrcAddress: valAddrsStr[1],
			ValidatorDstAddress: valAddrsStr[0],
		},
	}

	require.NoError(t, input.StakingKeeper.SetRedelegationQueueTimeSlice(ctx, completeTime, dvvTriplets))
	resDvvTriplets, err := input.StakingKeeper.GetRedelegationQueueTimeSlice(ctx, completeTime)
	require.NoError(t, err)
	require.Equal(t, dvvTriplets, resDvvTriplets)
	resDvvTriplets, err = input.StakingKeeper.DequeueAllMatureRedelegationQueue(ctx, completeTime.Add(time.Second))
	require.NoError(t, err)
	require.Equal(t, dvvTriplets, resDvvTriplets)

	completeTime1 := completeTime.Add(time.Second)
	completeTime2 := completeTime.Add(time.Hour)
	redelegationCoins1 := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	redelegationCoins2 := sdk.NewCoins(sdk.NewCoin("foo", math.NewInt(100)))
	redelegationShares1 := sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(100)))
	redelegationShares2 := sdk.NewDecCoins(sdk.NewDecCoin("foo", math.NewInt(100)))
	redelegation1 := types.NewRedelegation(addrsStr[0], valAddrsStr[0], valAddrsStr[1], 10, completeTime1, redelegationCoins1, redelegationShares1, 1)
	redelegation2 := types.NewRedelegation(addrsStr[1], valAddrsStr[1], valAddrsStr[0], 10, completeTime2, redelegationCoins2, redelegationShares2, 2)

	require.NoError(t, input.StakingKeeper.InsertRedelegationQueue(ctx, redelegation1, completeTime1))
	require.NoError(t, input.StakingKeeper.InsertRedelegationQueue(ctx, redelegation2, completeTime1))
	require.NoError(t, input.StakingKeeper.InsertRedelegationQueue(ctx, redelegation2, completeTime2))

	resDvvTriplets, err = input.StakingKeeper.GetRedelegationQueueTimeSlice(ctx, completeTime1)
	require.NoError(t, err)
	require.Equal(t, dvvTriplets, resDvvTriplets)

	resDvvTriplets, err = input.StakingKeeper.GetRedelegationQueueTimeSlice(ctx, completeTime2)
	require.NoError(t, err)
	require.Equal(t, []types.DVVTriplet{dvvTriplets[1]}, resDvvTriplets)
}

func Test_Delegate(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// create dex to register second bond denom
	baseDenom := bondDenom
	metadataLP := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1))

	secondBondDenom, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP)
	require.NoError(t, err)

	// update params
	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.BondDenoms = append(params.BondDenoms, secondBondDenom)
	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr)
	require.NoError(t, err)

	firstCoin := sdk.NewCoin(bondDenom, math.NewInt(1_000_000))
	secondCoin := sdk.NewCoin(secondBondDenom, math.NewInt(2_500_000))
	bondCoins := sdk.NewCoins(firstCoin, secondCoin)
	delAddr := input.Faucet.NewFundedAccount(ctx, firstCoin)
	delAddrStr, err := input.AccountKeeper.AddressCodec().BytesToString(delAddr)
	require.NoError(t, err)

	// mint not possible for second bond denom, so transfer from the 0x1
	require.NoError(t, input.BankKeeper.SendCoins(ctx, movetypes.TestAddr, delAddr, sdk.NewCoins(secondCoin)))

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	delegation, err := input.StakingKeeper.GetDelegation(ctx, delAddr, valAddr)
	require.NoError(t, err)
	require.Equal(t, types.Delegation{
		DelegatorAddress: delAddrStr,
		ValidatorAddress: valAddrStr,
		Shares:           shares,
	}, delegation)
}

func Test_Unbond(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// create dex to register second bond denom
	baseDenom := bondDenom
	metadataLP := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1))

	secondBondDenom, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP)
	require.NoError(t, err)

	// update params
	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.BondDenoms = append(params.BondDenoms, secondBondDenom)
	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)

	firstCoin := sdk.NewCoin(bondDenom, math.NewInt(1_000_000))
	secondCoin := sdk.NewCoin(secondBondDenom, math.NewInt(2_500_000))
	bondCoins := sdk.NewCoins(firstCoin, secondCoin)
	delAddr := input.Faucet.NewFundedAccount(ctx, firstCoin)

	// mint not possible for second bond denom, so transfer from the 0x1
	require.NoError(t, input.BankKeeper.SendCoins(ctx, movetypes.TestAddr, delAddr, sdk.NewCoins(secondCoin)))

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	// unbond half
	unbondedAmount, err := input.StakingKeeper.Unbond(ctx, delAddr, valAddr, shares.QuoDec(math.LegacyNewDec(2)))
	require.NoError(t, err)
	halfCoins, _ := sdk.NewDecCoinsFromCoins(bondCoins...).QuoDec(math.LegacyNewDec(2)).TruncateDecimal()
	require.Equal(t, halfCoins, unbondedAmount)
}

func Test_UnbondAfterSlash(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// create dex to register second bond denom
	baseDenom := bondDenom
	metadataLP := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1))

	secondBondDenom, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP)
	require.NoError(t, err)

	// update params
	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.BondDenoms = append(params.BondDenoms, secondBondDenom)
	input.StakingKeeper.SetParams(ctx, params)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)

	firstCoin := sdk.NewCoin(bondDenom, math.NewInt(1_000_000))
	secondCoin := sdk.NewCoin(secondBondDenom, math.NewInt(2_600_000))
	bondCoins := sdk.NewCoins(firstCoin, secondCoin)
	delAddr := input.Faucet.NewFundedAccount(ctx, firstCoin)

	// mint not possible for second bond denom, so transfer from the 0x1
	require.NoError(t, input.BankKeeper.SendCoins(ctx, movetypes.TestAddr, delAddr, sdk.NewCoins(secondCoin)))

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	pubkey, err := validator.ConsPubKey()
	require.NoError(t, err)

	// update validator for voting power update
	_, err = input.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	validator, err = input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	power := validator.ConsensusPower(input.StakingKeeper.PowerReduction(ctx))
	require.Equal(t, int64(3), power)

	// 50% slashing
	_, err = input.StakingKeeper.Slash(ctx, pubkey.Address().Bytes(), 100, math.LegacyNewDecWithPrec(5, 1))
	require.NoError(t, err)

	// unbond half
	unbondedAmount, err := input.StakingKeeper.Unbond(ctx, delAddr, valAddr, shares.QuoDec(math.LegacyNewDec(2)))
	require.NoError(t, err)
	quarterCoins, _ := sdk.NewDecCoinsFromCoins(bondCoins...).QuoDec(math.LegacyNewDec(4)).TruncateDecimal()
	require.Equal(t, quarterCoins, unbondedAmount)
}

func Test_Undelegate(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// create dex to register second bond denom
	baseDenom := bondDenom
	metadataLP := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1))

	secondBondDenom, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP)
	require.NoError(t, err)

	// update params
	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.BondDenoms = append(params.BondDenoms, secondBondDenom)
	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)

	firstCoin := sdk.NewCoin(bondDenom, math.NewInt(1_000_000))
	secondCoin := sdk.NewCoin(secondBondDenom, math.NewInt(2_500_000))
	bondCoins := sdk.NewCoins(firstCoin, secondCoin)
	delAddr := input.Faucet.NewFundedAccount(ctx, firstCoin)

	// mint not possible for second bond denom, so transfer from the 0x1
	require.NoError(t, input.BankKeeper.SendCoins(ctx, movetypes.TestAddr, delAddr, sdk.NewCoins(secondCoin)))

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	// unbond half
	completeTime, _, err := input.StakingKeeper.Undelegate(ctx, delAddr, valAddr, shares.QuoDec(math.LegacyNewDec(2)))
	require.NoError(t, err)
	require.Equal(t, ctx.BlockHeader().Time.Add(params.UnbondingTime), completeTime)

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(params.UnbondingTime))
	unbondedCoins, err := input.StakingKeeper.CompleteUnbonding(ctx, delAddr, valAddr)
	require.NoError(t, err)

	halfCoins, _ := sdk.NewDecCoinsFromCoins(bondCoins...).QuoDec(math.LegacyNewDec(2)).TruncateDecimal()
	require.Equal(t, halfCoins, unbondedCoins)
}

func Test_BeginRedelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// create dex to register second bond denom
	baseDenom := bondDenom
	metadataLP := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1))

	secondBondDenom, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP)
	require.NoError(t, err)

	// update params
	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.BondDenoms = append(params.BondDenoms, secondBondDenom)
	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 2)

	firstCoin := sdk.NewCoin(bondDenom, math.NewInt(1_000_000))
	secondCoin := sdk.NewCoin(secondBondDenom, math.NewInt(2_500_000))
	bondCoins := sdk.NewCoins(firstCoin, secondCoin)
	delAddr := input.Faucet.NewFundedAccount(ctx, firstCoin)

	// mint not possible for second bond denom, so transfer from the 0x1
	require.NoError(t, input.BankKeeper.SendCoins(ctx, movetypes.TestAddr, delAddr, sdk.NewCoins(secondCoin)))

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	// redelegate half
	completeTime, err := input.StakingKeeper.BeginRedelegation(ctx, delAddr, valAddr, valAddr2, shares.QuoDec(math.LegacyNewDec(2)))
	require.NoError(t, err)
	require.Equal(t, ctx.BlockHeader().Time.Add(params.UnbondingTime), completeTime)

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(params.UnbondingTime))
	redelegatedCoins, err := input.StakingKeeper.CompleteRedelegation(ctx, delAddr, valAddr, valAddr2)
	require.NoError(t, err)

	halfCoins, _ := sdk.NewDecCoinsFromCoins(bondCoins...).QuoDec(math.LegacyNewDec(2)).TruncateDecimal()
	require.Equal(t, halfCoins, redelegatedCoins)
}

func Test_ValidateUnbondAmount(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// create dex to register second bond denom
	baseDenom := bondDenom
	metadataLP := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1))

	secondBondDenom, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP)
	require.NoError(t, err)

	// update params
	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.BondDenoms = append(params.BondDenoms, secondBondDenom)
	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)

	firstCoin := sdk.NewCoin(bondDenom, math.NewInt(1_000_000))
	secondCoin := sdk.NewCoin(secondBondDenom, math.NewInt(2_500_000))
	bondCoins := sdk.NewCoins(firstCoin, secondCoin)
	delAddr := input.Faucet.NewFundedAccount(ctx, firstCoin)

	// mint not possible for second bond denom, so transfer from the 0x1
	require.NoError(t, input.BankKeeper.SendCoins(ctx, movetypes.TestAddr, delAddr, sdk.NewCoins(secondCoin)))

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	unbondShares, err := input.StakingKeeper.ValidateUnbondAmount(ctx, delAddr, valAddr, bondCoins)
	require.NoError(t, err)
	require.Equal(t, shares, unbondShares)

	_, err = input.StakingKeeper.ValidateUnbondAmount(ctx, delAddr, valAddr, bondCoins.Add(sdk.NewCoin(bondDenom, math.OneInt())))
	require.Error(t, err)
}
