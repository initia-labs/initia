package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestParams(t *testing.T) {
	p1 := DefaultParams()
	require.NoError(t, p1.Validate())

	p1.ContractSharedRevenueRatio = sdk.OneDec().Neg()
	require.Error(t, p1.Validate())

	p2 := DefaultParams()
	p2.BaseDenom = "v#ksdjf"
	err := p2.Validate()
	require.Error(t, err)

	p3 := DefaultParams()
	p3.BaseMinGasPrice = sdk.OneDec().Neg()
	err = p3.Validate()
	require.Error(t, err)
}

func TestRawParams(t *testing.T) {
	p1 := DefaultParams()
	require.NoError(t, p1.Validate())

	p1.ContractSharedRevenueRatio = sdk.OneDec()
	p1.BaseDenom = "venusinthemorning"
	p1.BaseMinGasPrice = sdk.OneDec()
	require.NoError(t, p1.Validate())

	rp := p1.ToRaw()
	p2 := rp.ToParams(p1.ArbitraryEnabled)
	require.NoError(t, p2.Validate())
	require.Equal(t, p1, p2)
}
