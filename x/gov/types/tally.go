package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// ValidatorGovInfo used for tallying
type ValidatorGovInfo struct {
	Address             sdk.ValAddress         // address of the validator operator
	BondedTokens        sdk.Coins              // Power of a Validator
	DelegatorShares     sdk.DecCoins           // Total outstanding delegator shares
	DelegatorDeductions sdk.DecCoins           // Delegator deductions from validator's delegators voting independently
	Vote                v1.WeightedVoteOptions // Vote of the validator
}

// NewValidatorGovInfo creates a ValidatorGovInfo instance
func NewValidatorGovInfo(address sdk.ValAddress, bondedTokens sdk.Coins, delegatorShares,
	delegatorDeductions sdk.DecCoins, options v1.WeightedVoteOptions,
) ValidatorGovInfo {
	return ValidatorGovInfo{
		Address:             address,
		BondedTokens:        bondedTokens,
		DelegatorShares:     delegatorShares,
		DelegatorDeductions: delegatorDeductions,
		Vote:                options,
	}
}
