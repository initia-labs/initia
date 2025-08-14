package keeper

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	movetypes "github.com/initia-labs/initia/x/move/types"
	"github.com/initia-labs/initia/x/mstaking/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

// RegisterMigration registers a migration of a delegation from one lp denom to another.
// Swap contract requires the following function:
// - swap(account: &signer, coin_in: Object<Metadata>, coin_out: Object<Metadata>, amount: u64)
func (k Keeper) RegisterMigration(ctx context.Context, lpDenomIn string, lpDenomOut string, denomIn string, denomOut string, swapContractStr string) error {
	lpMetadataIn, err := movetypes.MetadataAddressFromDenom(lpDenomIn)
	if err != nil {
		return err
	}

	hasPool, err := k.balancerKeeper.HasPool(ctx, lpMetadataIn)
	if err != nil {
		return err
	} else if !hasPool {
		return fmt.Errorf("lp metadata is not found in balancer")
	}

	lpMetadataOut, err := movetypes.MetadataAddressFromDenom(lpDenomOut)
	if err != nil {
		return err
	}

	hasPool, err = k.balancerKeeper.HasPool(ctx, lpMetadataOut)
	if err != nil {
		return err
	} else if !hasPool {
		return fmt.Errorf("lp metadata is not found in balancer")
	}

	swapContract := strings.Split(swapContractStr, "::")
	if len(swapContract) != 2 {
		return fmt.Errorf("invalid swap contract address: %s, expected format: <module_addr>::<module_name>", swapContractStr)
	}

	swapContractModuleAddress, err := movetypes.AccAddressFromString(k.authKeeper.AddressCodec(), swapContract[0])
	if err != nil {
		return err
	}

	// even if the migration is already registered, it will be overwritten
	err = k.Migrations.Set(ctx, collections.Join(lpMetadataIn[:], lpMetadataOut[:]), types.DelegationMigration{
		DenomIn:                   denomIn,
		DenomOut:                  denomOut,
		LpDenomIn:                 lpDenomIn,
		LpDenomOut:                lpDenomOut,
		SwapContractModuleAddress: swapContractModuleAddress[:],
		SwapContractModuleName:    swapContract[1],
	})
	if err != nil {
		return err
	}
	return nil
}

// MigrateDelegation migrates a delegator's staked LP tokens from one denomination to another.
// The migration process:
// 1. Unbonds the original delegation of LP tokens
// 2. Withdraws liquidity from the source DEX pool to get the underlying tokens
// 3. Swaps the underlying tokens through the registered swap contract
// 4. Provides liquidity to the target DEX pool to get new LP tokens
// 5. Re-delegates the new LP tokens to the same validator
// Returns both the original delegation shares and the new delegation shares.
func (k Keeper) MigrateDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, migration types.DelegationMigration) (sdk.DecCoins, sdk.DecCoins, error) {
	delegation, err := k.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}
	validator, err := k.Validators.Get(ctx, valAddr)
	if err != nil {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}

	lpDenomIn := migration.LpDenomIn
	lpMetadataIn, err := movetypes.MetadataAddressFromDenom(lpDenomIn)
	if err != nil {
		return nil, nil, err
	}
	denomSwapIn := migration.DenomIn
	metadataSwapIn, err := movetypes.MetadataAddressFromDenom(denomSwapIn)
	if err != nil {
		return nil, nil, err
	}
	lpDenomOut := migration.LpDenomOut
	lpMetadataOut, err := movetypes.MetadataAddressFromDenom(lpDenomOut)
	if err != nil {
		return nil, nil, err
	}
	denomSwapOut := migration.DenomOut
	metadataSwapOut, err := movetypes.MetadataAddressFromDenom(denomSwapOut)
	if err != nil {
		return nil, nil, err
	}
	swapContractModuleAddress := vmtypes.AccountAddress(migration.SwapContractModuleAddress)
	swapContractModuleName := migration.SwapContractModuleName

	// check if the lp denom out is in the bond denoms
	bondDenoms, err := k.BondDenoms(ctx)
	if err != nil {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}
	if !slices.Contains(bondDenoms, lpDenomOut) {
		return nil, nil, errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest, "invalid coin denomination: got %s, expected one of %s", lpDenomOut, bondDenoms,
		)
	}

	// compute the origin shares
	originShares := sdk.NewDecCoins(sdk.NewDecCoinFromDec(lpDenomIn, delegation.Shares.AmountOf(lpDenomIn)))
	if originShares.IsZero() {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap("origin shares is zero")
	}

	// unbond from a validator
	returnCoins, err := k.Unbond(ctx, delAddr, valAddr, originShares)
	if err != nil {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}

	// complete the unbonding
	if validator.IsBonded() {
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.BondedPoolName, delAddr, returnCoins)
	} else {
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.NotBondedPoolName, delAddr, returnCoins)
	}
	if err != nil {
		return nil, nil, err
	}

	// compute fixed counterparty token metadata and denom
	metadatas, err := k.balancerKeeper.PoolMetadata(ctx, lpMetadataIn)
	if err != nil {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}
	if len(metadatas) != 2 {
		return nil, nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid pool metadata")
	}
	baseMetadata := metadatas[0]
	if metadatas[0].Equals(metadataSwapIn) {
		baseMetadata = metadatas[1]
	} else if !metadatas[1].Equals(metadataSwapIn) {
		return nil, nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid pool metadata")
	}
	baseDenom, err := movetypes.DenomFromMetadataAddress(ctx, k.fungibleAssetKeeper, baseMetadata)
	if err != nil {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}

	// Step 0: initialize the balances
	balances0 := k.bankKeeper.GetAllBalances(ctx, delAddr)
	lpInBalance0 := balances0.AmountOf(lpDenomIn)
	baseDenomBalance0 := balances0.AmountOf(baseDenom)
	denomSwapInBalance0 := balances0.AmountOf(denomSwapIn)

	// Step 1: withdraw liquidity from the dex pool
	withdrawAmount := returnCoins.AmountOf(lpDenomIn)
	if err := k.balancerKeeper.WithdrawLiquidity(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(delAddr),
		lpMetadataIn,
		withdrawAmount,
	); err != nil {
		return nil, nil, err
	}

	balances1 := k.bankKeeper.GetAllBalances(ctx, delAddr)
	lpInBalance1 := balances1.AmountOf(lpDenomIn)
	baseDenomBalance1 := balances1.AmountOf(baseDenom)
	denomSwapInBalance1 := balances1.AmountOf(denomSwapIn)
	denomSwapOutBalance1 := balances1.AmountOf(migration.DenomOut)

	// check if the lp in balance is expected
	if !lpInBalance0.Sub(lpInBalance1).Equal(withdrawAmount) {
		return nil, nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "lp withdraw spend unexpected amount")
	}

	// Step 2: swap the denom in to the denom out
	swapAmount := denomSwapInBalance1.Sub(denomSwapInBalance0)
	if swapAmount.IsZero() {
		return nil, nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "swap amount is zero")
	}
	err = k.moveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(delAddr),
		vmtypes.AccountAddress(swapContractModuleAddress),
		swapContractModuleName,
		movetypes.FunctionNameMigrateDelegationSwapContractSwap,
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", metadataSwapIn.String()),
			fmt.Sprintf("\"%s\"", metadataSwapOut.String()),
			fmt.Sprintf("\"%s\"", swapAmount.String()),
		},
	)
	if err != nil {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}
	balances2 := k.bankKeeper.GetAllBalances(ctx, delAddr)
	denomSwapInBalance2 := balances2.AmountOf(denomSwapIn)
	denomSwapOutBalance2 := balances2.AmountOf(migration.DenomOut)

	// swap contract should convert same amount of denom in to denom out
	if !denomSwapInBalance1.Sub(denomSwapInBalance2).Equal(denomSwapOutBalance2.Sub(denomSwapOutBalance1)) {
		return nil, nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "swap contract did not convert equal amounts between input and output denominations")
	}

	// Step 3: update the swap fee rate to zero to prevent the swap fee during migration
	feeRateBefore, err := k.balancerKeeper.PoolFeeRate(ctx, lpMetadataOut)
	if err != nil {
		return nil, nil, err
	}
	if err := k.balancerKeeper.UpdateFeeRate(ctx, lpMetadataOut, math.LegacyZeroDec()); err != nil {
		return nil, nil, err
	}

	// Step 4: provide liquidity to the dex pool
	coinAAmount := baseDenomBalance1.Sub(baseDenomBalance0)
	coinBAmount := denomSwapOutBalance2.Sub(denomSwapOutBalance1)
	poolMetadata, err := k.balancerKeeper.PoolMetadata(ctx, lpMetadataOut)
	if err != nil {
		return nil, nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}
	if poolMetadata[0].Equals(metadataSwapOut) && poolMetadata[1].Equals(baseMetadata) {
		coinAAmount, coinBAmount = coinBAmount, coinAAmount
	} else if !poolMetadata[0].Equals(baseMetadata) || !poolMetadata[1].Equals(metadataSwapOut) {
		return nil, nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, fmt.Sprintf("pool metadata mismatch: expected (%s, %s), got (%s, %s)", baseMetadata, metadataSwapOut, poolMetadata[0], poolMetadata[1]))
	}
	if err := k.balancerKeeper.ProvideLiquidity(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(delAddr),
		lpMetadataOut,
		coinAAmount,
		coinBAmount,
	); err != nil {
		return nil, nil, err
	}

	// Step 5: rollback the swap fee rate
	if err := k.balancerKeeper.UpdateFeeRate(ctx, lpMetadataOut, feeRateBefore); err != nil {
		return nil, nil, err
	}

	// need to reload the validator for the delegate
	validator, err = k.Validators.Get(ctx, valAddr)
	if err != nil {
		return nil, nil, err
	}

	// Step 6: delegate the lp denom out
	lpDenomOutBalance := k.bankKeeper.GetBalance(ctx, delAddr, lpDenomOut)
	newShares, err := k.Delegate(ctx, delAddr, sdk.NewCoins(lpDenomOutBalance), types.Unbonded, validator, true)
	return originShares, newShares, err
}
