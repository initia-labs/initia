package move

import (
	"time"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	vmtypes "github.com/initia-labs/initiavm/types"
)

func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)

	// skip staking sweep operations when staking keeper is not registered,
	// this is for minitia
	if k.StakingKeeper == nil {
		return
	}

	// get rewards from active validators
	activeValidators := k.StakingKeeper.GetBondedValidatorsByPower(ctx)
	rewardDenom := k.RewardKeeper.GetParams(ctx).RewardDenom

	denoms := []string{}
	rewardVecMap := make(map[string][]uint64)
	valAddrVecMap := make(map[string][][]byte)
	for _, activeValidator := range activeValidators {
		valAddr := activeValidator.GetOperator()

		rewardPools, err := k.WithdrawRewards(ctx, valAddr)
		if err != nil {
			panic(err)
		}

		for _, pool := range rewardPools {
			poolDenom := pool.Denom
			if _, found := rewardVecMap[poolDenom]; !found {
				denoms = append(denoms, poolDenom)
				rewardVecMap[poolDenom] = make([]uint64, 0, len(activeValidators))
				valAddrVecMap[poolDenom] = make([][]byte, 0, len(activeValidators))
			}

			rewardAmount := pool.Coins.AmountOf(rewardDenom)
			if rewardAmount.IsZero() {
				continue
			}

			rewardVecMap[poolDenom] = append(rewardVecMap[poolDenom], rewardAmount.Uint64())
			valAddrVecMap[poolDenom] = append(valAddrVecMap[poolDenom], []byte(valAddr.String()))
		}
	}

	for _, poolDenom := range denoms {
		rewardArg, err := vmtypes.SerializeUint64Vector(rewardVecMap[poolDenom])
		if err != nil {
			panic(err)
		}

		valArg, err := vmtypes.SerializeBytesVector(valAddrVecMap[poolDenom])
		if err != nil {
			panic(err)
		}

		metadata, err := types.MetadataAddressFromDenom(poolDenom)
		if err != nil {
			panic(err)
		}

		args := [][]byte{metadata[:], valArg, rewardArg}
		if err = k.ExecuteEntryFunction(
			ctx,
			vmtypes.StdAddress,
			vmtypes.StdAddress,
			types.MoveModuleNameStaking,
			types.FunctionNameStakingDepositReward,
			[]vmtypes.TypeTag{},
			args,
		); err != nil {
			panic(err)
		}
	}
}
