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

// NewBalancerKeeper returns a balancer-specific keeper wrapper.
func NewBalancerKeeper(k *Keeper) BalancerKeeper {
	return BalancerKeeper{k}
}

// HasPool reports whether a balancer pool exists for the given LP metadata.
func (k BalancerKeeper) HasPool(ctx context.Context, metadataLP vmtypes.AccountAddress) (bool, error) {
	return k.HasResource(ctx, metadataLP, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameDex,
		Name:     types.ResourceNamePool,
		TypeArgs: []vmtypes.TypeTag{},
	})
}

// GetBaseSpotPrice returns the base asset spot price for the pool.
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

// SwapToBase executes a sudo swap from quote asset to base asset in the pool.
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
	return k.executeEntryFunction(
		ctx,
		[]vmtypes.AccountAddress{vmtypes.StdAddress, trader},
		vmtypes.StdAddress,
		types.MoveModuleNameDex,
		types.FunctionNameDexSudoSwap,
		[]vmtypes.TypeTag{},
		[][]byte{metadataLP[:], metadataQuote[:], offerAmountBz, {0}},
		false,
	)
}

// WhitelistStaking validates a balancer LP for staking whitelist registration.
func (k BalancerKeeper) WhitelistStaking(ctx context.Context, metadataLP vmtypes.AccountAddress) (bool, error) {
	_, ok, err := k.validation(ctx, metadataLP)
	return ok, err
}

// WhitelistGasPrice validates that the balancer LP can be whitelisted for gas-price usage.
// It returns (true, nil) on success, (false, nil) if the LP is not a balancer pool,
// and (false, err) on a validation error.
// Store operations are intentionally omitted; they are handled by whitelist.go.
func (k BalancerKeeper) WhitelistGasPrice(ctx context.Context, metadataQuote, metadataLP vmtypes.AccountAddress) (bool, error) {
	metadataQuotePtr, ok, err := k.validation(ctx, metadataLP)
	if !ok || err != nil {
		return ok, err
	}
	if metadataQuotePtr == nil {
		return false, moderrors.Wrap(types.ErrInvalidRequest, "failed to resolve quote metadata for the given LP")
	}

	// Ensure the provided quote metadata matches the quote asset in the LP pair.
	if metadataQuote != *metadataQuotePtr {
		return false, moderrors.Wrapf(
			types.ErrInvalidRequest,
			"invalid quote metadata: expected `%s`, got `%s`",
			metadataQuotePtr.String(),
			metadataQuote.String(),
		)
	}

	return true, nil
}

// DelistStaking is a no-op for balancer staking delist.
func (k BalancerKeeper) DelistStaking(
	ctx context.Context,
	metadataLP vmtypes.AccountAddress,
) error {
	// no-op for now
	return nil
}

// DelistGasPrice validates that the balancer LP contains metadataQuote.
// Returns (true, nil) on success, (false, nil) if the LP is not a balancer pool,
// and (false, err) on a validation error.
// Store operations are intentionally omitted; they are handled by whitelist.go.
func (k BalancerKeeper) DelistGasPrice(
	ctx context.Context,
	metadataQuote vmtypes.AccountAddress,
	metadataLP vmtypes.AccountAddress,
) (bool, error) {
	if ok, err := k.HasPool(ctx, metadataLP); err != nil {
		return false, err
	} else if !ok {
		return false, nil
	}

	metadata, err := k.poolMetadata(ctx, metadataLP)
	if err != nil {
		return false, err
	}

	if !slices.Contains(metadata, metadataQuote) {
		return false, moderrors.Wrapf(
			types.ErrInvalidRequest,
			"invalid quote metadata `%s` for LP `%s`",
			metadataQuote.String(),
			metadataLP.String(),
		)
	}

	return true, nil
}

// validation validates whether the LP belongs to a balancer pool that can be
// whitelisted and returns the quote metadata address when valid.
// The bool return is false when the LP does not belong to a balancer pool.
func (k BalancerKeeper) validation(ctx context.Context, metadataLP vmtypes.AccountAddress) (*vmtypes.AccountAddress, bool, error) {
	ok, err := k.HasPool(ctx, metadataLP)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	// assert base denom is exist in the dex pair

	denomBase, err := k.BaseDenom(ctx)
	if err != nil {
		return nil, false, err
	}

	metadataBase, err := types.MetadataAddressFromDenom(denomBase)
	if err != nil {
		return nil, false, err
	}

	metadata, err := k.poolMetadata(ctx, metadataLP)
	if err != nil {
		return nil, false, err
	}

	if !slices.Contains(metadata, metadataBase) {
		return nil, false, moderrors.Wrapf(
			types.ErrInvalidDexConfig,
			"to be whitelisted, a balancer pool should contain `%s` in its pair", denomBase,
		)
	}

	var metadataQuote vmtypes.AccountAddress
	switch metadataBase {
	case metadata[0]:
		metadataQuote = metadata[1]
	case metadata[1]:
		metadataQuote = metadata[0]
	default:
		return nil, false, moderrors.Wrapf(
			types.ErrInvalidDexConfig,
			"To be whitelisted, a dex should contain `%s` in its pair", denomBase,
		)
	}

	//
	// compute weights and assert base weight is bigger than quote weight
	//

	weights, err := k.poolWeights(ctx, metadataLP)
	if err != nil {
		return nil, false, err
	}

	if weights[0].LT(weights[1]) {
		return nil, false, moderrors.Wrapf(types.ErrInvalidDexConfig,
			"base weight `%s` must be bigger than quote weight `%s`", weights[0], weights[1])
	}

	return &metadataQuote, true, nil
}

// getPoolInfo returns balances and weights in base/quote order for the pool.
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

// poolMetadata returns the two metadata addresses stored in the pool.
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

// poolBalances returns pool balances in base/quote order.
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

// poolWeights returns pool weights in base/quote order at the current block time.
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

// isReverse reports whether the pool metadata order is quote/base.
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

	switch metadataBase {
	case metadata[0]:
		return false, nil
	case metadata[1]:
		return true, nil
	}

	return false, types.ErrInvalidDexConfig.Wrapf("the pair does not contain `%s`", denomBase)
}
