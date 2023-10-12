package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/types"
)

func Test_GetDexWeight(t *testing.T) {
	weightBase, weightQuote, err := types.GetPoolWeights(
		sdk.NewDecWithPrec(5, 1), // 50%
		sdk.NewDecWithPrec(5, 1), // 50%
		sdk.NewDecWithPrec(7, 1), // 70%
		sdk.NewDecWithPrec(3, 1), // 30%
		sdk.NewInt(1_000_000),
		sdk.NewInt(2_000_000),
		sdk.NewInt(1_500_000),
	)

	require.NoError(t, err)
	require.Equal(t, sdk.NewDecWithPrec(6, 1), weightBase)
	require.Equal(t, sdk.NewDecWithPrec(4, 1), weightQuote)
}

func Test_GetPoolSpotPrice(t *testing.T) {
	price := types.GetPoolSpotPrice(
		sdk.NewInt(1_000_000),
		sdk.NewInt(4_000_000),
		sdk.NewDecWithPrec(2, 1),
		sdk.NewDecWithPrec(8, 1),
	)

	require.Equal(t, sdk.OneDec(), price)
}

func Test_DeserializeUint128(t *testing.T) {
	num, err := types.DeserializeUint128([]byte{0x15, 0x14, 0x13, 0x12, 0x11, 0x10, 0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00})
	require.NoError(t, err)
	require.Equal(t, strings.TrimLeft("00010203040506070809101112131415", "0"), num.BigInt().Text(16))
}
