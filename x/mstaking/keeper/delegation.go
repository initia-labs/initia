package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/initia-labs/initia/x/mstaking/types"
)

// GetDelegation returns a specific delegation
func (k Keeper) GetDelegation(
	ctx context.Context,
	delAddr sdk.AccAddress,
	valAddr sdk.ValAddress,
) (delegation types.Delegation, err error) {
	return k.Delegations.Get(ctx, collections.Join([]byte(delAddr), []byte(valAddr)))
}

// IterateAllDelegations iterate through all of the delegations
func (k Keeper) IterateAllDelegations(
	ctx context.Context,
	cb func(delegation types.Delegation) (stop bool, err error),
) error {
	return k.Delegations.Walk(ctx, nil, func(key collections.Pair[[]byte, []byte], delegation types.Delegation) (stop bool, err error) {
		return cb(delegation)
	})
}

// GetAllDelegations returns all delegations used during genesis dump
func (k Keeper) GetAllDelegations(ctx context.Context) (delegations []types.Delegation, err error) {
	err = k.IterateAllDelegations(ctx, func(delegation types.Delegation) (bool, error) {
		delegations = append(delegations, delegation)
		return false, nil
	})

	return delegations, err
}

// GetValidatorDelegations returns all delegations to a specific validator. Useful for querier.
func (k Keeper) GetValidatorDelegations(
	ctx context.Context,
	valAddr sdk.ValAddress,
) (delegations []types.Delegation, err error) {
	err = k.DelegationsByValIndex.Walk(ctx, collections.NewPrefixedPairRange[[]byte, []byte](valAddr), func(key collections.Pair[[]byte, []byte], _ bool) (stop bool, err error) {
		valAddr, delAddr := key.K1(), key.K2()
		delegation, err := k.GetDelegation(ctx, delAddr, valAddr)
		if err != nil {
			return true, err
		}

		delegations = append(delegations, delegation)
		return false, nil
	})

	return delegations, err
}

// GetDelegatorDelegations returns a given amount of all the delegations from a delegator
func (k Keeper) GetDelegatorDelegations(
	ctx context.Context,
	delegator sdk.AccAddress,
	maxRetrieve uint16,
) (delegations []types.Delegation, err error) {
	delegations = make([]types.Delegation, 0, maxRetrieve)
	err = k.Delegations.Walk(ctx, collections.NewPrefixedPairRange[[]byte, []byte](delegator), func(key collections.Pair[[]byte, []byte], delegation types.Delegation) (stop bool, err error) {
		delegations = append(delegations, delegation)
		return len(delegations) == int(maxRetrieve), nil
	})

	return delegations, err
}

// SetDelegation sets a delegation
func (k Keeper) SetDelegation(ctx context.Context, delegation types.Delegation) error {
	delAddr, err := k.authKeeper.AddressCodec().StringToBytes(delegation.DelegatorAddress)
	if err != nil {
		return err
	}

	valAddr, err := k.validatorAddressCodec.StringToBytes(delegation.GetValidatorAddr())
	if err != nil {
		return err
	}

	if err := k.Delegations.Set(ctx, collections.Join(delAddr, valAddr), delegation); err != nil {
		return err
	}

	return k.DelegationsByValIndex.Set(ctx, collections.Join(valAddr, delAddr), true)
}

// RemoveDelegation removes a delegation
func (k Keeper) RemoveDelegation(ctx context.Context, delegation types.Delegation) error {
	delAddr, err := k.authKeeper.AddressCodec().StringToBytes(delegation.DelegatorAddress)
	if err != nil {
		return err
	}
	valAddr, err := k.validatorAddressCodec.StringToBytes(delegation.GetValidatorAddr())
	if err != nil {
		return err
	}

	// TODO: Consider calling hooks outside of the store wrapper functions, it's unobvious.
	if err := k.Hooks().BeforeDelegationRemoved(ctx, delAddr, valAddr); err != nil {
		return err
	}

	if err := k.Delegations.Remove(ctx, collections.Join(delAddr, valAddr)); err != nil {
		return err
	}

	return k.DelegationsByValIndex.Remove(ctx, collections.Join(valAddr, delAddr))
}

// GetUnbondingDelegations returns a given amount of all the delegator unbonding-delegations
func (k Keeper) GetUnbondingDelegations(
	ctx context.Context,
	delegator sdk.AccAddress,
	maxRetrieve uint16,
) ([]types.UnbondingDelegation, error) {
	unbondingDelegations := make([]types.UnbondingDelegation, 0, maxRetrieve)

	err := k.UnbondingDelegations.Walk(ctx, collections.NewPrefixedPairRange[[]byte, []byte](delegator), func(key collections.Pair[[]byte, []byte], unbondingDelegation types.UnbondingDelegation) (stop bool, err error) {
		unbondingDelegations = append(unbondingDelegations, unbondingDelegation)
		return len(unbondingDelegations) == int(maxRetrieve), nil
	})

	return unbondingDelegations, err
}

// GetUnbondingDelegation GetUnbondingDelegation returns a unbonding delegation
func (k Keeper) GetUnbondingDelegation(
	ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress,
) (types.UnbondingDelegation, error) {
	return k.UnbondingDelegations.Get(ctx, collections.Join[[]byte, []byte](delAddr, valAddr))
}

// GetUnbondingDelegationsFromValidator returns all unbonding delegations from a particular validator
func (k Keeper) GetUnbondingDelegationsFromValidator(ctx context.Context, valAddr sdk.ValAddress) (ubds []types.UnbondingDelegation, err error) {
	err = k.UnbondingDelegationsByValIndex.Walk(ctx, collections.NewPrefixedPairRange[[]byte, []byte](valAddr), func(key collections.Pair[[]byte, []byte], value bool) (stop bool, err error) {
		valAddr := key.K1()
		delAddr := key.K2()

		ubd, err := k.UnbondingDelegations.Get(ctx, collections.Join(delAddr, valAddr))
		if err != nil {
			return false, err
		}

		ubds = append(ubds, ubd)
		return false, nil
	})

	return ubds, err
}

// IterateUnbondingDelegations IterateUnbondingDelegations iterates through all of the unbonding delegations
func (k Keeper) IterateUnbondingDelegations(ctx context.Context, cb func(ubd types.UnbondingDelegation) (stop bool, err error)) error {
	return k.UnbondingDelegations.Walk(ctx, nil, func(key collections.Pair[[]byte, []byte], ubd types.UnbondingDelegation) (stop bool, err error) {
		return cb(ubd)
	})
}

// HasMaxUnbondingDelegationEntries - check if unbonding delegation has maximum number of entries
func (k Keeper) HasMaxUnbondingDelegationEntries(
	ctx context.Context,
	delegatorAddr sdk.AccAddress,
	validatorAddr sdk.ValAddress,
) (bool, error) {
	ubd, err := k.GetUnbondingDelegation(ctx, delegatorAddr, validatorAddr)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return false, err
	}

	maxEntries, err := k.MaxEntries(ctx)
	if err != nil {
		return false, err
	}

	return len(ubd.Entries) >= int(maxEntries), nil
}

// SetUnbondingDelegation sets the unbonding delegation and associated index
func (k Keeper) SetUnbondingDelegation(ctx context.Context, ubd types.UnbondingDelegation) error {
	delAddr, err := k.authKeeper.AddressCodec().StringToBytes(ubd.DelegatorAddress)
	if err != nil {
		return err
	}
	valAddr, err := k.validatorAddressCodec.StringToBytes(ubd.ValidatorAddress)
	if err != nil {
		return err
	}

	err = k.UnbondingDelegations.Set(ctx, collections.Join[[]byte, []byte](delAddr, valAddr), ubd)
	if err != nil {
		return err
	}

	err = k.UnbondingDelegationsByValIndex.Set(ctx, collections.Join[[]byte, []byte](valAddr, delAddr), true)
	if err != nil {
		return err
	}

	return nil
}

// RemoveUnbondingDelegation removes the unbonding delegation object and associated index
func (k Keeper) RemoveUnbondingDelegation(ctx context.Context, ubd types.UnbondingDelegation) error {
	delAddr, err := k.authKeeper.AddressCodec().StringToBytes(ubd.DelegatorAddress)
	if err != nil {
		return err
	}
	valAddr, err := k.validatorAddressCodec.StringToBytes(ubd.ValidatorAddress)
	if err != nil {
		return err
	}

	err = k.UnbondingDelegations.Remove(ctx, collections.Join(delAddr, valAddr))
	if err != nil {
		return err
	}

	err = k.UnbondingDelegationsByValIndex.Remove(ctx, collections.Join(valAddr, delAddr))
	if err != nil {
		return err
	}

	return nil
}

// SetUnbondingDelegationEntry adds an entry to the unbonding delegation at
// the given addresses. It creates the unbonding delegation if it does not exist
func (k Keeper) SetUnbondingDelegationEntry(
	ctx context.Context,
	delegatorAddr sdk.AccAddress,
	validatorAddr sdk.ValAddress,
	creationHeight int64,
	minTime time.Time,
	balance sdk.Coins,
) (ubd types.UnbondingDelegation, err error) {
	unbondingId, err := k.IncrementUnbondingId(ctx)
	if err != nil {
		return ubd, err
	}

	ubd, err = k.GetUnbondingDelegation(ctx, delegatorAddr, validatorAddr)
	if err == nil {
		ubd.AddEntry(creationHeight, minTime, balance, unbondingId)
	} else if errors.Is(err, collections.ErrNotFound) {
		delAddrStr, err := k.authKeeper.AddressCodec().BytesToString(delegatorAddr)
		if err != nil {
			return ubd, err
		}

		valAddrStr, err := k.validatorAddressCodec.BytesToString(validatorAddr)
		if err != nil {
			return ubd, err
		}

		ubd = types.NewUnbondingDelegation(delAddrStr, valAddrStr, creationHeight, minTime, balance, unbondingId)
	} else {
		return ubd, err
	}

	if err := k.SetUnbondingDelegation(ctx, ubd); err != nil {
		return ubd, err
	}

	// Add to the UBDByUnbondingOp index to look up the UBD by the UBDE ID
	if err := k.SetUnbondingDelegationByUnbondingId(ctx, ubd, unbondingId); err != nil {
		return ubd, err
	}

	if err := k.Hooks().AfterUnbondingInitiated(ctx, unbondingId); err != nil {
		return ubd, err
	}

	return ubd, nil
}

// unbonding delegation queue timeslice operations

// GetUBDQueueTimeSlice gets a specific unbonding queue timeslice. A timeslice is a slice of DVPairs
// corresponding to unbonding delegations that expire at a certain time.
func (k Keeper) GetUBDQueueTimeSlice(ctx context.Context, timestamp time.Time) ([]types.DVPair, error) {
	dvPairs, err := k.UnbondingQueue.Get(ctx, timestamp)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return []types.DVPair{}, nil
		}

		return nil, err
	}

	return dvPairs.Pairs, nil
}

// Sets a specific unbonding queue timeslice.
func (k Keeper) SetUBDQueueTimeSlice(ctx context.Context, timestamp time.Time, pairs []types.DVPair) error {
	return k.UnbondingQueue.Set(ctx, timestamp, types.DVPairs{
		Pairs: pairs,
	})
}

// Insert an unbonding delegation to the appropriate timeslice in the unbonding queue
func (k Keeper) InsertUBDQueue(
	ctx context.Context,
	ubd types.UnbondingDelegation,
	completionTime time.Time,
) error {
	dvPair := types.DVPair{DelegatorAddress: ubd.DelegatorAddress, ValidatorAddress: ubd.ValidatorAddress}

	timeSlice, err := k.GetUBDQueueTimeSlice(ctx, completionTime)
	if err != nil {
		return err
	}

	if len(timeSlice) == 0 {
		return k.SetUBDQueueTimeSlice(ctx, completionTime, []types.DVPair{dvPair})
	}

	timeSlice = append(timeSlice, dvPair)
	return k.SetUBDQueueTimeSlice(ctx, completionTime, timeSlice)
}

// Returns a concatenated list of all the timeslices inclusively previous to
// curTime, and deletes the timeslices from the queue
func (k Keeper) DequeueAllMatureUBDQueue(ctx context.Context, curTime time.Time) (matureUnbondings []types.DVPair, err error) {
	err = k.UnbondingQueue.Walk(ctx, new(collections.Range[time.Time]).EndInclusive(curTime), func(key time.Time, dvPairs types.DVPairs) (stop bool, err error) {
		matureUnbondings = append(matureUnbondings, dvPairs.Pairs...)
		return false, k.UnbondingQueue.Remove(ctx, key)
	})

	return matureUnbondings, err
}

// return a given amount of all the delegator redelegations
func (k Keeper) GetRedelegations(
	ctx context.Context,
	delegator sdk.AccAddress,
	maxRetrieve uint16,
) (redelegations []types.Redelegation, err error) {
	redelegations = make([]types.Redelegation, 0, maxRetrieve)

	err = k.Redelegations.Walk(ctx, collections.NewPrefixedTripleRange[[]byte, []byte, []byte](delegator), func(key collections.Triple[[]byte, []byte, []byte], redelegation types.Redelegation) (stop bool, err error) {
		redelegations = append(redelegations, redelegation)
		return len(redelegations) == int(maxRetrieve), nil
	})

	return redelegations, err
}

// return a redelegation
func (k Keeper) GetRedelegation(
	ctx context.Context,
	delAddr sdk.AccAddress,
	valSrcAddr, valDstAddr sdk.ValAddress,
) (red types.Redelegation, err error) {
	return k.Redelegations.Get(ctx, collections.Join3[[]byte, []byte, []byte](delAddr, valSrcAddr, valDstAddr))
}

// return all redelegations from a particular validator
func (k Keeper) GetRedelegationsFromSrcValidator(ctx context.Context, valAddr sdk.ValAddress) (redelegations []types.Redelegation, err error) {
	err = k.RedelegationsByValSrcIndex.Walk(ctx, collections.NewPrefixedTripleRange[[]byte, []byte, []byte](valAddr), func(key collections.Triple[[]byte, []byte, []byte], value bool) (stop bool, err error) {
		valSrcAddr := key.K1()
		delAddr := key.K2()
		valDstAddr := key.K3()

		redelegation, err := k.Redelegations.Get(ctx, collections.Join3(delAddr, valSrcAddr, valDstAddr))
		if err != nil {
			return true, err
		}

		redelegations = append(redelegations, redelegation)
		return false, nil
	})

	return redelegations, err
}

// check if validator is receiving a redelegation
func (k Keeper) HasReceivingRedelegation(
	ctx context.Context,
	delAddr sdk.AccAddress,
	valDstAddr sdk.ValAddress,
) (bool, error) {
	iterator, err := k.RedelegationsByValDstIndex.Iterate(ctx, collections.NewSuperPrefixedTripleRange[[]byte, []byte, []byte](valDstAddr, delAddr))
	if err != nil {
		return false, err
	}

	return iterator.Valid(), nil
}

// HasMaxRedelegationEntries - redelegation has maximum number of entries
func (k Keeper) HasMaxRedelegationEntries(
	ctx context.Context,
	delegatorAddr sdk.AccAddress,
	validatorSrcAddr, validatorDstAddr sdk.ValAddress,
) (bool, error) {
	red, err := k.GetRedelegation(ctx, delegatorAddr, validatorSrcAddr, validatorDstAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return false, nil
		}

		return false, err
	}

	maxEntries, err := k.MaxEntries(ctx)
	if err != nil {
		return false, err
	}

	return len(red.Entries) >= int(maxEntries), nil
}

// set a redelegation and associated index
func (k Keeper) SetRedelegation(ctx context.Context, red types.Redelegation) error {
	delAddr, err := k.authKeeper.AddressCodec().StringToBytes(red.DelegatorAddress)
	if err != nil {
		return err
	}

	valSrcAddr, err := k.validatorAddressCodec.StringToBytes(red.ValidatorSrcAddress)
	if err != nil {
		return err
	}

	valDstAddr, err := k.validatorAddressCodec.StringToBytes(red.ValidatorDstAddress)
	if err != nil {
		return err
	}

	if err := k.Redelegations.Set(ctx, collections.Join3(delAddr, valSrcAddr, valDstAddr), red); err != nil {
		return err
	}

	if err := k.RedelegationsByValSrcIndex.Set(ctx, collections.Join3(valSrcAddr, delAddr, valDstAddr), true); err != nil {
		return err
	}

	if err := k.RedelegationsByValDstIndex.Set(ctx, collections.Join3(valDstAddr, delAddr, valSrcAddr), true); err != nil {
		return err
	}

	return nil
}

// SetRedelegationEntry adds an entry to the redelegation at
// the given addresses. It creates the redelegation if it does not exist
func (k Keeper) SetRedelegationEntry(
	ctx context.Context,
	delegatorAddr sdk.AccAddress,
	validatorSrcAddr, validatorDstAddr sdk.ValAddress,
	creationHeight int64,
	minTime time.Time,
	balance sdk.Coins,
	sharesDst sdk.DecCoins,
) (red types.Redelegation, err error) {
	unbondingId, err := k.IncrementUnbondingId(ctx)
	if err != nil {
		return red, err
	}

	red, err = k.GetRedelegation(ctx, delegatorAddr, validatorSrcAddr, validatorDstAddr)
	if err == nil {
		red.AddEntry(
			creationHeight, minTime, balance,
			sharesDst, unbondingId,
		)
	} else if errors.Is(err, collections.ErrNotFound) {
		delAddrStr, err := k.authKeeper.AddressCodec().BytesToString(delegatorAddr)
		if err != nil {
			return red, err
		}
		valSrcAddrStr, err := k.validatorAddressCodec.BytesToString(validatorSrcAddr)
		if err != nil {
			return red, err
		}
		valDstAddrStr, err := k.validatorAddressCodec.BytesToString(validatorDstAddr)
		if err != nil {
			return red, err
		}

		red = types.NewRedelegation(
			delAddrStr, valSrcAddrStr,
			valDstAddrStr, creationHeight,
			minTime, balance, sharesDst, unbondingId,
		)
	} else {
		return red, err
	}

	if err := k.SetRedelegation(ctx, red); err != nil {
		return red, err
	}

	// Add to the UBDByEntry index to look up the UBD by the UBDE ID
	if err := k.SetRedelegationByUnbondingId(ctx, red, unbondingId); err != nil {
		return red, err
	}

	if err := k.Hooks().AfterUnbondingInitiated(ctx, unbondingId); err != nil {
		return red, err
	}

	return red, nil
}

// iterate through all redelegations
func (k Keeper) IterateRedelegations(ctx context.Context, cb func(red types.Redelegation) (stop bool, err error)) error {
	return k.Redelegations.Walk(ctx, nil, func(key collections.Triple[[]byte, []byte, []byte], redelegation types.Redelegation) (stop bool, err error) {
		return cb(redelegation)
	})
}

// remove a redelegation object and associated index
func (k Keeper) RemoveRedelegation(ctx context.Context, red types.Redelegation) error {
	delAddr, err := k.authKeeper.AddressCodec().StringToBytes(red.DelegatorAddress)
	if err != nil {
		return err
	}

	valSrcAddr, err := k.validatorAddressCodec.StringToBytes(red.ValidatorSrcAddress)
	if err != nil {
		return err
	}

	valDstAddr, err := k.validatorAddressCodec.StringToBytes(red.ValidatorDstAddress)
	if err != nil {
		return err
	}

	if err := k.Redelegations.Remove(ctx, collections.Join3(delAddr, valSrcAddr, valDstAddr)); err != nil {
		return err
	}

	if err := k.RedelegationsByValSrcIndex.Remove(ctx, collections.Join3(valSrcAddr, delAddr, valDstAddr)); err != nil {
		return err
	}

	if err := k.RedelegationsByValDstIndex.Remove(ctx, collections.Join3(valDstAddr, delAddr, valSrcAddr)); err != nil {
		return err
	}

	return nil
}

// redelegation queue timeslice operations

// Gets a specific redelegation queue timeslice. A timeslice is a slice of DVVTriplets corresponding to redelegations
// that expire at a certain time.
func (k Keeper) GetRedelegationQueueTimeSlice(ctx context.Context, timestamp time.Time) (dvvTriplets []types.DVVTriplet, err error) {
	triplets, err := k.RedelegationQueue.Get(ctx, timestamp)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return dvvTriplets, err
	}

	return triplets.Triplets, nil
}

// Sets a specific redelegation queue timeslice.
func (k Keeper) SetRedelegationQueueTimeSlice(ctx context.Context, timestamp time.Time, keys []types.DVVTriplet) error {
	return k.RedelegationQueue.Set(ctx, timestamp, types.DVVTriplets{Triplets: keys})
}

// Insert an redelegation delegation to the appropriate timeslice in the redelegation queue
func (k Keeper) InsertRedelegationQueue(
	ctx context.Context,
	red types.Redelegation,
	completionTime time.Time,
) error {
	timeSlice, err := k.GetRedelegationQueueTimeSlice(ctx, completionTime)
	if err != nil {
		return err
	}

	dvvTriplet := types.DVVTriplet{
		DelegatorAddress:    red.DelegatorAddress,
		ValidatorSrcAddress: red.ValidatorSrcAddress,
		ValidatorDstAddress: red.ValidatorDstAddress,
	}

	if len(timeSlice) == 0 {
		return k.SetRedelegationQueueTimeSlice(ctx, completionTime, []types.DVVTriplet{dvvTriplet})
	}

	timeSlice = append(timeSlice, dvvTriplet)
	return k.SetRedelegationQueueTimeSlice(ctx, completionTime, timeSlice)
}

// DequeueAllMatureRedelegationQueue returns a concatenated list of all the timeslices inclusively previous to
// curTime, and deletes the timeslices from the queue
func (k Keeper) DequeueAllMatureRedelegationQueue(ctx context.Context, curTime time.Time) (matureRedelegations []types.DVVTriplet, err error) {
	err = k.RedelegationQueue.Walk(ctx, new(collections.Range[time.Time]).EndInclusive(curTime), func(key time.Time, timeslice types.DVVTriplets) (stop bool, err error) {
		matureRedelegations = append(matureRedelegations, timeslice.Triplets...)
		return false, k.RedelegationQueue.Remove(ctx, key)
	})

	return matureRedelegations, err
}

// Delegate performs a delegation, set/update everything necessary within the store.
// tokenSrc indicates the bond status of the incoming funds.
func (k Keeper) Delegate(
	ctx context.Context,
	delAddr sdk.AccAddress,
	bondAmt sdk.Coins,
	tokenSrc types.BondStatus,
	validator types.Validator,
	subtractAccount bool,
) (sdk.DecCoins, error) {
	// In some situations, the exchange rate becomes invalid, e.g. if
	// Validator loses all tokens due to slashing. In this case,
	// make all future delegations invalid.
	if validator.InvalidExRate() {
		return nil, types.ErrDelegatorShareExRateInvalid
	}

	valAddr, err := k.validatorAddressCodec.StringToBytes(validator.GetOperator())
	if err != nil {
		return nil, err
	}

	// Get or create the delegation object
	delegation, err := k.GetDelegation(ctx, delAddr, valAddr)
	if err == nil {
		if err := k.Hooks().BeforeDelegationSharesModified(ctx, delAddr, valAddr); err != nil {
			return nil, err
		}
	} else if errors.Is(err, collections.ErrNotFound) {
		delAddrStr, err := k.authKeeper.AddressCodec().BytesToString(delAddr)
		if err != nil {
			return nil, err
		}

		delegation = types.NewDelegation(delAddrStr, validator.GetOperator(), sdk.NewDecCoins())
		if err := k.Hooks().BeforeDelegationCreated(ctx, delAddr, valAddr); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	// if subtractAccount is true then we are
	// performing a delegation and not a redelegation, thus the source tokens are
	// all non bonded
	if subtractAccount {
		if tokenSrc == types.Bonded {
			panic("delegation token source cannot be bonded")
		}

		var sendName string

		switch {
		case validator.IsBonded():
			sendName = types.BondedPoolName
		case validator.IsUnbonding(), validator.IsUnbonded():
			sendName = types.NotBondedPoolName
		default:
			panic("invalid validator status")
		}

		if err := k.bankKeeper.DelegateCoinsFromAccountToModule(ctx, delAddr, sendName, bondAmt); err != nil {
			return sdk.NewDecCoins(), err
		}
	} else {
		// potentially transfer tokens between pools, if
		switch {
		case tokenSrc == types.Bonded && validator.IsBonded():
			// do nothing
		case (tokenSrc == types.Unbonded || tokenSrc == types.Unbonding) && !validator.IsBonded():
			// do nothing
		case (tokenSrc == types.Unbonded || tokenSrc == types.Unbonding) && validator.IsBonded():
			// transfer pools
			if err = k.notBondedTokensToBonded(ctx, bondAmt); err != nil {
				return nil, err
			}
		case tokenSrc == types.Bonded && !validator.IsBonded():
			// transfer pools
			if err = k.bondedTokensToNotBonded(ctx, bondAmt); err != nil {
				return nil, err
			}
		default:
			panic("unknown token source bond status")
		}
	}

	_, newShares, err := k.AddValidatorTokensAndShares(ctx, validator, bondAmt)
	if err != nil {
		return nil, err
	}

	// Update delegation
	delegation.Shares = delegation.Shares.Add(newShares...)
	if err = k.SetDelegation(ctx, delegation); err != nil {
		return nil, err
	}

	// Call the after-modification hook
	if err = k.Hooks().AfterDelegationModified(ctx, delAddr, valAddr); err != nil {
		return newShares, err
	}

	return newShares, nil
}

// Unbond a particular delegation and perform associated store operations.
func (k Keeper) Unbond(
	ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, shares sdk.DecCoins,
) (sdk.Coins, error) {
	// check if a delegation object exists in the store
	delegation, err := k.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return nil, err
	}

	// call the before-delegation-modified hook
	if err := k.Hooks().BeforeDelegationSharesModified(ctx, delAddr, valAddr); err != nil {
		return nil, err
	}

	// ensure that we have enough shares to remove
	if _, hasNeg := delegation.Shares.SafeSub(shares); hasNeg {
		return nil, errorsmod.Wrap(types.ErrNotEnoughDelegationShares, delegation.Shares.String())
	}

	// get validator
	validator, err := k.Validators.Get(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	// subtract shares from delegation
	delegation.Shares = delegation.Shares.Sub(shares)

	delegatorAddress, err := k.authKeeper.AddressCodec().StringToBytes(delegation.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	isValidatorOperator := bytes.Equal(delegatorAddress, valAddr)

	// If the delegation is the operator of the validator and undelegation will decrease the validator's
	// self-delegation below their minimum, we jail the validator.
	if isValidatorOperator && !validator.Jailed {
		selfDelegation, _ := validator.TokensFromShares(delegation.Shares).TruncateDecimal()

		// recalculate voting power with new weights
		weights, err := k.GetVotingPowerWeights(ctx)
		if err != nil {
			return nil, err
		}
		votingPower, _ := types.CalculateVotingPower(selfDelegation, weights)
		consensusPower := k.VotingPowerToConsensusPower(ctx, votingPower)

		// min self delegation is constantly one.
		if consensusPower < 1 {
			if err = k.jailValidator(ctx, validator); err != nil {
				return nil, err
			}
			validator = k.mustGetValidator(ctx, valAddr)
		}
	}

	// remove the delegation
	if delegation.Shares.IsZero() {
		err = k.RemoveDelegation(ctx, delegation)
	} else {
		if err = k.SetDelegation(ctx, delegation); err != nil {
			return nil, err
		}
		// call the after delegation modification hook
		err = k.Hooks().AfterDelegationModified(ctx, delegatorAddress, valAddr)
	}
	if err != nil {
		return nil, err
	}

	// remove the shares and coins from the validator
	// NOTE that the amount is later (in keeper.Delegation) moved between staking module pools
	validator, amount, err := k.RemoveValidatorTokensAndShares(ctx, validator, shares)
	if err != nil {
		return nil, err
	}

	if validator.DelegatorShares.IsZero() && validator.IsUnbonded() {
		// if not unbonded, we must instead remove validator in EndBlocker once it finishes its unbonding period
		if err := k.RemoveValidator(ctx, valAddr); err != nil {
			return nil, err
		}
	}

	return amount, nil
}

// getBeginInfo returns the completion time and height of a redelegation, along
// with a boolean signaling if the redelegation is complete based on the source
// validator.
func (k Keeper) getBeginInfo(
	ctx context.Context, valSrcAddr sdk.ValAddress,
) (completionTime time.Time, height int64, completeNow bool, err error) {
	validator, err := k.Validators.Get(ctx, valSrcAddr)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return
	}

	// err == not found
	found := err == nil
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	unbondingTime, err := k.UnbondingTime(ctx)
	if err != nil {
		return
	}

	// TODO: When would the validator not be found?
	switch {
	case !found || validator.IsBonded():
		// the longest wait - just unbonding period from now
		completionTime = sdkCtx.BlockHeader().Time.Add(unbondingTime)
		height = sdkCtx.BlockHeight()

		return completionTime, height, false, nil

	case validator.IsUnbonded():
		return completionTime, height, true, nil

	case validator.IsUnbonding():
		return validator.UnbondingTime, validator.UnbondingHeight, false, nil

	default:
		panic(fmt.Sprintf("unknown validator status: %s", validator.Status))
	}
}

// Undelegate unbonds an amount of delegator shares from a given validator. It
// will verify that the unbonding entries between the delegator and validator
// are not exceeded and unbond the staked tokens (based on shares) by creating
// an unbonding object and inserting it into the unbonding queue which will be
// processed during the staking EndBlocker.
func (k Keeper) Undelegate(
	ctx context.Context,
	delAddr sdk.AccAddress,
	valAddr sdk.ValAddress,
	sharesAmount sdk.DecCoins,
) (time.Time, sdk.Coins, error) {
	validator, err := k.Validators.Get(ctx, valAddr)
	if err != nil {
		return time.Time{}, nil, err
	}

	if hasMax, err := k.HasMaxUnbondingDelegationEntries(ctx, delAddr, valAddr); err != nil {
		return time.Time{}, nil, err
	} else if hasMax {
		return time.Time{}, nil, types.ErrMaxUnbondingDelegationEntries
	}

	returnAmount, err := k.Unbond(ctx, delAddr, valAddr, sharesAmount)
	if err != nil {
		return time.Time{}, nil, err
	}

	// transfer the validator tokens to the not bonded pool
	if validator.IsBonded() {
		if err := k.bondedTokensToNotBonded(ctx, returnAmount); err != nil {
			return time.Time{}, nil, err
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	unbondingTime, err := k.UnbondingTime(ctx)
	if err != nil {
		return time.Time{}, nil, err
	}

	completionTime := sdkCtx.BlockHeader().Time.Add(unbondingTime)
	ubd, err := k.SetUnbondingDelegationEntry(ctx, delAddr, valAddr, sdkCtx.BlockHeight(), completionTime, returnAmount)
	if err != nil {
		return time.Time{}, nil, err
	}

	return completionTime, returnAmount, k.InsertUBDQueue(ctx, ubd, completionTime)
}

// CompleteUnbonding completes the unbonding of all mature entries in the
// retrieved unbonding delegation object and returns the total unbonding balance
// or an error upon failure.
func (k Keeper) CompleteUnbonding(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (sdk.Coins, error) {
	ubd, err := k.GetUnbondingDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return nil, err
	}

	balances := sdk.NewCoins()

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctxTime := sdkCtx.BlockHeader().Time

	delegatorAddress, err := k.authKeeper.AddressCodec().StringToBytes(ubd.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	// loop through all the entries and complete unbonding mature entries
	for i := 0; i < len(ubd.Entries); i++ {
		entry := ubd.Entries[i]
		if entry.IsMature(ctxTime) && !entry.OnHold() {
			ubd.RemoveEntry(int64(i))
			i--

			// remove UBDE index
			if err := k.DeleteUnbondingIndex(ctx, entry.UnbondingId); err != nil {
				return nil, err
			}

			// track undelegation only when remaining or truncated shares are non-zero
			if !entry.Balance.IsZero() {
				if err := k.bankKeeper.UndelegateCoinsFromModuleToAccount(
					ctx, types.NotBondedPoolName, delegatorAddress, entry.Balance,
				); err != nil {
					return nil, err
				}

				balances = balances.Add(entry.Balance...)
			}
		}
	}

	// set the unbonding delegation or remove it if there are no more entries
	if len(ubd.Entries) == 0 {
		err = k.RemoveUnbondingDelegation(ctx, ubd)
	} else {
		err = k.SetUnbondingDelegation(ctx, ubd)
	}
	if err != nil {
		return nil, err
	}

	return balances, nil
}

// begin unbonding / redelegation; create a redelegation record
func (k Keeper) BeginRedelegation(
	ctx context.Context, delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress, sharesAmount sdk.DecCoins,
) (completionTime time.Time, err error) {
	if bytes.Equal(valSrcAddr, valDstAddr) {
		return time.Time{}, types.ErrSelfRedelegation
	}

	dstValidator, err := k.Validators.Get(ctx, valDstAddr)
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return time.Time{}, types.ErrBadRedelegationDst
	} else if err != nil {
		return time.Time{}, err
	}

	srcValidator, err := k.Validators.Get(ctx, valSrcAddr)
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return time.Time{}, types.ErrBadRedelegationSrc
	} else if err != nil {
		return time.Time{}, err
	}

	// check if this is a transitive redelegation
	if has, err := k.HasReceivingRedelegation(ctx, delAddr, valSrcAddr); err != nil {
		return time.Time{}, err
	} else if has {
		return time.Time{}, types.ErrTransitiveRedelegation
	}

	if has, err := k.HasMaxRedelegationEntries(ctx, delAddr, valSrcAddr, valDstAddr); err != nil {
		return time.Time{}, err
	} else if has {
		return time.Time{}, types.ErrMaxRedelegationEntries
	}

	returnAmount, err := k.Unbond(ctx, delAddr, valSrcAddr, sharesAmount)
	if err != nil {
		return time.Time{}, err
	}

	if returnAmount.IsZero() {
		return time.Time{}, types.ErrTinyRedelegationAmount
	}

	sharesCreated, err := k.Delegate(ctx, delAddr, returnAmount, srcValidator.GetStatus(), dstValidator, false)
	if err != nil {
		return time.Time{}, err
	}

	// create the unbonding delegation
	completionTime, height, completeNow, err := k.getBeginInfo(ctx, valSrcAddr)
	if err != nil {
		return time.Time{}, err
	}

	if completeNow { // no need to create the redelegation object
		return completionTime, nil
	}

	red, err := k.SetRedelegationEntry(
		ctx, delAddr, valSrcAddr, valDstAddr,
		height, completionTime, returnAmount, sharesCreated,
	)
	if err != nil {
		return time.Time{}, err
	}

	if err := k.InsertRedelegationQueue(ctx, red, completionTime); err != nil {
		return time.Time{}, err
	}

	return completionTime, nil
}

// CompleteRedelegation completes the redelegations of all mature entries in the
// retrieved redelegation object and returns the total redelegation (initial)
// balance or an error upon failure.
func (k Keeper) CompleteRedelegation(
	ctx context.Context, delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress,
) (sdk.Coins, error) {
	red, err := k.GetRedelegation(ctx, delAddr, valSrcAddr, valDstAddr)
	if err != nil {
		return nil, err
	}

	balances := sdk.NewCoins()
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctxTime := sdkCtx.BlockHeader().Time

	// loop through all the entries and complete mature redelegation entries
	for i := 0; i < len(red.Entries); i++ {
		entry := red.Entries[i]
		if entry.IsMature(ctxTime) && !entry.OnHold() {
			red.RemoveEntry(int64(i))
			i--

			// remove UBDE index
			if err := k.DeleteUnbondingIndex(ctx, entry.UnbondingId); err != nil {
				return nil, err
			}

			if !entry.InitialBalance.IsZero() {
				balances = balances.Add(entry.InitialBalance...)
			}
		}
	}

	// set the redelegation or remove it if there are no more entries
	if len(red.Entries) == 0 {
		err = k.RemoveRedelegation(ctx, red)
	} else {
		err = k.SetRedelegation(ctx, red)
	}
	if err != nil {
		return nil, err
	}

	return balances, nil
}

// ValidateUnbondAmount validates that a given unbond or redelegation amount is
// valid based on upon the converted shares. If the amount is valid, the total
// amount of respective shares is returned, otherwise an error is returned.
func (k Keeper) ValidateUnbondAmount(
	ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt sdk.Coins,
) (shares sdk.DecCoins, err error) {
	validator, err := k.Validators.Get(ctx, valAddr)
	if err != nil {
		return shares, err
	}

	del, err := k.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return shares, err
	}

	shares, err = validator.SharesFromTokens(amt)
	if err != nil {
		return shares, err
	}

	sharesTruncated, err := validator.SharesFromTokensTruncated(amt)
	if err != nil {
		return shares, err
	}

	delShares := del.GetShares()
	if _, hasNeg := delShares.SafeSub(sharesTruncated); hasNeg {
		return shares, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid shares amount")
	}

	// Cap the shares at the delegation's shares. Shares being greater could occur
	// due to rounding, however we don't want to truncate the shares or take the
	// minimum because we want to allow for the full withdraw of shares from a
	// delegation.
	shares = shares.Intersect(delShares)

	return shares, nil
}
