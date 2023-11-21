package keeper

import (
	"bytes"
	"fmt"
	"sort"

	abci "github.com/cometbft/cometbft/abci/types"
	gogotypes "github.com/cosmos/gogoproto/types"

	"github.com/initia-labs/initia/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BlockValidatorUpdates calculates the ValidatorUpdates for the current block
// Called in each EndBlock
func (k Keeper) BlockValidatorUpdates(ctx sdk.Context) []abci.ValidatorUpdate {
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
		panic(err)
	}

	// unbond all mature validators from the unbonding queue
	k.UnbondAllMatureValidators(ctx)

	// Remove all mature unbonding delegations from the ubd queue.
	matureUnbonds := k.DequeueAllMatureUBDQueue(ctx, ctx.BlockHeader().Time)
	for _, dvPair := range matureUnbonds {
		addr, err := sdk.ValAddressFromBech32(dvPair.ValidatorAddress)
		if err != nil {
			panic(err)
		}
		delegatorAddress, err := sdk.AccAddressFromBech32(dvPair.DelegatorAddress)
		if err != nil {
			panic(err)
		}
		balances, err := k.CompleteUnbonding(ctx, delegatorAddress, addr)
		if err != nil {
			continue
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeCompleteUnbonding,
				sdk.NewAttribute(sdk.AttributeKeyAmount, balances.String()),
				sdk.NewAttribute(types.AttributeKeyValidator, dvPair.ValidatorAddress),
				sdk.NewAttribute(types.AttributeKeyDelegator, dvPair.DelegatorAddress),
			),
		)
	}

	// Remove all mature redelegations from the red queue.
	matureRedelegations := k.DequeueAllMatureRedelegationQueue(ctx, ctx.BlockHeader().Time)
	for _, dvvTriplet := range matureRedelegations {
		valSrcAddr, err := sdk.ValAddressFromBech32(dvvTriplet.ValidatorSrcAddress)
		if err != nil {
			panic(err)
		}
		valDstAddr, err := sdk.ValAddressFromBech32(dvvTriplet.ValidatorDstAddress)
		if err != nil {
			panic(err)
		}
		delegatorAddress := sdk.MustAccAddressFromBech32(dvvTriplet.DelegatorAddress)

		balances, err := k.CompleteRedelegation(
			ctx,
			delegatorAddress,
			valSrcAddr,
			valDstAddr,
		)
		if err != nil {
			continue
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeCompleteRedelegation,
				sdk.NewAttribute(sdk.AttributeKeyAmount, balances.String()),
				sdk.NewAttribute(types.AttributeKeyDelegator, dvvTriplet.DelegatorAddress),
				sdk.NewAttribute(types.AttributeKeySrcValidator, dvvTriplet.ValidatorSrcAddress),
				sdk.NewAttribute(types.AttributeKeyDstValidator, dvvTriplet.ValidatorDstAddress),
			),
		)
	}

	return validatorUpdates
}

// ApplyVotingPowerUpdates applies and return voting power weights changes to
// power update whitelist validators
func (k Keeper) ApplyVotingPowerUpdates(ctx sdk.Context) {
	weights := k.GetVotingPowerWeights(ctx)
	minVotingPower := k.MinVotingPower(ctx)
	k.IterateWhitelistValidator(ctx, func(validator types.Validator) bool {
		if validator.Jailed {
			panic("should never retrieve a jailed validator from the power whitelist")
		}

		k.DeleteValidatorByPowerIndex(ctx, validator)

		// Update validator power with its index.
		validator.VotingPower, validator.VotingPowers = types.CalculateVotingPower(validator.Tokens, weights)
		k.SetValidator(ctx, validator)
		k.SetValidatorByPowerIndex(ctx, validator)

		// Remove validator from the power update whitelist group.
		// Only way to get back to whitelist group is more delegations.
		if validator.VotingPower.LT(minVotingPower) {
			k.RemoveWhitelistValidator(ctx, validator)
		}

		return false
	})
}

// ApplyAndReturnValidatorSetUpdates applies and return accumulated updates to the bonded validator set. Also,
// * Updates the active valset as keyed by LastValidatorPowerKey.
// * Updates the total power as keyed by LastTotalPowerKey.
// * Updates validator status' according to updated powers.
// * Updates the fee pool bonded vs not-bonded tokens.
// * Updates relevant indices.
// It gets called once after genesis, another time maybe after genesis transactions,
// then once at every EndBlock.
//
// CONTRACT: Only validators with non-zero power or zero-power that were bonded
// at the previous block height or were removed from the validator set entirely
// are returned to Tendermint.
func (k Keeper) ApplyAndReturnValidatorSetUpdates(ctx sdk.Context) (updates []abci.ValidatorUpdate, err error) {
	k.ApplyVotingPowerUpdates(ctx)

	params := k.GetParams(ctx)
	maxValidators := params.MaxValidators
	powerReduction := k.PowerReduction(ctx)
	totalPower := sdk.ZeroInt()
	amtFromBondedToNotBonded, amtFromNotBondedToBonded := sdk.NewCoins(), sdk.NewCoins()

	// Retrieve the last validator set.
	// The persistent set is updated later in this function.
	// (see LastValidatorPowerKey).
	last, err := k.getLastValidatorsByAddr(ctx)
	if err != nil {
		return nil, err
	}

	// Iterate over validators, highest power to lowest.
	iterator := k.ValidatorsPowerStoreIterator(ctx)
	defer iterator.Close()

	for count := 0; iterator.Valid() && count < int(maxValidators); iterator.Next() {
		// everything that is iterated in this loop is becoming or already a
		// part of the bonded validator set
		valAddr := sdk.ValAddress(iterator.Value())
		validator := k.mustGetValidator(ctx, valAddr)
		if validator.Jailed {
			panic("should never retrieve a jailed validator from the power store")
		}

		// if we get to a zero-power validator (which we don't bond),
		// there are no more possible bonded validators
		if validator.PotentialConsensusPower(powerReduction) == 0 {
			break
		}

		// apply the appropriate state change if necessary
		switch {
		case validator.IsUnbonded():
			validator, err = k.unbondedToBonded(ctx, validator)
			if err != nil {
				return
			}
			amtFromNotBondedToBonded = amtFromNotBondedToBonded.Add(validator.GetTokens()...)
		case validator.IsUnbonding():
			validator, err = k.unbondingToBonded(ctx, validator)
			if err != nil {
				return
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
			return nil, err
		}
		oldPowerBytes, found := last[valAddrStr]
		newPower := validator.ConsensusPower(powerReduction)
		newPowerBytes := k.cdc.MustMarshal(&gogotypes.Int64Value{Value: newPower})

		// update the validator set if power has changed
		if !found || !bytes.Equal(oldPowerBytes, newPowerBytes) {
			updates = append(updates, validator.ABCIValidatorUpdate(powerReduction))

			k.SetLastValidatorPower(ctx, valAddr, newPower)
		}

		delete(last, valAddrStr)
		count++

		totalPower = totalPower.Add(sdk.NewInt(newPower))
	}

	noLongerBonded, err := sortNoLongerBonded(last)
	if err != nil {
		return nil, err
	}

	for _, valAddrBytes := range noLongerBonded {
		validator := k.mustGetValidator(ctx, sdk.ValAddress(valAddrBytes))
		validator, err = k.bondedToUnbonding(ctx, validator)
		if err != nil {
			return
		}
		amtFromBondedToNotBonded = amtFromBondedToNotBonded.Add(validator.GetTokens()...)
		k.DeleteLastValidatorPower(ctx, validator.GetOperator())
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
		k.notBondedTokensToBonded(ctx, amtFromNotBondedToBonded)
	}
	if !amtFromBondedToNotBonded.IsZero() {
		k.bondedTokensToNotBonded(ctx, amtFromBondedToNotBonded)
	}

	return updates, err
}

// Validator state transitions

func (k Keeper) bondedToUnbonding(ctx sdk.Context, validator types.Validator) (types.Validator, error) {
	if !validator.IsBonded() {
		panic(fmt.Sprintf("bad state transition bondedToUnbonding, validator: %v\n", validator))
	}

	return k.beginUnbondingValidator(ctx, validator)
}

func (k Keeper) unbondingToBonded(ctx sdk.Context, validator types.Validator) (types.Validator, error) {
	if !validator.IsUnbonding() {
		panic(fmt.Sprintf("bad state transition unbondingToBonded, validator: %v\n", validator))
	}

	return k.bondValidator(ctx, validator)
}

func (k Keeper) unbondedToBonded(ctx sdk.Context, validator types.Validator) (types.Validator, error) {
	if !validator.IsUnbonded() {
		panic(fmt.Sprintf("bad state transition unbondedToBonded, validator: %v\n", validator))
	}

	return k.bondValidator(ctx, validator)
}

// UnbondingToUnbonded switches a validator from unbonding state to unbonded state
func (k Keeper) UnbondingToUnbonded(ctx sdk.Context, validator types.Validator) types.Validator {
	if !validator.IsUnbonding() {
		panic(fmt.Sprintf("bad state transition unbondingToBonded, validator: %v\n", validator))
	}

	return k.completeUnbondingValidator(ctx, validator)
}

// send a validator to jail
func (k Keeper) jailValidator(ctx sdk.Context, validator types.Validator) {
	if validator.Jailed {
		panic(fmt.Sprintf("cannot jail already jailed validator, validator: %v\n", validator))
	}
	validator.Jailed = true
	k.SetValidator(ctx, validator)
	k.RemoveWhitelistValidator(ctx, validator)
}

// remove a validator from jail
func (k Keeper) unjailValidator(ctx sdk.Context, validator types.Validator) {
	if !validator.Jailed {
		panic(fmt.Sprintf("cannot unjail already unjailed validator, validator: %v\n", validator))
	}

	validator.Jailed = false
	k.SetValidator(ctx, validator)

	// check voting power is enough to get into power update whitelist.
	votingPower := k.VotingPower(ctx, validator.Tokens)
	if votingPower.GTE(k.MinVotingPower(ctx)) {
		k.AddWhitelistValidator(ctx, validator)
	}
}

// perform all the store operations for when a validator status becomes bonded
func (k Keeper) bondValidator(ctx sdk.Context, validator types.Validator) (types.Validator, error) {

	// delete from queue if present
	k.DeleteValidatorQueue(ctx, validator)

	// TODO - why sdk do not delete index?
	k.DeleteUnbondingIndex(ctx, validator.UnbondingId)

	// resetting unbonding info should be done
	// after state update.
	validator = validator.
		UpdateStatus(types.Bonded).
		ResetUnbondingInfos()

	// save the now bonded validator record to the two referenced stores
	k.SetValidator(ctx, validator)

	// trigger hook
	consAddr, err := validator.GetConsAddr()
	if err != nil {
		return validator, err
	}

	if err := k.Hooks().AfterValidatorBonded(ctx, consAddr, validator.GetOperator()); err != nil {
		return validator, err
	}

	return validator, err
}

// perform all the store operations for when a validator begins unbonding
func (k Keeper) beginUnbondingValidator(ctx sdk.Context, validator types.Validator) (types.Validator, error) {
	params := k.GetParams(ctx)

	// sanity check
	if validator.Status != types.Bonded {
		panic(fmt.Sprintf("should not already be unbonded or unbonding, validator: %v\n", validator))
	}

	unbondingId := k.IncrementUnbondingId(ctx)
	validator = validator.UpdateStatus(types.Unbonding)

	// set the unbonding completion time and completion height appropriately
	validator.UnbondingTime = ctx.BlockHeader().Time.Add(params.UnbondingTime)
	validator.UnbondingHeight = ctx.BlockHeader().Height

	// TODO - why sdk keep all unbonding ids?
	validator.UnbondingId = unbondingId

	// save the now unbonded validator record and power index
	k.SetValidator(ctx, validator)

	// Adds to unbonding validator queue
	k.InsertUnbondingValidatorQueue(ctx, validator)

	// trigger hook
	consAddr, err := validator.GetConsAddr()
	if err != nil {
		return validator, err
	}

	if err := k.Hooks().AfterValidatorBeginUnbonding(ctx, consAddr, validator.GetOperator()); err != nil {
		return validator, err
	}

	k.SetValidatorByUnbondingId(ctx, validator, unbondingId)
	if err := k.Hooks().AfterUnbondingInitiated(ctx, unbondingId); err != nil {
		return validator, err
	}

	return validator, nil
}

// perform all the store operations for when a validator status becomes unbonded
func (k Keeper) completeUnbondingValidator(ctx sdk.Context, validator types.Validator) types.Validator {
	validator = validator.UpdateStatus(types.Unbonded)
	k.SetValidator(ctx, validator)

	return validator
}

// map of operator bech32-addresses to serialized power
// We use bech32 strings here, because we can't have slices as keys: map[[]byte][]byte
type validatorsByAddr map[string][]byte

// get the last validator set
func (k Keeper) getLastValidatorsByAddr(ctx sdk.Context) (validatorsByAddr, error) {
	last := make(validatorsByAddr)

	iterator := k.LastValidatorsIterator(ctx)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// extract the validator address from the key (prefix is 1-byte, addrLen is 1-byte)
		valAddr := types.AddressFromLastValidatorPowerKey(iterator.Key())
		valAddrStr, err := sdk.Bech32ifyAddressBytes(sdk.GetConfig().GetBech32ValidatorAddrPrefix(), valAddr)
		if err != nil {
			return nil, err
		}

		powerBytes := iterator.Value()
		last[valAddrStr] = make([]byte, len(powerBytes))
		copy(last[valAddrStr], powerBytes)
	}

	return last, nil
}

// given a map of remaining validators to previous bonded power
// returns the list of validators to be unbonded, sorted by operator address
func sortNoLongerBonded(last validatorsByAddr) ([][]byte, error) {
	// sort the map keys for determinism
	noLongerBonded := make([][]byte, len(last))
	index := 0

	for valAddrStr := range last {
		valAddrBytes, err := sdk.ValAddressFromBech32(valAddrStr)
		if err != nil {
			return nil, err
		}
		noLongerBonded[index] = valAddrBytes
		index++
	}
	// sorted by address - order doesn't matter
	sort.SliceStable(noLongerBonded, func(i, j int) bool {
		// -1 means strictly less than
		return bytes.Compare(noLongerBonded[i], noLongerBonded[j]) == -1
	})

	return noLongerBonded, nil
}
