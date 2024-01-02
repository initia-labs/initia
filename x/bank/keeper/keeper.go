package keeper

import (
	"fmt"

	"cosmossdk.io/errors"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	cosmosbank "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	customtypes "github.com/initia-labs/initia/x/bank/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

var _ cosmosbank.Keeper = (*BaseKeeper)(nil)

// BaseKeeper manages transfers between accounts. It implements the Keeper interface.
type BaseKeeper struct {
	MoveSendKeeper

	ak                     types.AccountKeeper
	cdc                    codec.BinaryCodec
	storeKey               storetypes.StoreKey
	mintCoinsRestrictionFn cosmosbank.MintingRestrictionFn
}

// GetPaginatedTotalSupply queries for the supply, ignoring 0 coins, with a given pagination
func (k BaseKeeper) GetPaginatedTotalSupply(ctx sdk.Context, pagination *query.PageRequest) (sdk.Coins, *query.PageResponse, error) {
	issuers, err := k.mk.GetIssuers(ctx)
	if err != nil {
		return nil, nil, err
	}

	supply := sdk.NewCoins()

	pageRes, err := query.Paginate(issuers, pagination, func(key, value []byte) error {
		metadata, err := vmtypes.NewAccountAddressFromBytes(key)
		if err != nil {
			return err
		}

		denom, err := movetypes.DenomFromMetadataAddress(ctx, k.mk, metadata)
		if err != nil {
			return err
		}

		amount, err := k.mk.GetSupply(ctx, denom)
		if err != nil {
			return err
		}
		if amount.IsZero() {
			return nil
		}

		balance := sdk.Coin{
			Denom:  denom,
			Amount: amount,
		}

		// `Add` omits the 0 coins addition to the `supply`.
		supply = supply.Add(balance)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return supply, pageRes, nil
}

// NewBaseKeeper returns a new BaseKeeper object with a given codec, dedicated
// store key, an AccountKeeper implementation, and a parameter Subspace used to
// store and fetch module parameters. The BaseKeeper also accepts a
// blocklist map. This blocklist describes the set of addresses that are not allowed
// to receive funds through direct and explicit actions, for example, by using a MsgSend or
// by using a SendCoinsFromModuleToAccount execution.
func NewBaseKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	ak types.AccountKeeper,
	mk customtypes.MoveBankKeeper,
	blockedAddrs map[string]bool,
	authority string,
) BaseKeeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Errorf("invalid bank authority address: %w", err))
	}

	return BaseKeeper{
		MoveSendKeeper:         NewMoveSendKeeper(cdc, storeKey, ak, mk, blockedAddrs, authority),
		ak:                     ak,
		cdc:                    cdc,
		storeKey:               storeKey,
		mintCoinsRestrictionFn: func(ctx sdk.Context, coins sdk.Coins) error { return nil },
	}
}

// WithMintCoinsRestriction deprecated please use WithMintCoinsRestrictionV2
func (k BaseKeeper) WithMintCoinsRestriction(check cosmosbank.MintingRestrictionFn) cosmosbank.BaseKeeper {
	panic("not supported")
}

// WithMintCoinsRestrictionV2 restricts the bank Keeper used within a specific module to
// have restricted permissions on minting via function passed in parameter.
// Previous restriction functions can be nested as such:
//
//	bankKeeper.WithMintCoinsRestrictionV2(restriction1).WithMintCoinsRestrictionV2(restriction2)
func (k BaseKeeper) WithMintCoinsRestrictionV2(check cosmosbank.MintingRestrictionFn) cosmosbank.Keeper {
	oldRestrictionFn := k.mintCoinsRestrictionFn
	k.mintCoinsRestrictionFn = func(ctx sdk.Context, coins sdk.Coins) error {
		err := check(ctx, coins)
		if err != nil {
			return err
		}
		err = oldRestrictionFn(ctx, coins)
		if err != nil {
			return err
		}
		return nil
	}
	return k
}

// DelegateCoins performs delegation by deducting amt coins from an account with
// address addr. The coins are then transferred from the delegator
// address to a ModuleAccount address. If any of the delegation amounts are negative,
// an error is returned.
func (k BaseKeeper) DelegateCoins(ctx sdk.Context, delegatorAddr, moduleAccAddr sdk.AccAddress, amt sdk.Coins) error {
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

	// emit coin spent event
	ctx.EventManager().EmitEvent(
		types.NewCoinSpentEvent(delegatorAddr, amt),
	)

	// emit coin received event
	ctx.EventManager().EmitEvent(
		types.NewCoinReceivedEvent(moduleAccAddr, amt),
	)

	return nil
}

// UndelegateCoins performs undelegation by crediting amt coins to an account with
// address addr. The coins are then transferred from a ModuleAccount
// address to the delegator address. If any of the undelegation amounts are
// negative, an error is returned.
func (k BaseKeeper) UndelegateCoins(ctx sdk.Context, moduleAccAddr, delegatorAddr sdk.AccAddress, amt sdk.Coins) error {
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

	// emit coin spent event
	ctx.EventManager().EmitEvent(
		types.NewCoinSpentEvent(moduleAccAddr, amt),
	)

	// emit coin received event
	ctx.EventManager().EmitEvent(
		types.NewCoinReceivedEvent(delegatorAddr, amt),
	)

	return nil
}

// GetSupply retrieves the Supply from store
func (k BaseKeeper) GetSupply(ctx sdk.Context, denom string) sdk.Coin {
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
func (k BaseKeeper) HasSupply(ctx sdk.Context, denom string) bool {
	store := ctx.KVStore(k.storeKey)
	supplyStore := prefix.NewStore(store, types.SupplyKey)
	return supplyStore.Has([]byte(denom))
}

// GetDenomMetaData retrieves the denomination metadata. returns the metadata and true if the denom exists,
// false otherwise.
func (k BaseKeeper) GetDenomMetaData(ctx sdk.Context, denom string) (types.Metadata, bool) {
	store := ctx.KVStore(k.storeKey)
	store = prefix.NewStore(store, types.DenomMetadataPrefix)

	bz := store.Get([]byte(denom))
	if bz == nil {
		return types.Metadata{}, false
	}

	var metadata types.Metadata
	k.cdc.MustUnmarshal(bz, &metadata)

	return metadata, true
}

// HasDenomMetaData checks if the denomination metadata exists in store.
func (k BaseKeeper) HasDenomMetaData(ctx sdk.Context, denom string) bool {
	store := ctx.KVStore(k.storeKey)
	store = prefix.NewStore(store, types.DenomMetadataPrefix)
	return store.Has([]byte(denom))
}

// GetAllDenomMetaData retrieves all denominations metadata
func (k BaseKeeper) GetAllDenomMetaData(ctx sdk.Context) []types.Metadata {
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
func (k BaseKeeper) IterateAllDenomMetaData(ctx sdk.Context, cb func(types.Metadata) bool) {
	store := ctx.KVStore(k.storeKey)
	denomMetaDataStore := prefix.NewStore(store, types.DenomMetadataPrefix)

	iterator := denomMetaDataStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var metadata types.Metadata
		k.cdc.MustUnmarshal(iterator.Value(), &metadata)

		if cb(metadata) {
			break
		}
	}
}

// SetDenomMetaData sets the denominations metadata
func (k BaseKeeper) SetDenomMetaData(ctx sdk.Context, denomMetaData types.Metadata) {
	store := ctx.KVStore(k.storeKey)
	denomMetaDataStore := prefix.NewStore(store, types.DenomMetadataPrefix)

	m := k.cdc.MustMarshal(&denomMetaData)
	denomMetaDataStore.Set([]byte(denomMetaData.Base), m)
}

// SendCoinsFromModuleToAccount transfers coins from a ModuleAccount to an AccAddress.
// It will panic if the module account does not exist. An error is returned if
// the recipient address is black-listed or if sending the tokens fails.
func (k BaseKeeper) SendCoinsFromModuleToAccount(
	ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins,
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
	ctx sdk.Context, senderModule, recipientModule string, amt sdk.Coins,
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
	ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins,
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
	ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins,
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
	ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins,
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
func (k BaseKeeper) MintCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error {
	err := k.mintCoinsRestrictionFn(ctx, amounts)
	if err != nil {
		ctx.Logger().Error(fmt.Sprintf("Module %q attempted to mint coins %s it doesn't have permission for, error %v", moduleName, amounts, err))
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
	ctx.EventManager().EmitEvent(
		types.NewCoinMintEvent(acc.GetAddress(), amounts),
	)

	return nil
}

// BurnCoins burns coins deletes coins from the balance of the module account.
// It will panic if the module account does not exist or is unauthorized.
func (k BaseKeeper) BurnCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error {
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
	ctx.EventManager().EmitEvent(
		types.NewCoinBurnEvent(acc.GetAddress(), amounts),
	)

	return nil
}

// IterateTotalSupply iterates over the total supply calling the given cb (callback) function
// with the balance of each coin.
// The iteration stops if the callback returns true.
func (k MoveViewKeeper) IterateTotalSupply(ctx sdk.Context, cb func(sdk.Coin) bool) {
	issuers, err := k.mk.GetIssuers(ctx)
	if err != nil {
		panic(err)
	}

	iterator := issuers.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		metadata, err := vmtypes.NewAccountAddressFromBytes(key)
		if err != nil {
			panic(err)
		}
		denom, err := movetypes.DenomFromMetadataAddress(ctx, k.mk, metadata)
		if err != nil {
			panic(err)
		}

		amount, err := k.mk.GetSupply(ctx, denom)
		if err != nil {
			panic(err)
		}
		if amount.IsZero() {
			continue
		}

		balance := sdk.Coin{
			Denom:  denom,
			Amount: amount,
		}
		if cb(balance) {
			break
		}
	}
}
