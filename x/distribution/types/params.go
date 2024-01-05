package types

import (
	"fmt"

	"cosmossdk.io/math"
	yaml "gopkg.in/yaml.v3"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	DefaultCommunityTax        = math.LegacyNewDecWithPrec(2, 2) // 2%
	DefaultWithdrawAddrEnabled = true
	DefaultRewardWeights       = []RewardWeight{}
)

// DefaultParams returns default distribution parameters
func DefaultParams() Params {
	return Params{
		CommunityTax:        DefaultCommunityTax,
		WithdrawAddrEnabled: DefaultWithdrawAddrEnabled,
		RewardWeights:       DefaultRewardWeights,
	}
}

func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// ValidateBasic performs basic validation on distribution parameters.
func (p Params) ValidateBasic() error {
	if err := validateCommunityTax(p.CommunityTax); err != nil {
		return err
	}

	if err := validateWithdrawAddrEnabled(p.WithdrawAddrEnabled); err != nil {
		return err
	}

	if err := validateRewardWeights(p.RewardWeights); err != nil {
		return err
	}

	return nil
}

func validateCommunityTax(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("community tax must be not nil")
	}
	if v.IsNegative() {
		return fmt.Errorf("community tax must be positive: %s", v)
	}
	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("community tax too large: %s", v)
	}

	return nil
}

func validateWithdrawAddrEnabled(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateRewardWeights(i interface{}) error {
	v, ok := i.([]RewardWeight)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	for _, rw := range v {
		if err := sdk.ValidateDenom(rw.Denom); err != nil {
			return err
		}

		if rw.Weight.IsNegative() {
			return fmt.Errorf("reward weight must be positive: %s", rw)
		}
	}

	return nil
}
