package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmosbank "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	customtypes "github.com/initia-labs/initia/x/bank/types"
)

var _ cosmosbank.SendKeeper = (*MoveSendKeeper)(nil)

// MoveSendKeeper only allows transfers between accounts without the possibility of
// creating coins. It implements the SendKeeper interface.
type MoveSendKeeper struct {
	MoveViewKeeper

	cdc          codec.BinaryCodec
	ak           types.AccountKeeper
	storeService store.KVStoreService

	// list of addresses that are restricted from receiving transactions
	blockedAddrs map[string]bool

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string

	sendRestriction *sendRestriction
}

func NewMoveSendKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	ak types.AccountKeeper,
	mk customtypes.MoveBankKeeper,
	blockedAddrs map[string]bool,
	authority string,
) MoveSendKeeper {
	if _, err := ak.AddressCodec().StringToBytes(authority); err != nil {
		panic(fmt.Errorf("invalid bank authority address: %w", err))
	}

	return MoveSendKeeper{
		MoveViewKeeper:  NewMoveViewKeeper(cdc, storeService, ak, mk),
		cdc:             cdc,
		ak:              ak,
		storeService:    storeService,
		blockedAddrs:    blockedAddrs,
		authority:       authority,
		sendRestriction: newSendRestriction(),
	}
}

// AppendSendRestriction adds the provided SendRestrictionFn to run after previously provided restrictions.
func (k MoveSendKeeper) AppendSendRestriction(restriction types.SendRestrictionFn) {
	k.sendRestriction.append(restriction)
}

// PrependSendRestriction adds the provided SendRestrictionFn to run before previously provided restrictions.
func (k MoveSendKeeper) PrependSendRestriction(restriction types.SendRestrictionFn) {
	k.sendRestriction.prepend(restriction)
}

// ClearSendRestriction removes the send restriction (if there is one).
func (k MoveSendKeeper) ClearSendRestriction() {
	k.sendRestriction.clear()
}

// GetAuthority returns the x/bank module's authority.
func (k MoveSendKeeper) GetAuthority() string {
	return k.authority
}

// GetParams returns the total set of bank parameters.
func (k MoveSendKeeper) GetParams(ctx context.Context) (params types.Params) {
	p, _ := k.Params.Get(ctx)
	return p
}

// SetParams sets the total set of bank parameters.
//
// Note: params.SendEnabled is deprecated but it should be here regardless.
//

func (k MoveSendKeeper) SetParams(ctx context.Context, params types.Params) error {
	// Normally SendEnabled is deprecated but we still support it for backwards
	// compatibility. Using params.Validate() would fail due to the SendEnabled
	// deprecation.
	if len(params.SendEnabled) > 0 { //nolint:staticcheck // SA1019: params.SendEnabled is deprecated
		k.SetAllSendEnabled(ctx, params.SendEnabled) //nolint:staticcheck // SA1019: params.SendEnabled is deprecated

		// override params without SendEnabled
		params = types.NewParams(params.DefaultSendEnabled)
	}
	return k.Params.Set(ctx, params)
}

// InputOutputCoins performs multi-send functionality. It transfers coins from a single sender
// to multiple recipients. An error is returned upon failure.
func (k MoveSendKeeper) InputOutputCoins(ctx context.Context, input types.Input, outputs []types.Output) error {
	// Safety check ensuring that when sending coins the keeper must maintain the
	// Check supply invariant and validity of Coins.
	if err := types.ValidateInputOutputs(input, outputs); err != nil {
		return err
	}

	fromAddr, err := k.ak.AddressCodec().StringToBytes(input.Address)
	if err != nil {
		return err
	}

	// event emission
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(types.AttributeKeySender, input.Address),
		),
	)

	// emit coin spent event
	sdkCtx.EventManager().EmitEvent(
		types.NewCoinSpentEvent(fromAddr, input.Coins),
	)

	// emit coin received events and do address caching
	addrMap := make(map[string][]byte)
	for _, output := range outputs {
		addr, err := k.ak.AddressCodec().StringToBytes(output.Address)
		if err != nil {
			return err
		}

		// cache bytes address
		addrMap[output.Address] = addr

		// emit coin received event
		sdkCtx.EventManager().EmitEvent(
			types.NewCoinReceivedEvent(addr, output.Coins),
		)

		// emit transfer event (for compatibility with cosmos bank)
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeTransfer,
				sdk.NewAttribute(types.AttributeKeyRecipient, output.Address),
				sdk.NewAttribute(sdk.AttributeKeyAmount, output.Coins.String()),
			),
		)
	}

	for _, coin := range input.Coins {
		if !coin.Amount.IsPositive() {
			continue
		}

		recipients := make([]sdk.AccAddress, 0, len(outputs))
		amounts := make([]math.Int, 0, len(outputs))
		for _, output := range outputs {
			// Create account if recipient does not exist.
			outAddress := addrMap[output.Address]
			accExists := k.ak.HasAccount(ctx, outAddress)
			if !accExists {
				defer telemetry.IncrCounter(1, "new", "account")
				k.ak.SetAccount(ctx, k.ak.NewAccountWithAddress(ctx, outAddress))
			}

			amount := output.Coins.AmountOf(coin.Denom)
			if !amount.IsPositive() {
				continue
			}

			recipients = append(recipients, outAddress)
			amounts = append(amounts, output.Coins.AmountOf(coin.Denom))
		}

		if err = k.mk.MultiSend(ctx, fromAddr, coin.Denom, recipients, amounts); err != nil {
			return err
		}
	}

	return nil
}

// SendCoins transfers amt coins from a sending account to a receiving account.
// An error is returned upon failure.
func (k MoveSendKeeper) SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	toAddr, err := k.sendRestriction.apply(ctx, fromAddr, toAddr, amt)
	if err != nil {
		return err
	}

	err = k.mk.SendCoins(ctx, fromAddr, toAddr, amt)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// emit coin spent event
	sdkCtx.EventManager().EmitEvent(
		types.NewCoinSpentEvent(fromAddr, amt),
	)

	// emit coin received event
	sdkCtx.EventManager().EmitEvent(
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

	sdkCtx.EventManager().EmitEvents(sdk.Events{
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
func (k MoveSendKeeper) initBalances(ctx context.Context, addr sdk.AccAddress, balances sdk.Coins) error {
	return k.mk.MintCoins(ctx, addr, balances)
}

// IsSendEnabledCoins checks the coins provided and returns an ErrSendDisabled
// if any of the coins are not configured for sending. Returns nil if sending is
// enabled for all provided coins.
func (k MoveSendKeeper) IsSendEnabledCoins(ctx context.Context, coins ...sdk.Coin) error {
	if len(coins) == 0 {
		return nil
	}

	defaultVal := k.GetParams(ctx).DefaultSendEnabled

	for _, coin := range coins {
		if !k.getSendEnabledOrDefault(ctx, coin.Denom, defaultVal) {
			return types.ErrSendDisabled.Wrapf("%s transfers are currently disabled", coin.Denom)
		}
	}

	return nil
}

// IsSendEnabledCoin returns the current SendEnabled status of the provided coin's denom
func (k MoveSendKeeper) IsSendEnabledCoin(ctx context.Context, coin sdk.Coin) bool {
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
func (k MoveSendKeeper) IsSendEnabledDenom(ctx context.Context, denom string) bool {
	return k.getSendEnabledOrDefault(ctx, denom, k.GetParams(ctx).DefaultSendEnabled)
}

// GetSendEnabledEntry gets a SendEnabled entry for the given denom.
// The second return argument is true iff a specific entry exists for the given denom.
func (k MoveSendKeeper) GetSendEnabledEntry(ctx context.Context, denom string) (types.SendEnabled, bool) {
	sendEnabled, found := k.getSendEnabled(ctx, denom)
	if !found {
		return types.SendEnabled{}, false
	}

	return types.SendEnabled{Denom: denom, Enabled: sendEnabled}, true
}

// SetSendEnabled sets the SendEnabled flag for a denom to the provided value.
func (k MoveSendKeeper) SetSendEnabled(ctx context.Context, denom string, value bool) {
	_ = k.SendEnabled.Set(ctx, denom, value)
}

// SetAllSendEnabled sets all the provided SendEnabled entries in the bank store.
func (k MoveSendKeeper) SetAllSendEnabled(ctx context.Context, entries []*types.SendEnabled) {
	for _, entry := range entries {
		_ = k.SendEnabled.Set(ctx, entry.Denom, entry.Enabled)
	}
}

// DeleteSendEnabled deletes the SendEnabled flags for one or more denoms.
// If a denom is provided that doesn't have a SendEnabled entry, it is ignored.
func (k MoveSendKeeper) DeleteSendEnabled(ctx context.Context, denoms ...string) {
	for _, denom := range denoms {
		_ = k.SendEnabled.Remove(ctx, denom)
	}
}

// IterateSendEnabledEntries iterates over all the SendEnabled entries.
func (k MoveSendKeeper) IterateSendEnabledEntries(ctx context.Context, cb func(denom string, sendEnabled bool) bool) {
	err := k.SendEnabled.Walk(ctx, nil, func(key string, value bool) (stop bool, err error) {
		return cb(key, value), nil
	})
	if err != nil {
		panic(err)
	}
}

// GetAllSendEnabledEntries gets all the SendEnabled entries that are stored.
// Any denominations not returned use the default value (set in Params).
func (k MoveSendKeeper) GetAllSendEnabledEntries(ctx context.Context) []types.SendEnabled {
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
func (k MoveSendKeeper) getSendEnabled(ctx context.Context, denom string) (bool, bool) {
	has, err := k.SendEnabled.Has(ctx, denom)
	if err != nil || !has {
		return false, false
	}

	v, err := k.SendEnabled.Get(ctx, denom)
	if err != nil {
		return false, false
	}

	return v, true
}

// getSendEnabledOrDefault gets the SendEnabled value for a denom. If it's not
// in the store, this will return defaultVal.
func (k MoveSendKeeper) getSendEnabledOrDefault(ctx context.Context, denom string, defaultVal bool) bool {
	sendEnabled, found := k.getSendEnabled(ctx, denom)
	if found {
		return sendEnabled
	}

	return defaultVal
}

// sendRestriction is a struct that houses a SendRestrictionFn.
// It exists so that the SendRestrictionFn can be updated in the SendKeeper without needing to have a pointer receiver.
type sendRestriction struct {
	fn types.SendRestrictionFn
}

// newSendRestriction creates a new sendRestriction with nil send restriction.
func newSendRestriction() *sendRestriction {
	return &sendRestriction{
		fn: nil,
	}
}

// append adds the provided restriction to this, to be run after the existing function.
func (r *sendRestriction) append(restriction types.SendRestrictionFn) {
	r.fn = r.fn.Then(restriction)
}

// prepend adds the provided restriction to this, to be run before the existing function.
func (r *sendRestriction) prepend(restriction types.SendRestrictionFn) {
	r.fn = restriction.Then(r.fn)
}

// clear removes the send restriction (sets it to nil).
func (r *sendRestriction) clear() {
	r.fn = nil
}

var _ types.SendRestrictionFn = (*sendRestriction)(nil).apply

// apply applies the send restriction if there is one. If not, it's a no-op.
func (r *sendRestriction) apply(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) (sdk.AccAddress, error) {
	if r == nil || r.fn == nil {
		return toAddr, nil
	}
	return r.fn(ctx, fromAddr, toAddr, amt)
}
