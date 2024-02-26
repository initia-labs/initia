package types

// NewGenesisState creates a new ibc perm GenesisState instance.
func NewGenesisState(permissionedRelayers []PermissionedRelayer) *GenesisState {
	return &GenesisState{
		PermissionedRelayers: permissionedRelayers,
	}
}

// DefaultGenesisState returns a default empty GenesisState.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		PermissionedRelayers: []PermissionedRelayer{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return nil
}
