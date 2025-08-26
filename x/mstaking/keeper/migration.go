package keeper

import (
	"context"
	"fmt"
	"slices"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	movetypes "github.com/initia-labs/initia/x/move/types"
	"github.com/initia-labs/initia/x/mstaking/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

// RegisterMigration registers a migration of a delegation from one lp denom to another.
// Migration module requires the following function:
// - convert(account: &signer, coin_in: Object<Metadata>, coin_out: Object<Metadata>, amount: u64)
func (k Keeper) RegisterMigration(ctx context.Context, denomLpFrom string, denomLpTo string, moduleAddr string, moduleName string) error {
	hasPool, err := k.dexMigrationKeeper.HasPoolByDenom(ctx, denomLpFrom)
	if err != nil {
		return err
	} else if !hasPool {
		return fmt.Errorf("lp metadata is not found in balancer")
	}

	hasPool, err = k.dexMigrationKeeper.HasPoolByDenom(ctx, denomLpTo)
	if err != nil {
		return err
	} else if !hasPool {
		return fmt.Errorf("lp metadata is not found in balancer")
	}

	moduleAddress, err := movetypes.AccAddressFromString(k.authKeeper.AddressCodec(), moduleAddr)
	if err != nil {
		return err
	}

	// even if the migration is already registered, it will be overwritten
	err = k.Migrations.Set(ctx, collections.Join(denomLpFrom, denomLpTo), types.DelegationMigration{
		DenomLpFrom:   denomLpFrom,
		DenomLpTo:     denomLpTo,
		ModuleAddress: moduleAddress[:],
		ModuleName:    moduleName,
	})
	if err != nil {
		return err
	}
	return nil
}

// MigrateDelegation migrates a delegator's staked LP tokens from one denomination to another.
// The migration process:
// 1. Unbonds the original delegation of LP tokens
// 2. Migrates LP tokens through MigrateLP which handles withdrawing, converting and providing liquidity
// 3. Re-delegates the new LP tokens to the same validator
// Returns both the original delegation shares and the new delegation shares.
func (k Keeper) MigrateDelegation(
	ctx context.Context,
	delAddr sdk.AccAddress,
	valAddr sdk.ValAddress,
	migration types.DelegationMigration,
	newDelAddr sdk.AccAddress,
) (sdk.DecCoins, sdk.DecCoins, error) {
	delegation, err := k.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}
	validator, err := k.Validators.Get(ctx, valAddr)
	if err != nil {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}

	denomLpFrom := migration.DenomLpFrom
	metadataLpFrom, err := movetypes.MetadataAddressFromDenom(denomLpFrom)
	if err != nil {
		return nil, nil, err
	}
	denomLpTo := migration.DenomLpTo
	metadataLpTo, err := movetypes.MetadataAddressFromDenom(denomLpTo)
	if err != nil {
		return nil, nil, err
	}
	moduleAddress := vmtypes.AccountAddress(migration.ModuleAddress)
	moduleName := migration.ModuleName

	// check if the lp denom out is in the bond denoms
	bondDenoms, err := k.BondDenoms(ctx)
	if err != nil {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}
	if !slices.Contains(bondDenoms, denomLpTo) {
		return nil, nil, errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest, "invalid coin denomination: got %s, expected one of %s", denomLpTo, bondDenoms,
		)
	}

	// compute the origin shares
	originShares := sdk.NewDecCoins(sdk.NewDecCoinFromDec(denomLpFrom, delegation.Shares.AmountOf(denomLpFrom)))
	if originShares.IsZero() {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap("origin shares is zero")
	}

	// Step 1: unbond from a validator
	unbondedTokens, err := k.Unbond(ctx, delAddr, valAddr, originShares)
	if err != nil {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}

	// complete the unbonding
	if validator.IsBonded() {
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.BondedPoolName, newDelAddr, unbondedTokens)
	} else {
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.NotBondedPoolName, newDelAddr, unbondedTokens)
	}
	if err != nil {
		return nil, nil, err
	}

	// Step 2: migrate LP token from lpFrom to lpTo
	amountLpTo, err := k.dexMigrationKeeper.MigrateLP(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(newDelAddr),
		metadataLpFrom,
		metadataLpTo,
		moduleAddress,
		moduleName,
		unbondedTokens.AmountOf(denomLpFrom),
	)
	if err != nil {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}

	// need to reload the validator for the delegate
	validator, err = k.Validators.Get(ctx, valAddr)
	if err != nil {
		return nil, nil, err
	}

	// Step 3: delegate the denomLpTo
	newShares, err := k.Delegate(ctx, newDelAddr, sdk.NewCoins(sdk.NewCoin(denomLpTo, amountLpTo)), types.Unbonded, validator, true)
	return originShares, newShares, err
}
