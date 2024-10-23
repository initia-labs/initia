package keeper

import (
	"context"
	"errors"
	"slices"

	"cosmossdk.io/collections"
	moderrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

type BalancerKeeper struct {
	*Keeper
}

func NewBalancerKeeper(k *Keeper) BalancerKeeper {
	return BalancerKeeper{k}
}

func (k BalancerKeeper) HasPool(ctx context.Context, metadataLP vmtypes.AccountAddress) (bool, error) {
	return k.HasResource(ctx, metadataLP, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameDex,
		Name:     types.ResourceNamePool,
		TypeArgs: []vmtypes.TypeTag{},
	})
}

// GetBaseSpotPrice return base coin spot price
// `base_price` * `quote_amount` == `base_amount`
func (k BalancerKeeper) GetBaseSpotPrice(
	ctx context.Context,
	metadataLP vmtypes.AccountAddress,
) (math.LegacyDec, error) {
	balances, weights, err := k.getPoolInfo(ctx, metadataLP)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	return types.GetBaseSpotPrice(balances[0], balances[1], weights[0], weights[1]), nil
}

func (k BalancerKeeper) SwapToBase(
	ctx context.Context,
	trader vmtypes.AccountAddress,
	metadataLP, metadataQuote vmtypes.AccountAddress,
	quoteAmount math.Int,
) error {
	// build argument bytes
	offerAmountBz, err := vmtypes.SerializeUint64(quoteAmount.Uint64())
	if err != nil {
		return err
	}

	// swap quote coin to base coin
	return k.ExecuteEntryFunction(
		ctx,
		trader,
		vmtypes.StdAddress,
		types.MoveModuleNameDex,
		types.FunctionNameDexSwapScript,
		[]vmtypes.TypeTag{},
		[][]byte{metadataLP[:], metadataQuote[:], offerAmountBz, {0}},
	)
}

func (k BalancerKeeper) Whitelist(ctx context.Context, metadataLP vmtypes.AccountAddress) error {
	ok, err := k.HasPool(ctx, metadataLP)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	// assert base denom is exist in the dex pair

	denomBase, err := k.BaseDenom(ctx)
	if err != nil {
		return err
	}

	metadataBase, err := types.MetadataAddressFromDenom(denomBase)
	if err != nil {
		return err
	}

	metadata, err := k.poolMetadata(ctx, metadataLP)
	if err != nil {
		return err
	}

	if !slices.Contains(metadata, metadataBase) {
		return moderrors.Wrapf(
			types.ErrInvalidDexConfig,
			"To be whitelisted, a stableswap should contain `%s` in its pair", denomBase,
		)
	}

	var metadataQuote vmtypes.AccountAddress
	if metadataBase == metadata[0] {
		metadataQuote = metadata[1]
	} else if metadataBase == metadata[1] {
		metadataQuote = metadata[0]
	} else {
		return moderrors.Wrapf(
			types.ErrInvalidDexConfig,
			"To be whitelisted, a dex should contain `%s` in its pair", denomBase,
		)
	}

	//
	// compute weights and assert base weight is bigger than quote weight
	//

	weights, err := k.poolWeights(ctx, metadataLP)
	if err != nil {
		return err
	}

	if weights[0].LT(weights[1]) {
		return moderrors.Wrapf(types.ErrInvalidDexConfig,
			"base weight `%s` must be bigger than quote weight `%s`", weights[0], weights[1])
	}

	// check dex pair was registered

	if found, err := k.DexKeeper().hasDexPair(ctx, metadataQuote); err != nil {
		return err
	} else if found {
		return moderrors.Wrapf(types.ErrInvalidRequest, "coin `%s` was already whitelisted", metadataQuote.String())
	}

	// store dex pair
	err = k.DexKeeper().setDexPair(ctx, metadataQuote, metadataLP)
	if err != nil {
		return err
	}

	return nil
}

func (k BalancerKeeper) Delist(
	ctx context.Context,
	metadataLP vmtypes.AccountAddress,
) error {
	ok, err := k.HasPool(ctx, metadataLP)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	metadata, err := k.poolMetadata(ctx, metadataLP)
	if err != nil {
		return err
	}

	for _, metadata := range metadata {
		err = k.DexKeeper().deleteDexPair(ctx, metadata)
		if err != nil {
			return err
		}
	}

	return nil
}

func (k BalancerKeeper) getPoolInfo(ctx context.Context, metadataLP vmtypes.AccountAddress) (
	balances []math.Int,
	weights []math.LegacyDec,
	err error,
) {
	weights, err = k.poolWeights(ctx, metadataLP)
	if err != nil {
		return nil, nil, err
	}

	balances, err = k.poolBalances(ctx, metadataLP)
	if err != nil {
		return nil, nil, err
	}

	return
}

func (k BalancerKeeper) poolMetadata(ctx context.Context, metadataLP vmtypes.AccountAddress) ([]vmtypes.AccountAddress, error) {
	bz, err := k.GetResourceBytes(ctx, metadataLP, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameDex,
		Name:     types.ResourceNamePool,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil {
		return nil, err
	}

	storeA, storeB, err := types.ReadStoresFromPool(bz)
	if err != nil {
		return nil, err
	}

	metadataA, _, err := k.Keeper.MoveBankKeeper().Balance(ctx, storeA)
	if err != nil {
		return nil, err
	}

	metadataB, _, err := k.Keeper.MoveBankKeeper().Balance(ctx, storeB)
	if err != nil {
		return nil, err
	}

	return []vmtypes.AccountAddress{metadataA, metadataB}, nil
}

func (k BalancerKeeper) poolBalances(ctx context.Context, metadataLP vmtypes.AccountAddress) (balances []math.Int, err error) {
	bz, err := k.GetResourceBytes(ctx, metadataLP, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameDex,
		Name:     types.ResourceNamePool,
		TypeArgs: []vmtypes.TypeTag{},
	})

	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return []math.Int{math.ZeroInt(), math.ZeroInt()}, nil
	} else if err != nil {
		return nil, err
	}

	storeA, storeB, err := types.ReadStoresFromPool(bz)
	if err != nil {
		return nil, err
	}

	_, balanceA, err := k.Keeper.MoveBankKeeper().Balance(ctx, storeA)
	if err != nil {
		return nil, err
	}

	_, balanceB, err := k.Keeper.MoveBankKeeper().Balance(ctx, storeB)
	if err != nil {
		return nil, err
	}

	if ok, err := k.isReverse(ctx, metadataLP); err != nil {
		return nil, err
	} else if ok {
		return []math.Int{balanceB, balanceA}, nil
	}

	return []math.Int{balanceA, balanceB}, nil
}

// poolWeights return base, quote dex weights with quote denom struct tag
func (k BalancerKeeper) poolWeights(
	ctx context.Context,
	metadataLP vmtypes.AccountAddress,
) ([]math.LegacyDec, error) {
	bz, err := k.GetResourceBytes(ctx, metadataLP, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameDex,
		Name:     types.ResourceNameConfig,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return []math.LegacyDec{math.LegacyZeroDec(), math.LegacyZeroDec()}, nil
	} else if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	timestamp := math.NewInt(sdkCtx.BlockTime().Unix())
	weightA, weightB, err := types.ReadWeightsFromDexConfig(timestamp, bz)
	if err != nil {
		return nil, err
	}

	if ok, err := k.isReverse(ctx, metadataLP); err != nil {
		return nil, err
	} else if ok {
		return []math.LegacyDec{weightB, weightA}, nil
	}

	return []math.LegacyDec{weightA, weightB}, nil
}

func (k BalancerKeeper) isReverse(
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

	metadata, err := k.poolMetadata(ctx, metadataLP)
	if err != nil {
		return false, err
	}

	if metadataBase == metadata[0] {
		return false, nil
	} else if metadataBase == metadata[1] {
		return true, nil
	}

	return false, types.ErrInvalidDexConfig.Wrapf("the pair does not contain `%s`", denomBase)
}
