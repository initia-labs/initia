package types

// NewGenesisState creates a new ibc nft-transfer GenesisState instance.
func NewGenesisState(portID string, params Params) *GenesisState {
	return &GenesisState{
		PortId: portID,
		Params: params,
	}
}

// DefaultGenesisState returns a GenesisState with "nft-transfer" as the default PortID.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		PortId: PortID,
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
