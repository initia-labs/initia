package keeper_test

import (
	"encoding/hex"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmosbanktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func Test_GetBalance(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := keeper.NewMoveBankKeeper(&input.MoveKeeper)

	bz, err := hex.DecodeString("0000000000000000000000000000000000000002")
	require.NoError(t, err)
	twoAddr := sdk.AccAddress(bz)

	amount, err := moveBankKeeper.GetBalance(ctx, twoAddr, bondDenom)
	require.NoError(t, err)
	require.Equal(t, sdkmath.ZeroInt(), amount)

	// mint token
	mintAmount := sdkmath.NewInt(100)
	err = moveBankKeeper.MintCoins(ctx, twoAddr, sdk.NewCoins(sdk.NewCoin(bondDenom, mintAmount)))
	require.NoError(t, err)

	amount, err = moveBankKeeper.GetBalance(ctx, twoAddr, bondDenom)
	require.NoError(t, err)
	require.Equal(t, mintAmount, amount)
}

func Test_IterateBalances(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := keeper.NewMoveBankKeeper(&input.MoveKeeper)

	bz, err := hex.DecodeString("0000000000000000000000000000000000000002")
	require.NoError(t, err)
	twoAddr := sdk.AccAddress(bz)

	err = moveBankKeeper.MintCoins(ctx, twoAddr, sdk.NewCoins(sdk.NewInt64Coin(bondDenom, int64(1))))
	require.NoError(t, err)

	entered := false
	moveBankKeeper.IterateAccountBalances(ctx, twoAddr, func(amount sdk.Coin) (bool, error) {
		entered = true
		require.Equal(t, sdk.NewCoin(bondDenom, sdkmath.NewInt(1)), amount)
		return false, nil
	})
	require.True(t, entered)

	mintAmount := int64(100)

	err = moveBankKeeper.MintCoins(ctx, twoAddr, sdk.NewCoins(sdk.NewInt64Coin(bondDenom, mintAmount)))
	require.NoError(t, err)

	// check after mint
	entered = false
	moveBankKeeper.IterateAccountBalances(ctx, twoAddr, func(amount sdk.Coin) (bool, error) {
		entered = true
		require.Equal(t, sdk.NewCoin(bondDenom, sdkmath.NewInt(mintAmount+1)), amount)
		return false, nil
	})
	require.True(t, entered)
}

func Test_GetSupply(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := keeper.NewMoveBankKeeper(&input.MoveKeeper)

	amount, err := moveBankKeeper.GetSupply(ctx, bondDenom)
	require.NoError(t, err)
	require.Equal(t, initiaSupply, amount)

	// mint token
	mintAmount := sdkmath.NewIntFromUint64(math.MaxUint64)
	mintNum := 5

	for i := 0; i < mintNum; i++ {
		bz, err := hex.DecodeString(fmt.Sprintf("000000000000000000000000000000000000000%d", i))
		require.NoError(t, err)
		addr := sdk.AccAddress(bz)

		err = moveBankKeeper.MintCoins(ctx, addr, sdk.NewCoins(sdk.NewCoin(bondDenom, mintAmount)))
		require.NoError(t, err)
	}

	amount, err = moveBankKeeper.GetSupply(ctx, bondDenom)
	require.NoError(t, err)
	require.Equal(t, initiaSupply.Add(mintAmount.MulRaw(int64(mintNum))), amount)
}

func Test_IterateSupply(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := keeper.NewMoveBankKeeper(&input.MoveKeeper)

	testDenom := testDenoms[0]

	// mint token
	mintAmount := sdkmath.NewIntFromUint64(math.MaxUint64)
	mintNum := 5

	for i := 0; i < mintNum; i++ {
		bz, err := hex.DecodeString(fmt.Sprintf("000000000000000000000000000000000000000%d", i))
		require.NoError(t, err)
		addr := sdk.AccAddress(bz)

		err = moveBankKeeper.MintCoins(ctx, addr, sdk.NewCoins(sdk.NewCoin(testDenom, mintAmount)))
		require.NoError(t, err)
	}

	counter := 0
	moveBankKeeper.IterateSupply(ctx, func(supply sdk.Coin) (bool, error) {
		if supply.Denom == bondDenom {
			counter++
			require.Equal(t, initiaSupply, supply.Amount)
		} else if supply.Denom == testDenom {
			counter++
			require.Equal(t, initiaSupply.Add(mintAmount.MulRaw(int64(mintNum))), supply.Amount)
		} else {
			require.Equal(t, initiaSupply, supply.Amount)
		}

		return false, nil
	})
	require.Equal(t, 2, counter)
}

func Test_SendCoins(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := keeper.NewMoveBankKeeper(&input.MoveKeeper)

	bz, err := hex.DecodeString("0000000000000000000000000000000000000002")
	require.NoError(t, err)
	twoAddr := sdk.AccAddress(bz)

	bz, err = hex.DecodeString("0000000000000000000000000000000000000003")
	require.NoError(t, err)
	threeAddr := sdk.AccAddress(bz)

	amount := sdk.NewCoins(sdk.NewCoin(bondDenom, sdkmath.NewIntFromUint64(1_000_000)))
	input.Faucet.Fund(ctx, twoAddr, amount...)

	err = moveBankKeeper.SendCoins(ctx, twoAddr, threeAddr, amount)
	require.NoError(t, err)

	require.Equal(t, sdk.NewCoin(bondDenom, sdkmath.ZeroInt()), input.BankKeeper.GetBalance(ctx, twoAddr, bondDenom))
	require.Equal(t, amount, sdk.NewCoins(input.BankKeeper.GetBalance(ctx, threeAddr, bondDenom)))
}

func Test_GetMetadata(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := keeper.NewMoveBankKeeper(&input.MoveKeeper)

	metadata, err := moveBankKeeper.GetMetadata(ctx, bondDenom)
	require.NoError(t, err)

	require.Equal(t, "uinit", metadata.Base)
	require.Equal(t, "init", metadata.Display)
	require.Equal(t, "uinit Coin", metadata.Name)
	require.Equal(t, []*cosmosbanktypes.DenomUnit{
		{
			Denom:    bondDenom,
			Exponent: 0,
		},
		{
			Denom:    "init",
			Exponent: 6,
		},
	}, metadata.DenomUnits)
}

func Test_BurnCoins(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := keeper.NewMoveBankKeeper(&input.MoveKeeper)

	bz, err := hex.DecodeString("0000000000000000000000000000000000000002")
	require.NoError(t, err)
	twoAddr := sdk.AccAddress(bz)

	amount := sdk.NewCoins(sdk.NewCoin("foo", sdkmath.NewIntFromUint64(1_000_000)))
	input.Faucet.Fund(ctx, twoAddr, amount...)

	// create token
	err = input.MoveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		vmtypes.TestAddress,
		vmtypes.StdAddress,
		"managed_coin",
		"initialize",
		[]vmtypes.TypeTag{},
		[]string{`null`, `"test coin"`, `"bar"`, `6`, `""`, `""`},
	)
	require.NoError(t, err)

	// get bar metadata addr
	barMetadata := types.NamedObjectAddress(vmtypes.TestAddress, "bar")
	barDenom, err := types.DenomFromMetadataAddress(ctx, moveBankKeeper, barMetadata)
	require.NoError(t, err)

	// mint token
	err = input.MoveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		vmtypes.TestAddress,
		vmtypes.StdAddress,
		"managed_coin",
		"mint",
		[]vmtypes.TypeTag{},
		[]string{fmt.Sprintf("\"%s\"", vmtypes.TestAddress.String()), fmt.Sprintf("\"%s\"", barMetadata.String()), `"1000000"`},
	)
	require.NoError(t, err)

	err = moveBankKeeper.BurnCoins(ctx, twoAddr, sdk.NewCoins(sdk.NewCoin("foo", sdkmath.NewInt(500_000)), sdk.NewCoin(barDenom, sdkmath.NewInt(500_000))))
	require.NoError(t, err)

	require.Equal(t, sdk.NewCoin("foo", sdkmath.NewInt(500_000)), input.BankKeeper.GetBalance(ctx, twoAddr, "foo"))
	require.Equal(t, sdk.NewCoin(barDenom, sdkmath.NewInt(500_000)), input.BankKeeper.GetBalance(ctx, twoAddr, barDenom))
}
