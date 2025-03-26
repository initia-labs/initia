package types

import (
	"fmt"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	"github.com/pkg/errors"

	"gopkg.in/yaml.v3"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Default parameter values
const (
	DefaultBaseDenom     = "uinit"
	DefaultScriptEnabled = true
)

var (
	DefaultBaseMinGasPrice            = math.LegacyZeroDec()
	DefaultContractSharedRevenueRatio = math.LegacyZeroDec()
)

const (
	ModuleSizeHardLimit         = int(1024 * 1024) // 1MB
	ModuleNameLengthHardLimit   = int(128)
	FunctionNameLengthHardLimit = int(128)
	NumArgumentsHardLimit       = int(16)
)

// DefaultParams returns default move parameters
func DefaultParams() Params {
	return Params{
		BaseDenom:                  DefaultBaseDenom,
		BaseMinGasPrice:            DefaultBaseMinGasPrice,
		ContractSharedRevenueRatio: DefaultContractSharedRevenueRatio,
		ScriptEnabled:              DefaultScriptEnabled,
		AllowedPublishers:          nil,
	}
}

func (p Params) String() string {
	out, err := yaml.Marshal(p)
	if err != nil {
		panic(err)
	}
	return string(out)
}

// Validate performs basic validation on move parameters
func (p Params) Validate(ac address.Codec) error {
	if err := validateBaseDenom(p.BaseDenom); err != nil {
		return errors.Wrap(err, "invalid base_denom")
	}

	if err := validateBaseMinGasPrice(p.BaseMinGasPrice); err != nil {
		return errors.Wrap(err, "invalid base_min_gas_price")
	}

	if err := validateContractSharedRatio(p.ContractSharedRevenueRatio); err != nil {
		return errors.Wrap(err, "invalid shared_revenue_ratio")
	}

	if err := validateAllowedPublishers(ac, p.AllowedPublishers); err != nil {
		return errors.Wrap(err, "invalid allowed_publishers")
	}

	return nil
}

// ToRaw return RawParams from the Params
func (p Params) ToRaw() RawParams {
	return RawParams{
		BaseDenom:                  p.BaseDenom,
		BaseMinGasPrice:            p.BaseMinGasPrice,
		ContractSharedRevenueRatio: p.ContractSharedRevenueRatio,
		ScriptEnabled:              p.ScriptEnabled,
	}
}

// ToParams return Params from the RawParams
func (p RawParams) ToParams(allowedPublishers []string) Params {
	return Params{
		BaseDenom:                  p.BaseDenom,
		BaseMinGasPrice:            p.BaseMinGasPrice,
		ContractSharedRevenueRatio: p.ContractSharedRevenueRatio,
		AllowedPublishers:          allowedPublishers,
		ScriptEnabled:              p.ScriptEnabled,
	}
}

func validateBaseDenom(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if err := sdk.ValidateDenom(v); err != nil {
		return err
	}

	return nil
}

func validateBaseMinGasPrice(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("base_min_gas_price must be non-negative value: %v", v)
	}

	return nil
}

func validateContractSharedRatio(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("contract_share_ratio must be non-negative value: %v", v)
	}

	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("contract_share_ratio must be smaller than or equal to one: %v", v)
	}

	return nil
}

func validateAllowedPublishers(ac address.Codec, i interface{}) error {
	allowedPublishers, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	for _, addr := range allowedPublishers {
		if _, err := AccAddressFromString(ac, addr); err != nil {
			return err
		}
	}

	return nil
}
