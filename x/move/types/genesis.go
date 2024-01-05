package types

import (
	"encoding/json"

	"cosmossdk.io/core/address"
	"cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"

	vmprecom "github.com/initia-labs/initiavm/precompile"
)

// NewGenesisState creates a new GenesisState object
func NewGenesisState(params Params, modules []Module, resources []Resource) *GenesisState {
	return &GenesisState{
		Params:    params,
		Modules:   modules,
		Resources: resources,
	}
}

// DefaultGenesisState gets raw genesis raw message for testing
func DefaultGenesisState() *GenesisState {
	modules, err := vmprecom.ReadStdlib()
	if err != nil {
		panic(errors.Wrap(err, "failed to read stdlib from precompile"))
	}

	return &GenesisState{
		Stdlibs:          modules,
		Params:           DefaultParams(),
		ExecutionCounter: 0,
		Modules:          []Module{},
		Resources:        []Resource{},
		TableInfos:       []TableInfo{},
		TableEntries:     []TableEntry{},
		DexPairs:         []DexPair{},
	}
}

// ValidateGenesis performs basic validation of move genesis data returning an
// error for any failed validation criteria.
func ValidateGenesis(data *GenesisState, ac address.Codec) error {
	return data.Params.Validate(ac)
}

// GetGenesisStateFromAppState returns x/auth GenesisState given raw application
// genesis state.
func GetGenesisStateFromAppState(cdc codec.Codec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return &genesisState
}
