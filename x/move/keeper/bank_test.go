package keeper_test

import (
	"encoding/hex"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

func Test_GetBalance(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := keeper.NewMoveBankKeeper(&input.MoveKeeper)

	bz, err := hex.DecodeString("0000000000000000000000000000000000000002")
	require.NoError(t, err)
	twoAddr := sdk.AccAddress(bz)

	amount, err := moveBankKeeper.GetBalance(ctx, twoAddr, bondDenom)
	require.NoError(t, err)
	require.Equal(t, sdk.ZeroInt(), amount)

	// mint token
	mintAmount := sdk.NewInt(100)
	err = moveBankKeeper.MintCoins(ctx, twoAddr, sdk.NewCoins(sdk.NewCoin(bondDenom, mintAmount)))
	require.NoError(t, err)

	amount, err = moveBankKeeper.GetBalance(ctx, twoAddr, bondDenom)
	require.Equal(t, mintAmount, amount)
}

func Test_AccountCoinStore(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := keeper.NewMoveBankKeeper(&input.MoveKeeper)

	bz, err := hex.DecodeString("0000000000000000000000000000000000000002")
	require.NoError(t, err)
	twoAddr := sdk.AccAddress(bz)

	metadata, err := types.MetadataAddressFromDenom(bondDenom)
	require.NoError(t, err)

	err = moveBankKeeper.MintCoins(ctx, twoAddr, sdk.NewCoins(sdk.NewInt64Coin(bondDenom, int64(1))))
	require.NoError(t, err)

	coinStore, err := moveBankKeeper.GetUserStores(ctx, twoAddr)
	iter := coinStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		// first byte of key is TypeTag enum index
		key := iter.Key()
		mt, err := vmtypes.NewAccountAddressFromBytes(key)
		require.NoError(t, err)
		require.Equal(t, metadata, mt)

		value := iter.Value()
		storeAddr, err := vmtypes.NewAccountAddressFromBytes(value)
		mt, amount, err := moveBankKeeper.Balance(ctx, storeAddr)
		require.NoError(t, err)
		require.Equal(t, sdk.NewInt(1), amount)
		require.Equal(t, metadata, mt)
	}

	mintAmount := int64(100)

	err = moveBankKeeper.MintCoins(ctx, twoAddr, sdk.NewCoins(sdk.NewInt64Coin(bondDenom, mintAmount)))
	require.NoError(t, err)

	// check after mint
	iter2 := coinStore.Iterator(nil, nil)
	defer iter2.Close()

	for ; iter2.Valid(); iter2.Next() {
		// first byte of key is TypeTag enum index
		key := iter2.Key()
		mt, err := vmtypes.NewAccountAddressFromBytes(key)
		require.NoError(t, err)
		require.Equal(t, metadata, mt)

		value := iter2.Value()
		storeAddr, err := vmtypes.NewAccountAddressFromBytes(value)
		mt, amount, err := moveBankKeeper.Balance(ctx, storeAddr)
		require.NoError(t, err)
		require.Equal(t, sdk.NewInt(mintAmount+1), amount)
		require.Equal(t, metadata, mt)
	}
}

func Test_GetSupply(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := keeper.NewMoveBankKeeper(&input.MoveKeeper)

	amount, err := moveBankKeeper.GetSupply(ctx, bondDenom)
	require.NoError(t, err)
	require.Equal(t, initiaSupply, amount)

	// mint token
	mintAmount := sdk.NewIntFromUint64(math.MaxUint64)
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

func Test_GetIssuers(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveBankKeeper := keeper.NewMoveBankKeeper(&input.MoveKeeper)

	bondDenomMetadata, err := types.MetadataAddressFromDenom(bondDenom)
	require.NoError(t, err)

	testDenom := testDenoms[0]
	testDenomMetadata, err := types.MetadataAddressFromDenom(testDenom)
	require.NoError(t, err)

	// mint token
	mintAmount := sdk.NewIntFromUint64(math.MaxUint64)
	mintNum := 5

	for i := 0; i < mintNum; i++ {
		bz, err := hex.DecodeString(fmt.Sprintf("000000000000000000000000000000000000000%d", i))
		require.NoError(t, err)
		addr := sdk.AccAddress(bz)

		err = moveBankKeeper.MintCoins(ctx, addr, sdk.NewCoins(sdk.NewCoin(testDenom, mintAmount)))
		require.NoError(t, err)
	}

	issuers, err := moveBankKeeper.GetIssuers(ctx)
	iter := issuers.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		// first byte of key is TypeTag enum index
		key := iter.Key()
		mt, err := vmtypes.NewAccountAddressFromBytes(key)
		require.NoError(t, err)

		value := iter.Value()

		amount, err := keeper.NewMoveBankKeeper(&input.MoveKeeper).GetSupplyWithMetadata(ctx, mt)
		require.NoError(t, err)

		issuer, err := vmtypes.NewAccountAddressFromBytes(value)
		require.NoError(t, err)
		require.Equal(t, vmtypes.StdAddress, issuer)

		if mt == bondDenomMetadata {
			require.Equal(t, initiaSupply, amount)
		} else if mt == testDenomMetadata {
			require.Equal(t, initiaSupply.Add(mintAmount.MulRaw(int64(mintNum))), amount)
		}
	}
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

	amount := sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewIntFromUint64(1_000_000)))
	input.Faucet.Fund(ctx, twoAddr, amount...)

	err = moveBankKeeper.SendCoins(ctx, twoAddr, threeAddr, amount)
	require.NoError(t, err)

	require.Equal(t, sdk.NewCoin(bondDenom, sdk.ZeroInt()), input.BankKeeper.GetBalance(ctx, twoAddr, bondDenom))
	require.Equal(t, amount, sdk.NewCoins(input.BankKeeper.GetBalance(ctx, threeAddr, bondDenom)))
}
