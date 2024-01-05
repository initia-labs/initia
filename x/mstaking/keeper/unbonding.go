package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/mstaking/types"
)

func (k Keeper) SetNextUnbondingId(ctx context.Context, unbondingId uint64) error {
	return k.NextUnbondingId.Set(ctx, unbondingId)
}

func (k Keeper) GetNextUnbondingId(ctx context.Context) (uint64, error) {
	nextUnbondingId, err := k.NextUnbondingId.Peek(ctx)
	if err != nil {
		return 0, err
	}

	if nextUnbondingId == collections.DefaultSequenceStart {
		return types.DefaultUnbondingIdStart, nil
	}

	return nextUnbondingId, nil
}

// IncrementUnbondingId increments and returns a unique ID for an unbonding operation
func (k Keeper) IncrementUnbondingId(ctx context.Context) (uint64, error) {
	nextUnbondingId, err := k.NextUnbondingId.Next(ctx)
	if err != nil {
		return 0, err
	}

	if nextUnbondingId == collections.DefaultSequenceStart {
		if err := k.NextUnbondingId.Set(ctx, types.DefaultUnbondingIdStart+1); err != nil {
			return 0, err
		}

		return types.DefaultUnbondingIdStart, nil
	}

	return nextUnbondingId, nil

}

// DeleteUnbondingIndex removes a mapping from UnbondingId to unbonding operation
func (k Keeper) DeleteUnbondingIndex(ctx context.Context, id uint64) error {
	if err := k.UnbondingsIndex.Remove(ctx, id); err != nil {
		return err
	}

	// TODO - why sdk do not delete type?
	return k.DeleteUnbondingType(ctx, id)
}

func (k Keeper) GetUnbondingType(ctx context.Context, id uint64) (types.UnbondingType, error) {
	t, err := k.UnbondingsType.Get(ctx, id)
	if err != nil {
		return types.UnbondingType_Undefined, err
	}

	return types.UnbondingType(t), nil
}

func (k Keeper) SetUnbondingType(ctx context.Context, id uint64, unbondingType types.UnbondingType) error {
	return k.UnbondingsType.Set(ctx, id, uint32(unbondingType))
}

func (k Keeper) DeleteUnbondingType(ctx context.Context, id uint64) error {
	return k.UnbondingsType.Remove(ctx, id)
}

// GetUnbondingDelegationByUnbondingId returns a unbonding delegation that has an unbonding delegation entry with a certain ID
func (k Keeper) GetUnbondingDelegationByUnbondingId(ctx context.Context, id uint64) (types.UnbondingDelegation, error) {
	ubdKey, err := k.UnbondingsIndex.Get(ctx, id)
	if err != nil {
		return types.UnbondingDelegation{}, err
	}

	return k.UnbondingDelegations.Get(ctx, collections.Join(ubdKey.K1(), ubdKey.K2()))
}

// GetRedelegationByUnbondingId returns a unbonding delegation that has an unbonding delegation entry with a certain ID
func (k Keeper) GetRedelegationByUnbondingId(ctx context.Context, id uint64) (types.Redelegation, error) {
	redKey, err := k.UnbondingsIndex.Get(ctx, id)
	if err != nil {
		return types.Redelegation{}, err
	}

	return k.Redelegations.Get(ctx, redKey)
}

// GetValidatorByUnbondingId returns the validator that is unbonding with a certain unbonding op ID
func (k Keeper) GetValidatorByUnbondingId(ctx context.Context, id uint64) (types.Validator, error) {
	valKey, err := k.UnbondingsIndex.Get(ctx, id)
	if err != nil {
		return types.Validator{}, err
	}

	return k.Validators.Get(ctx, valKey.K1())
}

// SetUnbondingDelegationByUnbondingId sets an index to look up an UnbondingDelegation by the unbondingId of an UnbondingDelegationEntry that it contains
// Note, it does not set the unbonding delegation itself, use SetUnbondingDelegation(ctx, ubd) for that
func (k Keeper) SetUnbondingDelegationByUnbondingId(ctx context.Context, ubd types.UnbondingDelegation, id uint64) error {
	delAddr, err := k.authKeeper.AddressCodec().StringToBytes(ubd.DelegatorAddress)
	if err != nil {
		return err
	}

	valAddr, err := k.validatorAddressCodec.StringToBytes(ubd.ValidatorAddress)
	if err != nil {
		return err
	}

	if err := k.UnbondingsIndex.Set(ctx, id, collections.Join3(delAddr, valAddr, []byte{})); err != nil {
		return err
	}

	// Set unbonding type so that we know how to deserialize it later
	if err := k.SetUnbondingType(ctx, id, types.UnbondingType_UnbondingDelegation); err != nil {
		return err
	}

	return nil
}

// SetRedelegationByUnbondingId sets an index to look up an Redelegation by the unbondingId of an RedelegationEntry that it contains
// Note, it does not set the redelegation itself, use SetRedelegation(ctx, red) for that
func (k Keeper) SetRedelegationByUnbondingId(ctx context.Context, red types.Redelegation, id uint64) error {
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

	if err := k.UnbondingsIndex.Set(ctx, id, collections.Join3(delAddr, valSrcAddr, valDstAddr)); err != nil {
		return err
	}

	// Set unbonding type so that we know how to deserialize it later
	if err := k.SetUnbondingType(ctx, id, types.UnbondingType_Redelegation); err != nil {
		return err
	}

	return nil
}

// SetValidatorByUnbondingId sets an index to look up a Validator by the unbondingId corresponding to its current unbonding
// Note, it does not set the validator itself, use SetValidator(ctx, val) for that
func (k Keeper) SetValidatorByUnbondingId(ctx context.Context, val types.Validator, id uint64) error {
	valAddr, err := k.validatorAddressCodec.StringToBytes(val.GetOperator())
	if err != nil {
		return err
	}

	if err := k.UnbondingsIndex.Set(ctx, id, collections.Join3(valAddr, []byte{}, []byte{})); err != nil {
		return err
	}

	// Set unbonding type so that we know how to deserialize it later
	if err := k.SetUnbondingType(ctx, id, types.UnbondingType_ValidatorUnbonding); err != nil {
		return err
	}

	return nil
}

// unbondingDelegationEntryArrayIndex and redelegationEntryArrayIndex are utilities to find
// at which position in the Entries array the entry with a given id is
func unbondingDelegationEntryArrayIndex(ubd types.UnbondingDelegation, id uint64) (index int, found bool) {
	for i, entry := range ubd.Entries {
		// we find the entry with the right ID
		if entry.UnbondingId == id {
			return i, true
		}
	}

	return 0, false
}

func redelegationEntryArrayIndex(red types.Redelegation, id uint64) (index int, found bool) {
	for i, entry := range red.Entries {
		// we find the entry with the right ID
		if entry.UnbondingId == id {
			return i, true
		}
	}

	return 0, false
}

// UnbondingCanComplete allows a stopped unbonding operation, such as an
// unbonding delegation, a redelegation, or a validator unbonding to complete.
// In order for the unbonding operation with `id` to eventually complete, every call
// to PutUnbondingOnHold(id) must be matched by a call to UnbondingCanComplete(id).
func (k Keeper) UnbondingCanComplete(ctx context.Context, id uint64) error {
	unbondingType, err := k.GetUnbondingType(ctx, id)
	if err != nil {
		return err
	}

	switch unbondingType {
	case types.UnbondingType_UnbondingDelegation:
		if err := k.unbondingDelegationEntryCanComplete(ctx, id); err != nil {
			return err
		}
	case types.UnbondingType_Redelegation:
		if err := k.redelegationEntryCanComplete(ctx, id); err != nil {
			return err
		}
	case types.UnbondingType_ValidatorUnbonding:
		if err := k.validatorUnbondingCanComplete(ctx, id); err != nil {
			return err
		}
	default:
		return types.ErrUnbondingNotFound
	}

	return nil
}

func (k Keeper) unbondingDelegationEntryCanComplete(ctx context.Context, id uint64) error {
	ubd, err := k.GetUnbondingDelegationByUnbondingId(ctx, id)
	if err != nil {
		return err
	}

	i, found := unbondingDelegationEntryArrayIndex(ubd, id)
	if !found {
		return types.ErrUnbondingNotFound
	}

	// The entry must be on hold
	if !ubd.Entries[i].OnHold() {
		return errors.Wrapf(
			types.ErrUnbondingOnHoldRefCountNegative,
			"undelegation unbondingId(%d), expecting UnbondingOnHoldRefCount > 0, got %T",
			id, ubd.Entries[i].UnbondingOnHoldRefCount,
		)
	}
	ubd.Entries[i].UnbondingOnHoldRefCount--

	// Check if entry is matured.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !ubd.Entries[i].OnHold() && ubd.Entries[i].IsMature(sdkCtx.BlockHeader().Time) {
		// If matured, complete it.
		delegatorAddress, err := k.authKeeper.AddressCodec().StringToBytes(ubd.DelegatorAddress)
		if err != nil {
			return err
		}

		// track undelegation only when remaining or truncated shares are non-zero
		if !ubd.Entries[i].Balance.IsZero() {
			amt := ubd.Entries[i].Balance
			if err := k.bankKeeper.UndelegateCoinsFromModuleToAccount(
				ctx, types.NotBondedPoolName, delegatorAddress, amt,
			); err != nil {
				return err
			}
		}

		// Remove entry
		ubd.RemoveEntry(int64(i))
		// Remove from the UnbondingIndex
		if err := k.DeleteUnbondingIndex(ctx, id); err != nil {
			return err
		}
	}

	// set the unbonding delegation or remove it if there are no more entries
	if len(ubd.Entries) == 0 {
		if err := k.RemoveUnbondingDelegation(ctx, ubd); err != nil {
			return err
		}
	} else {
		if err := k.SetUnbondingDelegation(ctx, ubd); err != nil {
			return err
		}
	}

	// Successfully completed unbonding
	return nil
}

func (k Keeper) redelegationEntryCanComplete(ctx context.Context, id uint64) error {
	red, err := k.GetRedelegationByUnbondingId(ctx, id)
	if err != nil {
		return err
	}

	i, found := redelegationEntryArrayIndex(red, id)
	if !found {
		return types.ErrUnbondingNotFound
	}

	// The entry must be on hold
	if !red.Entries[i].OnHold() {
		return errors.Wrapf(
			types.ErrUnbondingOnHoldRefCountNegative,
			"redelegation unbondingId(%d), expecting UnbondingOnHoldRefCount > 0, got %T",
			id, red.Entries[i].UnbondingOnHoldRefCount,
		)
	}
	red.Entries[i].UnbondingOnHoldRefCount--

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !red.Entries[i].OnHold() && red.Entries[i].IsMature(sdkCtx.BlockHeader().Time) {
		// If matured, complete it.
		// Remove entry
		red.RemoveEntry(int64(i))

		// Remove from the Unbonding index
		if err := k.DeleteUnbondingIndex(ctx, id); err != nil {
			return err
		}
	}

	// set the redelegation or remove it if there are no more entries
	if len(red.Entries) == 0 {
		if err := k.RemoveRedelegation(ctx, red); err != nil {
			return err
		}
	} else {
		if err := k.SetRedelegation(ctx, red); err != nil {
			return err
		}
	}

	// Successfully completed unbonding
	return nil
}

func (k Keeper) validatorUnbondingCanComplete(ctx context.Context, id uint64) error {
	val, err := k.GetValidatorByUnbondingId(ctx, id)
	if err != nil {
		return err
	}

	if val.UnbondingOnHoldRefCount <= 0 {
		return errors.Wrapf(
			types.ErrUnbondingOnHoldRefCountNegative,
			"val(%s), expecting UnbondingOnHoldRefCount > 0, got %T",
			val.OperatorAddress, val.UnbondingOnHoldRefCount,
		)
	}

	val.UnbondingOnHoldRefCount--
	if err := k.SetValidator(ctx, val); err != nil {
		return err
	}

	return nil
}

// PutUnbondingOnHold allows an external module to stop an unbonding operation,
// such as an unbonding delegation, a redelegation, or a validator unbonding.
// In order for the unbonding operation with `id` to eventually complete, every call
// to PutUnbondingOnHold(id) must be matched by a call to UnbondingCanComplete(id).
func (k Keeper) PutUnbondingOnHold(ctx context.Context, id uint64) error {
	unbondingType, err := k.GetUnbondingType(ctx, id)
	if err != nil {
		return err
	}

	switch unbondingType {
	case types.UnbondingType_UnbondingDelegation:
		if err := k.putUnbondingDelegationEntryOnHold(ctx, id); err != nil {
			return err
		}
	case types.UnbondingType_Redelegation:
		if err := k.putRedelegationEntryOnHold(ctx, id); err != nil {
			return err
		}
	case types.UnbondingType_ValidatorUnbonding:
		if err := k.putValidatorOnHold(ctx, id); err != nil {
			return err
		}
	default:
		return types.ErrUnbondingNotFound
	}

	return nil
}

func (k Keeper) putUnbondingDelegationEntryOnHold(ctx context.Context, id uint64) error {
	ubd, err := k.GetUnbondingDelegationByUnbondingId(ctx, id)
	if err != nil {
		return err
	}

	i, found := unbondingDelegationEntryArrayIndex(ubd, id)
	if !found {
		return types.ErrUnbondingNotFound
	}

	ubd.Entries[i].UnbondingOnHoldRefCount++
	if err := k.SetUnbondingDelegation(ctx, ubd); err != nil {
		return err
	}

	return nil
}

func (k Keeper) putRedelegationEntryOnHold(ctx context.Context, id uint64) error {
	red, err := k.GetRedelegationByUnbondingId(ctx, id)
	if err != nil {
		return err
	}

	i, found := redelegationEntryArrayIndex(red, id)
	if !found {
		return types.ErrUnbondingNotFound
	}

	red.Entries[i].UnbondingOnHoldRefCount++
	if err := k.SetRedelegation(ctx, red); err != nil {
		return err
	}

	return nil
}

func (k Keeper) putValidatorOnHold(ctx context.Context, id uint64) error {
	val, err := k.GetValidatorByUnbondingId(ctx, id)
	if err != nil {
		return err
	}

	val.UnbondingOnHoldRefCount++
	if err := k.SetValidator(ctx, val); err != nil {
		return err
	}

	return nil
}
