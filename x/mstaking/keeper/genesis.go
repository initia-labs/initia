package keeper

import (
	"context"
	"fmt"
	"log"

	abci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/mstaking/types"
)

// InitGenesis sets the pool and parameters for the provided keeper.  For each
// validator in data, it sets that validator in the keeper along with manually
// setting the indexes. In addition, it also sets any delegations found in
// data. Finally, it updates the bonded validators.
// Returns final validator set after applying all declaration and delegations
func (k Keeper) InitGenesis(
	ctx context.Context, data *types.GenesisState,
) (res []abci.ValidatorUpdate) {
	bondedTokens := sdk.NewCoins()
	notBondedTokens := sdk.NewCoins()

	// We need to pretend to be "n blocks before genesis", where "n" is the
	// validator update delay, so that e.g. slashing periods are correctly
	// initialized for the validator set e.g. with a one-block offset - the
	// first TM block is at height 1, so state updates applied from
	// genesis.json are in block 0.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1 - sdk.ValidatorUpdateDelay)
	ctx = sdkCtx

	if err := k.SetParams(ctx, data.Params); err != nil {
		panic(err)
	}

	minVotingPower, err := k.MinVotingPower(ctx)
	if err != nil {
		panic(err)
	}

	for _, validator := range data.Validators {
		if err := k.SetValidator(ctx, validator); err != nil {
			panic(err)
		}

		votingPower, err := k.VotingPower(ctx, validator.Tokens)
		if err != nil {
			panic(err)
		}

		if votingPower.GTE(minVotingPower) {
			if err := k.AddWhitelistValidator(ctx, validator); err != nil {
				panic(err)
			}
		}

		// Manually set indices for the first time
		if err := k.SetValidatorByConsAddr(ctx, validator); err != nil {
			panic(err)
		}

		if err := k.SetValidatorByPowerIndex(ctx, validator); err != nil {
			panic(err)
		}

		// Call the creation hook if not exported
		if !data.Exported {
			valAddr, err := k.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
			if err != nil {
				panic(err)
			}

			if err := k.Hooks().AfterValidatorCreated(ctx, valAddr); err != nil {
				panic(err)
			}
		}

		// update timeslice if necessary
		if validator.IsUnbonding() {
			if err := k.InsertUnbondingValidatorQueue(ctx, validator); err != nil {
				panic(err)
			}
		}

		switch validator.GetStatus() {
		case types.Bonded:
			bondedTokens = bondedTokens.Add(validator.GetTokens()...)
		case types.Unbonding, types.Unbonded:
			notBondedTokens = notBondedTokens.Add(validator.GetTokens()...)
		default:
			panic("invalid validator status")
		}
	}

	for _, delegation := range data.Delegations {
		delAddr, err := k.authKeeper.AddressCodec().StringToBytes(delegation.DelegatorAddress)
		if err != nil {
			panic(fmt.Errorf("invalid delegator address: %s", err))
		}

		valAddr, err := k.validatorAddressCodec.StringToBytes(delegation.GetValidatorAddr())
		if err != nil {
			panic(err)
		}

		// Call the before-creation hook if not exported
		if !data.Exported {
			if err := k.Hooks().BeforeDelegationCreated(ctx, delAddr, valAddr); err != nil {
				panic(err)
			}
		}

		if err := k.SetDelegation(ctx, delegation); err != nil {
			panic(err)
		}

		// Call the after-modification hook if not exported
		if !data.Exported {
			if err := k.Hooks().AfterDelegationModified(ctx, delAddr, valAddr); err != nil {
				panic(err)
			}
		}
	}

	for _, ubd := range data.UnbondingDelegations {
		if err := k.SetUnbondingDelegation(ctx, ubd); err != nil {
			panic(err)
		}

		for _, entry := range ubd.Entries {
			if err := k.InsertUBDQueue(ctx, ubd, entry.CompletionTime); err != nil {
				panic(err)
			}

			notBondedTokens = notBondedTokens.Add(entry.Balance...)
		}
	}

	for _, red := range data.Redelegations {
		if err := k.SetRedelegation(ctx, red); err != nil {
			panic(err)
		}

		for _, entry := range red.Entries {
			if err := k.InsertRedelegationQueue(ctx, red, entry.CompletionTime); err != nil {
				panic(err)
			}
		}
	}

	// check if the unbonded and bonded pools accounts exists
	bondedPool := k.GetBondedPool(ctx)
	if bondedPool == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.BondedPoolName))
	}

	// TODO remove with genesis 2-phases refactor https://github.com/cosmos/cosmos-sdk/issues/2862
	bondedBalance := k.bankKeeper.GetAllBalances(ctx, bondedPool.GetAddress())
	if bondedBalance.IsZero() {
		k.authKeeper.SetModuleAccount(ctx, bondedPool)
	}

	// if balance is different from bonded coins panic because genesis is most likely malformed
	if !bondedBalance.Equal(bondedTokens) {
		panic(fmt.Sprintf("bonded pool balance is different from bonded coins: %s <-> %s", bondedBalance, bondedTokens))
	}

	notBondedPool := k.GetNotBondedPool(ctx)
	if notBondedPool == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.NotBondedPoolName))
	}

	notBondedBalance := k.bankKeeper.GetAllBalances(ctx, notBondedPool.GetAddress())
	if notBondedBalance.IsZero() {
		k.authKeeper.SetModuleAccount(ctx, notBondedPool)
	}

	// if balance is different from non bonded coins panic because genesis is most likely malformed
	if !notBondedBalance.Equal(notBondedTokens) {
		panic(fmt.Sprintf("not bonded pool balance is different from not bonded coins: %s <-> %s", notBondedBalance, notBondedTokens))
	}

	if err := k.SetNextUnbondingId(ctx, data.NextUnbondingId); err != nil {
		panic(err)
	}

	// don't need to run Tendermint updates if we exported
	if data.Exported {
		for _, lv := range data.LastValidatorPowers {
			valAddr, err := k.validatorAddressCodec.StringToBytes(lv.Address)
			if err != nil {
				panic(err)
			}

			err = k.SetLastValidatorConsPower(ctx, valAddr, lv.Power)
			if err != nil {
				panic(err)
			}

			validator, err := k.Validators.Get(ctx, valAddr)
			if err != nil {
				panic(fmt.Sprintf("validator %s not found", lv.Address))
			}

			update := validator.ABCIValidatorUpdate(k.PowerReduction(ctx))
			update.Power = lv.Power // keep the next-val-set offset, use the last power for the first block
			res = append(res, update)
		}
	} else {
		var err error

		res, err = k.ApplyAndReturnValidatorSetUpdates(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}

	return res
}

// ExportGenesis returns a GenesisState for a given context and keeper. The
// GenesisState will contain the pool, params, validators, and bonds found in
// the keeper.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	var unbondingDelegations []types.UnbondingDelegation

	err := k.IterateUnbondingDelegations(ctx, func(ubd types.UnbondingDelegation) (stop bool, err error) {
		unbondingDelegations = append(unbondingDelegations, ubd)
		return false, nil
	})
	if err != nil {
		panic(err)
	}

	var redelegations []types.Redelegation

	err = k.IterateRedelegations(ctx, func(red types.Redelegation) (stop bool, err error) {
		redelegations = append(redelegations, red)
		return false, nil
	})
	if err != nil {
		panic(err)
	}

	var lastValidatorPowers []types.LastValidatorPower

	err = k.IterateLastValidatorConsPowers(ctx, func(addr sdk.ValAddress, power int64) (stop bool, err error) {
		valAddr, err := k.validatorAddressCodec.BytesToString(addr)
		if err != nil {
			return true, err
		}

		lastValidatorPowers = append(lastValidatorPowers, types.LastValidatorPower{Address: valAddr, Power: power})
		return false, nil
	})
	if err != nil {
		panic(err)
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}

	allDelegations, err := k.GetAllDelegations(ctx)
	if err != nil {
		panic(err)
	}

	allValidators, err := k.GetAllValidators(ctx)
	if err != nil {
		panic(err)
	}

	nextUnbondingId, err := k.GetNextUnbondingId(ctx)
	if err != nil {
		panic(err)
	}

	return &types.GenesisState{
		Params:               params,
		LastValidatorPowers:  lastValidatorPowers,
		Validators:           allValidators,
		Delegations:          allDelegations,
		UnbondingDelegations: unbondingDelegations,
		Redelegations:        redelegations,
		Exported:             true,
		NextUnbondingId:      nextUnbondingId,
	}
}
