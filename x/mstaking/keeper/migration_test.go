package keeper_test

import (
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	"github.com/initia-labs/initia/x/mstaking/types"
	vmtypes "github.com/initia-labs/movevm/types"
	"github.com/stretchr/testify/require"
)

func Test_RegisterMigration(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	dexMigrationModule := ReadMoveFile("dex_migration")
	err := input.MoveKeeper.PublishModuleBundle(ctx, movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr), vmtypes.NewModuleBundle(vmtypes.Module{Code: dexMigrationModule}), movetypes.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	baseDenom := bondDenom
	metadataLPOld := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), true)
	metadataLPNew := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 100_000_000), sdk.NewInt64Coin("uusdc2", 250_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), true)

	denomLpOld, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPOld)
	require.NoError(t, err)
	denomLpNew, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPNew)
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewInt64Coin("uusdc2", 2_500_000_000))

	err = input.StakingKeeper.RegisterMigration(ctx, denomLpOld, denomLpNew, "invalid_module", "dex_migration")
	require.Error(t, err)

	err = input.StakingKeeper.RegisterMigration(ctx, denomLpOld, denomLpNew, movetypes.TestAddr.String(), "dex_migration")
	require.NoError(t, err)
}

func Test_RegisterMigration_Validation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// Create DEX pools for testing
	baseDenom := bondDenom
	metadataLP1 := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), true)
	metadataLP2 := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 100_000_000), sdk.NewInt64Coin("uusdc2", 250_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), true)

	lpDenom1, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP1)
	require.NoError(t, err)
	lpDenom2, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP2)
	require.NoError(t, err)

	// Test case 1: Register migration with invalid module format
	t.Run("InvalidModuleFormat", func(t *testing.T) {
		// Test with invalid module address
		err := input.StakingKeeper.RegisterMigration(ctx, lpDenom1, lpDenom2, "invalid_format", "dex_migration")
		require.Error(t, err)
		require.Contains(t, err.Error(), "decoding bech32 failed")
	})

	// Test case 2: Register migration with non-existent LP denoms
	t.Run("NonExistentLPDenoms", func(t *testing.T) {
		err := input.StakingKeeper.RegisterMigration(ctx, "non_existent_lp", "non_existent_lp_out", movetypes.TestAddr.String(), "dex_migration")
		require.Error(t, err)
		require.Contains(t, err.Error(), "lp metadata is not found in balancer")
	})

}

func Test_MigrateDelegation_CompleteFlow(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// Setup: Create DEX pools and publish migration module
	dexMigrationModule := ReadMoveFile("dex_migration")
	err := input.MoveKeeper.PublishModuleBundle(ctx, movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr), vmtypes.NewModuleBundle(vmtypes.Module{Code: dexMigrationModule}), movetypes.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	// Create DEX pools
	baseDenom := bondDenom
	metadataLPOld := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), false)
	metadataLPNew := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 100_000_000), sdk.NewInt64Coin("uusdc2", 250_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), false)

	denomLpOld, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPOld)
	require.NoError(t, err)
	denomLpNew, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPNew)
	require.NoError(t, err)

	// Initialize dex migration contract
	metadataUusdc, err := movetypes.MetadataAddressFromDenom("uusdc")
	require.NoError(t, err)
	metadataUusdc2, err := movetypes.MetadataAddressFromDenom("uusdc2")
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr),
		movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr),
		"dex_migration",
		"initialize",
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", metadataUusdc.String()),
			fmt.Sprintf("\"%s\"", metadataUusdc2.String()),
		},
	)
	require.NoError(t, err)

	// Fund account for liquidity provision
	fundedAccount := input.Faucet.NewFundedAccount(ctx, sdk.NewInt64Coin("uusdc2", 500_000_000))
	err = input.MoveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(fundedAccount),
		movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr),
		"dex_migration",
		"provide_liquidity",
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", metadataUusdc2.String()),
			"\"500000000\"",
		},
	)
	require.NoError(t, err)

	// Update params to include LP denoms
	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.BondDenoms = append(params.BondDenoms, denomLpOld, denomLpNew)
	require.NoError(t, input.StakingKeeper.SetParams(ctx, params))

	// Create validator
	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)

	// Create delegator and provide liquidity
	delAddr := input.Faucet.NewFundedAccount(ctx, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000))

	err = input.MoveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(delAddr),
		movetypes.ConvertSDKAddressToVMAddress(movetypes.StdAddr),
		movetypes.MoveModuleNameDex,
		movetypes.FunctionNameDexProvideLiquidity,
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", metadataLPOld.String()),
			"\"1000000\"",
			"\"2500000\"",
			"null",
		},
	)
	require.NoError(t, err)

	balanceLpOld := input.BankKeeper.GetBalance(ctx, delAddr, denomLpOld)

	// Whitelist LP denom (only if not already whitelisted)
	err = input.MoveKeeper.Whitelist(ctx, movetypes.MsgWhitelist{
		MetadataLP:   metadataLPOld.String(),
		RewardWeight: math.LegacyNewDecWithPrec(1, 1),
	})
	if err != nil && !strings.Contains(err.Error(), "was already registered") {
		require.NoError(t, err)
	}

	// Delegate LP tokens
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)
	_, err = input.StakingKeeper.Delegate(ctx, delAddr, sdk.NewCoins(balanceLpOld), types.Unbonded, validator, true)
	require.NoError(t, err)

	// Test case 1: Successful migration flow
	t.Run("SuccessfulMigration", func(t *testing.T) {
		// Register migration
		err = input.StakingKeeper.RegisterMigration(ctx, denomLpOld, denomLpNew, movetypes.TestAddr.String(), "dex_migration")
		require.NoError(t, err)

		// Whitelist target LP denom (only if not already whitelisted)
		err = input.MoveKeeper.Whitelist(ctx, movetypes.MsgWhitelist{
			MetadataLP:   metadataLPNew.String(),
			RewardWeight: math.LegacyNewDecWithPrec(1, 1),
		})
		if err != nil && !strings.Contains(err.Error(), "was already registered") {
			require.NoError(t, err)
		}

		// Get migration info
		migration, err := input.StakingKeeper.Migrations.Get(ctx, collections.Join(denomLpOld, denomLpNew))
		require.NoError(t, err)

		// Execute migration
		originShares, newShares, err := input.StakingKeeper.MigrateDelegation(ctx, delAddr, valAddr, migration)
		require.NoError(t, err)

		// Verify migration results
		require.NotNil(t, originShares)
		require.NotNil(t, newShares)
		require.False(t, originShares.IsZero())
		require.False(t, newShares.IsZero())

		// Verify delegation was updated
		delegation, err := input.StakingKeeper.GetDelegation(ctx, delAddr, valAddr)
		require.NoError(t, err)
		require.Equal(t, newShares, delegation.Shares)
	})

	// Test case 2: Migration with insufficient balance
	t.Run("InsufficientBalance", func(t *testing.T) {
		// Try to migrate more than available by creating a migration with non-existent pool
		migrationAddr, err := vmtypes.NewAccountAddress("2")
		require.NoError(t, err)

		largeMigration := types.DelegationMigration{
			DenomLpFrom:   "non_existent_pool",
			DenomLpTo:     denomLpNew,
			ModuleAddress: migrationAddr[:],
			ModuleName:    "dex_migration",
		}

		_, _, err = input.StakingKeeper.MigrateDelegation(ctx, delAddr, valAddr, largeMigration)
		require.Error(t, err)
	})

	// Test case 3: Migration with invalid pool metadata
	t.Run("InvalidPoolMetadata", func(t *testing.T) {
		// Create a migration with non-existent pool
		migrationAddr, err := vmtypes.NewAccountAddress("2")
		require.NoError(t, err)

		invalidMigration := types.DelegationMigration{
			DenomLpFrom:   "non_existent_pool",
			DenomLpTo:     denomLpNew,
			ModuleAddress: migrationAddr[:],
			ModuleName:    "dex_migration",
		}

		_, _, err = input.StakingKeeper.MigrateDelegation(ctx, delAddr, valAddr, invalidMigration)
		require.Error(t, err)
	})
}

func Test_MigrateDelegation_StateConsistency(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// Create DEX pools for testing
	baseDenom := bondDenom
	metadataLP1 := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), true)
	metadataLP2 := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 100_000_000), sdk.NewInt64Coin("uusdc2", 250_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), true)

	lpDenom1, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP1)
	require.NoError(t, err)
	lpDenom2, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLP2)
	require.NoError(t, err)

	// Test case 1: Verify migration registration state
	t.Run("MigrationRegistrationState", func(t *testing.T) {
		// Register a migration
		err := input.StakingKeeper.RegisterMigration(ctx, lpDenom1, lpDenom2, movetypes.TestAddr.String(), "dex_migration")
		require.NoError(t, err)

		// Verify migration was stored
		migration, err := input.StakingKeeper.Migrations.Get(ctx, collections.Join(lpDenom1, lpDenom2))
		require.NoError(t, err)
		require.Equal(t, lpDenom1, migration.DenomLpFrom)
		require.Equal(t, lpDenom2, migration.DenomLpTo)
		require.NotEmpty(t, migration.ModuleAddress)
		require.Equal(t, "dex_migration", migration.ModuleName)
	})

	// Test case 2: Verify migration overwrite behavior
	t.Run("MigrationOverwrite", func(t *testing.T) {
		// Register migration with same LP denom but different target
		err := input.StakingKeeper.RegisterMigration(ctx, lpDenom1, lpDenom2, movetypes.TestAddr.String(), "dex_migration")
		require.NoError(t, err)

		// Verify migration was overwritten
		migration, err := input.StakingKeeper.Migrations.Get(ctx, collections.Join(lpDenom1, lpDenom2))
		require.NoError(t, err)
		require.Equal(t, lpDenom2, migration.DenomLpTo)
		require.NotEmpty(t, migration.ModuleAddress)
		require.Equal(t, "dex_migration", migration.ModuleName)
	})

	// Test case 3: Verify migration cleanup
	t.Run("MigrationCleanup", func(t *testing.T) {
		// Register a migration
		err := input.StakingKeeper.RegisterMigration(ctx, lpDenom1, lpDenom2, movetypes.TestAddr.String(), "dex_migration")
		require.NoError(t, err)

		// Verify migration exists
		_, err = input.StakingKeeper.Migrations.Get(ctx, collections.Join(lpDenom1, lpDenom2))
		require.NoError(t, err)

		// Remove the migration (simulating cleanup)
		err = input.StakingKeeper.Migrations.Remove(ctx, collections.Join(lpDenom1, lpDenom2))
		require.NoError(t, err)

		// Verify migration was removed
		_, err = input.StakingKeeper.Migrations.Get(ctx, collections.Join(lpDenom1, lpDenom2))
		require.Error(t, err)
	})
}

func Test_MigrateDelegation_ErrorHandling(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// Test case 1: Migration with non-existent delegation
	t.Run("NonExistentDelegation", func(t *testing.T) {
		_, _, err := input.StakingKeeper.MigrateDelegation(ctx, addrs[0], valAddrs[0], types.DelegationMigration{
			DenomLpFrom:   "test_lp",
			DenomLpTo:     "test_lp_out",
			ModuleAddress: []byte("0x2"),
			ModuleName:    "dex_migration",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid request")
	})

	// Test case 2: Migration with non-existent validator
	t.Run("NonExistentValidator", func(t *testing.T) {
		// Create a delegation first
		delegation := types.NewDelegation(addrsStr[0], valAddrsStr[0], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(100))))
		require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation))

		// Try to migrate with non-existent validator
		nonExistentVal := sdk.ValAddress("non_existent_validator")
		_, _, err := input.StakingKeeper.MigrateDelegation(ctx, addrs[0], nonExistentVal, types.DelegationMigration{
			DenomLpFrom:   "test_lp",
			DenomLpTo:     "test_lp_out",
			ModuleAddress: []byte("0x2"),
			ModuleName:    "dex_migration",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid request")
	})

	// Test case 3: Migration with invalid bond denom
	t.Run("InvalidBondDenom", func(t *testing.T) {
		// Create a delegation
		delegation := types.NewDelegation(addrsStr[0], valAddrsStr[0], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(100))))
		require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation))

		// Try to migrate to a non-bond denom
		_, _, err := input.StakingKeeper.MigrateDelegation(ctx, addrs[0], valAddrs[0], types.DelegationMigration{
			DenomLpFrom:   "test_lp",
			DenomLpTo:     "test_lp_out",
			ModuleAddress: []byte("0x2"),
			ModuleName:    "dex_migration",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid request")
	})
}

func Test_MigrateDelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	dexMigrationModule := ReadMoveFile("dex_migration")
	err := input.MoveKeeper.PublishModuleBundle(ctx, movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr), vmtypes.NewModuleBundle(vmtypes.Module{Code: dexMigrationModule}), movetypes.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	baseDenom := bondDenom
	metadataLPOld := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), false)
	metadataLPNew := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 100_000_000), sdk.NewInt64Coin("uusdc2", 250_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), false)

	denomLpOld, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPOld)
	require.NoError(t, err)
	denomLpNew, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPNew)
	require.NoError(t, err)

	metadataUusdc, err := movetypes.MetadataAddressFromDenom("uusdc")
	require.NoError(t, err)

	metadataUusdc2, err := movetypes.MetadataAddressFromDenom("uusdc2")
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr),
		movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr),
		"dex_migration",
		"initialize",
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", metadataUusdc.String()),
			fmt.Sprintf("\"%s\"", metadataUusdc2.String()),
		},
	)
	require.NoError(t, err)

	fundedAccount := input.Faucet.NewFundedAccount(ctx, sdk.NewInt64Coin("uusdc2", 500_000_000))
	err = input.MoveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(fundedAccount),
		movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr),
		"dex_migration",
		"provide_liquidity",
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", metadataUusdc2.String()),
			"\"500000000\"",
		},
	)
	require.NoError(t, err)

	// update params
	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.BondDenoms = append(params.BondDenoms, denomLpOld)
	require.NoError(t, input.StakingKeeper.SetParams(ctx, params))
	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr)
	require.NoError(t, err)

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	delAddr := input.Faucet.NewFundedAccount(ctx, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000))

	err = input.MoveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(delAddr),
		movetypes.ConvertSDKAddressToVMAddress(movetypes.StdAddr),
		movetypes.MoveModuleNameDex,
		movetypes.FunctionNameDexProvideLiquidity,
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf("\"%s\"", metadataLPOld.String()),
			"\"1000000\"",
			"\"2500000\"",
			"null",
		},
	)
	require.NoError(t, err)

	balanceLpOld := input.BankKeeper.GetBalance(ctx, delAddr, denomLpOld)

	err = input.MoveKeeper.Whitelist(ctx, movetypes.MsgWhitelist{
		MetadataLP:   metadataLPOld.String(),
		RewardWeight: math.LegacyNewDecWithPrec(1, 1),
	})
	if err != nil && !strings.Contains(err.Error(), "was already registered") {
		require.NoError(t, err)
	}

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, sdk.NewCoins(balanceLpOld), types.Unbonded, validator, true)
	require.NoError(t, err)

	delegation, err := input.StakingKeeper.GetDelegation(ctx, delAddr, valAddr)
	require.NoError(t, err)

	delAddrStr, err := input.AccountKeeper.AddressCodec().BytesToString(delAddr)
	require.NoError(t, err)

	require.Equal(t, types.Delegation{
		DelegatorAddress: delAddrStr,
		ValidatorAddress: valAddrStr,
		Shares:           shares,
	}, delegation)

	metadataLPOldFeeRateBefore, err := input.MoveKeeper.BalancerKeeper().PoolFeeRate(ctx, metadataLPOld)
	require.NoError(t, err)
	metadataLPNewFeeRateBefore, err := input.MoveKeeper.BalancerKeeper().PoolFeeRate(ctx, metadataLPNew)
	require.NoError(t, err)

	// Get the registered migration
	// no migration registered
	_, err = input.StakingKeeper.Migrations.Get(ctx, collections.Join(denomLpOld, denomLpNew))
	require.Error(t, err)

	err = input.StakingKeeper.RegisterMigration(ctx, denomLpOld, denomLpNew, movetypes.TestAddr.String(), "dex_migration")
	require.NoError(t, err)

	migration, err := input.StakingKeeper.Migrations.Get(ctx, collections.Join(denomLpOld, denomLpNew))
	require.NoError(t, err)

	// denomLpNew is not in bond denoms
	_, _, err = input.StakingKeeper.MigrateDelegation(ctx, delAddr, valAddr, migration)
	require.Error(t, err)

	err = input.MoveKeeper.Whitelist(ctx, movetypes.MsgWhitelist{
		MetadataLP:   metadataLPNew.String(),
		RewardWeight: math.LegacyNewDecWithPrec(1, 1),
	})
	if err != nil && !strings.Contains(err.Error(), "was already registered") {
		require.NoError(t, err)
	}

	err = input.StakingKeeper.RegisterMigration(ctx, denomLpOld, denomLpNew, movetypes.TestAddr.String(), "dex_migration")
	require.NoError(t, err)

	// Get the registered migration
	migration, err = input.StakingKeeper.Migrations.Get(ctx, collections.Join(denomLpOld, denomLpNew))
	require.NoError(t, err)

	_, newShares, err := input.StakingKeeper.MigrateDelegation(ctx, delAddr, valAddr, migration)
	require.NoError(t, err)

	delegation, err = input.StakingKeeper.GetDelegation(ctx, delAddr, valAddr)
	require.NoError(t, err)

	require.Equal(t, types.Delegation{
		DelegatorAddress: delAddrStr,
		ValidatorAddress: valAddrStr,
		Shares:           newShares,
	}, delegation)

	metadataLPOldFeeRateAfter, err := input.MoveKeeper.BalancerKeeper().PoolFeeRate(ctx, metadataLPOld)
	require.NoError(t, err)
	metadataLPNewFeeRateAfter, err := input.MoveKeeper.BalancerKeeper().PoolFeeRate(ctx, metadataLPNew)
	require.NoError(t, err)

	require.Equal(t, metadataLPOldFeeRateBefore, metadataLPOldFeeRateAfter)
	require.Equal(t, metadataLPNewFeeRateBefore, metadataLPNewFeeRateAfter)
}
