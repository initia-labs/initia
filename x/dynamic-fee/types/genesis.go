package types

// NewGenesisState creates a new GenesisState object
func NewGenesisState(params Params) *GenesisState {
	return &GenesisState{
		Params: params,
	}
}

// DefaultGenesisState gets raw genesis raw message for testing
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

// ValidateGenesis performs basic validation of move genesis data returning an
// error for any failed validation criteria.
func ValidateGenesis(data *GenesisState) error {
	return data.Params.Validate()
}
