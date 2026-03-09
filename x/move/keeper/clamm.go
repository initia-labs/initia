package keeper

import (
	"context"
	"math/big"
	"slices"

	moderrors "cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

// CLAMMKeeper handles operations for CLAMM pools deployed at a
// user-specified module address (clamm_module_address).
type CLAMMKeeper struct {
	*Keeper
	clammModuleAddr vmtypes.AccountAddress
}

var (
	// tick_math::min_sqrt_ratio() + 1, tick_math::max_sqrt_ratio() - 1.
	clammMinSqrtPriceLimit = big.NewInt(4295048018)
	clammMaxSqrtPriceLimit = mustBigInt("79226673515401279992447579061")
)

// NewCLAMMKeeper creates a CLAMMKeeper for the given CLAMM module deployer address.
func NewCLAMMKeeper(k *Keeper, clammModuleAddr vmtypes.AccountAddress) CLAMMKeeper {
	return CLAMMKeeper{k, clammModuleAddr}
}

func mustBigInt(v string) *big.Int {
	out, ok := new(big.Int).SetString(v, 10)
	if !ok {
		panic("invalid big.Int literal: " + v)
	}
	return out
}

// HasPool reports whether a CLAMM Pool resource exists at metadataLP (the pool object address).
func (k CLAMMKeeper) HasPool(ctx context.Context, metadataLP vmtypes.AccountAddress) (bool, error) {
	return k.HasResource(ctx, metadataLP, vmtypes.StructTag{
		Address:  k.clammModuleAddr,
		Module:   types.MoveModuleNameCLAMMPool,
		Name:     types.ResourceNameCLAMMPool,
		TypeArgs: []vmtypes.TypeTag{},
	})
}

// GetPoolMetadata returns the two token metadata addresses stored in the CLAMM pool.
func (k CLAMMKeeper) GetPoolMetadata(ctx context.Context, metadataLP vmtypes.AccountAddress) (vmtypes.AccountAddress, vmtypes.AccountAddress, error) {
	bz, err := k.GetResourceBytes(ctx, metadataLP, vmtypes.StructTag{
		Address:  k.clammModuleAddr,
		Module:   types.MoveModuleNameCLAMMPool,
		Name:     types.ResourceNameCLAMMPool,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil {
		return vmtypes.AccountAddress{}, vmtypes.AccountAddress{}, err
	}

	metadata0, metadata1, _, err := types.ReadCLAMMPool(bz)
	return metadata0, metadata1, err
}

// GetBaseSpotPrice returns the base-asset spot price computed from the pool's current sqrt_price.
func (k CLAMMKeeper) GetBaseSpotPrice(ctx context.Context, metadataQuote, metadataLP vmtypes.AccountAddress) (math.LegacyDec, error) {
	bz, err := k.GetResourceBytes(ctx, metadataLP, vmtypes.StructTag{
		Address:  k.clammModuleAddr,
		Module:   types.MoveModuleNameCLAMMPool,
		Name:     types.ResourceNameCLAMMPool,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	metadata0, metadata1, sqrtPrice, err := types.ReadCLAMMPool(bz)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	denomBase, err := k.BaseDenom(ctx)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	metadataBase, err := types.MetadataAddressFromDenom(denomBase)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	switch metadataBase {
	case metadata0:
		return types.CLAMMBaseSpotPrice(sqrtPrice, true)
	case metadata1:
		return types.CLAMMBaseSpotPrice(sqrtPrice, false)
	default:
		return math.LegacyZeroDec(), moderrors.Wrapf(
			types.ErrInvalidDexConfig,
			"CLAMM pool `%s` does not contain base denom `%s`",
			metadataLP.String(), denomBase,
		)
	}
}

// SwapToBase swaps quote asset to base asset through clamm scripts::swap (exact-in mode).
func (k CLAMMKeeper) SwapToBase(
	ctx context.Context,
	trader vmtypes.AccountAddress,
	metadataLP, metadataQuote vmtypes.AccountAddress,
	quoteAmount math.Int,
) error {
	bz, err := k.GetResourceBytes(ctx, metadataLP, vmtypes.StructTag{
		Address:  k.clammModuleAddr,
		Module:   types.MoveModuleNameCLAMMPool,
		Name:     types.ResourceNameCLAMMPool,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil {
		return err
	}

	metadata0, metadata1, _, err := types.ReadCLAMMPool(bz)
	if err != nil {
		return err
	}

	denomBase, err := k.BaseDenom(ctx)
	if err != nil {
		return err
	}

	metadataBase, err := types.MetadataAddressFromDenom(denomBase)
	if err != nil {
		return err
	}

	if metadataQuote != metadata0 && metadataQuote != metadata1 {
		return moderrors.Wrapf(
			types.ErrInvalidRequest,
			"invalid quote metadata `%s` for CLAMM pool `%s`",
			metadataQuote.String(),
			metadataLP.String(),
		)
	}

	if metadataBase != metadata0 && metadataBase != metadata1 {
		return moderrors.Wrapf(
			types.ErrInvalidDexConfig,
			"CLAMM pool `%s` does not contain base denom `%s`",
			metadataLP.String(),
			denomBase,
		)
	}

	zeroForOne := metadataQuote == metadata0
	sqrtPriceLimitBI := new(big.Int)
	if zeroForOne {
		// use the loosest valid lower bound for zero-for-one direction
		sqrtPriceLimitBI.Set(clammMinSqrtPriceLimit)
	} else {
		// use the loosest valid upper bound for one-for-zero direction
		sqrtPriceLimitBI.Set(clammMaxSqrtPriceLimit)
	}

	if sqrtPriceLimitBI.Sign() <= 0 {
		return moderrors.Wrap(types.ErrInvalidRequest, "invalid CLAMM sqrt_price_limit")
	}

	amountInBz, err := vmtypes.SerializeUint64(quoteAmount.Uint64())
	if err != nil {
		return err
	}
	amountOutBz, err := vmtypes.SerializeUint64(0) // exact in with no min out
	if err != nil {
		return err
	}
	sqrtPriceLimitBz, err := serializeU128FromBigInt(sqrtPriceLimitBI)
	if err != nil {
		return err
	}
	exactInBz, err := vmtypes.SerializeBool(true)
	if err != nil {
		return err
	}
	zeroForOneBz, err := vmtypes.SerializeBool(zeroForOne)
	if err != nil {
		return err
	}
	integratorBz, err := vmtypes.SerializeString("")
	if err != nil {
		return err
	}

	return k.ExecuteEntryFunction(
		ctx,
		trader,
		k.clammModuleAddr,
		types.MoveModuleNameCLAMMScripts,
		types.FunctionNameCLAMMSwap,
		[]vmtypes.TypeTag{},
		[][]byte{
			metadataLP[:],
			amountInBz,
			amountOutBz,
			sqrtPriceLimitBz,
			exactInBz,
			zeroForOneBz,
			integratorBz,
		},
	)
}

func serializeU128FromBigInt(v *big.Int) ([]byte, error) {
	if v.Sign() < 0 || v.BitLen() > 128 {
		return nil, moderrors.Wrapf(types.ErrInvalidRequest, "invalid u128 value: %s", v.String())
	}

	mask := new(big.Int).SetUint64(^uint64(0))
	low := new(big.Int).And(new(big.Int).Set(v), mask).Uint64()
	high := new(big.Int).Rsh(new(big.Int).Set(v), 64).Uint64()

	return vmtypes.SerializeUint128(high, low)
}

// WhitelistGasPrice validates that the CLAMM pool at metadataLP:
//   - exists
//   - contains the base token
//   - has metadataQuote as the non-base asset
//
// It returns (true, nil) on success, (false, nil) if the pool doesn't exist,
// and (false, err) on a validation error.
// Store operations are intentionally omitted; they are handled by whitelist.go.
func (k CLAMMKeeper) WhitelistGasPrice(ctx context.Context, metadataQuote, metadataLP vmtypes.AccountAddress) (bool, error) {
	ok, err := k.HasPool(ctx, metadataLP)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	metadata0, metadata1, err := k.GetPoolMetadata(ctx, metadataLP)
	if err != nil {
		return false, err
	}

	denomBase, err := k.BaseDenom(ctx)
	if err != nil {
		return false, err
	}

	metadataBase, err := types.MetadataAddressFromDenom(denomBase)
	if err != nil {
		return false, err
	}

	switch metadataBase {
	case metadata0:
		if metadataQuote != metadata1 {
			return false, moderrors.Wrapf(
				types.ErrInvalidRequest,
				"invalid quote metadata `%s` for CLAMM pool `%s`: expected `%s`",
				metadataQuote.String(), metadataLP.String(), metadata1.String(),
			)
		}
	case metadata1:
		if metadataQuote != metadata0 {
			return false, moderrors.Wrapf(
				types.ErrInvalidRequest,
				"invalid quote metadata `%s` for CLAMM pool `%s`: expected `%s`",
				metadataQuote.String(), metadataLP.String(), metadata0.String(),
			)
		}
	default:
		return false, moderrors.Wrapf(
			types.ErrInvalidDexConfig,
			"to be whitelisted, a CLAMM pool should contain `%s` in its pair", denomBase,
		)
	}

	return true, nil
}

// DelistGasPrice validates that the CLAMM pool at metadataLP exists and contains metadataQuote.
// Returns (true, nil) on success, (false, nil) if the pool doesn't exist.
// Store operations are intentionally omitted; they are handled by whitelist.go.
func (k CLAMMKeeper) DelistGasPrice(ctx context.Context, metadataQuote, metadataLP vmtypes.AccountAddress) (bool, error) {
	ok, err := k.HasPool(ctx, metadataLP)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	metadata0, metadata1, err := k.GetPoolMetadata(ctx, metadataLP)
	if err != nil {
		return false, err
	}

	if !slices.Contains([]vmtypes.AccountAddress{metadata0, metadata1}, metadataQuote) {
		return false, moderrors.Wrapf(
			types.ErrInvalidRequest,
			"invalid quote metadata `%s` for CLAMM pool `%s`",
			metadataQuote.String(), metadataLP.String(),
		)
	}

	return true, nil
}
