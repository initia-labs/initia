package types_test

import (
	"math/big"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/move/types"
)

func Test_GetDexWeight(t *testing.T) {
	weightBase, weightQuote, err := types.GetPoolWeights(
		math.LegacyNewDecWithPrec(5, 1), // 50%
		math.LegacyNewDecWithPrec(5, 1), // 50%
		math.LegacyNewDecWithPrec(7, 1), // 70%
		math.LegacyNewDecWithPrec(3, 1), // 30%
		math.NewInt(1_000_000),
		math.NewInt(2_000_000),
		math.NewInt(1_500_000),
	)

	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecWithPrec(6, 1), weightBase)
	require.Equal(t, math.LegacyNewDecWithPrec(4, 1), weightQuote)
}

func Test_GetBaseSpotPrice(t *testing.T) {
	price := types.GetBaseSpotPrice(
		math.NewInt(1_000_000),
		math.NewInt(8_000_000),
		math.LegacyNewDecWithPrec(2, 1),
		math.LegacyNewDecWithPrec(8, 1),
	)

	require.Equal(t, math.LegacyNewDecWithPrec(5, 1), price)
}

func Test_DeserializeUint128(t *testing.T) {
	num, err := types.DeserializeUint128([]byte{0x15, 0x14, 0x13, 0x12, 0x11, 0x10, 0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00})
	require.NoError(t, err)
	require.Equal(t, strings.TrimLeft("00010203040506070809101112131415", "0"), num.BigInt().Text(16))
}

func Test_CLAMMBaseSpotPrice(t *testing.T) {
	two64 := new(big.Int).Lsh(big.NewInt(1), 64)
	two128 := new(big.Int).Lsh(big.NewInt(1), 128)
	maxU128 := new(big.Int).Sub(new(big.Int).Set(two128), big.NewInt(1))

	testCases := []struct {
		name            string
		sqrtPrice       *big.Int
		isBase0         bool
		expected        math.LegacyDec
		expectZeroPrice bool
	}{
		{
			name:      "sqrt_2_pow_64_base0",
			sqrtPrice: two64,
			isBase0:   true,
			expected:  math.LegacyOneDec(),
		},
		{
			name:      "sqrt_2_pow_64_base1",
			sqrtPrice: two64,
			isBase0:   false,
			expected:  math.LegacyOneDec(),
		},
		{
			name:      "sqrt_one_base0",
			sqrtPrice: big.NewInt(1),
			isBase0:   true,
			expected:  math.LegacyNewDecFromInt(math.NewIntFromBigInt(two128)),
		},
		{
			name:            "sqrt_one_base1",
			sqrtPrice:       big.NewInt(1),
			isBase0:         false,
			expected:        math.LegacyZeroDec(),
			expectZeroPrice: true,
		},
		{
			name:            "sqrt_max_u128_base0",
			sqrtPrice:       maxU128,
			isBase0:         true,
			expected:        math.LegacyZeroDec(),
			expectZeroPrice: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sqrtPrice := math.NewIntFromBigInt(new(big.Int).Set(tc.sqrtPrice))
			price, err := types.CLAMMBaseSpotPrice(sqrtPrice, tc.isBase0)
			require.NoError(t, err)
			require.Equal(t, tc.expected, price)

			// repeat calls to ensure deterministic output for edge values
			for i := 0; i < 5; i++ {
				repeated, err := types.CLAMMBaseSpotPrice(sqrtPrice, tc.isBase0)
				require.NoError(t, err)
				require.Equal(t, price, repeated)
			}

			if tc.expectZeroPrice {
				require.True(t, price.IsZero())
			}
		})
	}
}

func Test_CLAMMBaseSpotPrice_ZeroSqrt(t *testing.T) {
	price, err := types.CLAMMBaseSpotPrice(math.ZeroInt(), true)
	require.Error(t, err)
	require.ErrorContains(t, err, "sqrt_price is zero")
	require.Equal(t, math.LegacyZeroDec(), price)
}
