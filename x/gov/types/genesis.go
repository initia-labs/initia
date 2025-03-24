package types

import (
	"errors"

	"cosmossdk.io/core/address"

	"github.com/cosmos/cosmos-sdk/codec/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// NewGenesisState creates a new genesis state for the governance module
func NewGenesisState(startingProposalID uint64, params Params) *GenesisState {
	return &GenesisState{
		StartingProposalId: startingProposalID,
		Params:             &params,
	}
}

// DefaultGenesisState defines the default governance genesis state
func DefaultGenesisState() *GenesisState {
	return NewGenesisState(
		v1.DefaultStartingProposalID,
		DefaultParams(),
	)
}

// Empty returns true if a GenesisState is empty
func (data GenesisState) Empty() bool {
	return data.StartingProposalId == 0 || data.Params == nil
}

// ValidateGenesis checks if parameters are within valid ranges
func ValidateGenesis(data *GenesisState, ac address.Codec) error {
	if data.StartingProposalId == 0 {
		return errors.New("starting proposal id must be greater than 0")
	}

	return data.Params.Validate(ac)
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
