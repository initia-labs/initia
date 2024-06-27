package types

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/math"
	yaml "gopkg.in/yaml.v3"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Staking params default values
const (
	// DefaultUnbondingTime reflects three weeks in seconds as the default
	// unbonding time.
	// TODO: Justify our choice of default here.
	DefaultUnbondingTime time.Duration = time.Hour * 24 * 7 * 3

	// Default maximum number of bonded validators
	DefaultMaxValidators uint32 = 100

	// Default maximum entries in a UBD/RED pair
	DefaultMaxEntries uint32 = 7

	// DefaultHistorical entries is 10000. Apps that don't use IBC can ignore this
	// value by not adding the staking module to the application module manager's
	// SetOrderBeginBlockers.
	DefaultHistoricalEntries uint32 = 10000

	// DefaultMinVotingPower minimum voting power to get into power update validator set
	DefaultMinVotingPower uint64 = 1_000_000
)

// NewParams creates a new Params instance
func NewParams(unbondingTime time.Duration, maxValidators, maxEntries uint32, historicalEntries uint32, bondDenoms []string, minVotingPower uint64) Params {
	return Params{
		UnbondingTime:     unbondingTime,
		MaxValidators:     maxValidators,
		MaxEntries:        maxEntries,
		HistoricalEntries: historicalEntries,
		BondDenoms:        bondDenoms,
		MinVotingPower:    minVotingPower,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(
		DefaultUnbondingTime,
		DefaultMaxValidators,
		DefaultMaxEntries,
		DefaultHistoricalEntries,
		[]string{sdk.DefaultBondDenom},
		DefaultMinVotingPower,
	)
}

// String returns a human readable string representation of the parameters.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// MustUnmarshalParams unmarshal the current staking params value from store key or panic
func MustUnmarshalParams(cdc *codec.LegacyAmino, value []byte) Params {
	params, err := UnmarshalParams(cdc, value)
	if err != nil {
		panic(err)
	}

	return params
}

// UnmarshalParams unmarshal the current staking params value from store key
func UnmarshalParams(cdc *codec.LegacyAmino, value []byte) (params Params, err error) {
	err = cdc.Unmarshal(value, &params)
	if err != nil {
		return
	}

	return
}

// Validate validates a set of params
func (p Params) Validate() error {
	if err := validateUnbondingTime(p.UnbondingTime); err != nil {
		return err
	}

	if err := validateMaxValidators(p.MaxValidators); err != nil {
		return err
	}

	if err := validateMaxEntries(p.MaxEntries); err != nil {
		return err
	}

	if err := validateHistoricalEntries(p.HistoricalEntries); err != nil {
		return err
	}

	if err := validateBondDenoms(p.BondDenoms); err != nil {
		return err
	}

	if err := validateMinVotingPower(p.MinVotingPower); err != nil {
		return err
	}

	return nil
}

func validateUnbondingTime(i interface{}) error {
	v, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v <= 0 {
		return fmt.Errorf("unbonding time must be positive: %d", v)
	}

	return nil
}

func validateMaxValidators(i interface{}) error {
	v, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("max validators must be positive: %d", v)
	}

	return nil
}

func validateMaxEntries(i interface{}) error {
	v, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("max entries must be positive: %d", v)
	}

	return nil
}

func validateHistoricalEntries(i interface{}) error {
	_, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateBondDenoms(i interface{}) error {
	v, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	for _, denom := range v {
		if strings.TrimSpace(denom) == "" {
			return errors.New("bond denom cannot be blank")
		}

		if err := sdk.ValidateDenom(denom); err != nil {
			return err
		}
	}

	return nil
}

func ValidatePowerReduction(i interface{}) error {
	v, ok := i.(math.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.LT(math.OneInt()) {
		return fmt.Errorf("power reduction cannot be lower than 1")
	}

	return nil
}

func validateMinVotingPower(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("min voting power must be positive: %d", v)
	}

	return nil
}
