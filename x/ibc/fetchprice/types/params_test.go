package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateParams(t *testing.T) {
	require.NoError(t, DefaultParams().Validate())
	require.Error(t, NewParams(true, true, 0).Validate())
}
