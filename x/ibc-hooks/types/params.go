package types

import "gopkg.in/yaml.v3"

const (
	DefaultAllowed = false
)

// NewParams creates a new Params instance with given values.
func NewParams(defaultAllowed bool) Params {
	return Params{
		DefaultAllowed: defaultAllowed,
	}
}

// DefaultParams returns the default hook params
func DefaultParams() Params {
	return NewParams(
		DefaultAllowed,
	)
}

func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

func (p Params) Validate() error {
	return nil
}
