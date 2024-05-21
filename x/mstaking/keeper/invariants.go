package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"github.com/initia-labs/initia/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterInvariants registers all staking invariants
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "module-accounts",
		ModuleAccountInvariants(k))
	ir.RegisterRoute(types.ModuleName, "nonnegative-power",
		NonNegativePowerInvariant(k))
	ir.RegisterRoute(types.ModuleName, "positive-delegation",
		PositiveDelegationInvariant(k))
	ir.RegisterRoute(types.ModuleName, "delegator-shares",
		DelegatorSharesInvariant(k))
}

// AllInvariants runs all invariants of the staking module.
func AllInvariants(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		res, stop := ModuleAccountInvariants(k)(ctx)
		if stop {
			return res, stop
		}

		res, stop = NonNegativePowerInvariant(k)(ctx)
		if stop {
			return res, stop
		}

		res, stop = PositiveDelegationInvariant(k)(ctx)
		if stop {
			return res, stop
		}

		return DelegatorSharesInvariant(k)(ctx)
	}
}

// ModuleAccountInvariants checks that the bonded and notBonded ModuleAccounts pools
// reflects the tokens actively bonded and not bonded
func ModuleAccountInvariants(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		bonded := sdk.NewCoins()
		notBonded := sdk.NewCoins()
		bondedPool := k.GetBondedPool(ctx)
		notBondedPool := k.GetNotBondedPool(ctx)

		err := k.IterateValidators(ctx, func(validator types.ValidatorI) (bool, error) {
			switch validator.GetStatus() {
			case types.Bonded:
				bonded = bonded.Add(validator.GetTokens()...)
			case types.Unbonding, types.Unbonded:
				notBonded = notBonded.Add(validator.GetTokens()...)
			default:
				panic("invalid validator status")
			}
			return false, nil
		})
		if err != nil {
			panic(err)
		}

		err = k.IterateUnbondingDelegations(ctx, func(ubd types.UnbondingDelegation) (bool, error) {
			for _, entry := range ubd.Entries {
				notBonded = notBonded.Add(entry.Balance...)
			}
			return false, nil
		})
		if err != nil {
			panic(err)
		}

		poolBonded := k.bankKeeper.GetAllBalances(ctx, bondedPool.GetAddress())
		poolNotBonded := k.bankKeeper.GetAllBalances(ctx, notBondedPool.GetAddress())

		// It is hard to block fungible asset transfer from move side,
		// so we can't guarantee  the pool is always equal to the sum
		// of the bonded validators. Instead, we decide to check if the pool
		// is greater than or equal to the sum of the bonded validators.
		broken := !poolBonded.IsAllGTE(bonded) || !poolNotBonded.IsAllGTE(notBonded)

		// Bonded tokens should equal sum of tokens with bonded validators
		// Not-bonded tokens should equal unbonding delegations	plus tokens on unbonded validators
		return sdk.FormatInvariant(types.ModuleName, "bonded and not bonded module account coins", fmt.Sprintf(
			"\tPool's bonded tokens: %v\n"+
				"\tsum of bonded tokens: %v\n"+
				"not bonded token invariance:\n"+
				"\tPool's not bonded tokens: %v\n"+
				"\tsum of not bonded tokens: %v\n"+
				"module accounts total (bonded + not bonded):\n"+
				"\tModule Accounts' tokens: %v\n"+
				"\tsum tokens:              %v\n",
			poolBonded, bonded, poolNotBonded, notBonded, poolBonded.Add(poolNotBonded...), bonded.Add(notBonded...))), broken
	}
}

// NonNegativePowerInvariant checks that all stored validators have >= 0 power.
func NonNegativePowerInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			msg    string
			broken bool
		)
		err := k.ValidatorsByConsPowerIndex.Walk(ctx, nil, func(key collections.Pair[int64, []byte], _ bool) (stop bool, err error) {
			power := key.K1()
			valAddr := key.K2()

			validator, err := k.Validators.Get(ctx, valAddr)
			if err != nil {
				panic(fmt.Sprintf("validator record not found for address: %X\n", valAddr))
			}

			if validator.GetConsensusPower(k.PowerReduction(ctx)) != power {
				broken = true
				msg += fmt.Sprintf("power store invariance:\n\tvalidator.Power: %v"+
					"\n\tkey should be: %v\n\tkey in store: %v\n",
					validator.GetConsensusPower(k.PowerReduction(ctx)), power, key)
			}

			if validator.Tokens.IsAnyNegative() {
				broken = true
				msg += fmt.Sprintf("\tnegative tokens for validator: %v\n", validator)
			}

			return false, nil
		})
		if err != nil {
			panic(err)
		}

		return sdk.FormatInvariant(types.ModuleName, "nonnegative power", fmt.Sprintf("found invalid validator powers\n%s", msg)), broken
	}
}

// PositiveDelegationInvariant checks that all stored delegations have > 0 shares.
func PositiveDelegationInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			msg   string
			count int
		)

		delegations, err := k.GetAllDelegations(ctx)
		if err != nil {
			panic(err)
		}

		for _, delegation := range delegations {
			if delegation.Shares.IsAnyNegative() {
				count++

				msg += fmt.Sprintf("\tdelegation with negative shares: %+v\n", delegation)
			}

			if delegation.Shares.IsZero() {
				count++

				msg += fmt.Sprintf("\tdelegation with zero shares: %+v\n", delegation)
			}
		}

		broken := count != 0

		return sdk.FormatInvariant(types.ModuleName, "positive delegations", fmt.Sprintf(
			"%d invalid delegations found\n%s", count, msg)), broken
	}
}

// DelegatorSharesInvariant checks whether all the delegator shares which persist
// in the delegator object add up to the correct total delegator shares
// amount stored in each validator.
func DelegatorSharesInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			msg    string
			broken bool
		)

		validators, err := k.GetAllValidators(ctx)
		if err != nil {
			panic(err)
		}

		validatorsDelegationShares := map[string]sdk.DecCoins{}

		for _, validator := range validators {
			validatorsDelegationShares[validator.GetOperator()] = sdk.NewDecCoins()
		}

		// iterate through all the delegations to calculate the total delegation shares for each validator
		delegations, err := k.GetAllDelegations(ctx)
		if err != nil {
			panic(err)
		}

		for _, delegation := range delegations {
			delegationValidatorAddr := delegation.GetValidatorAddr()
			validatorDelegationShares := validatorsDelegationShares[delegationValidatorAddr]
			validatorsDelegationShares[delegationValidatorAddr] = validatorDelegationShares.Add(delegation.Shares...)
		}

		// for each validator, check if its total delegation shares calculated from the step above equals to its expected delegation shares
		for _, validator := range validators {
			expValTotalDelShares := validator.GetDelegatorShares()
			calculatedValTotalDelShares := validatorsDelegationShares[validator.GetOperator()]
			if !calculatedValTotalDelShares.Equal(expValTotalDelShares) {
				broken = true
				msg += fmt.Sprintf("broken delegator shares invariance:\n"+
					"\tvalidator.DelegatorShares: %v\n"+
					"\tsum of Delegator.Shares: %v\n", expValTotalDelShares, calculatedValTotalDelShares)
			}
		}

		return sdk.FormatInvariant(types.ModuleName, "delegator shares", msg), broken
	}
}
