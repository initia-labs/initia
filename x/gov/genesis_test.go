package gov_test

import (
	"testing"

	"github.com/initia-labs/initia/x/gov"
	"github.com/stretchr/testify/require"

	customtypes "github.com/initia-labs/initia/x/gov/types"
)

func TestExportDefaultState(t *testing.T) {
	app := createAppWithSimpleValidators(t)
	ctx := app.BaseApp.NewContext(true)
	exportedState, err := gov.ExportGenesis(ctx, app.GovKeeper)
	require.NoError(t, err)
	require.Equal(t, exportedState, customtypes.DefaultGenesisState())
}
