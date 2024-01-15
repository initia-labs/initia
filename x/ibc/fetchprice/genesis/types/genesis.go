package types

import (
	consumertypes "github.com/initia-labs/initia/x/ibc/fetchprice/consumer/types"
	providertypes "github.com/initia-labs/initia/x/ibc/fetchprice/provider/types"
)

func NewGenesisState(
	consumerGenesisState consumertypes.GenesisState,
	providerGenesisState providertypes.GenesisState,
) *GenesisState {
	return &GenesisState{
		ConsumerGenesisState: consumerGenesisState,
		ProviderGenesisState: providerGenesisState,
	}
}

func DefaultGenesis() *GenesisState {
	return NewGenesisState(
		consumertypes.DefaultGenesisState(),
		providertypes.DefaultGenesisState(),
	)
}

func (gs GenesisState) Validate() error {
	if err := gs.ConsumerGenesisState.Validate(); err != nil {
		return err
	}

	if err := gs.ProviderGenesisState.Validate(); err != nil {
		return err
	}

	return nil
}
