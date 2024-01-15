package types

import (
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"

	"github.com/initia-labs/initia/x/ibc/fetchprice/types"
)

// NewGenesisState creates a new ibc fetchprice provider GenesisState instance.
func NewGenesisState(portID string) *GenesisState {
	return &GenesisState{
		PortId: portID,
	}
}

// DefaultGenesisState returns a GenesisState with "fetchprice provider" as the default PortID.
func DefaultGenesisState() GenesisState {
	return GenesisState{
		PortId: types.ProviderPortID,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := host.PortIdentifierValidator(gs.PortId); err != nil {
		return err
	}

	return nil
}
