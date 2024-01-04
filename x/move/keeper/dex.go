package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	distrtypes "github.com/initia-labs/initia/x/distribution/types"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
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
func (k Keeper) GetMetadataLP(
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
func (k Keeper) getMetadataLP(
	ctx context.Context,
	metadataQuote vmtypes.AccountAddress,
) (vmtypes.AccountAddress, error) {
	bz, err := k.DexPairs.Get(ctx, metadataQuote[:])
	if err != nil {
		return vmtypes.AccountAddress{}, err
	}

	return vmtypes.NewAccountAddressFromBytes(bz)
}

// GetPoolSpotPrice return quote price in base unit
// `price` * `quote_amount` == `quote_value_in_base_unit`
func (k DexKeeper) GetPoolSpotPrice(
	ctx context.Context,
	denomQuote string,
) (math.LegacyDec, error) {
	metadataLP, err := k.GetMetadataLP(ctx, denomQuote)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	balanceBase, balanceQuote, weightBase, weightQuote, err := k.getPoolInfo(ctx, metadataLP)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	return types.GetPoolSpotPrice(balanceBase, balanceQuote, weightBase, weightQuote), nil
}

func (k DexKeeper) getPoolInfo(ctx context.Context, metadataLP vmtypes.AccountAddress) (
	balanceBase math.Int,
	balanceQuote math.Int,
	weightBase math.LegacyDec,
	weightQuote math.LegacyDec,
	err error,
) {
	weightBase, weightQuote, err = k.getPoolWeights(ctx, metadataLP)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), math.LegacyZeroDec(), math.LegacyZeroDec(), err
	}

	balanceBase, balanceQuote, err = k.getPoolBalances(ctx, metadataLP)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), math.LegacyZeroDec(), math.LegacyZeroDec(), err
	}

	return
}

// GetPoolBalances return move dex pool info
func (k DexKeeper) GetPoolBalances(
	ctx context.Context,
	denom string,
) (
	balanceBase math.Int,
	balanceQuote math.Int,
	err error,
) {
	metadataLP, err := k.GetMetadataLP(ctx, denom)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	return k.getPoolBalances(ctx, metadataLP)
}

func (k DexKeeper) isReverse(
	ctx context.Context,
	metadataLP vmtypes.AccountAddress,
) (bool, error) {
	denomBase, err := k.BaseDenom(ctx)
	if err != nil {
		return false, err
	}

	metadataBase, err := types.MetadataAddressFromDenom(denomBase)
	if err != nil {
		return false, err
	}

	metadataA, metadataB, err := k.GetPoolMetadata(ctx, metadataLP)
	if err != nil {
		return false, err
	}

	if metadataBase == metadataA {
		return false, nil
	} else if metadataBase == metadataB {
		return true, nil
	}

	return false, types.ErrInvalidDexConfig.Wrapf("the pair does not contain `%s`", denomBase)
}

// getPoolBalances return move dex pool info
func (k DexKeeper) getPoolBalances(
	ctx context.Context,
	metadataLP vmtypes.AccountAddress,
) (balanceBase, balanceQuote math.Int, err error) {
	bz, err := k.GetResourceBytes(ctx, metadataLP, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameDex,
		Name:     types.ResourceNamePool,
		TypeArgs: []vmtypes.TypeTag{},
	})

	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return math.ZeroInt(), math.ZeroInt(), nil
	} else if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	storeA, storeB, err := types.ReadStoresFromPool(bz)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	_, balanceA, err := NewMoveBankKeeper(k.Keeper).Balance(ctx, storeA)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	_, balanceB, err := NewMoveBankKeeper(k.Keeper).Balance(ctx, storeB)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	if ok, err := k.isReverse(ctx, metadataLP); err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	} else if ok {
		return balanceB, balanceA, nil
	}

	return balanceA, balanceB, nil
}

// GetPoolWeights return base, quote dex weights
func (k DexKeeper) GetPoolWeights(
	ctx context.Context,
	denomQuote string,
) (weightBase math.LegacyDec, weightB math.LegacyDec, err error) {
	metadataLP, err := k.GetMetadataLP(ctx, denomQuote)
	if err != nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), err
	}

	return k.getPoolWeights(ctx, metadataLP)
}

// getPoolWeights return base, quote dex weights with quote denom struct tag
func (k DexKeeper) getPoolWeights(
	ctx context.Context,
	metadataLP vmtypes.AccountAddress,
) (weightBase math.LegacyDec, weightQuote math.LegacyDec, err error) {
	bz, err := k.GetResourceBytes(ctx, metadataLP, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameDex,
		Name:     types.ResourceNameConfig,
		TypeArgs: []vmtypes.TypeTag{},
	})

	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), nil
	} else if err != nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	timestamp := math.NewInt(sdkCtx.BlockTime().Unix())
	weightA, weightB, err := types.ReadWeightsFromDexConfig(timestamp, bz)
	if err != nil {
		return math.LegacyDec{}, math.LegacyDec{}, err
	}

	if ok, err := k.isReverse(ctx, metadataLP); err != nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), err
	} else if ok {
		return weightB, weightA, nil
	}

	return weightA, weightB, nil
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

	// build argument bytes
	offerAmountBz, err := vmtypes.SerializeUint64(quoteCoin.Amount.Uint64())
	if err != nil {
		return err
	}

	// swap quote coin to base coin
	return k.ExecuteEntryFunction(
		ctx,
		vmAddr,
		vmtypes.StdAddress,
		types.MoveModuleNameDex,
		types.FunctionNameDexSwap,
		[]vmtypes.TypeTag{},
		[][]byte{metadataLP[:], metadataQuote[:], offerAmountBz, {0}},
	)
}

func (k DexKeeper) GetPoolMetadata(
	ctx context.Context,
	metadataLP vmtypes.AccountAddress,
) (metadataA, metadataB vmtypes.AccountAddress, err error) {
	bz, err := k.GetResourceBytes(ctx, metadataLP, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameDex,
		Name:     types.ResourceNamePool,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil {
		return vmtypes.AccountAddress{}, vmtypes.AccountAddress{}, err
	}

	storeA, storeB, err := types.ReadStoresFromPool(bz)
	if err != nil {
		return vmtypes.AccountAddress{}, vmtypes.AccountAddress{}, err
	}

	metadataA, _, err = NewMoveBankKeeper(k.Keeper).Balance(ctx, storeA)
	if err != nil {
		return vmtypes.AccountAddress{}, vmtypes.AccountAddress{}, err
	}

	metadataB, _, err = NewMoveBankKeeper(k.Keeper).Balance(ctx, storeB)
	if err != nil {
		return vmtypes.AccountAddress{}, vmtypes.AccountAddress{}, err
	}

	return metadataA, metadataB, nil
}
