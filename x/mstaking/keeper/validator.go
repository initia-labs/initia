package keeper

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	"github.com/initia-labs/initia/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// get a single validator
func (k Keeper) GetValidator(ctx context.Context, addr sdk.ValAddress) (validator types.Validator, err error) {
	return k.Validators.Get(ctx, addr)
}

func (k Keeper) mustGetValidator(ctx context.Context, addr sdk.ValAddress) types.Validator {
	validator, err := k.GetValidator(ctx, addr)
	if err != nil {
		panic(fmt.Sprintf("validator record not found for address: %X\n", addr))
	}

	return validator
}

// get a single validator by consensus address
func (k Keeper) GetValidatorByConsAddr(ctx context.Context, consAddr sdk.ConsAddress) (validator types.Validator, err error) {
	valAddr, err := k.ValidatorsByConsAddr.Get(ctx, consAddr)
	if err != nil {
		return validator, err
	}

	return k.GetValidator(ctx, valAddr)
}

func (k Keeper) mustGetValidatorByConsAddr(ctx context.Context, consAddr sdk.ConsAddress) types.Validator {
	validator, err := k.GetValidatorByConsAddr(ctx, consAddr)
	if err != nil {
		panic(fmt.Errorf("validator with consensus-Address %s not found: %v", consAddr, err))
	}

	return validator
}

// set the main record holding validator details
func (k Keeper) SetValidator(ctx context.Context, validator types.Validator) error {
	valAddr, err := k.validatorAddressCodec.StringToBytes(validator.GetOperator())
	if err != nil {
		return err
	}

	return k.Validators.Set(ctx, valAddr, validator)
}

// validator index
func (k Keeper) SetValidatorByConsAddr(ctx context.Context, validator types.Validator) error {
	consPk, err := validator.GetConsAddr()
	if err != nil {
		return err
	}

	valAddr, err := k.validatorAddressCodec.StringToBytes(validator.GetOperator())
	if err != nil {
		return err
	}

	return k.ValidatorsByConsAddr.Set(ctx, consPk, valAddr)
}

// validator index
func (k Keeper) SetValidatorByPowerIndex(ctx context.Context, validator types.Validator) error {
	powerReduction := k.PowerReduction(ctx)
	consensusPower := sdk.TokensToConsensusPower(validator.VotingPower, powerReduction)
	valAddr, err := k.validatorAddressCodec.StringToBytes(validator.GetOperator())
	if err != nil {
		return err
	}

	return k.ValidatorsByConsPowerIndex.Set(ctx, collections.Join(consensusPower, valAddr), true)
}

// validator index
func (k Keeper) DeleteValidatorByPowerIndex(ctx context.Context, validator types.Validator) error {
	powerReduction := k.PowerReduction(ctx)
	consensusPower := sdk.TokensToConsensusPower(validator.VotingPower, powerReduction)
	valAddr, err := k.validatorAddressCodec.StringToBytes(validator.GetOperator())
	if err != nil {
		return err
	}

	return k.ValidatorsByConsPowerIndex.Remove(ctx, collections.Join(consensusPower, valAddr))
}

// Update the tokens of an existing validator.
// NOTE: validators power index key not updated here
func (k Keeper) AddValidatorTokensAndShares(
	ctx context.Context,
	validator types.Validator,
	tokensToAdd sdk.Coins,
) (valOut types.Validator, addedShares sdk.DecCoins, err error) {
	valOut, addedShares = validator.AddTokensFromDel(tokensToAdd)

	// add a validator to whitelist group
	if ok, err := k.IsWhitelist(ctx, valOut); err != nil {
		return valOut, addedShares, err
	} else if !ok {
		minPower, err := k.MinVotingPower(ctx)
		if err != nil {
			return valOut, addedShares, err
		}

		votingPower, err := k.VotingPower(ctx, valOut.Tokens)
		if err != nil {
			return valOut, addedShares, err
		}

		if votingPower.GTE(minPower) {
			err = k.AddWhitelistValidator(ctx, valOut)
			if err != nil {
				return valOut, addedShares, err
			}
		}
	}

	err = k.SetValidator(ctx, valOut)
	return valOut, addedShares, err
}

// Update the tokens of an existing validator.
// NOTE: validators power index key not updated here
func (k Keeper) RemoveValidatorTokensAndShares(
	ctx context.Context,
	validator types.Validator,
	sharesToRemove sdk.DecCoins,
) (valOut types.Validator, removedTokens sdk.Coins, err error) {
	validator, removedTokens = validator.RemoveDelShares(sharesToRemove)

	err = k.SetValidator(ctx, validator)
	return validator, removedTokens, err
}

// Update the tokens of an existing validator.
// NOTE: validators power index key not updated here
func (k Keeper) RemoveValidatorTokens(
	ctx context.Context,
	validator types.Validator,
	tokensToRemove sdk.Coins,
) (types.Validator, error) {
	validator = validator.RemoveTokens(tokensToRemove)

	err := k.SetValidator(ctx, validator)
	return validator, err
}

// UpdateValidatorCommission attempts to update a validator's commission rate.
// An error is returned if the new commission rate is invalid.
func (k Keeper) UpdateValidatorCommission(
	ctx context.Context,
	validator types.Validator,
	newRate math.LegacyDec,
) (types.Commission, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	commission := validator.Commission
	blockTime := sdkCtx.BlockHeader().Time

	if err := commission.ValidateNewRate(newRate, blockTime); err != nil {
		return commission, err
	}

	if minCommission, err := k.MinCommissionRate(ctx); err != nil {
		return commission, err
	} else if newRate.LT(minCommission) {
		return commission, fmt.Errorf("cannot set validator commission to less than minimum rate of %s", minCommission)
	}

	commission.Rate = newRate
	commission.UpdateTime = blockTime

	return commission, nil
}

// RemoveValidator removes the validator record and associated indexes
// except for the bonded validator index which is only handled in ApplyAndReturnTendermintUpdates
func (k Keeper) RemoveValidator(ctx context.Context, valAddr sdk.ValAddress) error {
	// first retrieve the old validator record
	validator, err := k.GetValidator(ctx, valAddr)
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	if !validator.IsUnbonded() {
		return types.ErrBadRemoveValidator.Wrap("cannot call RemoveValidator on bonded or unbonding validators")
	}

	if !validator.Tokens.IsZero() {
		return types.ErrBadRemoveValidator.Wrap("attempting to remove a validator which still contains tokens")
	}

	valConsAddr, err := validator.GetConsAddr()
	if err != nil {
		return err
	}

	powerReduction := k.PowerReduction(ctx)
	consensusPower := sdk.TokensToConsensusPower(validator.VotingPower, powerReduction)

	// delete the old validator record
	if err := k.Validators.Remove(ctx, valAddr); err != nil {
		return err
	}

	if err := k.ValidatorsByConsAddr.Remove(ctx, valConsAddr); err != nil {
		return err
	}

	if err := k.ValidatorsByConsPowerIndex.Remove(ctx, collections.Join(consensusPower, []byte(valAddr))); err != nil {
		return err
	}

	// call hooks
	if err := k.Hooks().AfterValidatorRemoved(ctx, valConsAddr, valAddr); err != nil {
		k.Logger(ctx).Error("error in after validator removed hook", "error", err)
	}

	return nil
}

// get groups of validators

// get the set of all validators with no limits, used during genesis dump
func (k Keeper) GetAllValidators(ctx context.Context) (validators []types.Validator, err error) {
	err = k.Validators.Walk(ctx, nil, func(key []byte, validator types.Validator) (stop bool, err error) {
		validators = append(validators, validator)
		return false, nil
	})

	return
}

// return a given amount of all the validators
func (k Keeper) GetValidators(ctx context.Context, maxRetrieve uint32) (validators []types.Validator, err error) {
	validators = make([]types.Validator, 0, maxRetrieve)
	err = k.Validators.Walk(ctx, nil, func(key []byte, validator types.Validator) (stop bool, err error) {
		validators = append(validators, validator)
		return len(validators) == int(maxRetrieve), nil
	})

	return validators, err
}

// get the current group of bonded validators sorted by power-rank
func (k Keeper) GetBondedValidatorsByPower(ctx context.Context) ([]types.Validator, error) {
	maxValidators, err := k.MaxValidators(ctx)
	if err != nil {
		return nil, err
	}

	validators := make([]types.Validator, 0, maxValidators)
	err = k.ValidatorsByConsPowerIndex.Walk(ctx, new(collections.PairRange[int64, []byte]).Descending(), func(key collections.Pair[int64, []byte], value bool) (stop bool, err error) {
		validator, err := k.GetValidator(ctx, key.K2())
		if err != nil {
			return true, err
		}

		if validator.IsBonded() {
			validators = append(validators, validator)
		}

		return len(validators) == int(maxValidators), nil
	})

	return validators, err
}

// Last Validator Index

// Load the last validator power.
// Returns zero if the operator was not a validator last block.
func (k Keeper) GetLastValidatorConsPower(ctx context.Context, valAddr sdk.ValAddress) (int64, error) {
	power, err := k.LastValidatorConsPowers.Get(ctx, valAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return 0, nil
		}

		return 0, err
	}

	return power, nil
}

// Set the last validator power.
func (k Keeper) SetLastValidatorConsPower(ctx context.Context, valAddr sdk.ValAddress, power int64) error {
	return k.LastValidatorConsPowers.Set(ctx, valAddr, power)
}

// Delete the last validator power.
func (k Keeper) DeleteLastValidatorConsPower(ctx context.Context, valAddr sdk.ValAddress) error {
	return k.LastValidatorConsPowers.Remove(ctx, valAddr)
}

// Iterate over last validator powers.
func (k Keeper) IterateLastValidatorConsPowers(ctx context.Context, handler func(operator sdk.ValAddress, power int64) (stop bool, err error)) error {
	return k.LastValidatorConsPowers.Walk(ctx, nil, func(valAddr []byte, power int64) (stop bool, err error) {
		return handler(valAddr, power)
	})
}

// get the group of the bonded validators
func (k Keeper) GetLastValidators(ctx context.Context) (validators []types.Validator, err error) {
	maxValidators, err := k.MaxValidators(ctx)
	if err != nil {
		return nil, err
	}

	validators = make([]types.Validator, 0, maxValidators)
	err = k.LastValidatorConsPowers.Walk(ctx, nil, func(valAddr []byte, power int64) (stop bool, err error) {
		validator, err := k.GetValidator(ctx, valAddr)
		if err != nil {
			return true, err
		}

		validators = append(validators, validator)
		return false, nil
	})

	return validators, err
}

// GetUnbondingValidators returns a slice of mature validator addresses that
// complete their unbonding at a given time and height.
func (k Keeper) GetUnbondingValidators(ctx context.Context, endTime time.Time) ([]string, error) {
	valAddresses, err := k.ValidatorQueue.Get(ctx, endTime)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return []string{}, nil
		}

		return nil, err
	}

	return valAddresses.Addresses, nil
}

// SetUnbondingValidatorsQueue sets a given slice of validator addresses into
// the unbonding validator queue by a given height and time.
func (k Keeper) SetUnbondingValidatorsQueue(ctx context.Context, endTime time.Time, addrs []string) error {
	return k.ValidatorQueue.Set(ctx, endTime, types.ValAddresses{Addresses: addrs})
}

// InsertUnbondingValidatorQueue inserts a given unbonding validator address into
// the unbonding validator queue for a given height and time.
func (k Keeper) InsertUnbondingValidatorQueue(ctx context.Context, val types.Validator) error {
	addrs, err := k.GetUnbondingValidators(ctx, val.UnbondingTime)
	if err != nil {
		return err
	}

	addrs = append(addrs, val.OperatorAddress)
	return k.SetUnbondingValidatorsQueue(ctx, val.UnbondingTime, addrs)
}

// DeleteValidatorQueueTimeSlice deletes all entries in the queue indexed by a
// given height and time.
func (k Keeper) DeleteValidatorQueueTimeSlice(ctx context.Context, endTime time.Time, endHeight int64) error {
	return k.ValidatorQueue.Remove(ctx, endTime)
}

// DeleteValidatorQueue removes a validator by address from the unbonding queue
// indexed by a given height and time.
func (k Keeper) DeleteValidatorQueue(ctx context.Context, val types.Validator) error {
	addrs, err := k.GetUnbondingValidators(ctx, val.UnbondingTime)
	if err != nil {
		return err
	}

	newAddrs := []string{}
	for _, addr := range addrs {
		if addr != val.OperatorAddress {
			newAddrs = append(newAddrs, addr)
		}
	}

	if len(newAddrs) == 0 {
		return k.DeleteValidatorQueueTimeSlice(ctx, val.UnbondingTime, val.UnbondingHeight)
	}

	return k.SetUnbondingValidatorsQueue(ctx, val.UnbondingTime, newAddrs)
}

// UnbondAllMatureValidators unbonds all the mature unbonding validators that
// have finished their unbonding period.
func (k Keeper) UnbondAllMatureValidators(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := sdkCtx.BlockTime()
	return k.ValidatorQueue.Walk(ctx, new(collections.Range[time.Time]).EndInclusive(blockTime), func(key time.Time, valAddresses types.ValAddresses) (stop bool, err error) {
		for _, valAddr := range valAddresses.Addresses {
			addr, err := k.validatorAddressCodec.StringToBytes(valAddr)
			if err != nil {
				return true, err
			}

			val, err := k.GetValidator(ctx, addr)
			if err != nil {
				return true, err
			}

			if !val.IsUnbonding() {
				return true, fmt.Errorf("unexpected validator in unbonding queue; status was not unbonding")
			}

			if val.UnbondingOnHoldRefCount == 0 {
				if err := k.DeleteUnbondingIndex(ctx, val.UnbondingId); err != nil {
					return true, err
				}

				// TODO - sdk is not calling k.SetValidator
				// clear val.UnbondingId before k.SetValidator (called in k.UnbondingToUnbonded)
				val.UnbondingId = 0
				val, err = k.UnbondingToUnbonded(ctx, val)
				if err != nil {
					return true, err
				}

				if val.GetDelegatorShares().IsZero() {
					valAddr, err := k.validatorAddressCodec.StringToBytes(val.GetOperator())
					if err != nil {
						return true, err
					}

					if err := k.RemoveValidator(ctx, valAddr); err != nil {
						return true, err
					}
				}

				// remove validator from queue
				if err := k.DeleteValidatorQueue(ctx, val); err != nil {
					return true, err
				}
			}
		}

		return false, nil
	})
}

func (k Keeper) IsValidatorJailed(ctx context.Context, addr sdk.ConsAddress) (bool, error) {
	v, err := k.GetValidatorByConsAddr(ctx, addr)
	if err != nil {
		return false, err
	}

	return v.Jailed, nil
}

// IsWhitelist return whether a validator is already in whitelist
func (k Keeper) IsWhitelist(ctx context.Context, val types.Validator) (bool, error) {
	valAddr, err := k.validatorAddressCodec.StringToBytes(val.GetOperator())
	if err != nil {
		return false, err
	}

	return k.WhitelistedValidators.Has(ctx, valAddr)
}

// AddWhitelistValidator add validator to power update whitelist
func (k Keeper) AddWhitelistValidator(ctx context.Context, val types.Validator) error {
	// jailed validators are not kept in the power update whitelist
	if val.Jailed {
		return nil
	}

	valAddr, err := k.validatorAddressCodec.StringToBytes(val.GetOperator())
	if err != nil {
		return err
	}

	return k.WhitelistedValidators.Set(ctx, valAddr, true)
}

// RemoveWhitelistValidator remove validator from power update whitelist
func (k Keeper) RemoveWhitelistValidator(ctx context.Context, val types.Validator) error {
	valAddr, err := k.validatorAddressCodec.StringToBytes(val.GetOperator())
	if err != nil {
		return err
	}

	if err := k.WhitelistedValidators.Remove(ctx, valAddr); err != nil {
		return err
	}

	// remove power index
	return k.DeleteValidatorByPowerIndex(ctx, val)
}

// IterateWhitelistValidator iterate all whitelist validators
func (k Keeper) IterateWhitelistValidator(ctx context.Context, handler func(val types.Validator) (stop bool, err error)) error {
	return k.WhitelistedValidators.Walk(ctx, nil, func(valAddr []byte, value bool) (stop bool, err error) {
		val, err := k.GetValidator(ctx, valAddr)
		if err != nil {
			return true, err
		}

		return handler(val)
	})
}
