package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/stretchr/testify/require"
)

func TestParams(t *testing.T) {
	ac := address.NewBech32Codec("init")

	p1 := DefaultParams()
	require.NoError(t, p1.Validate(ac))

	p1.ContractSharedRevenueRatio = math.LegacyOneDec().Neg()
	require.Error(t, p1.Validate(ac))

	p2 := DefaultParams()
	p2.BaseDenom = "v#ksdjf"
	err := p2.Validate(ac)
	require.Error(t, err)

	p3 := DefaultParams()
	p3.AllowedPublishers = []string{"abc"}
	err = p3.Validate(ac)
	require.Error(t, err)
}

func TestRawParams(t *testing.T) {
	ac := address.NewBech32Codec("init")

	p1 := DefaultParams()
	require.NoError(t, p1.Validate(ac))

	p1.ContractSharedRevenueRatio = math.LegacyOneDec()
	p1.BaseDenom = "venusinthemorning"
	require.NoError(t, p1.Validate(ac))

	rp := p1.ToRaw()
	p2 := rp.ToParams(p1.AllowedPublishers)
	require.NoError(t, p2.Validate(ac))
	require.Equal(t, p1, p2)
}
