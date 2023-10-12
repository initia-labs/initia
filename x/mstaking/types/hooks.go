package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// combine multiple staking hooks, all hook functions are run in array sequence
var _ StakingHooks = &MultiStakingHooks{}

type MultiStakingHooks []StakingHooks

func NewMultiStakingHooks(hooks ...StakingHooks) MultiStakingHooks {
	return hooks
}

func (h MultiStakingHooks) AfterValidatorCreated(ctx sdk.Context, valAddr sdk.ValAddress) error {
	for i := range h {
		h[i].AfterValidatorCreated(ctx, valAddr)
	}
	return nil
}
func (h MultiStakingHooks) BeforeValidatorModified(ctx sdk.Context, valAddr sdk.ValAddress) error {
	for i := range h {
		h[i].BeforeValidatorModified(ctx, valAddr)
	}
	return nil
}
func (h MultiStakingHooks) AfterValidatorRemoved(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error {
	for i := range h {
		h[i].AfterValidatorRemoved(ctx, consAddr, valAddr)
	}
	return nil
}
func (h MultiStakingHooks) AfterValidatorBonded(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error {
	for i := range h {
		h[i].AfterValidatorBonded(ctx, consAddr, valAddr)
	}
	return nil
}
func (h MultiStakingHooks) AfterValidatorBeginUnbonding(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error {
	for i := range h {
		h[i].AfterValidatorBeginUnbonding(ctx, consAddr, valAddr)
	}
	return nil
}
func (h MultiStakingHooks) BeforeDelegationCreated(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	for i := range h {
		h[i].BeforeDelegationCreated(ctx, delAddr, valAddr)
	}
	return nil
}
func (h MultiStakingHooks) BeforeDelegationSharesModified(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	for i := range h {
		h[i].BeforeDelegationSharesModified(ctx, delAddr, valAddr)
	}
	return nil
}
func (h MultiStakingHooks) BeforeDelegationRemoved(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	for i := range h {
		h[i].BeforeDelegationRemoved(ctx, delAddr, valAddr)
	}
	return nil
}
func (h MultiStakingHooks) AfterDelegationModified(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	for i := range h {
		h[i].AfterDelegationModified(ctx, delAddr, valAddr)
	}
	return nil
}
func (h MultiStakingHooks) BeforeValidatorSlashed(ctx sdk.Context, valAddr sdk.ValAddress, fractions sdk.DecCoins) error {
	for i := range h {
		h[i].BeforeValidatorSlashed(ctx, valAddr, fractions)
	}
	return nil
}

func (h MultiStakingHooks) AfterUnbondingInitiated(ctx sdk.Context, id uint64) error {
	for i := range h {
		if err := h[i].AfterUnbondingInitiated(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// combine multiple slashing hooks, all hook functions are run in array sequence
var _ SlashingHooks = &MultiSlashingHooks{}

type MultiSlashingHooks []SlashingHooks

func NewMultiSlashingHooks(hooks ...SlashingHooks) MultiSlashingHooks {
	return hooks
}

func (h MultiSlashingHooks) SlashUnbondingDelegations(ctx sdk.Context, valAddr sdk.ValAddress, fraction sdk.Dec) error {
	for i := range h {
		h[i].SlashUnbondingDelegations(ctx, valAddr, fraction)
	}
	return nil
}
