package types

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	"gopkg.in/yaml.v3"

	"github.com/pkg/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Default parameter values
const (
	DefaultDilutionPeriod = time.Hour * 24 * 365 // a year
	DefaultReleaseEnabled = false
)

var (
	DefaultRewardDenom  = sdk.DefaultBondDenom
	DefaultReleaseRate  = math.LegacyNewDecWithPrec(7, 2) // 7%
	DefaultDilutionRate = math.LegacyNewDecWithPrec(5, 1) // 50%
)

func NewParams(
	rewardDenom string, releaseRate, dilutionRate math.LegacyDec,
	dilutionPeriod time.Duration, releaseEnabled bool,
) Params {
	return Params{
		RewardDenom:    rewardDenom,
		ReleaseRate:    releaseRate,
		DilutionRate:   dilutionRate,
		DilutionPeriod: dilutionPeriod,
		ReleaseEnabled: releaseEnabled,
	}
}

// DefaultParams returns default move parameters
func DefaultParams() Params {
	return Params{
		RewardDenom:    DefaultRewardDenom,
		ReleaseRate:    DefaultReleaseRate,
		DilutionRate:   DefaultDilutionRate,
		DilutionPeriod: DefaultDilutionPeriod,
		ReleaseEnabled: DefaultReleaseEnabled,
	}
}

// String returns a human readable string representation of the parameters.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// Validate performs basic validation on move parameters
func (p Params) Validate() error {
	if err := validateRewardDenom(p.RewardDenom); err != nil {
		return errors.Wrap(err, "invalid mint denom")
	}

	if err := validateDilutionPeriod(p.DilutionPeriod); err != nil {
		return errors.Wrap(err, "invalid dilution period")
	}

	if err := validateReleaseRate(p.ReleaseRate); err != nil {
		return errors.Wrap(err, "invalid release rate")
	}

	if err := validateDilutionRate(p.DilutionRate); err != nil {
		return errors.Wrap(err, "invalid dilution rate")
	}

	if err := validateReleaseEnabled(p.ReleaseEnabled); err != nil {
		return errors.Wrap(err, "invalid release enabled")
	}

	return nil
}

func validateRewardDenom(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if err := sdk.ValidateDenom(v); err != nil {
		return err
	}

	return nil
}

func validateDilutionPeriod(i interface{}) error {
	v, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("DilutionPeriod must be bigger than 0")
	}

	return nil
}

func validateReleaseRate(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.GT(math.LegacyZeroDec()) {
		return fmt.Errorf("ReleaseRate should be smaller than 1.0")
	}

	if v.LT(math.LegacyZeroDec()) {
		return fmt.Errorf("ReleaseRate should be bigger than 0.0")
	}

	return nil
}

func validateDilutionRate(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.GT(math.LegacyZeroDec()) {
		return fmt.Errorf("DilutionRate should be smaller than 1.0")
	}

	if v.LT(math.LegacyZeroDec()) {
		return fmt.Errorf("DilutionRate should be bigger than 0.0")
	}

	return nil
}

func validateReleaseEnabled(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}
