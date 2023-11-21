package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	banktypes "github.com/initia-labs/initia/x/bank/types"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

var _ banktypes.MoveBankKeeper = MoveBankKeeper{}

// NftKeeper implements move wrapper for types.MoveBankKeeper interface
type MoveBankKeeper struct {
	*Keeper
}

// NewMoveBankKeeper return new MoveBankKeeper instance
func NewMoveBankKeeper(k *Keeper) MoveBankKeeper {
	return MoveBankKeeper{k}
}

// GetBalance return move coin balance
func (k MoveBankKeeper) GetBalance(
	ctx sdk.Context,
	addr sdk.AccAddress,
	denom string,
) (math.Int, error) {
	userAddr, err := vmtypes.NewAccountAddressFromBytes(addr[:])
	if err != nil {
		return sdk.ZeroInt(), err
	}

	metadata, err := types.MetadataAddressFromDenom(denom)
	if err != nil {
		return sdk.ZeroInt(), err
	}

	storeAddr := types.UserDerivedObjectAddress(userAddr, metadata)
	_, balance, err := k.Balance(ctx, storeAddr)
	return balance, err
}

// GetUserStores return a prefix store of table,
// which holds all primary stores of a user.
func (k MoveBankKeeper) GetUserStores(
	ctx sdk.Context,
	addr sdk.AccAddress,
) (*prefix.Store, error) {
	bz, err := k.GetResourceBytes(ctx, vmtypes.StdAddress, vmtypes.StructTag{
		Address: vmtypes.StdAddress,
		Module:  types.MoveModuleNamePrimaryFungibleStore,
		Name:    types.ResourceNameModuleStore,
	})
	if err != nil {
		return nil, err
	}

	tableAddr, err := types.ReadUserStoresTableHandleFromModuleStore(bz)
	if err != nil {
		return nil, err
	}

	userAddr, err := vmtypes.NewAccountAddressFromBytes(addr[:])
	if err != nil {
		return nil, err
	}

	// check user has a store entry
	if !k.HasTableEntry(ctx, tableAddr, userAddr[:]) {
		return nil, nil
	}

	tableEntry, err := k.GetTableEntryBytes(ctx, tableAddr, userAddr[:])
	if err != nil {
		return nil, err
	}

	tableAddr, err = types.ReadTableHandleFromTable(tableEntry.ValueBytes)
	if err != nil {
		return nil, err
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.PrefixKeyVMStore, types.GetTableEntryPrefix(tableAddr)...))
	return &store, nil
}

// GetSupply return move coin supply
func (k MoveBankKeeper) GetSupply(
	ctx sdk.Context,
	denom string,
) (math.Int, error) {
	metadata, err := types.MetadataAddressFromDenom(denom)
	if err != nil {
		return sdk.ZeroInt(), err
	}

	return k.GetSupplyWithMetadata(ctx, metadata)
}

func (k MoveBankKeeper) GetSupplyWithMetadata(ctx sdk.Context, metadata vmtypes.AccountAddress) (math.Int, error) {
	bz, err := k.GetResourceBytes(ctx, metadata, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameFungibleAsset,
		Name:     types.ResourceNameSupply,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err == sdkerrors.ErrNotFound {
		return sdk.ZeroInt(), nil
	}
	if err != nil {
		return sdk.ZeroInt(), err
	}

	num, err := types.ReadSupplyFromSupply(bz)
	if err != nil {
		return sdk.ZeroInt(), err
	}

	return num, nil
}

// GetIssuers return 0x1 primary_fungible_store's Issuers table prefix store.
// The caller can consume key of the iterator, which is the metadata address.
func (k MoveBankKeeper) GetIssuers(ctx sdk.Context) (prefix.Store, error) {
	bz, err := k.GetResourceBytes(ctx, vmtypes.StdAddress, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNamePrimaryFungibleStore,
		Name:     types.ResourceNameModuleStore,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil {
		return prefix.Store{}, err
	}

	tableHandle, err := types.ReadIssuersTableHandleFromModuleStore(bz)
	if err != nil {
		return prefix.Store{}, err
	}

	return prefix.NewStore(ctx.KVStore(k.storeKey), append(types.PrefixKeyVMStore, types.GetTableEntryPrefix(tableHandle)...)), nil
}

// SendCoins transfer coins to recipient
func (k MoveBankKeeper) SendCoins(
	ctx sdk.Context,
	fromAddr sdk.AccAddress,
	toAddr sdk.AccAddress,
	coins sdk.Coins,
) error {
	for _, coin := range coins {
		if err := k.SendCoin(ctx, fromAddr, toAddr, coin.Denom, coin.Amount); err != nil {
			return err
		}
	}

	return nil
}

// BurnCoins burn coins or send to community pool.
func (k MoveBankKeeper) BurnCoins(
	ctx sdk.Context,
	accAddr sdk.AccAddress,
	coins sdk.Coins,
) error {
	for _, coin := range coins {
		// if a coin is not generated from 0x1, then send the coin to community pool
		// because we don't have burn capability.
		if types.IsMoveCoin(coin) {
			if err := k.communityPoolKeeper.FundCommunityPool(ctx, coins, accAddr); err != nil {
				return err
			}

			continue
		}

		// send tokens to 0x1
		err := k.SendCoin(ctx, accAddr, types.StdAddr, coin.Denom, coin.Amount)
		if err != nil {
			return err
		}

		// execute burn
		metadata, err := types.MetadataAddressFromDenom(coin.Denom)
		if err != nil {
			return err
		}

		amountBz, err := vmtypes.SerializeUint64(coin.Amount.Uint64())
		if err != nil {
			return err
		}

		err = k.ExecuteEntryFunction(
			ctx,
			vmtypes.StdAddress,
			vmtypes.StdAddress,
			types.MoveModuleNameManagedCoin,
			types.FunctionNameManagedCoinBurn,
			[]vmtypes.TypeTag{},
			[][]byte{metadata[:], amountBz},
		)

		if err != nil {
			return err
		}
	}

	return nil
}

// MintCoins mint coins to the address
func (k MoveBankKeeper) MintCoins(
	ctx sdk.Context,
	accAddr sdk.AccAddress,
	coins sdk.Coins,
) error {
	for _, coin := range coins {
		if types.IsMoveCoin(coin) {
			return errors.Wrapf(types.ErrInvalidRequest, "cannot mint move coin: %s", coin.Denom)
		}

		metadata, err := types.MetadataAddressFromDenom(coin.Denom)
		if err != nil {
			return err
		}

		if ok, err := k.HasResource(ctx, metadata, vmtypes.StructTag{
			Address: vmtypes.StdAddress,
			Module:  types.MoveModuleNameFungibleAsset,
			Name:    types.ResourceNameMetadata,
		}); err != nil {
			return err
		} else if !ok {
			if err := k.InitializeCoin(ctx, coin.Denom); err != nil {
				return err
			}
		}

		amountBz, err := vmtypes.SerializeUint64(coin.Amount.Uint64())
		if err != nil {
			return err
		}

		recipientAddr, err := vmtypes.NewAccountAddressFromBytes(accAddr)
		if err != nil {
			return err
		}

		err = k.ExecuteEntryFunction(
			ctx,
			vmtypes.StdAddress,
			vmtypes.StdAddress,
			types.MoveModuleNameManagedCoin,
			types.FunctionNameManagedCoinMint,
			[]vmtypes.TypeTag{},
			[][]byte{recipientAddr[:], metadata[:], amountBz},
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (k MoveBankKeeper) InitializeCoin(
	ctx sdk.Context,
	denom string,
) error {
	if types.IsMoveDenom(denom) {
		return errors.Wrapf(types.ErrInvalidRequest, "cannot initialize move coin: %s", denom)
	}

	nameBz, err := vmtypes.SerializeString(fmt.Sprintf("%s Coin", denom))
	if err != nil {
		return err
	}

	symbolBz, err := vmtypes.SerializeString(denom)
	if err != nil {
		return err
	}

	if err := k.ExecuteEntryFunction(
		ctx,
		vmtypes.StdAddress,
		vmtypes.StdAddress,
		types.MoveModuleNameManagedCoin,
		types.FunctionNameManagedCoinInitialize,
		[]vmtypes.TypeTag{},
		[][]byte{{0}, nameBz, symbolBz, {0}, {0}, {0}},
	); err != nil {
		return err
	}

	return nil
}

func (k MoveBankKeeper) SendCoin(
	ctx sdk.Context,
	fromAddr sdk.AccAddress,
	toAddr sdk.AccAddress,
	denom string,
	amount math.Int,
) error {
	metadata, err := types.MetadataAddressFromDenom(denom)
	if err != nil {
		return err
	}

	fromVmAddr, err := vmtypes.NewAccountAddressFromBytes(fromAddr)
	if err != nil {
		return err
	}

	toVmAddr, err := vmtypes.NewAccountAddressFromBytes(toAddr)
	if err != nil {
		return err
	}

	amountBz, err := vmtypes.SerializeUint64(amount.Uint64())
	if err != nil {
		return err
	}

	return k.ExecuteEntryFunction(
		ctx,
		fromVmAddr,
		vmtypes.StdAddress,
		types.MoveModuleNameCoin,
		types.FunctionNameCoinTransfer,
		[]vmtypes.TypeTag{},
		[][]byte{toVmAddr[:], metadata[:], amountBz},
	)
}
