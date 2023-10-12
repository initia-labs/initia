package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"cosmossdk.io/math"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

var _ types.FungibleAssetKeeper = MoveBankKeeper{}

func (k MoveBankKeeper) Balance(ctx sdk.Context, store vmtypes.AccountAddress) (vmtypes.AccountAddress, math.Int, error) {
	bz, err := k.GetResourceBytes(ctx, store, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameFungibleAsset,
		Name:     types.ResourceNameFungibleStore,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err == sdkerrors.ErrNotFound {
		return vmtypes.AccountAddress{}, sdk.ZeroInt(), nil
	}
	if err != nil {
		return vmtypes.AccountAddress{}, sdk.ZeroInt(), err
	}

	return types.ReadBalanceFromFungibleStore(bz)
}

func (k MoveBankKeeper) Issuer(ctx sdk.Context, metadata vmtypes.AccountAddress) (vmtypes.AccountAddress, error) {
	bz, err := k.GetResourceBytes(ctx, vmtypes.StdAddress, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNamePrimaryFungibleStore,
		Name:     types.ResourceNameModuleStore,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil {
		return vmtypes.AccountAddress{}, err
	}

	tableHandle, err := types.ReadIssuersTableHandleFromModuleStore(bz)
	if err != nil {
		return vmtypes.AccountAddress{}, err
	}

	tableEntry, err := k.GetTableEntryBytes(ctx, tableHandle, metadata[:])
	if err != nil {
		return vmtypes.AccountAddress{}, err
	}

	return vmtypes.NewAccountAddressFromBytes(tableEntry.ValueBytes)
}

func (k MoveBankKeeper) Symbol(ctx sdk.Context, metadata vmtypes.AccountAddress) (string, error) {
	if bz, err := k.GetResourceBytes(ctx, metadata, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameFungibleAsset,
		Name:     types.ResourceNameMetadata,
		TypeArgs: []vmtypes.TypeTag{},
	}); err != nil {
		return "", err
	} else {
		return types.ReadSymbolFromMetadata(bz), nil
	}
}
