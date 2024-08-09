package types_test

import (
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

func Test_GetQuoteSpotPrice(t *testing.T) {
	price := types.GetQuoteSpotPrice(
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
