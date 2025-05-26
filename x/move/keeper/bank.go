package keeper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	moderrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

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

	// query balance from primary_fungible_store
	output, _, err := k.ExecuteViewFunctionJSON(
		ctx,
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
		return sdkmath.ZeroInt(), err
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

	// get user address
	userAddr, err := vmtypes.NewAccountAddressFromBytes(addr[:])
	if err != nil {
		return err
	}

	const fetchLimit = 100
	startAfter := "null"

BALANCE_LOOP:
	for {
		coins, nextKey, err := k.balances(ctx, userAddr, startAfter, fetchLimit, true)
		if err != nil {
			return err
		} else if len(coins) == 0 {
			break BALANCE_LOOP
		}

		for _, coin := range coins {
			if coin.Amount.IsZero() {
				continue
			}

			if stop, err := cb(coin); err != nil {
				return err
			} else if stop {
				break BALANCE_LOOP
			}
		}

		// no more coins to fetch
		if nextKey == nil {
			break BALANCE_LOOP
		}

		startAfter = *nextKey
	}

	return nil
}

func (k MoveBankKeeper) GetPaginatedBalances(ctx context.Context, pageReq *query.PageRequest, addr sdk.AccAddress) (sdk.Coins, *query.PageResponse, error) {
	// check user has stores table
	tableAddr, tableLength, err := k.GetUserStoresTableHandleWithLength(ctx, addr)
	if err != nil {
		return nil, nil, err
	} else if tableAddr == nil || tableLength.IsZero() {
		return sdk.NewCoins(), &query.PageResponse{}, nil
	}

	// get user address
	userAddr, err := vmtypes.NewAccountAddressFromBytes(addr[:])
	if err != nil {
		return nil, nil, err
	}

	startAfter := "null"
	if pageReq != nil && len(pageReq.Key) > 0 {
		startAfter = string(pageReq.Key)
	}

	limit := uint64(100)
	if pageReq != nil && pageReq.Limit > 0 {
		limit = min(pageReq.Limit, tableLength.Uint64())
	}

	const fetchLimit = 100
	coins := make([]sdk.Coin, 0, limit)

	for {
		coins_, nextKey, err := k.balances(ctx, userAddr, startAfter, fetchLimit, true)
		if err != nil {
			return nil, nil, err
		}

		coins = append(coins, coins_...)

		// if the number of coins is less than the limit, break the loop
		if len(coins) >= int(limit) {
			break
		}

		// no more coins to fetch
		if nextKey == nil {
			break
		}

		startAfter = *nextKey
	}

	// prepare page response
	pageRes := &query.PageResponse{
		Total: 0,
	}

	// truncate coins to the limit
	coinsLength := uint64(len(coins))
	if coinsLength > limit {
		coins = coins[:limit]
	}

	// check has more coins to fetch
	if coinsLength >= limit {
		metadata, err := types.MetadataAddressFromDenom(coins[len(coins)-1].Denom)
		if err != nil {
			return nil, nil, err
		}
		startAfter = fmt.Sprintf("\"%s\"", metadata)
		if coins, _, err := k.balances(ctx, userAddr, startAfter, 1, false); err != nil {
			return nil, nil, err
		} else if len(coins) > 0 {
			pageRes.NextKey = []byte(startAfter)
		}
	}

	return sdk.Coins(coins).Sort(), pageRes, nil
}

// balances fetches token balances from primary_fungible_store for a given address.
// Returns unsorted coins and pagination info.
// Use startAfter="null" for first page, limit controls max results.
// Note: Sort coins before using sdk.Coins functions.
func (k MoveBankKeeper) balances(ctx context.Context, addr vmtypes.AccountAddress, startAfter string, limit uint64, filterZero bool) ([]sdk.Coin, *string, error) {
	output, _, err := k.ExecuteViewFunctionJSON(
		ctx,
		vmtypes.StdAddress,
		types.MoveModuleNamePrimaryFungibleStore,
		types.FunctionNamePrimaryFungibleStoreBalances,
		[]vmtypes.TypeTag{},
		[]string{fmt.Sprintf("\"%s\"", addr), startAfter, fmt.Sprintf("%d", limit)},
	)
	if err != nil {
		return nil, nil, err
	}

	var results [][]string
	err = json.Unmarshal([]byte(output.Ret), &results)
	if err != nil {
		return nil, nil, err
	}

	if len(results) != 2 {
		return nil, nil, moderrors.Wrapf(types.ErrInvalidResponse, "invalid balance results: %s", output.Ret)
	}

	metadataArray := results[0]
	amountArray := results[1]

	resLength := len(metadataArray)
	if resLength != len(amountArray) {
		return nil, nil, moderrors.Wrapf(types.ErrInvalidResponse, "invalid balance results: %s", output.Ret)
	} else if resLength == 0 {
		return nil, nil, nil
	}

	coins := make([]sdk.Coin, 0, resLength)
	for i := range resLength {
		amount, ok := sdkmath.NewIntFromString(amountArray[i])
		if !ok {
			return nil, nil, moderrors.Wrapf(types.ErrInvalidResponse, "invalid balance amount: %s", amountArray[i])
		}

		if filterZero && amount.IsZero() {
			continue
		}

		metadata, err := vmtypes.NewAccountAddress(metadataArray[i])
		if err != nil {
			return nil, nil, moderrors.Wrapf(types.ErrInvalidResponse, "invalid balance metadata: %s", metadataArray[i])
		}

		denom, err := types.DenomFromMetadataAddress(
			ctx, k, metadata,
		)
		if err != nil {
			return nil, nil, moderrors.Wrapf(types.ErrInvalidResponse, "invalid balance metadata: %s", metadataArray[i])
		}

		// add coin to coins
		coins = append(coins, sdk.NewCoin(denom, amount))
	}

	// update startAfter to the last metadata address
	if limit > uint64(resLength) {
		return coins, nil, nil
	}

	startAfter = fmt.Sprintf("\"%s\"", metadataArray[resLength-1])
	return coins, &startAfter, nil
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
