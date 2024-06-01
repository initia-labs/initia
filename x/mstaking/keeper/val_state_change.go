package keeper

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/initia-labs/initia/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BlockValidatorUpdates calculates the ValidatorUpdates for the current block
// Called in each EndBlock
func (k Keeper) BlockValidatorUpdates(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	// Calculate validator set changes.
	//
	// NOTE: ApplyAndReturnValidatorSetUpdates has to come before
	// UnbondAllMatureValidatorQueue.
	// This fixes a bug when the unbonding period is instant (is the case in
	// some of the tests). The test expected the validator to be completely
	// unbonded after the Endblocker (go from Bonded -> Unbonding during
	// ApplyAndReturnValidatorSetUpdates and then Unbonding -> Unbonded during
	// UnbondAllMatureValidatorQueue).
	validatorUpdates, err := k.ApplyAndReturnValidatorSetUpdates(ctx)
	if err != nil {
		return nil, err
	}

	// unbond all mature validators from the unbonding queue
	if err := k.UnbondAllMatureValidators(ctx); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Remove all mature unbonding delegations from the ubd queue.
	matureUnbonds, err := k.DequeueAllMatureUBDQueue(ctx, sdkCtx.BlockHeader().Time)
	if err != nil {
		return nil, err
	}

	for _, dvPair := range matureUnbonds {
		addr, err := k.validatorAddressCodec.StringToBytes(dvPair.ValidatorAddress)
		if err != nil {
			return nil, err
		}

		delegatorAddress, err := k.authKeeper.AddressCodec().StringToBytes(dvPair.DelegatorAddress)
		if err != nil {
			return nil, err
		}

		balances, err := k.CompleteUnbonding(ctx, delegatorAddress, addr)
		if err != nil {
			continue
		}

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeCompleteUnbonding,
				sdk.NewAttribute(sdk.AttributeKeyAmount, balances.String()),
				sdk.NewAttribute(types.AttributeKeyValidator, dvPair.ValidatorAddress),
				sdk.NewAttribute(types.AttributeKeyDelegator, dvPair.DelegatorAddress),
			),
		)
	}

	// Remove all mature redelegations from the red queue.
	matureRedelegations, err := k.DequeueAllMatureRedelegationQueue(ctx, sdkCtx.BlockHeader().Time)
	if err != nil {
		return nil, err
	}

	for _, dvvTriplet := range matureRedelegations {
		valSrcAddr, err := k.validatorAddressCodec.StringToBytes(dvvTriplet.ValidatorSrcAddress)
		if err != nil {
			return nil, err
		}
		valDstAddr, err := k.validatorAddressCodec.StringToBytes(dvvTriplet.ValidatorDstAddress)
		if err != nil {
			return nil, err
		}
		delegatorAddress, err := k.authKeeper.AddressCodec().StringToBytes(dvvTriplet.DelegatorAddress)
		if err != nil {
			return nil, err
		}

		balances, err := k.CompleteRedelegation(
			ctx,
			delegatorAddress,
			valSrcAddr,
			valDstAddr,
		)
		if err != nil {
			continue
		}

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeCompleteRedelegation,
				sdk.NewAttribute(sdk.AttributeKeyAmount, balances.String()),
				sdk.NewAttribute(types.AttributeKeyDelegator, dvvTriplet.DelegatorAddress),
				sdk.NewAttribute(types.AttributeKeySrcValidator, dvvTriplet.ValidatorSrcAddress),
				sdk.NewAttribute(types.AttributeKeyDstValidator, dvvTriplet.ValidatorDstAddress),
			),
		)
	}

	return validatorUpdates, nil
}

// ApplyVotingPowerUpdates applies and return voting power weights changes to
// power update whitelist validators
func (k Keeper) ApplyVotingPowerUpdates(ctx context.Context) error {
	weights, err := k.GetVotingPowerWeights(ctx)
	if err != nil {
		return err
	}

	minVotingPower, err := k.MinVotingPower(ctx)
	if err != nil {
		return err
	}

	return k.IterateWhitelistValidator(ctx, func(validator types.Validator) (bool, error) {
		if validator.Jailed {
			panic("should never retrieve a jailed validator from the power whitelist")
		}

		if err := k.DeleteValidatorByPowerIndex(ctx, validator); err != nil {
			return true, err
		}

		// Update validator power with its index.
		validator.VotingPower, validator.VotingPowers = types.CalculateVotingPower(validator.Tokens, weights)
		if err := k.SetValidator(ctx, validator); err != nil {
			return true, err
		}

		if err := k.SetValidatorByPowerIndex(ctx, validator); err != nil {
			return true, err
		}

		// Remove validator from the power update whitelist group.
		// Only way to get back to whitelist group is more delegations.
		if validator.VotingPower.LT(minVotingPower) {
			if err := k.RemoveWhitelistValidator(ctx, validator); err != nil {
				return true, err
			}
		}

		return false, nil
	})
}

// ApplyAndReturnValidatorSetUpdates applies and return accumulated updates to the bonded validator set. Also,
// * Updates the active valset as keyed by LastValidatorConsPowerKey.
// * Updates validator status' according to updated powers.
// * Updates the fee pool bonded vs not-bonded tokens.
// * Updates relevant indices.
// It gets called once after genesis, another time maybe after genesis transactions,
// then once at every EndBlock.
//
// CONTRACT: Only validators with non-zero power or zero-power that were bonded
// at the previous block height or were removed from the validator set entirely
// are returned to Tendermint.
func (k Keeper) ApplyAndReturnValidatorSetUpdates(ctx context.Context) (updates []abci.ValidatorUpdate, err error) {
	if err := k.ApplyVotingPowerUpdates(ctx); err != nil {
		return nil, err
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	maxValidators := params.MaxValidators
	powerReduction := k.PowerReduction(ctx)
	amtFromBondedToNotBonded, amtFromNotBondedToBonded := sdk.NewCoins(), sdk.NewCoins()

	// Retrieve the last validator set.
	// The persistent set is updated later in this function.
	// (see LastValidatorPowerKey).
	last, err := k.getLastValidatorsByAddr(ctx)
	if err != nil {
		return nil, err
	}

	counter := 0

	// Iterate over validators, highest power to lowest.
	err = k.ValidatorsByConsPowerIndex.Walk(ctx, new(collections.PairRange[int64, []byte]).Descending(), func(key collections.Pair[int64, []byte], value bool) (stop bool, err error) {
		valAddr := key.K2()
		validator := k.mustGetValidator(ctx, valAddr)

		if validator.Jailed {
			panic("should never retrieve a jailed validator from the power store")
		}

		// if we get to a zero-power validator (which we don't bond),
		// there are no more possible bonded validators
		if validator.PotentialConsensusPower(powerReduction) == 0 {
			return true, nil
		}

		// apply the appropriate state change if necessary
		switch {
		case validator.IsUnbonded():
			validator, err = k.unbondedToBonded(ctx, validator)
			if err != nil {
				return true, err
			}
			amtFromNotBondedToBonded = amtFromNotBondedToBonded.Add(validator.GetTokens()...)
		case validator.IsUnbonding():
			validator, err = k.unbondingToBonded(ctx, validator)
			if err != nil {
				return true, err
			}
			amtFromNotBondedToBonded = amtFromNotBondedToBonded.Add(validator.GetTokens()...)
		case validator.IsBonded():
			// no state change
		default:
			panic("unexpected validator status")
		}

		// fetch the old power bytes
		valAddrStr, err := sdk.Bech32ifyAddressBytes(sdk.GetConfig().GetBech32ValidatorAddrPrefix(), valAddr)
		if err != nil {
			return true, err
		}

		oldPower, found := last[valAddrStr]
		newPower := validator.ConsensusPower(powerReduction)

		// update the validator set if power has changed
		if !found || oldPower != newPower {
			updates = append(updates, validator.ABCIValidatorUpdate(powerReduction))

			err = k.SetLastValidatorConsPower(ctx, valAddr, newPower)
			if err != nil {
				return true, err
			}
		}

		delete(last, valAddrStr)
		counter++

		return counter == int(maxValidators), nil
	})
	if err != nil {
		return nil, err
	}

	noLongerBonded, err := sortNoLongerBonded(last, k.validatorAddressCodec)
	if err != nil {
		return nil, err
	}

	for _, valAddr := range noLongerBonded {
		validator := k.mustGetValidator(ctx, sdk.ValAddress(valAddr))
		validator, err = k.bondedToUnbonding(ctx, validator)
		if err != nil {
			return nil, err
		}

		amtFromBondedToNotBonded = amtFromBondedToNotBonded.Add(validator.GetTokens()...)
		if err := k.DeleteLastValidatorConsPower(ctx, valAddr); err != nil {
			return nil, err
		}
		updates = append(updates, validator.ABCIValidatorUpdateZero())
	}

	// Update the pools based on the recent updates in the validator set:
	// - The tokens from the non-bonded candidates that enter the new validator set need to be transferred
	// to the Bonded pool.
	// - The tokens from the bonded validators that are being kicked out from the validator set
	// need to be transferred to the NotBonded pool.
	diff, _ := amtFromNotBondedToBonded.SafeSub(amtFromBondedToNotBonded...)
	amtFromNotBondedToBonded, amtFromBondedToNotBonded = sdk.NewCoins(), sdk.NewCoins()
	for _, coin := range diff {
		if coin.IsNegative() {
			amtFromBondedToNotBonded = append(amtFromBondedToNotBonded, sdk.NewCoin(coin.Denom, coin.Amount.Neg()))
		} else if coin.IsPositive() {
			amtFromNotBondedToBonded = append(amtFromNotBondedToBonded, sdk.NewCoin(coin.Denom, coin.Amount))
		}
	}
	if !amtFromNotBondedToBonded.IsZero() {
		if err := k.notBondedTokensToBonded(ctx, amtFromNotBondedToBonded); err != nil {
			return nil, err
		}
	}
	if !amtFromBondedToNotBonded.IsZero() {
		if err := k.bondedTokensToNotBonded(ctx, amtFromBondedToNotBonded); err != nil {
			return nil, err
		}
	}

	return updates, err
}

// Validator state transitions

func (k Keeper) bondedToUnbonding(ctx context.Context, validator types.Validator) (types.Validator, error) {
	if !validator.IsBonded() {
		panic(fmt.Sprintf("bad state transition bondedToUnbonding, validator: %v\n", validator))
	}

	return k.beginUnbondingValidator(ctx, validator)
}

func (k Keeper) unbondingToBonded(ctx context.Context, validator types.Validator) (types.Validator, error) {
	if !validator.IsUnbonding() {
		panic(fmt.Sprintf("bad state transition unbondingToBonded, validator: %v\n", validator))
	}

	return k.bondValidator(ctx, validator)
}

func (k Keeper) unbondedToBonded(ctx context.Context, validator types.Validator) (types.Validator, error) {
	if !validator.IsUnbonded() {
		panic(fmt.Sprintf("bad state transition unbondedToBonded, validator: %v\n", validator))
	}

	return k.bondValidator(ctx, validator)
}

// UnbondingToUnbonded switches a validator from unbonding state to unbonded state
func (k Keeper) UnbondingToUnbonded(ctx context.Context, validator types.Validator) (types.Validator, error) {
	if !validator.IsUnbonding() {
		panic(fmt.Sprintf("bad state transition unbondingToBonded, validator: %v\n", validator))
	}

	return k.completeUnbondingValidator(ctx, validator)
}

// send a validator to jail
func (k Keeper) jailValidator(ctx context.Context, validator types.Validator) error {
	if validator.Jailed {
		return types.ErrValidatorJailed.Wrapf("cannot jail already jailed validator, validator: %v", validator)
	}

	validator.Jailed = true
	if err := k.SetValidator(ctx, validator); err != nil {
		return err
	}
	if err := k.RemoveWhitelistValidator(ctx, validator); err != nil {
		return err
	}

	return nil
}

// remove a validator from jail
func (k Keeper) unjailValidator(ctx context.Context, validator types.Validator) error {
	if !validator.Jailed {
		return fmt.Errorf("cannot unjail already unjailed validator, validator: %v", validator)
	}

	validator.Jailed = false
	if err := k.SetValidator(ctx, validator); err != nil {
		return err
	}

	// check voting power is enough to get into power update whitelist.
	votingPower, err := k.VotingPower(ctx, validator.Tokens)
	if err != nil {
		return err
	}

	minVotingPower, err := k.MinVotingPower(ctx)
	if err != nil {
		return err
	}

	if votingPower.GTE(minVotingPower) {
		if err := k.AddWhitelistValidator(ctx, validator); err != nil {
			return err
		}
	}

	return nil
}

// perform all the store operations for when a validator status becomes bonded
func (k Keeper) bondValidator(ctx context.Context, validator types.Validator) (types.Validator, error) {

	// delete from queue if present
	if err := k.DeleteValidatorQueue(ctx, validator); err != nil {
		return types.Validator{}, err
	}

	// TODO - why sdk do not delete index?
	if err := k.DeleteUnbondingIndex(ctx, validator.UnbondingId); err != nil {
		return types.Validator{}, err
	}

	// resetting unbonding info should be done
	// after state update.
	validator = validator.
		UpdateStatus(types.Bonded).
		ResetUnbondingInfos()

	// save the now bonded validator record to the two referenced stores
	if err := k.SetValidator(ctx, validator); err != nil {
		return types.Validator{}, err
	}

	// trigger hook
	consAddr, err := validator.GetConsAddr()
	if err != nil {
		return types.Validator{}, err
	}

	valAddr, err := k.validatorAddressCodec.StringToBytes(validator.GetOperator())
	if err != nil {
		return types.Validator{}, err
	}

	if err := k.Hooks().AfterValidatorBonded(ctx, consAddr, valAddr); err != nil {
		return types.Validator{}, err
	}

	return validator, nil
}

// perform all the store operations for when a validator begins unbonding
func (k Keeper) beginUnbondingValidator(ctx context.Context, validator types.Validator) (types.Validator, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return types.Validator{}, err
	}

	// sanity check
	if validator.Status != types.Bonded {
		panic(fmt.Sprintf("should not already be unbonded or unbonding, validator: %v\n", validator))
	}

	unbondingId, err := k.IncrementUnbondingId(ctx)
	if err != nil {
		return types.Validator{}, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	validator = validator.UpdateStatus(types.Unbonding)

	// set the unbonding completion time and completion height appropriately
	validator.UnbondingTime = sdkCtx.BlockHeader().Time.Add(params.UnbondingTime)
	validator.UnbondingHeight = sdkCtx.BlockHeader().Height

	// TODO - why sdk keep all unbonding ids?
	validator.UnbondingId = unbondingId

	// save the now unbonded validator record and power index
	if err := k.SetValidator(ctx, validator); err != nil {
		return types.Validator{}, err
	}

	// Adds to unbonding validator queue
	if err := k.InsertUnbondingValidatorQueue(ctx, validator); err != nil {
		return types.Validator{}, err
	}

	// trigger hook
	consAddr, err := validator.GetConsAddr()
	if err != nil {
		return types.Validator{}, err
	}

	valAddr, err := k.validatorAddressCodec.StringToBytes(validator.GetOperator())
	if err != nil {
		return types.Validator{}, err
	}

	if err := k.Hooks().AfterValidatorBeginUnbonding(ctx, consAddr, valAddr); err != nil {
		return types.Validator{}, err
	}

	if err := k.SetValidatorByUnbondingId(ctx, validator, unbondingId); err != nil {
		return types.Validator{}, err
	}

	if err := k.Hooks().AfterUnbondingInitiated(ctx, unbondingId); err != nil {
		return types.Validator{}, err
	}

	return validator, nil
}

// perform all the store operations for when a validator status becomes unbonded
func (k Keeper) completeUnbondingValidator(ctx context.Context, validator types.Validator) (types.Validator, error) {
	validator = validator.UpdateStatus(types.Unbonded)
	if err := k.SetValidator(ctx, validator); err != nil {
		return types.Validator{}, err
	}

	return validator, nil
}

// map of operator bech32-addresses to serialized power
// We use bech32 strings here, because we can't have slices as keys: map[[]byte]int64
type validatorsByAddr map[string]int64

// get the last validator set
func (k Keeper) getLastValidatorsByAddr(ctx context.Context) (validatorsByAddr, error) {
	last := make(validatorsByAddr)

	err := k.LastValidatorConsPowers.Walk(ctx, nil, func(valAddr []byte, power int64) (stop bool, err error) {
		if valAddrStr, err := k.validatorAddressCodec.BytesToString(valAddr); err != nil {
			return true, err
		} else {
			last[valAddrStr] = power
		}

		return false, nil
	})

	return last, err
}

// given a map of remaining validators to previous bonded power
// returns the list of validators to be unbonded, sorted by operator address
func sortNoLongerBonded(last validatorsByAddr, vc address.Codec) ([][]byte, error) {
	// sort the map keys for determinism
	noLongerBonded := make([][]byte, len(last))
	index := 0

	for valAddrStr := range last {
		valAddr, err := vc.StringToBytes(valAddrStr)
		if err != nil {
			return nil, err
		}

		noLongerBonded[index] = valAddr
		index++
	}

	// sorted by address - order doesn't matter
	sort.SliceStable(noLongerBonded, func(i, j int) bool {
		// -1 means strictly less than
		return bytes.Compare(noLongerBonded[i], noLongerBonded[j]) == -1
	})

	return noLongerBonded, nil
}
