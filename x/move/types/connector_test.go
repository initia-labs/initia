package types_test

import (
	"encoding/base64"
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
			name:      "sqrt_testnet_sample_base1",
			sqrtPrice: mustBigInt("19039117487731833422"),
			isBase0:   false,
			expected:  math.LegacyMustNewDecFromStr("1.065256475061127495"),
		},
		{
			name:      "sqrt_testnet_sample_base0",
			sqrtPrice: mustBigInt("19039117487731833422"),
			isBase0:   true,
			expected:  math.LegacyMustNewDecFromStr("0.938741066974145460"),
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
			for range 5 {
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

func Test_ReadCLAMMPool_TestnetSample(t *testing.T) {
	rawBytes := "KYJNlS4DVJD651Z97qXxW1BKaPpzYQBjwWCrH6h91gmORzO9q899Svw9FPDdRsm/UvsPzp5LmWyTnhlbi8iR2aan2opf8er2hPx6I3osHXNRnyVY1WGdvjK36yE3n1B3pqfail/x6vaE/Hojeiwdc1GfJVjVYZ2+MrfrITefUHfu4677dQ85kI4+ztrEbJgPLaWmoPk5OishzmdVjpCFiQUAAAAAAAAACgAAAAAAAAARUryG2qNmBiMP6Xsl45pjG1NKg8WhFUdqZ3u8QW/OPPeyXvm7jXgDAQAAAAAAAAASAQAAAAAAAE76LkV7iDgIAQAAAAAAAAB4AgAAAAAAABIqQV/L9/cyxY6jAFyeRKOXisCyQ/czmnvvfZNA1nUOXAAAAAAAAABnlNs5cO3Xlcgobp4xpeqGU4uX0TbBJdkjajHFt542mwkAAAAAAAAAMo/HBQgAAAAAAAAAAAAAALEuVV1IAgiAAwAAAAAAAAB0oxtV9rwSgFoAAAAAAAAAAA=="

	bz, err := base64.StdEncoding.DecodeString(rawBytes)
	require.NoError(t, err)

	metadata0, metadata1, sqrtPrice, err := types.ReadCLAMMPool(bz)
	require.NoError(t, err)
	require.Equal(t, "0x29824d952e035490fae7567deea5f15b504a68fa73610063c160ab1fa87dd609", metadata0.String())
	require.Equal(t, "0x8e4733bdabcf7d4afc3d14f0dd46c9bf52fb0fce9e4b996c939e195b8bc891d9", metadata1.String())
	require.Equal(t, "19039117487731833422", sqrtPrice.String())
}

func mustBigInt(v string) *big.Int {
	out, ok := new(big.Int).SetString(v, 10)
	if !ok {
		panic("invalid big.Int literal")
	}

	return out
}
