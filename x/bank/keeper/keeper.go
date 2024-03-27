package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	cosmosbank "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	customtypes "github.com/initia-labs/initia/x/bank/types"
)

var _ cosmosbank.Keeper = (*BaseKeeper)(nil)

// BaseKeeper manages transfers between accounts. It implements the Keeper interface.
type BaseKeeper struct {
	MoveSendKeeper

	ak                     types.AccountKeeper
	cdc                    codec.BinaryCodec
	storeService           store.KVStoreService
	mintCoinsRestrictionFn types.MintingRestrictionFn
}

// GetPaginatedTotalSupply queries for the supply, ignoring 0 coins, with a given pagination
func (k BaseKeeper) GetPaginatedTotalSupply(ctx context.Context, pagination *query.PageRequest) (sdk.Coins, *query.PageResponse, error) {
	return k.mk.GetPaginatedSupply(ctx, pagination)
}

// NewBaseKeeper returns a new BaseKeeper object with a given codec, dedicated
// store key, an AccountKeeper implementation, and a parameter Subspace used to
// store and fetch module parameters. The BaseKeeper also accepts a
// blocklist map. This blocklist describes the set of addresses that are not allowed
// to receive funds through direct and explicit actions, for example, by using a MsgSend or
// by using a SendCoinsFromModuleToAccount execution.
func NewBaseKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	ak types.AccountKeeper,
	mk customtypes.MoveBankKeeper,
	blockedAddrs map[string]bool,
	authority string,
) BaseKeeper {
	if _, err := ak.AddressCodec().StringToBytes(authority); err != nil {
		panic(fmt.Errorf("invalid bank authority address: %w", err))
	}

	return BaseKeeper{
		MoveSendKeeper:         NewMoveSendKeeper(cdc, storeService, ak, mk, blockedAddrs, authority),
		ak:                     ak,
		cdc:                    cdc,
		storeService:           storeService,
		mintCoinsRestrictionFn: types.NoOpMintingRestrictionFn,
	}
}

// DO NOT USE IT
func (k BaseKeeper) WithMintCoinsRestriction(check types.MintingRestrictionFn) cosmosbank.BaseKeeper {
	panic("not supported")
}

// WithMintCoinsRestrictionV2 restricts the bank Keeper used within a specific module to
// have restricted permissions on minting via function passed in parameter.
// Previous restriction functions can be nested as such:
//
// bankKeeper.WithMintCoinsRestriction(restriction1).WithMintCoinsRestriction(restriction2)
func (k BaseKeeper) WithMintCoinsRestrictionV2(check types.MintingRestrictionFn) BaseKeeper {
	k.mintCoinsRestrictionFn = check
	return k
}

// DelegateCoins performs delegation by deducting amt coins from an account with
// address addr. The coins are then transferred from the delegator
// address to a ModuleAccount address. If any of the delegation amounts are negative,
// an error is returned.
func (k BaseKeeper) DelegateCoins(ctx context.Context, delegatorAddr, moduleAccAddr sdk.AccAddress, amt sdk.Coins) error {
	moduleAcc := k.ak.GetAccount(ctx, moduleAccAddr)
	if moduleAcc == nil {
		return errors.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s does not exist", moduleAccAddr)
	}

	if !amt.IsValid() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, amt.String())
	}

	// transfer coins to module account
	err := k.mk.SendCoins(ctx, delegatorAddr, moduleAccAddr, amt)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// emit coin spent event
	sdkCtx.EventManager().EmitEvent(
		types.NewCoinSpentEvent(delegatorAddr, amt),
	)

	// emit coin received event
	sdkCtx.EventManager().EmitEvent(
		types.NewCoinReceivedEvent(moduleAccAddr, amt),
	)

	return nil
}

// UndelegateCoins performs undelegation by crediting amt coins to an account with
// address addr. The coins are then transferred from a ModuleAccount
// address to the delegator address. If any of the undelegation amounts are
// negative, an error is returned.
func (k BaseKeeper) UndelegateCoins(ctx context.Context, moduleAccAddr, delegatorAddr sdk.AccAddress, amt sdk.Coins) error {
	moduleAcc := k.ak.GetAccount(ctx, moduleAccAddr)
	if moduleAcc == nil {
		return errors.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s does not exist", moduleAccAddr)
	}

	if !amt.IsValid() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, amt.String())
	}

	// transfer coins to delegator account
	err := k.mk.SendCoins(ctx, moduleAccAddr, delegatorAddr, amt)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// emit coin spent event
	sdkCtx.EventManager().EmitEvent(
		types.NewCoinSpentEvent(moduleAccAddr, amt),
	)

	// emit coin received event
	sdkCtx.EventManager().EmitEvent(
		types.NewCoinReceivedEvent(delegatorAddr, amt),
	)

	return nil
}

// GetSupply retrieves the Supply from store
func (k BaseKeeper) GetSupply(ctx context.Context, denom string) sdk.Coin {
	amount, err := k.mk.GetSupply(ctx, denom)
	if err != nil {
		panic(err)
	}

	return sdk.Coin{
		Denom:  denom,
		Amount: amount,
	}
}

// HasSupply checks if the supply coin exists in store.
func (k BaseKeeper) HasSupply(ctx context.Context, denom string) bool {
	ok, err := k.mk.HasSupply(ctx, denom)
	if err != nil {
		panic(err)
	}

	return ok
}

// GetDenomMetaData retrieves the denomination metadata. returns the metadata and true if the denom exists,
// false otherwise.
func (k BaseKeeper) GetDenomMetaData(ctx context.Context, denom string) (types.Metadata, bool) {
	m, err := k.MoveViewKeeper.GetDenomMetaData(ctx, denom)
	return m, err == nil
}

// HasDenomMetaData checks if the denomination metadata exists in store.
func (k BaseKeeper) HasDenomMetaData(ctx context.Context, denom string) bool {
	has, err := k.MoveViewKeeper.HasDenomMetaData(ctx, denom)
	return has && err == nil
}

// GetAllDenomMetaData retrieves all denominations metadata
func (k BaseKeeper) GetAllDenomMetaData(ctx context.Context) []types.Metadata {
	denomMetaData := make([]types.Metadata, 0)
	k.IterateAllDenomMetaData(ctx, func(metadata types.Metadata) bool {
		denomMetaData = append(denomMetaData, metadata)
		return false
	})

	return denomMetaData
}

// IterateAllDenomMetaData iterates over all the denominations metadata and
// provides the metadata to a callback. If true is returned from the
// callback, iteration is halted.
func (k BaseKeeper) IterateAllDenomMetaData(ctx context.Context, cb func(types.Metadata) bool) {
	err := k.MoveViewKeeper.DenomMetadata.Walk(ctx, nil, func(_ string, metadata types.Metadata) (stop bool, err error) {
		return cb(metadata), nil
	})
	if err != nil {
		panic(err)
	}
}

// SetDenomMetaData sets the denominations metadata
func (k BaseKeeper) SetDenomMetaData(ctx context.Context, denomMetaData types.Metadata) {
	_ = k.MoveViewKeeper.DenomMetadata.Set(ctx, denomMetaData.Base, denomMetaData)
}

// SendCoinsFromModuleToAccount transfers coins from a ModuleAccount to an AccAddress.
// It will panic if the module account does not exist. An error is returned if
// the recipient address is black-listed or if sending the tokens fails.
func (k BaseKeeper) SendCoinsFromModuleToAccount(
	ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins,
) error {
	senderAddr := k.ak.GetModuleAddress(senderModule)
	if senderAddr == nil {
		panic(errors.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s does not exist", senderModule))
	}

	if k.BlockedAddr(recipientAddr) {
		return errors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive funds", recipientAddr)
	}

	return k.SendCoins(ctx, senderAddr, recipientAddr, amt)
}

// SendCoinsFromModuleToModule transfers coins from a ModuleAccount to another.
// It will panic if either module account does not exist.
func (k BaseKeeper) SendCoinsFromModuleToModule(
	ctx context.Context, senderModule, recipientModule string, amt sdk.Coins,
) error {
	senderAddr := k.ak.GetModuleAddress(senderModule)
	if senderAddr == nil {
		panic(errors.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s does not exist", senderModule))
	}

	recipientAcc := k.ak.GetModuleAccount(ctx, recipientModule)
	if recipientAcc == nil {
		panic(errors.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s does not exist", recipientModule))
	}

	return k.SendCoins(ctx, senderAddr, recipientAcc.GetAddress(), amt)
}

// SendCoinsFromAccountToModule transfers coins from an AccAddress to a ModuleAccount.
// It will panic if the module account does not exist.
func (k BaseKeeper) SendCoinsFromAccountToModule(
	ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins,
) error {
	recipientAcc := k.ak.GetModuleAccount(ctx, recipientModule)
	if recipientAcc == nil {
		panic(errors.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s does not exist", recipientModule))
	}

	return k.SendCoins(ctx, senderAddr, recipientAcc.GetAddress(), amt)
}

// DelegateCoinsFromAccountToModule delegates coins and transfers them from a
// delegator account to a module account. It will panic if the module account
// does not exist or is unauthorized.
func (k BaseKeeper) DelegateCoinsFromAccountToModule(
	ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins,
) error {
	recipientAcc := k.ak.GetModuleAccount(ctx, recipientModule)
	if recipientAcc == nil {
		panic(errors.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s does not exist", recipientModule))
	}

	if !recipientAcc.HasPermission(authtypes.Staking) {
		panic(errors.Wrapf(sdkerrors.ErrUnauthorized, "module account %s does not have permissions to receive delegated coins", recipientModule))
	}

	return k.DelegateCoins(ctx, senderAddr, recipientAcc.GetAddress(), amt)
}

// UndelegateCoinsFromModuleToAccount undelegates the unbonding coins and transfers
// them from a module account to the delegator account. It will panic if the
// module account does not exist or is unauthorized.
func (k BaseKeeper) UndelegateCoinsFromModuleToAccount(
	ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins,
) error {
	acc := k.ak.GetModuleAccount(ctx, senderModule)
	if acc == nil {
		panic(errors.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s does not exist", senderModule))
	}

	if !acc.HasPermission(authtypes.Staking) {
		panic(errors.Wrapf(sdkerrors.ErrUnauthorized, "module account %s does not have permissions to undelegate coins", senderModule))
	}

	return k.UndelegateCoins(ctx, acc.GetAddress(), recipientAddr, amt)
}

// MintCoins creates new coins from thin air and adds it to the module account.
// It will panic if the module account does not exist or is unauthorized.
func (k BaseKeeper) MintCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error {
	err := k.mintCoinsRestrictionFn(ctx, amounts)
	if err != nil {
		k.Logger(ctx).Error(fmt.Sprintf("Module %q attempted to mint coins %s it doesn't have permission for, error %v", moduleName, amounts, err))
		return err
	}
	acc := k.ak.GetModuleAccount(ctx, moduleName)
	if acc == nil {
		panic(errors.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s does not exist", moduleName))
	}

	if !acc.HasPermission(authtypes.Minter) {
		panic(errors.Wrapf(sdkerrors.ErrUnauthorized, "module account %s does not have permissions to mint tokens", moduleName))
	}

	err = k.mk.MintCoins(ctx, acc.GetAddress(), amounts)
	if err != nil {
		return err
	}

	logger := k.Logger(ctx)
	logger.Info("minted coins from module account", "amount", amounts.String(), "from", moduleName)

	// emit mint event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		types.NewCoinMintEvent(acc.GetAddress(), amounts),
	)

	return nil
}

// BurnCoins burns coins deletes coins from the balance of the module account.
// It will panic if the module account does not exist or is unauthorized.
func (k BaseKeeper) BurnCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error {
	acc := k.ak.GetModuleAccount(ctx, moduleName)
	if acc == nil {
		panic(errors.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s does not exist", moduleName))
	}

	if !acc.HasPermission(authtypes.Burner) {
		panic(errors.Wrapf(sdkerrors.ErrUnauthorized, "module account %s does not have permissions to burn tokens", moduleName))
	}

	err := k.mk.BurnCoins(ctx, acc.GetAddress(), amounts)
	if err != nil {
		return err
	}

	logger := k.Logger(ctx)
	logger.Info("burned tokens from module account", "amount", amounts.String(), "from", moduleName)

	// emit burn event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		types.NewCoinBurnEvent(acc.GetAddress(), amounts),
	)

	return nil
}

// IterateTotalSupply iterates over the total supply calling the given cb (callback) function
// with the balance of each coin.
// The iteration stops if the callback returns true.
func (k MoveViewKeeper) IterateTotalSupply(ctx context.Context, cb func(sdk.Coin) bool) {
	err := k.mk.IterateSupply(ctx, func(supply sdk.Coin) (bool, error) {
		return cb(supply), nil
	})
	if err != nil {
		panic(err)
	}
}
