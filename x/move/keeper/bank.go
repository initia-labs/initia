package keeper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	moderrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	banktypes "github.com/initia-labs/initia/x/bank/types"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
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

// GetBalance retrieves the balance of a specific denomination for an account from the primary fungible store.
// It supports both standard fungible assets and dispatchable fungible assets.
// Returns the balance amount as sdkmath.Int and any error encountered.
func (k MoveBankKeeper) GetBalance(
	ctx context.Context,
	addr sdk.AccAddress,
	denom string,
) (sdkmath.Int, error) {
	userAddr, err := vmtypes.NewAccountAddressFromBytes(addr[:])
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

	metadata, err := types.MetadataAddressFromDenom(denom)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

	return k.GetBalanceWithMetadata(ctx, userAddr, metadata)
}

// GetBalanceWithMetadata retrieves the balance of a specific denomination for an account from the primary fungible store.
// It supports both standard fungible assets and dispatchable fungible assets.
// Returns the balance amount as sdkmath.Int and any error encountered.
func (k MoveBankKeeper) GetBalanceWithMetadata(
	ctx context.Context,
	userAddr vmtypes.AccountAddress,
	metadata vmtypes.AccountAddress,
) (sdkmath.Int, error) {

	// if it is not a dispatchable fungible asset, return
	hasDispatchFunctionStore, err := k.HasDispatchFunctionStore(ctx, metadata)
	if err != nil {
		return sdkmath.ZeroInt(), err
	} else if !hasDispatchFunctionStore {
		storeAddr := types.UserDerivedObjectAddress(userAddr, metadata)
		_, balance, err := k.Balance(ctx, storeAddr)
		return balance, err
	}

	// use limited gas for dispatchable fungible assets balance query
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	gasMeter := sdkCtx.GasMeter()

	const getBalanceMaxGas = storetypes.Gas(100000)
	sdkCtx = sdkCtx.WithGasMeter(storetypes.NewGasMeter(min(gasMeter.GasRemaining(), getBalanceMaxGas)))
	defer func() {
		// ignore panic
		_ = recover()

		usedGas := sdkCtx.GasMeter().GasConsumedToLimit()
		gasMeter.ConsumeGas(usedGas, "GetBalance")
	}()

	// query balance from primary_fungible_store
	output, _, err := k.ExecuteViewFunctionJSON(
		sdkCtx,
		vmtypes.StdAddress,
		types.MoveModuleNamePrimaryFungibleStore,
		types.FunctionNamePrimaryFungibleStoreBalance,
		[]vmtypes.TypeTag{
			&vmtypes.TypeTag__Struct{
				Value: vmtypes.StructTag{
					Address:  vmtypes.StdAddress,
					Module:   types.MoveModuleNameFungibleAsset,
					Name:     types.ResourceNameMetadata,
					TypeArgs: []vmtypes.TypeTag{},
				},
			}},
		[]string{fmt.Sprintf("\"%s\"", userAddr), fmt.Sprintf("\"%s\"", metadata)},
	)
	if err != nil {
		// ignore fetching error due to dispatchable fungible assets
		return sdkmath.ZeroInt(), nil
	}

	var amountStr string
	if err := json.Unmarshal([]byte(output.Ret), &amountStr); err != nil {
		return sdkmath.ZeroInt(), err
	}

	amount, ok := sdkmath.NewIntFromString(amountStr)
	if !ok {
		return sdkmath.ZeroInt(), moderrors.Wrapf(types.ErrInvalidResponse, "invalid balance amount: %s", amountStr)
	}

	return amount, nil
}

// IterateAccountBalances iterates over the balances of a single account and
// provides the token balance to a callback. If true is returned from the
// callback, iteration is halted.
func (k MoveBankKeeper) IterateAccountBalances(
	ctx context.Context,
	addr sdk.AccAddress,
	cb func(sdk.Coin) (bool, error),
) error {
	// check user has stores table
	tableAddr, tableLength, err := k.GetUserStoresTableHandleWithLength(ctx, addr)
	if err != nil {
		return err
	} else if tableAddr == nil || tableLength.IsZero() {
		return nil
	}

	userAddr, err := vmtypes.NewAccountAddressFromBytes(addr[:])
	if err != nil {
		return err
	}

	prefix := types.GetTableEntryPrefix(*tableAddr)
	return k.VMStore.Walk(ctx, new(collections.Range[[]byte]).Prefix(collections.NewPrefix(prefix)), func(_, value []byte) (stop bool, err error) {
		storeAddr, err := vmtypes.NewAccountAddressFromBytes(value)
		if err != nil {
			return true, err
		}

		// load metadata from fungible store
		metadata, _, err := k.Balance(ctx, storeAddr)
		if err != nil {
			return true, err
		}

		// load balance from primary fungible store
		amount, err := k.GetBalanceWithMetadata(ctx, userAddr, metadata)
		if err != nil {
			return true, err
		}
		if !amount.IsPositive() {
			return false, nil
		}

		// load denom from metadata
		denom, err := types.DenomFromMetadataAddress(
			ctx, k, metadata,
		)
		if err != nil {
			return true, err
		}

		return cb(sdk.NewCoin(denom, amount))
	})
}

func (k MoveBankKeeper) GetPaginatedBalances(ctx context.Context, pageReq *query.PageRequest, addr sdk.AccAddress) (sdk.Coins, *query.PageResponse, error) {
	// check user has stores table
	tableAddr, tableLength, err := k.GetUserStoresTableHandleWithLength(ctx, addr)
	if err != nil {
		return nil, nil, err
	} else if tableAddr == nil || tableLength.IsZero() {
		return sdk.NewCoins(), &query.PageResponse{}, nil
	}

	var coin sdk.Coin
	coins, pageRes, err := query.CollectionFilteredPaginate(ctx, k.VMStore, pageReq, func(_, value []byte) (bool, error) {
		storeAddr, err := vmtypes.NewAccountAddressFromBytes(value)
		if err != nil {
			return false, err
		}

		// load metadata from fungible store
		metadata, _, err := k.Balance(ctx, storeAddr)
		if err != nil {
			return true, err
		}

		// load denom from metadata
		denom, err := types.DenomFromMetadataAddress(
			ctx, k, metadata,
		)
		if err != nil {
			return true, err
		}

		// load balance from primary fungible store
		amount, err := k.GetBalance(ctx, addr, denom)
		if err != nil {
			return true, err
		}
		if !amount.IsPositive() {
			return false, nil
		}

		coin = sdk.NewCoin(denom, amount)
		return true, nil
	}, func(_, value []byte) (sdk.Coin, error) {
		return coin, nil
	}, func(o *query.CollectionsPaginateOptions[[]byte]) {
		prefix := types.GetTableEntryPrefix(*tableAddr)
		o.Prefix = &prefix
	})
	if err != nil {
		return nil, nil, err
	}

	return sdk.Coins(coins).Sort(), pageRes, nil
}

// GetSupply return move coin supply
func (k MoveBankKeeper) GetSupply(
	ctx context.Context,
	denom string,
) (sdkmath.Int, error) {
	metadata, err := types.MetadataAddressFromDenom(denom)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

	return k.GetSupplyWithMetadata(ctx, metadata)
}

func (k MoveBankKeeper) HasSupply(
	ctx context.Context,
	denom string,
) (bool, error) {
	metadata, err := types.MetadataAddressFromDenom(denom)
	if err != nil {
		return false, err
	}

	return k.HasResource(ctx, metadata, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNameFungibleAsset,
		Name:     types.ResourceNameSupply,
		TypeArgs: []vmtypes.TypeTag{},
	})
}

func (k MoveBankKeeper) GetSupplyWithMetadata(ctx context.Context, metadata vmtypes.AccountAddress) (sdkmath.Int, error) {
	// if it is not a dispatchable fungible asset, return supply from supply store
	hasDispatchSupplyStore, err := k.HasDispatchSupplyStore(ctx, metadata)
	if err != nil {
		return sdkmath.ZeroInt(), err
	} else if !hasDispatchSupplyStore {
		bz, err := k.GetResourceBytes(ctx, metadata, vmtypes.StructTag{
			Address:  vmtypes.StdAddress,
			Module:   types.MoveModuleNameFungibleAsset,
			Name:     types.ResourceNameSupply,
			TypeArgs: []vmtypes.TypeTag{},
		})
		if err != nil && errors.Is(err, collections.ErrNotFound) {
			return sdkmath.ZeroInt(), nil
		} else if err != nil {
			return sdkmath.ZeroInt(), err
		}

		num, err := types.ReadSupplyFromSupply(bz)
		if err != nil {
			return sdkmath.ZeroInt(), err
		}

		return num, nil
	}

	// use limited gas for dispatchable fungible assets supply query
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	gasMeter := sdkCtx.GasMeter()

	const getSupplyMaxGas = storetypes.Gas(100000)
	sdkCtx = sdkCtx.WithGasMeter(storetypes.NewGasMeter(min(gasMeter.GasRemaining(), getSupplyMaxGas)))
	defer func() {
		// ignore panic
		_ = recover()

		usedGas := sdkCtx.GasMeter().GasConsumedToLimit()
		gasMeter.ConsumeGas(usedGas, "GetSupply")
	}()

	// query balance from primary_fungible_store
	output, _, err := k.ExecuteViewFunctionJSON(
		sdkCtx,
		vmtypes.StdAddress,
		types.MoveModuleNameDispatchableFungibleAsset,
		types.FunctionNameDispatchableFungibleAssetDerivedSupply,
		[]vmtypes.TypeTag{
			&vmtypes.TypeTag__Struct{
				Value: vmtypes.StructTag{
					Address:  vmtypes.StdAddress,
					Module:   types.MoveModuleNameFungibleAsset,
					Name:     types.ResourceNameMetadata,
					TypeArgs: []vmtypes.TypeTag{},
				},
			}},
		[]string{fmt.Sprintf("\"%s\"", metadata)},
	)
	if err != nil {
		// ignore fetching error due to dispatchable fungible assets
		return sdkmath.ZeroInt(), nil
	}

	var optionSupplyStr string
	if err := json.Unmarshal([]byte(output.Ret), &optionSupplyStr); err != nil {
		return sdkmath.ZeroInt(), err
	} else if optionSupplyStr == "null" {
		// return is option, so return zero if it is null
		return sdkmath.ZeroInt(), nil
	}

	supply, ok := sdkmath.NewIntFromString(optionSupplyStr)
	if !ok {
		return sdkmath.ZeroInt(), moderrors.Wrapf(types.ErrInvalidResponse, "invalid supply: %s", optionSupplyStr)
	}

	return supply, nil
}

func (k MoveBankKeeper) GetIssuersTableHandle(ctx context.Context) (*vmtypes.AccountAddress, error) {
	bz, err := k.GetResourceBytes(ctx, vmtypes.StdAddress, vmtypes.StructTag{
		Address:  vmtypes.StdAddress,
		Module:   types.MoveModuleNamePrimaryFungibleStore,
		Name:     types.ResourceNameModuleStore,
		TypeArgs: []vmtypes.TypeTag{},
	})
	if err != nil {
		return nil, err
	}

	tableHandle, err := types.ReadIssuersTableHandleFromModuleStore(bz)
	if err != nil {
		return nil, err
	}

	return &tableHandle, nil
}

func (k MoveBankKeeper) IterateSupply(ctx context.Context, cb func(supply sdk.Coin) (bool, error)) error {
	tableAddr, err := k.GetIssuersTableHandle(ctx)
	if err != nil {
		return err
	}

	prefixBytes := types.GetTableEntryPrefix(*tableAddr)
	return k.VMStore.Walk(ctx, new(collections.Range[[]byte]).Prefix(collections.NewPrefix[[]byte](prefixBytes)), func(key, value []byte) (stop bool, err error) {
		key = key[len(prefixBytes):]

		metadata, err := vmtypes.NewAccountAddressFromBytes(key)
		if err != nil {
			return true, err
		}

		denom, err := types.DenomFromMetadataAddress(ctx, k, metadata)
		if err != nil {
			return true, err
		}

		amount, err := k.GetSupply(ctx, denom)
		if err != nil {
			return true, err
		}
		if amount.IsZero() {
			return false, nil
		}

		return cb(sdk.Coin{
			Denom:  denom,
			Amount: amount,
		})
	})
}

func (k MoveBankKeeper) GetPaginatedSupply(ctx context.Context, pageReq *query.PageRequest) (sdk.Coins, *query.PageResponse, error) {
	tableAddr, err := k.GetIssuersTableHandle(ctx)
	if err != nil {
		return nil, nil, err
	}

	prefixBytes := types.GetTableEntryPrefix(*tableAddr)
	var coin sdk.Coin
	coins, pageRes, err := query.CollectionFilteredPaginate(ctx, k.VMStore, pageReq, func(key, value []byte) (bool, error) {
		key = key[len(prefixBytes):]

		metadata, err := vmtypes.NewAccountAddressFromBytes(key)
		if err != nil {
			return false, err
		}

		denom, err := types.DenomFromMetadataAddress(ctx, k, metadata)
		if err != nil {
			return false, err
		}

		amount, err := k.GetSupply(ctx, denom)
		if err != nil {
			return false, err
		}
		if !amount.IsPositive() {
			return false, nil
		}

		coin = sdk.NewCoin(denom, amount)
		return true, nil
	}, func(key, value []byte) (sdk.Coin, error) {
		return coin, nil
	}, func(opt *query.CollectionsPaginateOptions[[]byte]) {
		opt.Prefix = &prefixBytes
	})
	if err != nil {
		return nil, nil, err
	}

	return sdk.Coins(coins).Sort(), pageRes, nil
}

// SendCoins transfer coins to recipient
func (k MoveBankKeeper) SendCoins(
	ctx context.Context,
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
	ctx context.Context,
	accAddr sdk.AccAddress,
	coins sdk.Coins,
) error {

	communityPoolFunds := sdk.NewCoins()
	for _, coin := range coins {
		// if a coin is not generated from 0x1, then send the coin to community pool
		// because we don't have burn capability.
		if types.IsMoveCoin(coin) {
			communityPoolFunds = communityPoolFunds.Add(coin)
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

	// fund community pool with the coins that are not generated from 0x1
	if !communityPoolFunds.IsZero() {
		if err := k.communityPoolKeeper.FundCommunityPool(ctx, communityPoolFunds, accAddr); err != nil {
			return err
		}
	}

	return nil
}

// MintCoins mint coins to the address
func (k MoveBankKeeper) MintCoins(
	ctx context.Context,
	accAddr sdk.AccAddress,
	coins sdk.Coins,
) error {
	for _, coin := range coins {
		if types.IsMoveCoin(coin) {
			return moderrors.Wrapf(types.ErrInvalidRequest, "cannot mint move coin: %s", coin.Denom)
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
			types.FunctionNameManagedCoinSudoMint,
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
	ctx context.Context,
	denom string,
) error {
	if types.IsMoveDenom(denom) {
		return moderrors.Wrapf(types.ErrInvalidRequest, "cannot initialize move coin: %s", denom)
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
	ctx context.Context,
	fromAddr sdk.AccAddress,
	toAddr sdk.AccAddress,
	denom string,
	amount sdkmath.Int,
) error {
	metadata, err := types.MetadataAddressFromDenom(denom)
	if err != nil {
		return err
	}

	if err := k.assertDispatchableFungibleAssetAllowed(ctx, metadata); err != nil {
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

	return k.executeEntryFunction(
		ctx,
		[]vmtypes.AccountAddress{vmtypes.StdAddress, fromVmAddr},
		vmtypes.StdAddress,
		types.MoveModuleNameCoin,
		types.FunctionNameCoinSudoTransfer,
		[]vmtypes.TypeTag{},
		[][]byte{toVmAddr[:], metadata[:], amountBz},
		false,
	)
}

func (k MoveBankKeeper) MultiSend(
	ctx context.Context,
	sender sdk.AccAddress,
	denom string,
	recipients []sdk.AccAddress,
	amounts []sdkmath.Int,
) error {
	if len(recipients) != len(amounts) {
		return moderrors.Wrapf(types.ErrInvalidRequest, "recipients and amounts length mismatch")
	} else if len(recipients) == 0 {
		return moderrors.Wrapf(types.ErrInvalidRequest, "recipients and amounts length should be greater than 0")
	}

	senderVMAddr, err := vmtypes.NewAccountAddressFromBytes(sender)
	if err != nil {
		return err
	}

	metadata, err := types.MetadataAddressFromDenom(denom)
	if err != nil {
		return err
	}

	if err := k.assertDispatchableFungibleAssetAllowed(ctx, metadata); err != nil {
		return err
	}

	metadataArg, err := json.Marshal(metadata.String())
	if err != nil {
		return err
	}

	recipientAddrs := make([]string, len(recipients))
	for i, toAddr := range recipients {
		toVmAddr, err := vmtypes.NewAccountAddressFromBytes(toAddr)
		if err != nil {
			return err
		}

		recipientAddrs[i] = toVmAddr.String()
	}
	recipientsArg, err := json.Marshal(recipientAddrs)
	if err != nil {
		return err
	}

	amountsArg, err := json.Marshal(amounts)
	if err != nil {
		return err
	}

	return k.executeEntryFunction(
		ctx,
		[]vmtypes.AccountAddress{vmtypes.StdAddress, senderVMAddr},
		vmtypes.StdAddress,
		types.MoveModuleNameCoin,
		types.FunctionNameCoinSudoMultiSend,
		[]vmtypes.TypeTag{},
		[][]byte{metadataArg, recipientsArg, amountsArg},
		true,
	)
}

func (k MoveBankKeeper) GetUserStoresTableHandleWithLength(
	ctx context.Context,
	addr sdk.AccAddress,
) (*vmtypes.AccountAddress, sdkmath.Int, error) {
	bz, err := k.GetResourceBytes(ctx, vmtypes.StdAddress, vmtypes.StructTag{
		Address: vmtypes.StdAddress,
		Module:  types.MoveModuleNamePrimaryFungibleStore,
		Name:    types.ResourceNameModuleStore,
	})
	if err != nil {
		return nil, sdkmath.ZeroInt(), err
	}

	tableAddr, err := types.ReadUserStoresTableHandleFromModuleStore(bz)
	if err != nil {
		return nil, sdkmath.ZeroInt(), err
	}

	userAddr, err := vmtypes.NewAccountAddressFromBytes(addr[:])
	if err != nil {
		return nil, sdkmath.ZeroInt(), err
	}

	// check user has a store entry
	if ok, err := k.HasTableEntry(ctx, tableAddr, userAddr[:]); err != nil {
		return nil, sdkmath.ZeroInt(), err
	} else if !ok {
		return nil, sdkmath.ZeroInt(), err
	}

	tableEntry, err := k.GetTableEntryBytes(ctx, tableAddr, userAddr[:])
	if err != nil {
		return nil, sdkmath.ZeroInt(), err
	}

	tableAddr, err = types.ReadTableHandleFromTable(tableEntry.ValueBytes)
	if err != nil {
		return nil, sdkmath.ZeroInt(), err
	}

	length, err := types.ReadTableLengthFromTable(tableEntry.ValueBytes)
	if err != nil {
		return nil, sdkmath.ZeroInt(), err
	}

	return &tableAddr, length, nil
}

// assertDispatchableFungibleAssetAllowed checks if a dispatchable fungible asset is allowed to be used in the current context.
//
// Dispatchable fungible assets have custom logic that can be executed during transfers and other operations.
// For safety, we maintain a whitelist of contexts where these assets are permitted to prevent potential security issues
// or unexpected behavior. This function enforces that whitelist by checking if the current context allows dispatchable assets.
//
// If the asset is not dispatchable (has no dispatch function store), it is always allowed.
// Otherwise, the context must explicitly allow dispatchable assets via the AllowDispatchableContextKey.
func (k MoveBankKeeper) assertDispatchableFungibleAssetAllowed(ctx context.Context, metadata vmtypes.AccountAddress) error {
	hasDispatchFunctionStore, err := k.HasDispatchFunctionStore(ctx, metadata)
	if err != nil {
		return err
	} else if !hasDispatchFunctionStore {
		return nil
	}

	// Skip in CheckTx/ReCheckTx because IBC ante runs recv/ack/timeout messages during CheckTx
	// without msg context decorators.
	if sdkCtx := sdk.UnwrapSDKContext(ctx); sdkCtx.IsCheckTx() || sdkCtx.IsReCheckTx() {
		return nil
	}

	// check context value of allow dispatchable
	value := ctx.Value(types.AllowDispatchableContextKey)
	if value == nil || !value.(bool) {
		return moderrors.Wrapf(types.ErrInvalidRequest, "dispatchable fungible asset is not allowed in this context")
	}

	return nil
}
