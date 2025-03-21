package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmoskeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	customtypes "github.com/initia-labs/initia/x/bank/types"
)

var _ cosmoskeeper.ViewKeeper = (*MoveViewKeeper)(nil)

// MoveViewKeeper implements a read only keeper implementation of ViewKeeper.
type MoveViewKeeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	ak           types.AccountKeeper
	mk           customtypes.MoveBankKeeper

	Schema        collections.Schema
	DenomMetadata collections.Map[string, types.Metadata]
	SendEnabled   collections.Map[string, bool]
	Params        collections.Item[types.Params]
}

// NewMoveViewKeeper returns a new MoveViewKeeper.
func NewMoveViewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	ak types.AccountKeeper,
	mk customtypes.MoveBankKeeper,
) MoveViewKeeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := MoveViewKeeper{
		cdc:          cdc,
		storeService: storeService,
		ak:           ak,
		mk:           mk,

		DenomMetadata: collections.NewMap(sb, types.DenomMetadataPrefix, "denom_metadata", collections.StringKey, codec.CollValue[types.Metadata](cdc)),
		SendEnabled:   collections.NewMap(sb, types.SendEnabledPrefix, "send_enabled", collections.StringKey, codec.BoolValue), // NOTE: we use a bool value which uses protobuf to retain state backwards compat
		Params:        collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

// Logger returns a module-specific logger.
func (k MoveViewKeeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// GetDenomMetaData returns the metadata of a specific denomination.
func (k MoveViewKeeper) GetDenomMetaData(ctx context.Context, denom string) (types.Metadata, error) {
	metadata, err := k.DenomMetadata.Get(ctx, denom)
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return k.mk.GetMetadata(ctx, denom)
	} else if err != nil {
		return types.Metadata{}, err
	}

	return metadata, nil
}

// HasDenomMetaData returns whether or not the metadata for a denomination exists.
func (k MoveViewKeeper) HasDenomMetaData(ctx context.Context, denom string) (bool, error) {
	found, err := k.DenomMetadata.Has(ctx, denom)
	if err != nil {
		return false, err
	} else if !found {
		return k.mk.HasMetadata(ctx, denom)
	}

	return found, nil
}

// HasBalance returns whether or not an account has at least amt balance.
func (k MoveViewKeeper) HasBalance(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool {
	return k.GetBalance(ctx, addr, amt.Denom).IsGTE(amt)
}

// GetAllBalances returns all the account balances for the given account address.
func (k MoveViewKeeper) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	balances := sdk.NewCoins()
	k.IterateAccountBalances(ctx, addr, func(balance sdk.Coin) bool {
		balances = balances.Add(balance)
		return false
	})

	return balances.Sort()
}

// GetAccountsBalances returns all the accounts balances from the store.
func (k MoveViewKeeper) GetAccountsBalances(ctx context.Context) []types.Balance {
	balances := make([]types.Balance, 0)
	mapAddressToBalancesIdx := make(map[string]int)

	k.IterateAllBalances(ctx, func(addr sdk.AccAddress, balance sdk.Coin) bool {
		idx, ok := mapAddressToBalancesIdx[addr.String()]
		if ok {
			// address is already on the set of accounts balances
			balances[idx].Coins = balances[idx].Coins.Add(balance)
			balances[idx].Coins.Sort()
			return false
		}

		accountBalance := types.Balance{
			Address: addr.String(),
			Coins:   sdk.NewCoins(balance),
		}
		balances = append(balances, accountBalance)
		mapAddressToBalancesIdx[addr.String()] = len(balances) - 1
		return false
	})

	return balances
}

// GetBalance returns the balance of a specific denomination for a given account
// by address.
func (k MoveViewKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	balance, err := k.mk.GetBalance(ctx, addr, denom)
	if err != nil {
		panic(err)
	}

	return sdk.NewCoin(denom, balance)
}

// IterateAccountBalances iterates over the balances of a single account and
// provides the token balance to a callback. If true is returned from the
// callback, iteration is halted.
func (k MoveViewKeeper) IterateAccountBalances(ctx context.Context, addr sdk.AccAddress, cb func(sdk.Coin) bool) {
	err := k.mk.IterateAccountBalances(ctx, addr, func(c sdk.Coin) (bool, error) {
		return cb(c), nil
	})
	if err != nil {
		panic(err)
	}
}

// IterateAllBalances iterates over all the balances of all accounts and
// denominations that are provided to a callback. If true is returned from the
// callback, iteration is halted.
func (k MoveViewKeeper) IterateAllBalances(ctx context.Context, cb func(sdk.AccAddress, sdk.Coin) bool) {
	k.ak.IterateAccounts(ctx, func(account sdk.AccountI) bool {
		addr := account.GetAddress()
		err := k.mk.IterateAccountBalances(ctx, addr, func(coin sdk.Coin) (bool, error) {
			return cb(addr, coin), nil
		})
		if err != nil {
			panic(err)
		}

		return false
	})
}

// LockedCoins returns all the coins that are not spendable (i.e. locked) for an
// account by address. For standard accounts, the result will always be no coins.
func (k MoveViewKeeper) LockedCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return sdk.NewCoins()
}

// SpendableCoins returns the total balances of spendable coins for an account
// by address. If the account has no spendable coins, an empty Coins slice is
// returned.
func (k MoveViewKeeper) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	spendable, _ := k.spendableCoins(ctx, addr)
	return spendable
}

// SpendableCoin returns the balance of specific denomination of spendable coins
// for an account by address. If the account has no spendable coin, a zero Coin
// is returned.
func (k MoveViewKeeper) SpendableCoin(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	balance := k.GetBalance(ctx, addr, denom)
	locked := k.LockedCoins(ctx, addr)
	return balance.SubAmount(locked.AmountOf(denom))
}

// spendableCoins returns the coins the given address can spend alongside the total amount of coins it holds.
// It exists for gas efficiency, in order to avoid to have to get balance multiple times.
func (k MoveViewKeeper) spendableCoins(ctx context.Context, addr sdk.AccAddress) (spendable, total sdk.Coins) {
	total = k.GetAllBalances(ctx, addr)
	locked := k.LockedCoins(ctx, addr)

	spendable, hasNeg := total.SafeSub(locked...)
	if hasNeg {
		spendable = sdk.NewCoins()
		return
	}

	return
}

// ValidateBalance validates all balances for a given account address returning
// an error if any balance is invalid.
//
// CONTRACT: ValidateBalance should only be called upon genesis state.
func (k MoveViewKeeper) ValidateBalance(ctx context.Context, addr sdk.AccAddress) error {
	balances := k.GetAllBalances(ctx, addr)
	if !balances.IsValid() {
		return fmt.Errorf("account balance of %s is invalid", balances)
	}

	return nil
}
