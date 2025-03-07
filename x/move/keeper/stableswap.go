package keeper

import (
	"context"
	"slices"

	moderrors "cosmossdk.io/errors"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

type StableSwapKeeper struct {
	*Keeper
}

func NewStableSwapKeeper(k *Keeper) StableSwapKeeper {
	return StableSwapKeeper{k}
}

func (k StableSwapKeeper) HasPool(ctx context.Context, metadataLP vmtypes.AccountAddress) (bool, error) {
	return k.HasResource(ctx, metadataLP, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameStableSwap,
		Name:     types.ResourceNamePool,
		TypeArgs: []vmtypes.TypeTag{},
	})
}

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

// Whitelist checks if the stableswap pool is valid to be whitelisted
func (k StableSwapKeeper) Whitelist(ctx context.Context, metadataLP vmtypes.AccountAddress) (bool, error) {
	ok, err := k.HasPool(ctx, metadataLP)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	// assert base denom is exist in the dex pair

	denomBase, err := k.BaseDenom(ctx)
	if err != nil {
		return false, err
	}

	metadataBase, err := types.MetadataAddressFromDenom(denomBase)
	if err != nil {
		return false, err
	}

	metadata, err := k.GetPoolMetadata(ctx, metadataLP)
	if err != nil {
		return false, err
	}

	if !slices.Contains(metadata, metadataBase) {
		return false, moderrors.Wrapf(
			types.ErrInvalidDexConfig,
			"To be whitelisted, a stableswap should contain `%s` in its pair", denomBase,
		)
	}

	return true, nil
}

// Delist removes the stableswap pool from the whitelist
func (k StableSwapKeeper) Delist(ctx context.Context, metadataLP vmtypes.AccountAddress) error {
	// no-op for now
	return nil
}
