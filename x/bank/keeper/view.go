package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	cosmoskeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	customtypes "github.com/initia-labs/initia/x/bank/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

var _ cosmoskeeper.ViewKeeper = (*MoveViewKeeper)(nil)

// MoveViewKeeper implements a read only keeper implementation of ViewKeeper.
type MoveViewKeeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
	ak       types.AccountKeeper
	mk       customtypes.MoveBankKeeper
}

// NewMoveViewKeeper returns a new MoveViewKeeper.
func NewMoveViewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	ak types.AccountKeeper,
	mk customtypes.MoveBankKeeper,
) MoveViewKeeper {
	return MoveViewKeeper{
		cdc:      cdc,
		storeKey: storeKey,
		ak:       ak,
		mk:       mk,
	}
}

// Logger returns a module-specific logger.
func (k MoveViewKeeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// HasBalance returns whether or not an account has at least amt balance.
func (k MoveViewKeeper) HasBalance(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coin) bool {
	return k.GetBalance(ctx, addr, amt.Denom).IsGTE(amt)
}

// GetAllBalances returns all the account balances for the given account address.
func (k MoveViewKeeper) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	balances := sdk.NewCoins()
	k.IterateAccountBalances(ctx, addr, func(balance sdk.Coin) bool {
		balances = balances.Add(balance)
		return false
	})

	return balances.Sort()
}

// GetAccountsBalances returns all the accounts balances from the store.
func (k MoveViewKeeper) GetAccountsBalances(ctx sdk.Context) []types.Balance {
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
func (k MoveViewKeeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	balance, err := k.mk.GetBalance(ctx, addr, denom)
	if err != nil {
		panic(err)
	}

	return sdk.NewCoin(denom, balance)
}

// IterateAccountBalances iterates over the balances of a single account and
// provides the token balance to a callback. If true is returned from the
// callback, iteration is halted.
func (k MoveViewKeeper) IterateAccountBalances(ctx sdk.Context, addr sdk.AccAddress, cb func(sdk.Coin) bool) {
	userStores, err := k.mk.GetUserStores(ctx, addr)
	if err != nil {
		panic(err)
	}

	// stores not found
	if userStores == nil {
		return
	}

	iterator := userStores.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		value := iterator.Value()
		storeAddr, err := vmtypes.NewAccountAddressFromBytes(value)
		if err != nil {
			panic(err)
		}

		metadata, amount, err := k.mk.Balance(ctx, storeAddr)
		if err != nil {
			panic(err)
		}

		denom, err := movetypes.DenomFromMetadataAddress(
			ctx,
			k.mk,
			metadata,
		)
		if err != nil {
			panic(err)
		}

		if cb(sdk.NewCoin(denom, amount)) {
			break
		}
	}
}

// IterateAllBalances iterates over all the balances of all accounts and
// denominations that are provided to a callback. If true is returned from the
// callback, iteration is halted.
func (k MoveViewKeeper) IterateAllBalances(ctx sdk.Context, cb func(sdk.AccAddress, sdk.Coin) bool) {
	k.ak.IterateAccounts(ctx, func(account authtypes.AccountI) bool {
		addr := account.GetAddress()
		userStores, err := k.mk.GetUserStores(ctx, addr)
		if err != nil {
			panic(err)
		}

		// stores not found
		if userStores == nil {
			return false
		}

		iterator := userStores.Iterator(nil, nil)
		defer iterator.Close()

		for ; iterator.Valid(); iterator.Next() {
			value := iterator.Value()
			storeAddr, err := vmtypes.NewAccountAddressFromBytes(value)
			if err != nil {
				panic(err)
			}

			metadata, amount, err := k.mk.Balance(ctx, storeAddr)
			if err != nil {
				panic(err)
			}
			if amount.IsZero() {
				continue
			}

			denom, err := movetypes.DenomFromMetadataAddress(
				ctx,
				k.mk,
				metadata,
			)
			if err != nil {
				panic(err)
			}

			if cb(addr, sdk.NewCoin(denom, amount)) {
				break
			}
		}

		return false
	})
}

// LockedCoins returns all the coins that are not spendable (i.e. locked) for an
// account by address. For standard accounts, the result will always be no coins.
func (k MoveViewKeeper) LockedCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return sdk.NewCoins()
}

// SpendableCoins returns the total balances of spendable coins for an account
// by address. If the account has no spendable coins, an empty Coins slice is
// returned.
func (k MoveViewKeeper) SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	spendable, _ := k.spendableCoins(ctx, addr)
	return spendable
}

// SpendableCoin returns the balance of specific denomination of spendable coins
// for an account by address. If the account has no spendable coin, a zero Coin
// is returned.
func (k MoveViewKeeper) SpendableCoin(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	balance := k.GetBalance(ctx, addr, denom)
	locked := k.LockedCoins(ctx, addr)
	return balance.SubAmount(locked.AmountOf(denom))
}

// spendableCoins returns the coins the given address can spend alongside the total amount of coins it holds.
// It exists for gas efficiency, in order to avoid to have to get balance multiple times.
func (k MoveViewKeeper) spendableCoins(ctx sdk.Context, addr sdk.AccAddress) (spendable, total sdk.Coins) {
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
func (k MoveViewKeeper) ValidateBalance(ctx sdk.Context, addr sdk.AccAddress) error {
	balances := k.GetAllBalances(ctx, addr)
	if !balances.IsValid() {
		return fmt.Errorf("account balance of %s is invalid", balances)
	}

	return nil
}
