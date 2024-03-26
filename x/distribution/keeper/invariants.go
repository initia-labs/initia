package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

// register all distribution invariants
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "nonnegative-outstanding",
		NonNegativeOutstandingInvariant(k))
	ir.RegisterRoute(types.ModuleName, "can-withdraw",
		CanWithdrawInvariant(k))
	ir.RegisterRoute(types.ModuleName, "reference-count",
		ReferenceCountInvariant(k))
	ir.RegisterRoute(types.ModuleName, "module-account",
		ModuleAccountInvariant(k))
}

// AllInvariants runs all invariants of the distribution module
func AllInvariants(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		res, stop := CanWithdrawInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		res, stop = NonNegativeOutstandingInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		res, stop = ReferenceCountInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		return ModuleAccountInvariant(k)(ctx)
	}
}

// NonNegativeOutstandingInvariant checks that outstanding unwithdrawn fees are never negative
func NonNegativeOutstandingInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var msg string
		var count int
		var outstanding customtypes.DecPools

		err := k.ValidatorOutstandingRewards.Walk(ctx, nil, func(addr []byte, rewards customtypes.ValidatorOutstandingRewards) (stop bool, err error) {
			outstanding = rewards.GetRewards()
			if outstanding.IsAnyNegative() {
				count++
				msg += fmt.Sprintf("\t%v has negative outstanding coins: %v\n", sdk.ValAddress(addr), outstanding)
			}
			return false, nil
		})
		if err != nil {
			panic(err)
		}
		broken := count != 0

		return sdk.FormatInvariant(types.ModuleName, "nonnegative outstanding",
			fmt.Sprintf("found %d validators with negative outstanding rewards\n%s", count, msg)), broken
	}
}

// CanWithdrawInvariant checks that current rewards can be completely withdrawn
func CanWithdrawInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {

		// cache, we don't want to write changes
		ctx, _ = ctx.CacheContext()

		var remaining customtypes.DecPools

		valDelegationAddrs := make(map[string][]sdk.AccAddress)
		allDelegations, err := k.stakingKeeper.GetAllSDKDelegations(ctx)
		if err != nil {
			panic(err)
		}
		for _, del := range allDelegations {
			delAddr, err := k.authKeeper.AddressCodec().StringToBytes(del.GetDelegatorAddr())
			if err != nil {
				panic(err)
			}
			valAddr := del.GetValidatorAddr()
			valDelegationAddrs[valAddr] = append(valDelegationAddrs[valAddr], delAddr)
		}

		// iterate over all validators
		err = k.stakingKeeper.IterateValidators(ctx, func(val stakingtypes.ValidatorI) (stop bool, err error) {
			valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
			if err != nil {
				return false, err
			}

			_, _ = k.WithdrawValidatorCommission(ctx, valAddr)

			delegationAddrs, ok := valDelegationAddrs[val.GetOperator()]
			if ok {
				for _, delAddr := range delegationAddrs {
					if _, err := k.WithdrawDelegationRewards(ctx, delAddr, valAddr); err != nil {
						return false, err
					}
				}
			}

			remaining, err = k.GetValidatorOutstandingRewardsPools(ctx, valAddr)
			if err != nil {
				return false, err
			}

			return remaining.IsAnyNegative(), nil
		})
		if err != nil {
			panic(err)
		}

		broken := remaining.IsAnyNegative()
		return sdk.FormatInvariant(types.ModuleName, "can withdraw",
			fmt.Sprintf("remaining coins: %v\n", remaining)), broken
	}
}

// ReferenceCountInvariant checks that the number of historical rewards records is correct
func ReferenceCountInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {

		valCount := uint64(0)
		err := k.stakingKeeper.IterateValidators(ctx, func(val stakingtypes.ValidatorI) (stop bool, err error) {
			valCount++
			return false, nil
		})
		if err != nil {
			panic(err)
		}

		dels, err := k.stakingKeeper.GetAllSDKDelegations(ctx)
		if err != nil {
			panic(err)
		}

		slashCount := uint64(0)
		err = k.ValidatorSlashEvents.Walk(ctx, nil,
			func(_ collections.Triple[[]byte, uint64, uint64], _ customtypes.ValidatorSlashEvent) (stop bool, err error) {
				slashCount++
				return false, nil
			})
		if err != nil {
			panic(err)
		}

		// one record per validator (last tracked period), one record per
		// delegation (previous period), one record per slash (previous period)
		expected := valCount + uint64(len(dels)) + slashCount
		count, err := k.GetValidatorHistoricalReferenceCount(ctx)
		if err != nil {
			panic(err)
		}

		broken := count != expected

		return sdk.FormatInvariant(types.ModuleName, "reference count",
			fmt.Sprintf("expected historical reference count: %d = %v validators + %v delegations + %v slashes\n"+
				"total validator historical reference count: %d\n",
				expected, valCount, len(dels), slashCount, count)), broken
	}
}

// ModuleAccountInvariant checks that the coins held by the distr ModuleAccount
// is consistent with the sum of validator outstanding rewards
func ModuleAccountInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {

		var expectedCoins sdk.DecCoins
		err := k.ValidatorOutstandingRewards.Walk(ctx, nil, func(_ []byte, rewards customtypes.ValidatorOutstandingRewards) (stop bool, err error) {
			expectedCoins = expectedCoins.Add(rewards.Rewards.Sum()...)
			return false, nil
		})
		if err != nil {
			panic(err)
		}

		feePool, err := k.FeePool.Get(ctx)
		if err != nil {
			panic(err)
		}

		communityPool := feePool.CommunityPool
		expectedInt, _ := expectedCoins.Add(communityPool...).TruncateDecimal()

		macc := k.GetDistributionAccount(ctx)
		balances := k.bankKeeper.GetAllBalances(ctx, macc.GetAddress())

		// It is hard to block fungible asset transfer from move side,
		// so we can't guarantee the ModuleAccount is always equal to the sum
		// of the distribution rewards and community pool. Instead, we decide to check
		// if the ModuleAccount is greater than or equal to the sum of the distribution rewards
		// and community pool.
		broken := !balances.IsAllGTE(expectedInt)
		return sdk.FormatInvariant(
			types.ModuleName, "ModuleAccount coins",
			fmt.Sprintf("\texpected ModuleAccount coins:     %s\n"+
				"\tdistribution ModuleAccount coins: %s\n",
				expectedInt, balances,
			),
		), broken
	}
}
