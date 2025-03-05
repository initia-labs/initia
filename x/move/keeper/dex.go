package keeper

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	distrtypes "github.com/initia-labs/initia/v1/x/distribution/types"
	"github.com/initia-labs/initia/v1/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

var _ types.AnteKeeper = DexKeeper{}
var _ distrtypes.DexKeeper = DexKeeper{}

// DexKeeper implement dex features
type DexKeeper struct {
	*Keeper
}

// NewDexKeeper create new DexKeeper instance
func NewDexKeeper(k *Keeper) DexKeeper {
	return DexKeeper{k}
}

// SetDexPair store DexPair for both counterpart
// and LP coins
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

// setDexPair store DexPair for both counterpart
// and LP coins
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

// IterateDexPair iterate DexPair store for genesis export
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

// getMetadataLP return types.DexPair with the given
// metadata
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
	metadataLP, err := k.GetMetadataLP(ctx, denomQuote)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	return k.getBaseSpotPrice(ctx, metadataLP)
}

func (k DexKeeper) getBaseSpotPrice(
	ctx context.Context,
	metadataLP vmtypes.AccountAddress,
) (math.LegacyDec, error) {
	// for now, we only support balancer dex
	return k.BalancerKeeper().GetBaseSpotPrice(ctx, metadataLP)
}

// GasPrices return gas prices for all dex pairs
func (k DexKeeper) GasPrices(
	ctx context.Context,
) (sdk.DecCoins, error) {
	baseGasPrice, err := k.BaseMinGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	gasPrices := sdk.NewDecCoins()
	err = k.DexPairs.Walk(ctx, nil, func(key, value []byte) (stop bool, err error) {
		metadataQuote, err := vmtypes.NewAccountAddressFromBytes(key)
		if err != nil {
			return true, err
		}
		denomQuote, err := types.DenomFromMetadataAddress(ctx, k.MoveBankKeeper(), metadataQuote)
		if err != nil {
			return true, err
		}
		metadataLP, err := vmtypes.NewAccountAddressFromBytes(value)
		if err != nil {
			return true, err
		}
		baseSpotPrice, err := k.getBaseSpotPrice(ctx, metadataLP)
		if err != nil {
			return true, err
		}
		if baseSpotPrice.IsZero() {
			return true, errors.New("baseSpotPrice is zero")
		}

		gasPrice := baseGasPrice.Quo(baseSpotPrice)
		gasPrices = gasPrices.Add(sdk.NewDecCoinFromDec(denomQuote, gasPrice))
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return gasPrices, nil
}

// GasPrice return gas price for the given denom
func (k DexKeeper) GasPrice(
	ctx context.Context,
	denomQuote string,
) (sdk.DecCoin, error) {
	baseGasPrice, err := k.BaseMinGasPrice(ctx)
	if err != nil {
		return sdk.NewDecCoin(denomQuote, math.ZeroInt()), err
	}

	baseSpotPrice, err := k.GetBaseSpotPrice(ctx, denomQuote)
	if err != nil {
		return sdk.NewDecCoin(denomQuote, math.ZeroInt()), err
	}
	if baseSpotPrice.IsZero() {
		return sdk.NewDecCoin(denomQuote, math.ZeroInt()), errors.New("baseSpotPrice is zero")
	}

	return sdk.NewDecCoinFromDec(denomQuote, baseGasPrice.Quo(baseSpotPrice)), nil
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

	// for now, we only support balancer dex
	return k.BalancerKeeper().SwapToBase(ctx, vmAddr, metadataLP, metadataQuote, quoteCoin.Amount)
}

func (k DexKeeper) PoolBalances(
	ctx context.Context,
	denomQuote string,
) ([]math.Int, error) {
	metadataLP, err := k.GetMetadataLP(ctx, denomQuote)
	if err != nil {
		return nil, err
	}

	// for now, we only support balancer dex
	return k.BalancerKeeper().poolBalances(ctx, metadataLP)
}

func (k DexKeeper) PoolWeights(
	ctx context.Context,
	denomQuote string,
) ([]math.LegacyDec, error) {
	metadataLP, err := k.GetMetadataLP(ctx, denomQuote)
	if err != nil {
		return nil, err
	}

	// for now, we only support balancer dex
	return k.BalancerKeeper().poolWeights(ctx, metadataLP)
}
