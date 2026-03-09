package keeper

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	distrtypes "github.com/initia-labs/initia/x/distribution/types"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

var _ distrtypes.DexKeeper = DexKeeper{}

// DexKeeper implement dex features
type DexKeeper struct {
	*Keeper
}

// NewDexKeeper create new DexKeeper instance
func NewDexKeeper(k *Keeper) DexKeeper {
	return DexKeeper{k}
}

// SetDexPair store DexPair for Quote => LP
func (k DexKeeper) SetDexPair(
	ctx context.Context,
	dexPair types.DexPair,
) error {
	metadataQuote, err := types.AccAddressFromString(k.ac, dexPair.MetadataQuote)
	if err != nil {
		return err
	}

	metadataLP, err := types.AccAddressFromString(k.ac, dexPair.MetadataLP)
	if err != nil {
		return err
	}

	return k.setDexPair(ctx, metadataQuote, metadataLP)
}

// setDexPair stores a dex pair: value = metadataLP[:] (32 bytes).
func (k DexKeeper) setDexPair(
	ctx context.Context,
	metadataQuote vmtypes.AccountAddress,
	metadataLP vmtypes.AccountAddress,
) error {
	return k.DexPairs.Set(ctx, metadataQuote[:], metadataLP[:])
}

// DeleteDexPair remove types.DexPair from the store with the given denom
func (k DexKeeper) DeleteDexPair(
	ctx context.Context,
	denomQuote string,
) error {
	metadata, err := types.MetadataAddressFromDenom(denomQuote)
	if err != nil {
		return err
	}

	return k.deleteDexPair(ctx, metadata)
}

// deleteDexPair remove types.DexPair from the store
func (k DexKeeper) deleteDexPair(
	ctx context.Context,
	metadataQuote vmtypes.AccountAddress,
) error {
	return k.DexPairs.Remove(ctx, metadataQuote[:])
}

// HasDexPair check whether types.DexPair exists or not with
// the given denom
func (k DexKeeper) HasDexPair(
	ctx context.Context,
	denomQuote string,
) (bool, error) {
	metadata, err := types.MetadataAddressFromDenom(denomQuote)
	if err != nil {
		return false, err
	}

	return k.hasDexPair(ctx, metadata)
}

// hasDexPair check whether types.DexPair exists
// or not with the given denom
func (k DexKeeper) hasDexPair(
	ctx context.Context,
	metadataQuote vmtypes.AccountAddress,
) (bool, error) {
	return k.DexPairs.Has(ctx, metadataQuote[:])
}

// IterateDexPair iterate DexPair store for genesis export.
func (k DexKeeper) IterateDexPair(ctx context.Context, cb func(types.DexPair) (bool, error)) error {
	return k.DexPairs.Walk(ctx, nil, func(key, value []byte) (stop bool, err error) {
		metadataQuote, err := vmtypes.NewAccountAddressFromBytes(key)
		if err != nil {
			return true, err
		}

		metadataLP, err := vmtypes.NewAccountAddressFromBytes(value)
		if err != nil {
			return true, err
		}

		return cb(types.DexPair{
			MetadataQuote: metadataQuote.CanonicalString(),
			MetadataLP:    metadataLP.CanonicalString(),
		})
	})
}

// GetMetadataLP return types.DexPair with the given denom
func (k DexKeeper) GetMetadataLP(
	ctx context.Context,
	denomQuote string,
) (vmtypes.AccountAddress, error) {
	metadata, err := types.MetadataAddressFromDenom(denomQuote)
	if err != nil {
		return vmtypes.AccountAddress{}, err
	}

	return k.getMetadataLP(ctx, metadata)
}

// getMetadataLP returns the LP metadata address for the given quote.
func (k DexKeeper) getMetadataLP(
	ctx context.Context,
	metadataQuote vmtypes.AccountAddress,
) (vmtypes.AccountAddress, error) {
	bz, err := k.DexPairs.Get(ctx, metadataQuote[:])
	if err != nil {
		return vmtypes.AccountAddress{}, err
	}

	return vmtypes.NewAccountAddressFromBytes(bz)
}

// GetBaseSpotPrice return base coin spot price
// `base_price` * `quote_amount` == `base_amount`
func (k DexKeeper) GetBaseSpotPrice(
	ctx context.Context,
	denomQuote string,
) (math.LegacyDec, error) {
	metadataQuote, err := types.MetadataAddressFromDenom(denomQuote)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	metadataLP, err := k.getMetadataLP(ctx, metadataQuote)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	return k.getBaseSpotPrice(ctx, metadataQuote, metadataLP)
}

func (k DexKeeper) getBaseSpotPrice(
	ctx context.Context,
	metadataQuote vmtypes.AccountAddress,
	metadataLP vmtypes.AccountAddress,
) (math.LegacyDec, error) {
	if ok, err := k.BalancerKeeper().HasPool(ctx, metadataLP); err != nil {
		return math.LegacyZeroDec(), err
	} else if ok {
		cacheKey := spotPriceCacheKey{
			poolType:      types.MoveModuleNameDex,
			metadataLP:    metadataLP,
			metadataQuote: metadataQuote,
		}
		if cached, found := k.getCachedBaseSpotPrice(ctx, cacheKey); found {
			return cached, nil
		}

		price, err := k.BalancerKeeper().GetBaseSpotPrice(ctx, metadataLP)
		if err != nil {
			return math.LegacyZeroDec(), err
		}
		k.setCachedBaseSpotPrice(ctx, cacheKey, price)
		return price, nil
	}

	if ok, err := k.StableSwapKeeper().HasPool(ctx, metadataLP); err != nil {
		return math.LegacyZeroDec(), err
	} else if ok {
		cacheKey := spotPriceCacheKey{
			poolType:      types.MoveModuleNameStableSwap,
			metadataLP:    metadataLP,
			metadataQuote: metadataQuote,
		}
		if cached, found := k.getCachedBaseSpotPrice(ctx, cacheKey); found {
			return cached, nil
		}

		price, err := k.StableSwapKeeper().GetBaseSpotPrice(ctx, metadataQuote, metadataLP)
		if err != nil {
			return math.LegacyZeroDec(), err
		}
		k.setCachedBaseSpotPrice(ctx, cacheKey, price)
		return price, nil
	}

	// CLAMM pool: module address comes from params
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.LegacyZeroDec(), err
	}
	if params.ClammModuleAddress != "" {
		clammModuleAddr, err := types.AccAddressFromString(k.ac, params.ClammModuleAddress)
		if err != nil {
			return math.LegacyZeroDec(), err
		}
		clammKeeper := NewCLAMMKeeper(k.Keeper, clammModuleAddr)
		if ok, err := clammKeeper.HasPool(ctx, metadataLP); err != nil {
			return math.LegacyZeroDec(), err
		} else if ok {
			cacheKey := spotPriceCacheKey{
				poolType:      types.MoveModuleNameCLAMMPool,
				metadataLP:    metadataLP,
				metadataQuote: metadataQuote,
			}
			if cached, found := k.getCachedBaseSpotPrice(ctx, cacheKey); found {
				return cached, nil
			}
			price, err := clammKeeper.GetBaseSpotPrice(ctx, metadataQuote, metadataLP)
			if err != nil {
				return math.LegacyZeroDec(), err
			}
			k.setCachedBaseSpotPrice(ctx, cacheKey, price)
			return price, nil
		}
	}

	return math.LegacyZeroDec(), types.ErrInvalidRequest.Wrapf("LP `%s` is not a supported DEX pool", metadataLP.String())
}

func (k DexKeeper) SwapToBase(
	ctx context.Context,
	addr sdk.AccAddress,
	quoteCoin sdk.Coin,
) error {
	vmAddr, err := vmtypes.NewAccountAddressFromBytes(addr[:])
	if err != nil {
		return err
	}

	metadataQuote, err := types.MetadataAddressFromDenom(quoteCoin.Denom)
	if err != nil {
		return err
	}

	// if the quote denom is not whitelisted, then skip operation
	if found, err := k.hasDexPair(ctx, metadataQuote); err != nil {
		return err
	} else if !found {
		return nil
	}

	metadataLP, err := k.getMetadataLP(ctx, metadataQuote)
	if err != nil {
		return err
	}

	if ok, err := k.BalancerKeeper().HasPool(ctx, metadataLP); err != nil {
		return err
	} else if ok {
		return k.BalancerKeeper().SwapToBase(ctx, vmAddr, metadataLP, metadataQuote, quoteCoin.Amount)
	}

	if ok, err := k.StableSwapKeeper().HasPool(ctx, metadataLP); err != nil {
		return err
	} else if ok {
		return k.StableSwapKeeper().SwapToBase(ctx, vmAddr, metadataLP, metadataQuote, quoteCoin.Amount)
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	if params.ClammModuleAddress != "" {
		clammModuleAddr, err := types.AccAddressFromString(k.ac, params.ClammModuleAddress)
		if err != nil {
			return err
		}

		clammKeeper := NewCLAMMKeeper(k.Keeper, clammModuleAddr)
		if ok, err := clammKeeper.HasPool(ctx, metadataLP); err != nil {
			return err
		} else if ok {
			return clammKeeper.SwapToBase(ctx, vmAddr, metadataLP, metadataQuote, quoteCoin.Amount)
		}
	}

	return types.ErrInvalidRequest.Wrapf("LP `%s` is not a supported DEX pool", metadataLP.String())
}
