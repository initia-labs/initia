package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestParams(t *testing.T) {
	p1 := DefaultParams()
	require.NoError(t, p1.Validate())

	p1.MaxModuleSize = 1000
	err := p1.Validate()
	require.Error(t, err)

	p2 := DefaultParams()
	p2.BaseDenom = "v#ksdjf"
	err = p2.Validate()
	require.Error(t, err)

	p3 := DefaultParams()
	p3.BaseMinGasPrice = sdk.OneDec().Neg()
	err = p3.Validate()
	require.Error(t, err)
}
