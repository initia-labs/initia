package keeper

import (
	"context"
	"sync"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	vmtypes "github.com/initia-labs/movevm/types"
)

type spotPriceCacheKey struct {
	poolType      string
	metadataLP    vmtypes.AccountAddress
	metadataQuote vmtypes.AccountAddress
}

// spotPriceCache is used only in CheckTx/ReCheckTx to keep reads deterministic
// without affecting DeliverTx state.
type spotPriceCache struct {
	mu     sync.RWMutex
	height int64
	values map[spotPriceCacheKey]math.LegacyDec
}

func (k Keeper) getCachedBaseSpotPrice(ctx context.Context, key spotPriceCacheKey) (math.LegacyDec, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !sdkCtx.IsCheckTx() && !sdkCtx.IsReCheckTx() {
		return math.LegacyZeroDec(), false
	}

	cache := k.spotPriceCache
	if cache == nil {
		return math.LegacyZeroDec(), false
	}

	height := sdkCtx.BlockHeight()
	cache.mu.RLock()
	if cache.height == height {
		val, ok := cache.values[key]
		cache.mu.RUnlock()
		return val, ok
	}
	cache.mu.RUnlock()

	cache.mu.Lock()
	defer cache.mu.Unlock()

	if cache.height != height {
		cache.height = height
		cache.values = make(map[spotPriceCacheKey]math.LegacyDec)
	}

	val, ok := cache.values[key]
	return val, ok
}

func (k Keeper) setCachedBaseSpotPrice(ctx context.Context, key spotPriceCacheKey, price math.LegacyDec) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !sdkCtx.IsCheckTx() && !sdkCtx.IsReCheckTx() {
		return
	}

	cache := k.spotPriceCache
	if cache == nil {
		return
	}

	height := sdkCtx.BlockHeight()
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if cache.height != height {
		cache.height = height
		cache.values = make(map[spotPriceCacheKey]math.LegacyDec)
	}

	cache.values[key] = price
}
