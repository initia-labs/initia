package types

import (
	"cosmossdk.io/core/address"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// NewGenesisState creates a new ibc perm GenesisState instance.
func NewGenesisState(channelStates []ChannelState) *GenesisState {
	return &GenesisState{
		ChannelStates: channelStates,
	}
}

// DefaultGenesisState returns a default empty GenesisState.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		ChannelStates: []ChannelState{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate(ac address.Codec) error {
	for _, channelState := range gs.ChannelStates {
		if err := channelState.Validate(ac); err != nil {
			return err
		}
	}

	return nil
}

func (cs ChannelState) Validate(ac address.Codec) error {
	if err := host.ChannelIdentifierValidator(cs.ChannelId); err != nil {
		return err
	}

	if err := host.PortIdentifierValidator(cs.PortId); err != nil {
		return err
	}

	if _, err := ac.StringToBytes(cs.Admin); err != nil {
		return err
	}

	for _, relayer := range cs.Relayers {
		if _, err := ac.StringToBytes(relayer); err != nil {
			return err
		}
	}

	return nil
}
