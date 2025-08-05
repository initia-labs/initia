package types

import (
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// NewGenesisState creates a new ibc nft-transfer GenesisState instance.
func NewGenesisState(portID string, classTraces Traces, params Params) *GenesisState {
	return &GenesisState{
		PortId:      portID,
		ClassTraces: classTraces,
		Params:      params,
	}
}

// DefaultGenesisState returns a GenesisState with "nft-transfer" as the default PortID.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		PortId:      PortID,
		ClassTraces: Traces{},
		Params:      DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := host.PortIdentifierValidator(gs.PortId); err != nil {
		return err
	}
	if err := gs.ClassTraces.Validate(); err != nil {
		return err
	}
	return gs.Params.Validate()
}
