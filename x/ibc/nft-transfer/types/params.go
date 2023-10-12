package types

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultSendEnabled enabled
	DefaultSendEnabled = true
	// DefaultReceiveEnabled enabled
	DefaultReceiveEnabled = true
)

// NewParams creates a new parameter configuration for the ibc transfer module
func NewParams(enableSend, enableReceive bool) Params {
	return Params{
		SendEnabled:    enableSend,
		ReceiveEnabled: enableReceive,
	}
}

// DefaultParams is the default parameter configuration for the ibc-transfer module
func DefaultParams() Params {
	return NewParams(DefaultSendEnabled, DefaultReceiveEnabled)
}

func (p Params) String() string {
	out, err := yaml.Marshal(p)
	if err != nil {
		panic(err)
	}
	return string(out)
}

// Validate all ibc-transfer module parameters
func (p Params) Validate() error {
	if err := validateEnabled(p.SendEnabled); err != nil {
		return err
	}

	return validateEnabled(p.ReceiveEnabled)
}

func validateEnabled(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}
