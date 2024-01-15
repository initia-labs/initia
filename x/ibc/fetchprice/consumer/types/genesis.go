package types

import (
	errorsmod "cosmossdk.io/errors"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"

	"github.com/initia-labs/initia/x/ibc/fetchprice/types"

	oracletypes "github.com/skip-mev/slinky/x/oracle/types"
)

// NewGenesisState creates a new ibc fetchprice consumer GenesisState instance.
func NewGenesisState(portID string, currencyPairs []types.CurrencyPrice) *GenesisState {
	return &GenesisState{
		PortId:         portID,
		CurrencyPrices: currencyPairs,
	}
}

// DefaultGenesisState returns a GenesisState with "fetchprice consumer" as the default PortID.
func DefaultGenesisState() GenesisState {
	return GenesisState{
		PortId:         types.ConsumerPortID,
		CurrencyPrices: []types.CurrencyPrice{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := host.PortIdentifierValidator(gs.PortId); err != nil {
		return err
	}

	for _, cp := range gs.CurrencyPrices {
		if _, err := oracletypes.CurrencyPairFromString(cp.CurrencyId); err != nil {
			return errorsmod.Wrap(types.ErrInvalidCurrencyId, err.Error())
		}

		if err := cp.QuotePrice.ToOracleQuotePrice().ValidateBasic(); err != nil {
			return errorsmod.Wrap(types.ErrInvalidQuotePrice, err.Error())
		}
	}

	return nil
}
