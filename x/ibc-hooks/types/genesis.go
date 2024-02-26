package types

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Acls:   []ACL{},
	}
}

func (gs GenesisState) ValidateGenesis() error {
	return gs.Params.Validate()
}
