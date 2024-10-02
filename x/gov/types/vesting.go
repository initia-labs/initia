package types

import (
	"errors"

	"cosmossdk.io/core/address"
	"gopkg.in/yaml.v3"
)

// Vesting struct to hold the vesting contract information
func NewVesting(moduleAddr, moduleName, creatorAddr string) Vesting {
	return Vesting{moduleAddr, moduleName, creatorAddr}
}

// String implements the stringer interface for a Vesting struct
func (v Vesting) String() string {
	out, _ := yaml.Marshal(v)
	return string(out)
}

// Validate checks for the validity of the Vesting struct
func (v Vesting) Validate(ac address.Codec) error {
	if _, err := ac.StringToBytes(v.ModuleAddr); err != nil {
		return err
	}

	if _, err := ac.StringToBytes(v.CreatorAddr); err != nil {
		return err
	}

	if l := len(v.ModuleName); l == 0 || l > 128 {
		return errors.New("module name cannot be empty or exceed 128 characters")
	}

	return nil
}
