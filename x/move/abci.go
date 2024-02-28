package move

import (
	"context"
	"time"

	"cosmossdk.io/core/address"
	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"

	"github.com/cosmos/cosmos-sdk/telemetry"

	vmtypes "github.com/initia-labs/movevm/types"
)

func BeginBlocker(ctx context.Context, k keeper.Keeper, vc address.Codec) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)

	// skip staking sweep operations when staking keeper is not registered,
	// this is for minitia
	if k.StakingKeeper == nil {
		return nil
	}

	// get rewards from active validators
	activeValidators, err := k.StakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return err
	}

	params, err := k.RewardKeeper.GetParams(ctx)
	if err != nil {
		return err
	}
	rewardDenom := params.RewardDenom

	denoms := []string{}
	rewardVecMap := make(map[string][]uint64)
	valAddrVecMap := make(map[string][][]byte)
	for _, activeValidator := range activeValidators {
		valAddrStr := activeValidator.GetOperator()
		valAddr, err := vc.StringToBytes(activeValidator.GetOperator())
		if err != nil {
			return err
		}

		rewardPools, err := k.WithdrawRewards(ctx, valAddr)
		if err != nil {
			return err
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
			valAddrVecMap[poolDenom] = append(valAddrVecMap[poolDenom], []byte(valAddrStr))
		}
	}

	for _, poolDenom := range denoms {
		rewardArg, err := vmtypes.SerializeUint64Vector(rewardVecMap[poolDenom])
		if err != nil {
			return err
		}

		valArg, err := vmtypes.SerializeBytesVector(valAddrVecMap[poolDenom])
		if err != nil {
			return err
		}

		metadata, err := types.MetadataAddressFromDenom(poolDenom)
		if err != nil {
			return err
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
			return err
		}
	}

	return nil
}
