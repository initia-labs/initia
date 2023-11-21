package keeper

import (
	"fmt"

	gogotypes "github.com/cosmos/gogoproto/types"

	"cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	cosmosbank "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	customtypes "github.com/initia-labs/initia/x/bank/types"
)

var _ cosmosbank.SendKeeper = (*MoveSendKeeper)(nil)

// MoveSendKeeper only allows transfers between accounts without the possibility of
// creating coins. It implements the SendKeeper interface.
type MoveSendKeeper struct {
	MoveViewKeeper

	cdc      codec.BinaryCodec
	ak       types.AccountKeeper
	storeKey storetypes.StoreKey

	// list of addresses that are restricted from receiving transactions
	blockedAddrs map[string]bool

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

func NewMoveSendKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	ak types.AccountKeeper,
	mk customtypes.MoveBankKeeper,
	blockedAddrs map[string]bool,
	authority string,
) MoveSendKeeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Errorf("invalid bank authority address: %w", err))
	}

	return MoveSendKeeper{
		MoveViewKeeper: NewMoveViewKeeper(cdc, storeKey, ak, mk),
		cdc:            cdc,
		ak:             ak,
		storeKey:       storeKey,
		blockedAddrs:   blockedAddrs,
		authority:      authority,
	}
}

// GetAuthority returns the x/bank module's authority.
func (k MoveSendKeeper) GetAuthority() string {
	return k.authority
}

// GetParams returns the total set of bank parameters.
func (k MoveSendKeeper) GetParams(ctx sdk.Context) (params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the total set of bank parameters.
//
// Note: params.SendEnabled is deprecated but it should be here regardless.
//
//nolint:staticcheck
func (k MoveSendKeeper) SetParams(ctx sdk.Context, params types.Params) error {
	// Normally SendEnabled is deprecated but we still support it for backwards
	// compatibility. Using params.Validate() would fail due to the SendEnabled
	// deprecation.
	if len(params.SendEnabled) > 0 {
		k.SetAllSendEnabled(ctx, params.SendEnabled)

		// override params without SendEnabled
		params = types.NewParams(params.DefaultSendEnabled)
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}

	store.Set(types.ParamsKey, bz)
	return nil
}

// InputOutputCoins performs multi-send functionality. It accepts a series of
// inputs that correspond to a series of outputs. It returns an error if the
// inputs and outputs don't lineup or if any single transfer of tokens fails.
func (k MoveSendKeeper) InputOutputCoins(ctx sdk.Context, inputs []types.Input, outputs []types.Output) error {
	return sdkerrors.ErrNotSupported
}

// SendCoins transfers amt coins from a sending account to a receiving account.
// An error is returned upon failure.
func (k MoveSendKeeper) SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	err := k.mk.SendCoins(ctx, fromAddr, toAddr, amt)
	if err != nil {
		return err
	}

	// emit coin spent event
	ctx.EventManager().EmitEvent(
		types.NewCoinSpentEvent(fromAddr, amt),
	)

	// emit coin received event
	ctx.EventManager().EmitEvent(
		types.NewCoinReceivedEvent(toAddr, amt),
	)

	// Create account if recipient does not exist.
	//
	// NOTE: This should ultimately be removed in favor a more flexible approach
	// such as delegated fee messages.
	accExists := k.ak.HasAccount(ctx, toAddr)
	if !accExists {
		defer telemetry.IncrCounter(1, "new", "account")
		k.ak.SetAccount(ctx, k.ak.NewAccountWithAddress(ctx, toAddr))
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTransfer,
			sdk.NewAttribute(types.AttributeKeyRecipient, toAddr.String()),
			sdk.NewAttribute(types.AttributeKeySender, fromAddr.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, amt.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(types.AttributeKeySender, fromAddr.String()),
		),
	})

	return nil
}

// initBalances sets the balance (multiple coins) for an account by address.
// An error is returned upon failure.
func (k MoveSendKeeper) initBalances(ctx sdk.Context, addr sdk.AccAddress, balances sdk.Coins) error {
	return k.mk.MintCoins(ctx, addr, balances)
}

// IsSendEnabledCoins checks the coins provide and returns an ErrSendDisabled if
// any of the coins are not configured for sending.  Returns nil if sending is enabled
// for all provided coin
func (k MoveSendKeeper) IsSendEnabledCoins(ctx sdk.Context, coins ...sdk.Coin) error {
	for _, coin := range coins {
		if !k.IsSendEnabledCoin(ctx, coin) {
			return errors.Wrapf(types.ErrSendDisabled, "%s transfers are currently disabled", coin.Denom)
		}
	}
	return nil
}

// IsSendEnabledCoin returns the current SendEnabled status of the provided coin's denom
func (k MoveSendKeeper) IsSendEnabledCoin(ctx sdk.Context, coin sdk.Coin) bool {
	return k.IsSendEnabledDenom(ctx, coin.Denom)
}

// BlockedAddr checks if a given address is restricted from
// receiving funds.
func (k MoveSendKeeper) BlockedAddr(addr sdk.AccAddress) bool {
	return k.blockedAddrs[addr.String()]
}

// GetBlockedAddresses returns the full list of addresses restricted from receiving funds.
func (k MoveSendKeeper) GetBlockedAddresses() map[string]bool {
	return k.blockedAddrs
}

// IsSendEnabledDenom returns the current SendEnabled status of the provided denom.
func (k MoveSendKeeper) IsSendEnabledDenom(ctx sdk.Context, denom string) bool {
	return k.getSendEnabledOrDefault(ctx.KVStore(k.storeKey), denom, k.GetParams(ctx).DefaultSendEnabled)
}

// GetSendEnabledEntry gets a SendEnabled entry for the given denom.
// The second return argument is true iff a specific entry exists for the given denom.
func (k MoveSendKeeper) GetSendEnabledEntry(ctx sdk.Context, denom string) (types.SendEnabled, bool) {
	sendEnabled, found := k.getSendEnabled(ctx.KVStore(k.storeKey), denom)
	if !found {
		return types.SendEnabled{}, false
	}

	return types.SendEnabled{Denom: denom, Enabled: sendEnabled}, true
}

// SetSendEnabled sets the SendEnabled flag for a denom to the provided value.
func (k MoveSendKeeper) SetSendEnabled(ctx sdk.Context, denom string, value bool) {
	store := ctx.KVStore(k.storeKey)
	k.setSendEnabledEntry(store, denom, value)
}

// SetAllSendEnabled sets all the provided SendEnabled entries in the bank store.
func (k MoveSendKeeper) SetAllSendEnabled(ctx sdk.Context, entries []*types.SendEnabled) {
	store := ctx.KVStore(k.storeKey)
	for _, entry := range entries {
		k.setSendEnabledEntry(store, entry.Denom, entry.Enabled)
	}
}

// setSendEnabledEntry sets SendEnabled for the given denom to the give value in the provided store.
func (k MoveSendKeeper) setSendEnabledEntry(store sdk.KVStore, denom string, value bool) {
	key := types.CreateSendEnabledKey(denom)

	bz := k.cdc.MustMarshal(&gogotypes.BoolValue{Value: value})
	store.Set(key, bz)
}

// DeleteSendEnabled deletes the SendEnabled flags for one or more denoms.
// If a denom is provided that doesn't have a SendEnabled entry, it is ignored.
func (k MoveSendKeeper) DeleteSendEnabled(ctx sdk.Context, denoms ...string) {
	store := ctx.KVStore(k.storeKey)
	for _, denom := range denoms {
		store.Delete(types.CreateSendEnabledKey(denom))
	}
}

// getSendEnabledPrefixStore gets a prefix store for the SendEnabled entries.
func (k MoveSendKeeper) getSendEnabledPrefixStore(ctx sdk.Context) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.SendEnabledPrefix)
}

// IterateSendEnabledEntries iterates over all the SendEnabled entries.
func (k MoveSendKeeper) IterateSendEnabledEntries(ctx sdk.Context, cb func(denom string, sendEnabled bool) bool) {
	seStore := k.getSendEnabledPrefixStore(ctx)

	iterator := seStore.Iterator(nil, nil)
	defer sdk.LogDeferred(ctx.Logger(), func() error { return iterator.Close() })

	for ; iterator.Valid(); iterator.Next() {
		denom := string(iterator.Key())

		var enabled gogotypes.BoolValue
		k.cdc.MustUnmarshal(iterator.Value(), &enabled)

		if cb(denom, enabled.Value) {
			break
		}
	}
}

// GetAllSendEnabledEntries gets all the SendEnabled entries that are stored.
// Any denominations not returned use the default value (set in Params).
func (k MoveSendKeeper) GetAllSendEnabledEntries(ctx sdk.Context) []types.SendEnabled {
	var rv []types.SendEnabled
	k.IterateSendEnabledEntries(ctx, func(denom string, sendEnabled bool) bool {
		rv = append(rv, types.SendEnabled{Denom: denom, Enabled: sendEnabled})
		return false
	})

	return rv
}

// getSendEnabled returns whether send is enabled and whether that flag was set
// for a denom.
//
// Example usage:
//
//	store := ctx.KVStore(k.storeKey)
//	sendEnabled, found := getSendEnabled(store, "atom")
//	if !found {
//	    sendEnabled = DefaultSendEnabled
//	}
func (k MoveSendKeeper) getSendEnabled(store sdk.KVStore, denom string) (bool, bool) {
	key := types.CreateSendEnabledKey(denom)
	if !store.Has(key) {
		return false, false
	}

	bz := store.Get(key)
	if bz == nil {
		return false, false
	}

	var enabled gogotypes.BoolValue
	k.cdc.MustUnmarshal(bz, &enabled)

	return enabled.Value, true
}

// getSendEnabledOrDefault gets the SendEnabled value for a denom. If it's not
// in the store, this will return defaultVal.
func (k MoveSendKeeper) getSendEnabledOrDefault(store sdk.KVStore, denom string, defaultVal bool) bool {
	sendEnabled, found := k.getSendEnabled(store, denom)
	if found {
		return sendEnabled
	}

	return defaultVal
}
