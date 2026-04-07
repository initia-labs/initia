package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	moderrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/initia-labs/initia/x/move/types"
)

type StableSwapKeeper struct {
	*Keeper
}

// NewStableSwapKeeper returns a stableswap-specific keeper wrapper.
func NewStableSwapKeeper(k *Keeper) StableSwapKeeper {
	return StableSwapKeeper{k}
}

// HasPool reports whether a stableswap pool exists for the given LP metadata.
func (k StableSwapKeeper) HasPool(ctx context.Context, metadataLP vmtypes.AccountAddress) (bool, error) {
	return k.HasResource(ctx, metadataLP, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameStableSwap,
		Name:     types.ResourceNamePool,
		TypeArgs: []vmtypes.TypeTag{},
	})
}

// GetBaseSpotPrice returns the base asset spot price from the stableswap pool.
func (k StableSwapKeeper) GetBaseSpotPrice(
	ctx context.Context,
	metadataQuote vmtypes.AccountAddress,
	metadataLP vmtypes.AccountAddress,
) (math.LegacyDec, error) {
	denomBase, err := k.BaseDenom(ctx)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	metadataBase, err := types.MetadataAddressFromDenom(denomBase)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	output, _, err := k.ExecuteViewFunctionJSON(
		ctx,
		vmtypes.StdAddress,
		types.MoveModuleNameStableSwap,
		types.FunctionNameStableSwapSpotPrice,
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", metadataLP),
			fmt.Sprintf("\"%s\"", metadataBase),
			fmt.Sprintf("\"%s\"", metadataQuote),
		},
	)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	var spotPriceStr string
	if err := json.Unmarshal([]byte(output.Ret), &spotPriceStr); err != nil {
		return math.LegacyZeroDec(), err
	}

	spotPrice, err := math.LegacyNewDecFromStr(spotPriceStr)
	if err != nil {
		return math.LegacyZeroDec(), moderrors.Wrapf(types.ErrInvalidResponse, "invalid spot price: %s", spotPriceStr)
	}

	return spotPrice, nil
}

// GetPoolMetadata returns metadata addresses of assets in the stableswap pool.
func (k StableSwapKeeper) GetPoolMetadata(ctx context.Context, metadataLP vmtypes.AccountAddress) ([]vmtypes.AccountAddress, error) {
	bz, err := k.GetResourceBytes(ctx, metadataLP, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameStableSwap,
		Name:     types.ResourceNamePool,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil {
		return nil, err
	}

	metadata, err := types.ReadStableSwapPool(bz)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// SwapToBase swaps quote asset to base asset through stableswap::swap_script.
func (k StableSwapKeeper) SwapToBase(
	ctx context.Context,
	trader vmtypes.AccountAddress,
	metadataLP, metadataQuote vmtypes.AccountAddress,
	quoteAmount math.Int,
) error {
	denomBase, err := k.BaseDenom(ctx)
	if err != nil {
		return err
	}

	metadataBase, err := types.MetadataAddressFromDenom(denomBase)
	if err != nil {
		return err
	}

	offerAmountBz, err := vmtypes.SerializeUint64(quoteAmount.Uint64())
	if err != nil {
		return err
	}

	// min_return_amount = None
	return k.ExecuteEntryFunction(
		ctx,
		trader,
		vmtypes.StdAddress,
		types.MoveModuleNameStableSwap,
		types.FunctionNameStableSwapSwap,
		[]vmtypes.TypeTag{},
		[][]byte{metadataLP[:], metadataQuote[:], metadataBase[:], offerAmountBz, {0}},
	)
}

// WhitelistStaking validates a stableswap LP for staking whitelist registration.
func (k StableSwapKeeper) WhitelistStaking(ctx context.Context, metadataLP vmtypes.AccountAddress) (bool, error) {
	_, ok, err := k.validation(ctx, metadataLP)
	return ok, err
}

// WhitelistGasPrice validates that the stableswap LP contains both the base token
// and the provided quote token.
// Returns (true, nil) on success, (false, nil) if the LP is not a stableswap pool,
// and (false, err) on a validation error.
// Store operations are intentionally omitted; they are handled by whitelist.go.
func (k StableSwapKeeper) WhitelistGasPrice(ctx context.Context, metadataQuote, metadataLP vmtypes.AccountAddress) (bool, error) {
	metadata, ok, err := k.validation(ctx, metadataLP)
	if !ok || err != nil {
		return ok, err
	}

	denomBase, err := k.BaseDenom(ctx)
	if err != nil {
		return false, err
	}

	metadataBase, err := types.MetadataAddressFromDenom(denomBase)
	if err != nil {
		return false, err
	}

	// Ensure the provided quote metadata belongs to the LP pair.
	if !slices.Contains(metadata, metadataQuote) {
		return false, moderrors.Wrapf(
			types.ErrInvalidRequest,
			"invalid quote metadata `%s` for LP `%s`",
			metadataQuote.String(),
			metadataLP.String(),
		)
	}

	if metadataQuote == metadataBase {
		return false, moderrors.Wrapf(
			types.ErrInvalidRequest,
			"quote metadata `%s` cannot be base denom `%s`",
			metadataQuote.String(),
			denomBase,
		)
	}

	return true, nil
}

// DelistStaking is a no-op for stableswap staking delist.
func (k StableSwapKeeper) DelistStaking(ctx context.Context, metadataLP vmtypes.AccountAddress) error {
	// no-op for now
	return nil
}

// DelistGasPrice validates that the stableswap LP contains metadataQuote.
// Returns (true, nil) on success, (false, nil) if the LP is not a stableswap pool,
// and (false, err) on a validation error.
// Store operations are intentionally omitted; they are handled by whitelist.go.
func (k StableSwapKeeper) DelistGasPrice(ctx context.Context, metadataQuote, metadataLP vmtypes.AccountAddress) (bool, error) {
	ok, err := k.HasPool(ctx, metadataLP)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	metadata, err := k.GetPoolMetadata(ctx, metadataLP)
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

// validation checks if metadataLP is a stableswap pool and contains the base denom.
func (k StableSwapKeeper) validation(ctx context.Context, metadataLP vmtypes.AccountAddress) ([]vmtypes.AccountAddress, bool, error) {
	ok, err := k.HasPool(ctx, metadataLP)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}

	// Ensure the base denom exists in the pool pair.

	denomBase, err := k.BaseDenom(ctx)
	if err != nil {
		return nil, false, err
	}

	metadataBase, err := types.MetadataAddressFromDenom(denomBase)
	if err != nil {
		return nil, false, err
	}

	metadata, err := k.GetPoolMetadata(ctx, metadataLP)
	if err != nil {
		return nil, false, err
	}

	if !slices.Contains(metadata, metadataBase) {
		return nil, false, moderrors.Wrapf(
			types.ErrInvalidDexConfig,
			"To be whitelisted, a stableswap should contain `%s` in its pair", denomBase,
		)
	}

	return metadata, true, err
}
