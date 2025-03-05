package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/v1/x/mstaking/types"
)

func TestParamsEqual(t *testing.T) {
	p1 := types.DefaultParams()
	p2 := types.DefaultParams()

	ok := p1.Equal(p2)
	require.True(t, ok)

	p2.UnbondingTime = 60 * 60 * 24 * 2
	p2.BondDenoms = []string{"soup"}

	ok = p1.Equal(p2)
	require.False(t, ok)
}
