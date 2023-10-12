package types

// NewGenesisState creates a new ibc perm GenesisState instance.
func NewGenesisState(channelRelayers []ChannelRelayer) *GenesisState {
	return &GenesisState{
		ChannelRelayers: channelRelayers,
	}
}

// DefaultGenesisState returns a default empty GenesisState.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		ChannelRelayers: []ChannelRelayer{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return nil
}
