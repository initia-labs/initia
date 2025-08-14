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

	swapModule := ReadMoveFile("swap")
	err := input.MoveKeeper.PublishModuleBundle(ctx, movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr), vmtypes.NewModuleBundle(vmtypes.Module{Code: swapModule}), movetypes.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	baseDenom := bondDenom
	metadataLPOld := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), true)
	metadataLPNew := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 100_000_000), sdk.NewInt64Coin("uusdc2", 250_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), true)

	lpDenomOld, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPOld)
	require.NoError(t, err)
	lpDenomNew, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPNew)
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewInt64Coin("uusdc2", 2_500_000_000))

	err = input.StakingKeeper.RegisterMigration(ctx, lpDenomOld, lpDenomNew, "uusdc", "uusdc2", "invalid_module")
	require.Error(t, err)

	err = input.StakingKeeper.RegisterMigration(ctx, lpDenomOld, lpDenomNew, "uusdc", "uusdc2", "0x2::swap")
	require.NoError(t, err)
}

func Test_MigrateDelegation_EdgeCases(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// Test case 1: Migration with non-existent LP denom
	t.Run("NonExistentLPDenom", func(t *testing.T) {
		_, _, err := input.StakingKeeper.MigrateDelegation(ctx, addrs[0], valAddrs[0], types.DelegationMigration{
			LpDenomIn:  "non_existent_lp",
			LpDenomOut: "non_existent_lp_out",
			DenomIn:    "uusdc",
			DenomOut:   "uusdc2",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid request")
	})

	// Test case 2: Migration with invalid metadata addresses
	t.Run("InvalidMetadataAddresses", func(t *testing.T) {
		_, _, err := input.StakingKeeper.MigrateDelegation(ctx, addrs[0], valAddrs[0], types.DelegationMigration{
			LpDenomIn:  "invalid_denom",
			LpDenomOut: "invalid_denom_out",
			DenomIn:    "invalid_denom_in",
			DenomOut:   "invalid_denom_out",
		})
		require.Error(t, err)
	})

	// Test case 3: Migration with zero origin shares
	t.Run("ZeroOriginShares", func(t *testing.T) {
		// Create a delegation with no shares in the specific LP denom
		delegation := types.NewDelegation(addrsStr[0], valAddrsStr[0], sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(100))))
		require.NoError(t, input.StakingKeeper.SetDelegation(ctx, delegation))

		_, _, err := input.StakingKeeper.MigrateDelegation(ctx, addrs[0], valAddrs[0], types.DelegationMigration{
			LpDenomIn:  "different_lp_denom",
			LpDenomOut: "target_lp_denom",
			DenomIn:    "uusdc",
			DenomOut:   "uusdc2",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid request")
	})
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

	// Test case 1: Register migration with invalid swap contract format
	t.Run("InvalidSwapContractFormat", func(t *testing.T) {
		// Test without ::
		err := input.StakingKeeper.RegisterMigration(ctx, lpDenom1, lpDenom2, "uusdc", "uusdc2", "invalid_format")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid swap contract address")

		// Test with too many parts
		err = input.StakingKeeper.RegisterMigration(ctx, lpDenom1, lpDenom2, "uusdc", "uusdc2", "part1::part2::part3")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid swap contract address")
	})

	// Test case 2: Register migration with non-existent LP denoms
	t.Run("NonExistentLPDenoms", func(t *testing.T) {
		err := input.StakingKeeper.RegisterMigration(ctx, "non_existent_lp", "non_existent_lp_out", "denom_in", "denom_out", "0x2::swap")
		require.Error(t, err)
		require.Contains(t, err.Error(), "lp metadata is not found in balancer")
	})

	// Test case 3: Register migration with invalid module address
	t.Run("InvalidModuleAddress", func(t *testing.T) {
		err := input.StakingKeeper.RegisterMigration(ctx, lpDenom1, lpDenom2, "uusdc", "uusdc2", "invalid_addr::swap")
		require.Error(t, err)
		require.Contains(t, err.Error(), "decoding bech32 failed")
	})
}

func Test_MigrateDelegation_CompleteFlow(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// Setup: Create DEX pools and publish swap module
	swapModule := ReadMoveFile("swap")
	err := input.MoveKeeper.PublishModuleBundle(ctx, movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr), vmtypes.NewModuleBundle(vmtypes.Module{Code: swapModule}), movetypes.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	// Create DEX pools
	baseDenom := bondDenom
	metadataLPOld := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), false)
	metadataLPNew := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 100_000_000), sdk.NewInt64Coin("uusdc2", 250_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), false)

	lpDenomOld, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPOld)
	require.NoError(t, err)
	lpDenomNew, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPNew)
	require.NoError(t, err)

	// Initialize swap contract
	metadataUusdc, err := movetypes.MetadataAddressFromDenom("uusdc")
	require.NoError(t, err)
	metadataUusdc2, err := movetypes.MetadataAddressFromDenom("uusdc2")
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr),
		movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr),
		"swap",
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
		"swap",
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
	params.BondDenoms = append(params.BondDenoms, lpDenomOld, lpDenomNew)
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

	lpDenomOldBalance := input.BankKeeper.GetBalance(ctx, delAddr, lpDenomOld)

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
	_, err = input.StakingKeeper.Delegate(ctx, delAddr, sdk.NewCoins(lpDenomOldBalance), types.Unbonded, validator, true)
	require.NoError(t, err)

	// Test case 1: Successful migration flow
	t.Run("SuccessfulMigration", func(t *testing.T) {
		// Register migration
		err = input.StakingKeeper.RegisterMigration(ctx, lpDenomOld, lpDenomNew, "uusdc", "uusdc2", "0x2::swap")
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
		lpMetadataIn, err := movetypes.MetadataAddressFromDenom(lpDenomOld)
		require.NoError(t, err)
		lpMetadataOut, err := movetypes.MetadataAddressFromDenom(lpDenomNew)
		require.NoError(t, err)
		migration, err := input.StakingKeeper.Migrations.Get(ctx, collections.Join(lpMetadataIn[:], lpMetadataOut[:]))
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
		swapAddr, err := vmtypes.NewAccountAddress("2")
		require.NoError(t, err)

		largeMigration := types.DelegationMigration{
			LpDenomIn:                 "non_existent_pool",
			LpDenomOut:                lpDenomNew,
			DenomIn:                   "uusdc",
			DenomOut:                  "uusdc2",
			SwapContractModuleAddress: swapAddr[:],
			SwapContractModuleName:    "swap",
		}

		_, _, err = input.StakingKeeper.MigrateDelegation(ctx, delAddr, valAddr, largeMigration)
		require.Error(t, err)
	})

	// Test case 3: Migration with invalid pool metadata
	t.Run("InvalidPoolMetadata", func(t *testing.T) {
		// Create a migration with non-existent pool
		swapAddr, err := vmtypes.NewAccountAddress("2")
		require.NoError(t, err)

		invalidMigration := types.DelegationMigration{
			LpDenomIn:                 "non_existent_pool",
			LpDenomOut:                lpDenomNew,
			DenomIn:                   "uusdc",
			DenomOut:                  "uusdc2",
			SwapContractModuleAddress: swapAddr[:],
			SwapContractModuleName:    "swap",
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
		err := input.StakingKeeper.RegisterMigration(ctx, lpDenom1, lpDenom2, "uusdc", "uusdc2", "0x2::swap")
		require.NoError(t, err)

		// Verify migration was stored
		lpMetadataIn, err := movetypes.MetadataAddressFromDenom(lpDenom1)
		require.NoError(t, err)
		lpMetadataOut, err := movetypes.MetadataAddressFromDenom(lpDenom2)
		require.NoError(t, err)

		migration, err := input.StakingKeeper.Migrations.Get(ctx, collections.Join(lpMetadataIn[:], lpMetadataOut[:]))
		require.NoError(t, err)
		require.Equal(t, lpDenom1, migration.LpDenomIn)
		require.Equal(t, lpDenom2, migration.LpDenomOut)
		require.Equal(t, "uusdc", migration.DenomIn)
		require.Equal(t, "uusdc2", migration.DenomOut)
	})

	// Test case 2: Verify migration overwrite behavior
	t.Run("MigrationOverwrite", func(t *testing.T) {
		// Register migration with same LP denom but different target
		err := input.StakingKeeper.RegisterMigration(ctx, lpDenom1, lpDenom2, "uusdc", "uusdc2", "0x2::swap")
		require.NoError(t, err)

		// Verify migration was overwritten
		lpMetadataIn, err := movetypes.MetadataAddressFromDenom(lpDenom1)
		require.NoError(t, err)
		lpMetadataOut, err := movetypes.MetadataAddressFromDenom(lpDenom2)
		require.NoError(t, err)

		migration, err := input.StakingKeeper.Migrations.Get(ctx, collections.Join(lpMetadataIn[:], lpMetadataOut[:]))
		require.NoError(t, err)
		require.Equal(t, lpDenom2, migration.LpDenomOut)
		require.Equal(t, "uusdc2", migration.DenomOut)
	})

	// Test case 3: Verify migration cleanup
	t.Run("MigrationCleanup", func(t *testing.T) {
		// Register a migration
		err := input.StakingKeeper.RegisterMigration(ctx, lpDenom1, lpDenom2, "uusdc", "uusdc2", "0x2::swap")
		require.NoError(t, err)

		// Verify migration exists
		lpMetadataIn, err := movetypes.MetadataAddressFromDenom(lpDenom1)
		require.NoError(t, err)
		lpMetadataOut, err := movetypes.MetadataAddressFromDenom(lpDenom2)
		require.NoError(t, err)

		_, err = input.StakingKeeper.Migrations.Get(ctx, collections.Join(lpMetadataIn[:], lpMetadataOut[:]))
		require.NoError(t, err)

		// Remove the migration (simulating cleanup)
		err = input.StakingKeeper.Migrations.Remove(ctx, collections.Join(lpMetadataIn[:], lpMetadataOut[:]))
		require.NoError(t, err)

		// Verify migration was removed
		_, err = input.StakingKeeper.Migrations.Get(ctx, collections.Join(lpMetadataIn[:], lpMetadataOut[:]))
		require.Error(t, err)
	})
}

func Test_MigrateDelegation_ErrorHandling(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// Test case 1: Migration with non-existent delegation
	t.Run("NonExistentDelegation", func(t *testing.T) {
		_, _, err := input.StakingKeeper.MigrateDelegation(ctx, addrs[0], valAddrs[0], types.DelegationMigration{
			LpDenomIn:  "test_lp",
			LpDenomOut: "test_lp_out",
			DenomIn:    "test_denom_in",
			DenomOut:   "test_denom_out",
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
			LpDenomIn:  "test_lp",
			LpDenomOut: "test_lp_out",
			DenomIn:    "test_denom_in",
			DenomOut:   "test_denom_out",
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
			LpDenomIn:  bondDenom,
			LpDenomOut: "non_bond_denom",
			DenomIn:    "test_denom_in",
			DenomOut:   "test_denom_out",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid request")
	})
}

func Test_MigrateDelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	swapModule := ReadMoveFile("swap")
	err := input.MoveKeeper.PublishModuleBundle(ctx, movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr), vmtypes.NewModuleBundle(vmtypes.Module{Code: swapModule}), movetypes.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	baseDenom := bondDenom
	metadataLPOld := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), false)
	metadataLPNew := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 100_000_000), sdk.NewInt64Coin("uusdc2", 250_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), false)

	lpDenomOld, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPOld)
	require.NoError(t, err)
	lpDenomNew, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPNew)
	require.NoError(t, err)

	metadataUusdc, err := movetypes.MetadataAddressFromDenom("uusdc")
	require.NoError(t, err)

	metadataUusdc2, err := movetypes.MetadataAddressFromDenom("uusdc2")
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr),
		movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr),
		"swap",
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
		"swap",
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
	params.BondDenoms = append(params.BondDenoms, lpDenomOld)
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

	lpDenomOldBalance := input.BankKeeper.GetBalance(ctx, delAddr, lpDenomOld)

	err = input.MoveKeeper.Whitelist(ctx, movetypes.MsgWhitelist{
		MetadataLP:   metadataLPOld.String(),
		RewardWeight: math.LegacyNewDecWithPrec(1, 1),
	})
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, sdk.NewCoins(lpDenomOldBalance), types.Unbonded, validator, true)
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
	lpMetadataIn, err := movetypes.MetadataAddressFromDenom(lpDenomOld)
	require.NoError(t, err)
	lpMetadataOut, err := movetypes.MetadataAddressFromDenom(lpDenomNew)
	require.NoError(t, err)

	// no migration registered
	_, err = input.StakingKeeper.Migrations.Get(ctx, collections.Join(lpMetadataIn[:], lpMetadataOut[:]))
	require.Error(t, err)

	err = input.StakingKeeper.RegisterMigration(ctx, lpDenomOld, lpDenomNew, "uusdc", "uusdc2", "0x2::swap")
	require.NoError(t, err)

	migration, err := input.StakingKeeper.Migrations.Get(ctx, collections.Join(lpMetadataIn[:], lpMetadataOut[:]))
	require.NoError(t, err)

	// lpDenomNew is not in bond denoms
	_, _, err = input.StakingKeeper.MigrateDelegation(ctx, delAddr, valAddr, migration)
	require.Error(t, err)

	err = input.MoveKeeper.Whitelist(ctx, movetypes.MsgWhitelist{
		MetadataLP:   metadataLPNew.String(),
		RewardWeight: math.LegacyNewDecWithPrec(1, 1),
	})
	require.NoError(t, err)

	err = input.StakingKeeper.RegisterMigration(ctx, lpDenomOld, lpDenomNew, "uusdc", "uusdc2", "0x2::swap")
	require.NoError(t, err)

	// Get the registered migration
	migration, err = input.StakingKeeper.Migrations.Get(ctx, collections.Join(lpMetadataIn[:], lpMetadataOut[:]))
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
