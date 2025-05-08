package types

import (
	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/staking module sentinel errors
//
// Many of these errors have been removed and replaced by sdkerrors.ErrInvalidRequest
// according to https://github.com/cosmos/cosmos-sdk/issues/5450
var (
	ErrEmptyValidatorAddr              = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty validator address")
	ErrNoValidatorFound                = errorsmod.Register(ModuleName, 3, "validator does not exist")
	ErrValidatorOwnerExists            = errorsmod.Register(ModuleName, 4, "validator already exist for this operator address; must use new validator operator address")
	ErrValidatorPubKeyExists           = errorsmod.Register(ModuleName, 5, "validator already exist for this pubkey; must use new validator pubkey")
	ErrValidatorPubKeyTypeNotSupported = errorsmod.Register(ModuleName, 6, "validator pubkey type is not supported")
	ErrValidatorJailed                 = errorsmod.Register(ModuleName, 7, "validator for this address is currently jailed")
	ErrBadRemoveValidator              = errorsmod.Register(ModuleName, 8, "failed to remove validator")
	ErrCommissionNegative              = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "commission must be positive")
	ErrCommissionHuge                  = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "commission cannot be more than 100%")
	ErrCommissionGTMaxRate             = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "commission cannot be more than the max rate")
	ErrCommissionUpdateTime            = errorsmod.Register(ModuleName, 12, "commission cannot be changed more than once in 24h")
	ErrCommissionChangeRateNegative    = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "commission change rate must be positive")
	ErrCommissionChangeRateGTMaxRate   = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "commission change rate cannot be more than the max rate")
	ErrCommissionGTMaxChangeRate       = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "commission cannot be changed more than max change rate")
	ErrSelfDelegationBelowMinimum      = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "validator's self delegation must be greater than their minimum self delegation")
	ErrMinSelfDelegationDecreased      = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "minimum self delegation cannot be decrease")
	ErrEmptyDelegatorAddr              = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty delegator address")
	ErrNoDelegation                    = errorsmod.Register(ModuleName, 19, "no delegation for (address, validator) tuple")
	ErrBadDelegatorAddr                = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "delegator does not exist with address")
	ErrNoDelegatorForAddress           = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "delegator does not contain delegation")
	ErrInsufficientShares              = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "insufficient delegation shares")
	ErrDelegationValidatorEmpty        = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "cannot delegate to an empty validator")
	ErrNotEnoughDelegationShares       = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "not enough delegation shares")
	ErrNotMature                       = errorsmod.Register(ModuleName, 25, "entry not mature")
	ErrNoUnbondingDelegation           = errorsmod.Register(ModuleName, 26, "no unbonding delegation found")
	ErrMaxUnbondingDelegationEntries   = errorsmod.Register(ModuleName, 27, "too many unbonding delegation entries for (delegator, validator) tuple")
	ErrNoRedelegation                  = errorsmod.Register(ModuleName, 28, "no redelegation found")
	ErrSelfRedelegation                = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "cannot redelegate to the same validator")
	ErrTinyRedelegationAmount          = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "too few tokens to redelegate (truncates to zero tokens)")
	ErrBadRedelegationDst              = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "redelegation destination validator not found")
	ErrTransitiveRedelegation          = errorsmod.Register(ModuleName, 32, "redelegation to this validator already in progress; first redelegation to this validator must complete before next redelegation")
	ErrMaxRedelegationEntries          = errorsmod.Register(ModuleName, 33, "too many redelegation entries for (delegator, src-validator, dst-validator) tuple")
	ErrDelegatorShareExRateInvalid     = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "cannot delegate to validators with invalid (zero) ex-rate")
	ErrBothShareMsgsGiven              = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "both shares amount and shares percent provided")
	ErrNeitherShareMsgsGiven           = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "neither shares amount nor shares percent provided")
	ErrInvalidHistoricalInfo           = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid historical info")
	ErrNoHistoricalInfo                = errorsmod.Register(ModuleName, 38, "no historical info found")
	ErrEmptyValidatorPubKey            = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty validator public key")
	ErrCommissionLTMinRate             = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "commission cannot be less than min rate")
	ErrUnbondingNotFound               = errorsmod.Register(ModuleName, 41, "unbonding operation not found")
	ErrUnbondingOnHoldRefCountNegative = errorsmod.Register(ModuleName, 42, "cannot un-hold unbonding operation that is not on hold")
	ErrInvalidSigner                   = errorsmod.Register(ModuleName, 43, "expected authority account as only signer for proposal message")
	ErrBadRedelegationSrc              = errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "redelegation source validator not found")
	ErrNoUnbondingType                 = errorsmod.Register(ModuleName, 45, "unbonding type not found")
)
