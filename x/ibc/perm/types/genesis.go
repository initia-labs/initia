package types

// NewGenesisState creates a new ibc perm GenesisState instance.
func NewGenesisState(permissionedRelayers []PermissionedRelayersSet) *GenesisState {
	return &GenesisState{
		PermissionedRelayerSets: permissionedRelayers,
	}
}

// DefaultGenesisState returns a default empty GenesisState.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		PermissionedRelayerSets: []PermissionedRelayersSet{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return nil
}
