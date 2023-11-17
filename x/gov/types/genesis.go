package types

import (
	"errors"
	"time"

	"github.com/cosmos/cosmos-sdk/codec/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// NewGenesisState creates a new genesis state for the governance module
func NewGenesisState(startingProposalID uint64, params Params, lastEmergencyProposalTallyTimestamp time.Time) *GenesisState {
	return &GenesisState{
		StartingProposalId:                  startingProposalID,
		Params:                              &params,
		LastEmergencyProposalTallyTimestamp: lastEmergencyProposalTallyTimestamp,
	}
}

// DefaultGenesisState defines the default governance genesis state
func DefaultGenesisState() *GenesisState {
	return NewGenesisState(
		v1.DefaultStartingProposalID,
		DefaultParams(),
		time.Now().UTC(),
	)
}

// Empty returns true if a GenesisState is empty
func (data GenesisState) Empty() bool {
	return data.StartingProposalId == 0 || data.Params == nil
}

// ValidateGenesis checks if parameters are within valid ranges
func ValidateGenesis(data *GenesisState) error {
	if data.StartingProposalId == 0 {
		return errors.New("starting proposal id must be greater than 0")
	}

	return data.Params.ValidateBasic()
}

var _ types.UnpackInterfacesMessage = GenesisState{}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (data GenesisState) UnpackInterfaces(unpacker types.AnyUnpacker) error {
	for _, p := range data.Proposals {
		err := p.UnpackInterfaces(unpacker)
		if err != nil {
			return err
		}
	}
	return nil
}
