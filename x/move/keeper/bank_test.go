package keeper_test

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	cosmosbanktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/initia-labs/initia/app/upgrades/v1_1_1"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func Test_GetBalance(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := input.MoveKeeper.MoveBankKeeper()

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
	moveBankKeeper := input.MoveKeeper.MoveBankKeeper()

	bz, err := hex.DecodeString("0000000000000000000000000000000000000002")
	require.NoError(t, err)
	twoAddr := sdk.AccAddress(bz)

	// create 200 tokens
	tokenNum := 200
	for i := range tokenNum {
		denom := fmt.Sprintf("test%d", i)
		err = moveBankKeeper.MintCoins(ctx, twoAddr, sdk.NewCoins(sdk.NewInt64Coin(denom, int64(i))))
		require.NoError(t, err)
	}

	counter := 0
	coins := sdk.NewCoins()
	moveBankKeeper.IterateAccountBalances(ctx, twoAddr, func(amount sdk.Coin) (bool, error) {
		// extract amount from denom
		amountStr := strings.Split(amount.Denom, "test")[1]
		expectedAmount, err := strconv.ParseInt(amountStr, 10, 64)
		require.NoError(t, err)
		require.Equal(t, sdk.NewCoin(amount.Denom, sdkmath.NewInt(expectedAmount)), amount)

		counter++

		coins = coins.Add(amount)
		return false, nil
	})

	// except zero amount coin
	require.Equal(t, tokenNum-1, counter)

	// transfer coins to other address except last one
	err = moveBankKeeper.SendCoins(ctx, twoAddr, types.StdAddr, coins[:len(coins)-1])
	require.NoError(t, err)

	// should count only last one
	counter = 0
	moveBankKeeper.IterateAccountBalances(ctx, twoAddr, func(amount sdk.Coin) (bool, error) {
		// extract amount from denom
		amountStr := strings.Split(amount.Denom, "test")[1]
		expectedAmount, err := strconv.ParseInt(amountStr, 10, 64)
		require.NoError(t, err)
		require.Equal(t, sdk.NewCoin(amount.Denom, sdkmath.NewInt(expectedAmount)), amount)

		counter++

		return false, nil
	})

	require.Equal(t, 1, counter)
}

func Test_GetPaginatedBalances(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := input.MoveKeeper.MoveBankKeeper()

	bz, err := hex.DecodeString("0000000000000000000000000000000000000002")
	require.NoError(t, err)
	twoAddr := sdk.AccAddress(bz)

	// create 200 tokens
	tokenNum := 200
	for i := range tokenNum {
		denom := fmt.Sprintf("test%d", i)
		err = moveBankKeeper.MintCoins(ctx, twoAddr, sdk.NewCoins(sdk.NewInt64Coin(denom, int64(i))))
		require.NoError(t, err)
	}

	counter := int(0)
	fetchUint := uint64(3)
	pageReq := &query.PageRequest{
		Limit: fetchUint,
	}

	totalCoins := sdk.Coins{}
	for i := 0; i < tokenNum; i += int(fetchUint) {
		coins, pageRes, err := moveBankKeeper.GetPaginatedBalances(ctx, pageReq, twoAddr)
		require.NoError(t, err)

		pageReq.Key = pageRes.NextKey
		counter += int(coins.Len())
		totalCoins = totalCoins.Add(coins...)
	}

	for _, coin := range totalCoins {
		amountStr := strings.Split(coin.Denom, "test")[1]
		expectedAmount, err := strconv.ParseInt(amountStr, 10, 64)
		require.NoError(t, err)
		require.Equal(t, sdk.NewCoin(coin.Denom, sdkmath.NewInt(expectedAmount)), coin)
	}

	// except zero amount coin
	require.Equal(t, tokenNum-1, counter)
}

func Test_GetSupply(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := input.MoveKeeper.MoveBankKeeper()

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
	moveBankKeeper := input.MoveKeeper.MoveBankKeeper()

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
	moveBankKeeper := input.MoveKeeper.MoveBankKeeper()

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
	moveBankKeeper := input.MoveKeeper.MoveBankKeeper()

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
	moveBankKeeper := input.MoveKeeper.MoveBankKeeper()

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
		"mint_to",
		[]vmtypes.TypeTag{},
		[]string{fmt.Sprintf("\"%s\"", vmtypes.TestAddress.String()), fmt.Sprintf("\"%s\"", barMetadata.String()), `"1000000"`},
	)
	require.NoError(t, err)

	err = moveBankKeeper.BurnCoins(ctx, twoAddr, sdk.NewCoins(sdk.NewCoin("foo", sdkmath.NewInt(500_000)), sdk.NewCoin(barDenom, sdkmath.NewInt(500_000))))
	require.NoError(t, err)

	require.Equal(t, sdk.NewCoin("foo", sdkmath.NewInt(500_000)), input.BankKeeper.GetBalance(ctx, twoAddr, "foo"))
	require.Equal(t, sdk.NewCoin(barDenom, sdkmath.NewInt(500_000)), input.BankKeeper.GetBalance(ctx, twoAddr, barDenom))
}

func Test_MultiSend(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := input.MoveKeeper.MoveBankKeeper()

	bz, err := hex.DecodeString("0000000000000000000000000000000000000002")
	require.NoError(t, err)
	twoAddr := sdk.AccAddress(bz)

	bz, err = hex.DecodeString("0000000000000000000000000000000000000003")
	require.NoError(t, err)
	threeAddr := sdk.AccAddress(bz)

	bz, err = hex.DecodeString("0000000000000000000000000000000000000004")
	require.NoError(t, err)
	fourAddr := sdk.AccAddress(bz)

	bz, err = hex.DecodeString("0000000000000000000000000000000000000005")
	require.NoError(t, err)
	fiveAddr := sdk.AccAddress(bz)

	amount := sdk.NewCoins(sdk.NewCoin(bondDenom, sdkmath.NewIntFromUint64(1_000_000)))
	input.Faucet.Fund(ctx, twoAddr, amount...)

	err = moveBankKeeper.MultiSend(ctx, twoAddr, bondDenom, []sdk.AccAddress{threeAddr, fourAddr, fiveAddr}, []sdkmath.Int{sdkmath.NewIntFromUint64(300_000), sdkmath.NewIntFromUint64(400_000), sdkmath.NewIntFromUint64(300_000)})
	require.NoError(t, err)

	require.Equal(t, sdk.NewCoin(bondDenom, sdkmath.ZeroInt()), input.BankKeeper.GetBalance(ctx, twoAddr, bondDenom))
	require.Equal(t, uint64(300_000), input.BankKeeper.GetBalance(ctx, threeAddr, bondDenom).Amount.Uint64())
	require.Equal(t, uint64(400_000), input.BankKeeper.GetBalance(ctx, fourAddr, bondDenom).Amount.Uint64())
	require.Equal(t, uint64(300_000), input.BankKeeper.GetBalance(ctx, fiveAddr, bondDenom).Amount.Uint64())
}

func Test_DispatchableToken(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// TODO - remove this after movevm version update
	////////////////////////////////////////////////////
	moduleBytes, err := v1_1_1.GetModuleBytes()
	require.NoError(t, err)

	var modules []vmtypes.Module
	for _, module := range moduleBytes {
		modules = append(modules, vmtypes.NewModule(module))
	}

	err = input.MoveKeeper.PublishModuleBundle(ctx, vmtypes.StdAddress, vmtypes.NewModuleBundle(modules...), types.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)
	////////////////////////////////////////////////////
	deployer, err := vmtypes.NewAccountAddress("0xcafe")
	require.NoError(t, err)

	moveBankKeeper := input.MoveKeeper.MoveBankKeeper()

	err = input.MoveKeeper.PublishModuleBundle(ctx, deployer, vmtypes.NewModuleBundle(vmtypes.NewModule(dispatchableTokenModule)), types.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	// execute initialize
	err = input.MoveKeeper.ExecuteEntryFunctionJSON(ctx, deployer, deployer, "test_dispatchable_token", "initialize", []vmtypes.TypeTag{}, []string{})
	require.NoError(t, err)

	// check supply
	metadata := types.NamedObjectAddress(deployer, "test_token")
	supply, err := moveBankKeeper.GetSupplyWithMetadata(ctx, metadata)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(0), supply)

	// mint token
	err = input.MoveKeeper.ExecuteEntryFunctionJSON(ctx, deployer, deployer, "test_dispatchable_token", "mint", []vmtypes.TypeTag{}, []string{fmt.Sprintf("\"%s\"", deployer.String()), `"1000000"`})
	require.NoError(t, err)

	// get supply
	supply, err = moveBankKeeper.GetSupplyWithMetadata(ctx, metadata)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(1_000_000).MulRaw(10), supply)

	// get balance
	denom, err := types.DenomFromMetadataAddress(ctx, moveBankKeeper, metadata)
	require.NoError(t, err)
	balance, err := moveBankKeeper.GetBalance(ctx, deployer[:], denom)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(1_000_000).MulRaw(10), balance)
}
