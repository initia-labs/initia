package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/math"
	"github.com/initia-labs/initia/x/move/types"

	vmtypes "github.com/initia-labs/movevm/types"
)

type BalancerMigrationKeeper struct {
	BalancerKeeper
	types.FungibleAssetKeeper
}

func NewBalancerMigrationKeeper(k *Keeper) BalancerMigrationKeeper {
	balancerKeeper := NewBalancerKeeper(k)
	moveBankKeeper := NewMoveBankKeeper(k)
	return BalancerMigrationKeeper{balancerKeeper, moveBankKeeper}
}

// HasPoolByDenom checks if a pool exists for a given denomLP.
func (k BalancerMigrationKeeper) HasPoolByDenom(ctx context.Context, denomLP string) (bool, error) {
	metadataLP, err := types.MetadataAddressFromDenom(denomLP)
	if err != nil {
		return false, err
	}

	return k.HasPool(ctx, metadataLP)
}

// ProvideLiquidity provide liquidity to a dex pair
func (k BalancerMigrationKeeper) ProvideLiquidity(ctx context.Context, provider vmtypes.AccountAddress, metadataLP vmtypes.AccountAddress, amountA, amountB math.Int) error {
	return k.ExecuteEntryFunctionJSON(
		ctx,
		provider,
		vmtypes.StdAddress,
		types.MoveModuleNameDex,
		types.FunctionNameDexProvideLiquidity,
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", metadataLP.String()),
			fmt.Sprintf("\"%s\"", amountA.String()),
			fmt.Sprintf("\"%s\"", amountB.String()),
			"null",
		},
	)
}

// WithdrawLiquidity withdraw liquidity from a dex pair
func (k BalancerMigrationKeeper) WithdrawLiquidity(ctx context.Context, provider vmtypes.AccountAddress, metadataLP vmtypes.AccountAddress, amount math.Int) error {
	return k.ExecuteEntryFunctionJSON(
		ctx,
		provider,
		vmtypes.StdAddress,
		types.MoveModuleNameDex,
		types.FunctionNameDexWithdrawLiquidity,
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", metadataLP.String()),
			fmt.Sprintf("\"%s\"", amount.String()),
			"null",
			"null",
		},
	)
}

// UpdateFeeRate update fee rate of a dex pair
func (k BalancerMigrationKeeper) UpdateFeeRate(ctx context.Context, metadataLP vmtypes.AccountAddress, feeRate math.LegacyDec) error {
	return k.ExecuteEntryFunctionJSON(
		ctx,
		vmtypes.StdAddress,
		vmtypes.StdAddress,
		types.MoveModuleNameDex,
		types.FunctionNameDexUpdateSwapFeeRate,
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", metadataLP.String()),
			fmt.Sprintf("\"%s\"", feeRate.String()),
		},
	)
}

// converts the denom in to the denom out
func (k BalancerMigrationKeeper) convert(
	ctx context.Context,
	caller vmtypes.AccountAddress,
	convertModuleAddr vmtypes.AccountAddress,
	convertModuleName string,
	metadataSwapIn vmtypes.AccountAddress,
	metadataSwapOut vmtypes.AccountAddress,
	amount math.Int,
) error {
	return k.ExecuteEntryFunctionJSON(
		ctx,
		caller,
		convertModuleAddr,
		convertModuleName,
		types.FunctionNameDexMigrationConvert,
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", metadataSwapIn.String()),
			fmt.Sprintf("\"%s\"", metadataSwapOut.String()),
			fmt.Sprintf("\"%s\"", amount.String()),
		},
	)
}

// MigrateLP migrates liquidity from one dex pair to another and returns the target LP tokens minted.
// The migration process:
// 1. Withdraws liquidity from the source LP pool (fromLP), returning base and quote tokens
// 2. Converts the withdrawn quote token (denomFrom) to target quote token (denomTo)
// 3. Temporarily sets fee rate to zero to avoid swap fees during migration
// 4. Provides liquidity to destination LP pool (toLP), minting new LP tokens
// 5. Restores the original fee rate
//
// The base token (denomBase) remains the same for both pools.
// Returns the amount of destination LP tokens minted and sent to the provider's account.
func (k BalancerMigrationKeeper) MigrateLP(
	ctx context.Context,
	provider vmtypes.AccountAddress,
	lpFrom vmtypes.AccountAddress,
	lpTo vmtypes.AccountAddress,
	convertModuleAddr vmtypes.AccountAddress,
	convertModuleName string,
	amountLpFrom math.Int,
) (math.Int, error) {
	if !amountLpFrom.IsPositive() {
		return math.ZeroInt(), errors.New("amountLpFrom must be positive")
	}

	// get pool metadata
	metadataFrom, err := k.poolMetadata(ctx, lpFrom)
	if err != nil {
		return math.ZeroInt(), err
	}
	if reverse, err := k.isReverse(ctx, lpFrom); err != nil {
		return math.ZeroInt(), err
	} else if reverse {
		metadataFrom[0], metadataFrom[1] = metadataFrom[1], metadataFrom[0]
	}
	metadataTo, err := k.poolMetadata(ctx, lpTo)
	if err != nil {
		return math.ZeroInt(), err
	}
	if reverse, err := k.isReverse(ctx, lpTo); err != nil {
		return math.ZeroInt(), err
	} else if reverse {
		metadataTo[0], metadataTo[1] = metadataTo[1], metadataTo[0]
	}

	// compute denoms from metadata
	denomFrom, err := types.DenomFromMetadataAddress(ctx, k, metadataFrom[1])
	if err != nil {
		return math.ZeroInt(), err
	}
	denomTo, err := types.DenomFromMetadataAddress(ctx, k, metadataTo[1])
	if err != nil {
		return math.ZeroInt(), err
	}
	denomLpTo, err := types.DenomFromMetadataAddress(ctx, k, lpTo)
	if err != nil {
		return math.ZeroInt(), err
	}
	denomBase, err := k.BaseDenom(ctx)
	if err != nil {
		return math.ZeroInt(), err
	}

	providerSDK := types.ConvertVMAddressToSDKAddress(provider)
	balances0 := k.bankKeeper.GetAllBalances(ctx, providerSDK)
	// Step 1: withdraw liquidity from the fromLP
	if err := k.WithdrawLiquidity(ctx, provider, lpFrom, amountLpFrom); err != nil {
		return math.ZeroInt(), err
	}

	balances1 := k.bankKeeper.GetAllBalances(ctx, providerSDK)
	amountBase := balances1.AmountOf(denomBase).Sub(balances0.AmountOf(denomBase))
	amountFrom := balances1.AmountOf(denomFrom).Sub(balances0.AmountOf(denomFrom))
	if amountBase.IsZero() || amountFrom.IsZero() {
		return math.ZeroInt(), errors.New("swap amount is zero")
	}

	// Step 2: convert fromDenom to toDenom
	err = k.convert(
		ctx,
		provider,
		convertModuleAddr,
		convertModuleName,
		metadataFrom[1],
		metadataTo[1],
		amountFrom,
	)
	if err != nil {
		return math.ZeroInt(), err
	}

	// compute amounts
	balances2 := k.bankKeeper.GetAllBalances(ctx, providerSDK)
	amountTo := balances2.AmountOf(denomTo).Sub(balances1.AmountOf(denomTo))
	amounts := []math.Int{amountBase, amountTo}
	if reverse, err := k.isReverse(ctx, lpTo); err != nil {
		return math.ZeroInt(), err
	} else if reverse {
		amounts[0], amounts[1] = amounts[1], amounts[0]
	}

	// get fee rate before migration
	feeRateBefore, err := k.poolFeeRate(ctx, lpTo)
	if err != nil {
		return math.ZeroInt(), err
	}

	// Step 3: update fee rate to zero to prevent the swap fee during migration
	if err := k.UpdateFeeRate(ctx, lpTo, math.LegacyZeroDec()); err != nil {
		return math.ZeroInt(), err
	}

	// Step 4: provide liquidity to the toLP
	if err := k.ProvideLiquidity(ctx, provider, lpTo, amounts[0], amounts[1]); err != nil {
		return math.ZeroInt(), err
	}

	// Step 5: rollback the fee rate
	if err := k.UpdateFeeRate(ctx, lpTo, feeRateBefore); err != nil {
		return math.ZeroInt(), err
	}

	balances3 := k.bankKeeper.GetAllBalances(ctx, providerSDK)
	amountLpTo := balances3.AmountOf(denomLpTo).Sub(balances2.AmountOf(denomLpTo))

	return amountLpTo, nil
}
