package types

import "time"

// NewGenesisState creates a new GenesisState object
func NewGenesisState(params Params, lastReleaseTimestamp time.Time, lastDilutionTimestamp time.Time) *GenesisState {
	return &GenesisState{
		Params:                params,
		LastReleaseTimestamp:  lastReleaseTimestamp,
		LastDilutionTimestamp: lastDilutionTimestamp,
	}
}

// DefaultGenesisState creates a default GenesisState object
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:                DefaultParams(),
		LastReleaseTimestamp:  time.Now().UTC(),
		LastDilutionTimestamp: time.Now().UTC(),
	}
}

// ValidateGenesis validates the provided genesis state to ensure the
// expected invariants holds.
func ValidateGenesis(data GenesisState) error {
	return data.Params.Validate()
}
