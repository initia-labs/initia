package types

import (
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultFetchEnabled enabled
	DefaultFetchEnabled    = true
	DefaultFetchActivated  = false
	DefaultTimeoutDuration = time.Second * 30
)

// NewParams creates a new parameter configuration for the ibc transfer module
func NewParams(fetchEnabled, fetchActivated bool, timeoutDuration time.Duration) Params {
	return Params{
		FetchEnabled:    fetchEnabled,
		FetchActivated:  fetchActivated,
		TimeoutDuration: timeoutDuration,
	}
}

// DefaultParams is the default parameter configuration for the ibc-transfer module
func DefaultParams() Params {
	return NewParams(DefaultFetchEnabled, DefaultFetchActivated, DefaultTimeoutDuration)
}

func (p Params) String() string {
	out, err := yaml.Marshal(p)
	if err != nil {
		panic(err)
	}
	return string(out)
}

func (p Params) Validate() error {
	if p.TimeoutDuration == 0 {
		return ErrInvalidPacketTimeout.Wrap("timeout cannot be zero")
	}

	return nil
}
