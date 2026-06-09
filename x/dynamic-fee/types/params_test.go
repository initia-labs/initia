package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/dynamic-fee/types"
)

func TestParamsValidateTargetGas(t *testing.T) {
	params := types.DefaultParams()
	require.NoError(t, params.Validate())

	params.TargetGas = 0
	require.ErrorContains(t, params.Validate(), "target gas must be positive")

	params.TargetGas = -1
	require.ErrorContains(t, params.Validate(), "target gas must be positive")
}
