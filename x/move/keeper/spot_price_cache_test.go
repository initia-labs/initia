package keeper

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	vmtypes "github.com/initia-labs/movevm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestSpotPriceCache_NonCheckTxIgnored(t *testing.T) {
	k := Keeper{
		spotPriceCache: &spotPriceCache{values: make(map[spotPriceCacheKey]math.LegacyDec)},
	}

	key := spotPriceCacheKey{
		poolType:      "dex",
		metadataLP:    vmtypes.AccountAddress{31: 0x01},
		metadataQuote: vmtypes.AccountAddress{31: 0x02},
	}
	price := math.LegacyMustNewDecFromStr("1.23")

	ctx := sdk.WrapSDKContext(sdk.Context{}.WithBlockHeight(1))
	k.setCachedBaseSpotPrice(ctx, key, price)

	got, found := k.getCachedBaseSpotPrice(ctx, key)
	require.False(t, found)
	require.Equal(t, math.LegacyZeroDec(), got)
}

func TestSpotPriceCache_CheckTxAndReCheckTx(t *testing.T) {
	key := spotPriceCacheKey{
		poolType:      "stableswap",
		metadataLP:    vmtypes.AccountAddress{31: 0x03},
		metadataQuote: vmtypes.AccountAddress{31: 0x04},
	}
	price := math.LegacyMustNewDecFromStr("2.5")

	t.Run("checktx", func(t *testing.T) {
		k := Keeper{
			spotPriceCache: &spotPriceCache{values: make(map[spotPriceCacheKey]math.LegacyDec)},
		}
		ctx := sdk.WrapSDKContext(sdk.Context{}.WithBlockHeight(10).WithIsCheckTx(true))

		k.setCachedBaseSpotPrice(ctx, key, price)
		got, found := k.getCachedBaseSpotPrice(ctx, key)
		require.True(t, found)
		require.Equal(t, price, got)
	})

	t.Run("rechecktx", func(t *testing.T) {
		k := Keeper{
			spotPriceCache: &spotPriceCache{values: make(map[spotPriceCacheKey]math.LegacyDec)},
		}
		ctx := sdk.WrapSDKContext(
			sdk.Context{}.WithBlockHeight(11).WithIsCheckTx(true).WithIsReCheckTx(true),
		)

		k.setCachedBaseSpotPrice(ctx, key, price)
		got, found := k.getCachedBaseSpotPrice(ctx, key)
		require.True(t, found)
		require.Equal(t, price, got)
	})
}

func TestSpotPriceCache_HeightInvalidation(t *testing.T) {
	k := Keeper{
		spotPriceCache: &spotPriceCache{values: make(map[spotPriceCacheKey]math.LegacyDec)},
	}

	key := spotPriceCacheKey{
		poolType:      "pool",
		metadataLP:    vmtypes.AccountAddress{31: 0x05},
		metadataQuote: vmtypes.AccountAddress{31: 0x06},
	}
	price := math.LegacyMustNewDecFromStr("3.14")

	ctxH1 := sdk.WrapSDKContext(sdk.Context{}.WithBlockHeight(1).WithIsCheckTx(true))
	k.setCachedBaseSpotPrice(ctxH1, key, price)

	got, found := k.getCachedBaseSpotPrice(ctxH1, key)
	require.True(t, found)
	require.Equal(t, price, got)

	ctxH2 := sdk.WrapSDKContext(sdk.Context{}.WithBlockHeight(2).WithIsCheckTx(true))
	got, found = k.getCachedBaseSpotPrice(ctxH2, key)
	require.False(t, found)
	_ = got

	// cache map should be recreated for new height
	k.setCachedBaseSpotPrice(ctxH2, key, price)
	got, found = k.getCachedBaseSpotPrice(ctxH2, key)
	require.True(t, found)
	require.Equal(t, price, got)
}

func TestSpotPriceCache_NilCache(t *testing.T) {
	k := Keeper{spotPriceCache: nil}
	key := spotPriceCacheKey{
		poolType:      "dex",
		metadataLP:    vmtypes.AccountAddress{31: 0x07},
		metadataQuote: vmtypes.AccountAddress{31: 0x08},
	}

	ctx := sdk.WrapSDKContext(sdk.Context{}.WithBlockHeight(1).WithIsCheckTx(true))
	k.setCachedBaseSpotPrice(ctx, key, math.LegacyOneDec())
	got, found := k.getCachedBaseSpotPrice(ctx, key)
	require.False(t, found)
	require.Equal(t, math.LegacyZeroDec(), got)
}
