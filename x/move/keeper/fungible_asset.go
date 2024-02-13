package keeper

import (
	"context"
	"errors"
	"strings"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	cosmosbanktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

var _ types.FungibleAssetKeeper = MoveBankKeeper{}

func (k MoveBankKeeper) Balance(ctx context.Context, store vmtypes.AccountAddress) (vmtypes.AccountAddress, math.Int, error) {
	bz, err := k.GetResourceBytes(ctx, store, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameFungibleAsset,
		Name:     types.ResourceNameFungibleStore,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return vmtypes.AccountAddress{}, math.ZeroInt(), nil
	} else if err != nil {
		return vmtypes.AccountAddress{}, math.ZeroInt(), err
	}

	return types.ReadBalanceFromFungibleStore(bz)
}

func (k MoveBankKeeper) Issuer(ctx context.Context, metadata vmtypes.AccountAddress) (vmtypes.AccountAddress, error) {
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

func (k MoveBankKeeper) Symbol(ctx context.Context, metadata vmtypes.AccountAddress) (string, error) {
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

// GetMetadata interprets move fungible asset metadata
// to cosmos metadata.
func (k MoveBankKeeper) GetMetadata(
	ctx context.Context,
	denom string,
) (cosmosbanktypes.Metadata, error) {
	metadata, err := types.MetadataAddressFromDenom(denom)
	if err != nil {
		return cosmosbanktypes.Metadata{}, err
	}

	bz, err := k.GetResourceBytes(ctx, metadata, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameFungibleAsset,
		Name:     types.ResourceNameMetadata,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil {
		return cosmosbanktypes.Metadata{}, err
	}

	name, symbol, decimals := types.ReadFungibleAssetMetadata(bz)
	denomUnits := []*cosmosbanktypes.DenomUnit{
		{
			Denom:    denom,
			Exponent: 0,
		},
	}

	base := denom
	display := denom
	if decimals == 0 {
		if !strings.Contains(denom, "/") && denom[0] == 'u' {
			display = denom[1:]
			denomUnits = append(denomUnits, &cosmosbanktypes.DenomUnit{
				Denom:    display,
				Exponent: 6,
			})
		} else if !strings.Contains(denom, "/") && denom[0] == 'm' {
			display = denom[1:]
			denomUnits = append(denomUnits, &cosmosbanktypes.DenomUnit{
				Denom:    display,
				Exponent: 3,
			})
		}
	}

	return cosmosbanktypes.Metadata{
		Name:       name,
		Symbol:     symbol,
		Base:       base,
		Display:    display,
		DenomUnits: denomUnits,
	}, nil
}
