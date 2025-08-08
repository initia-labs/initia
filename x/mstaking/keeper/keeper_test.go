package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

	metadataUusdc, err := movetypes.MetadataAddressFromDenom("uusdc")
	require.NoError(t, err)

	metadataUusdc2, err := movetypes.MetadataAddressFromDenom("uusdc2")
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewInt64Coin("uusdc2", 2_500_000_000))

	err = input.StakingKeeper.RegisterMigration(ctx, lpDenomOld, lpDenomNew, "uusdc", "uusdc2", "0x2::swap")
	require.Error(t, err)

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

	err = input.StakingKeeper.RegisterMigration(ctx, lpDenomOld, lpDenomNew, "uusdc", "uusdc2", "invalid_module")
	require.Error(t, err)

	err = input.StakingKeeper.RegisterMigration(ctx, lpDenomOld, lpDenomNew, "uusdc", "uusdc2", "0x2::invalid_swap_module")
	require.Error(t, err)

	err = input.StakingKeeper.RegisterMigration(ctx, lpDenomOld, lpDenomNew, "uusdc", "uusdc2", "0x2::swap")
	require.NoError(t, err)
}
