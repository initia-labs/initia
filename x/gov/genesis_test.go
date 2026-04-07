package gov_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/gov"

	customtypes "github.com/initia-labs/initia/x/gov/types"
)

func TestExportImportState(t *testing.T) {
	app := createDefaultApp(t)
	ctx := app.NewContext(true)

	exportedState, err := gov.ExportGenesis(ctx, app.GovKeeper)
	require.NoError(t, err)

	genesisState := customtypes.DefaultGenesisState()
	genesisState.Params.MinDeposit[0].Denom = bondDenom
	genesisState.Params.ExpeditedMinDeposit[0].Denom = bondDenom
	genesisState.Params.EmergencyMinDeposit[0].Denom = bondDenom

	require.Equal(t, exportedState, genesisState)
}
