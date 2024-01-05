package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	stakingkeeper "github.com/initia-labs/initia/x/mstaking/keeper"

	vmtypes "github.com/initia-labs/initiavm/types"
)

func Test_GetAccountInfo(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	vmaddr, err := vmtypes.NewAccountAddressFromBytes(addrs[0])
	require.NoError(t, err)

	api := keeper.NewApi(input.MoveKeeper, ctx)

	input.AccountKeeper.SetAccount(ctx, input.AccountKeeper.NewAccountWithAddress(ctx, addrs[0]))
	found, accountNumber, sequence, accountType := api.GetAccountInfo(vmaddr)
	require.True(t, found)

	acc := input.AccountKeeper.GetAccount(ctx, addrs[0])
	require.Equal(t, acc.GetAccountNumber(), accountNumber)
	require.Equal(t, acc.GetSequence(), sequence)
	require.Equal(t, vmtypes.AccountType_Base, accountType)

	vmaddr, err = vmtypes.NewAccountAddress("0x3")
	require.NoError(t, err)

	found, _, _, _ = api.GetAccountInfo(vmaddr)
	require.False(t, found)
}

func Test_CreateTypedAccounts(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	vmaddr, err := vmtypes.NewAccountAddressFromBytes(addrs[0])
	require.NoError(t, err)

	api := keeper.NewApi(input.MoveKeeper, ctx)

	input.AccountKeeper.SetAccount(ctx, input.AccountKeeper.NewAccountWithAddress(ctx, addrs[0]))
	found, accountNumber, sequence, accountType := api.GetAccountInfo(vmaddr)
	require.True(t, found)

	acc := input.AccountKeeper.GetAccount(ctx, addrs[0])
	require.Equal(t, acc.GetAccountNumber(), accountNumber)
	require.Equal(t, acc.GetSequence(), sequence)
	require.Equal(t, vmtypes.AccountType_Base, accountType)

	vmaddr, err = vmtypes.NewAccountAddress("0x3")
	require.NoError(t, err)

	found, _, _, _ = api.GetAccountInfo(vmaddr)
	require.False(t, found)
}

func Test_AmountToShareAPI(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := valAddrs[0]
	valPubKey := valPubKeys[0]

	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddrs[0])
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))

	sh := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	_, err = sh.CreateValidator(ctx, newTestMsgCreateValidator(valAddrStr, valPubKey, math.NewInt(100_000)))
	require.NoError(t, err)

	metadata, err := types.MetadataAddressFromDenom(bondDenom)
	require.NoError(t, err)

	api := keeper.NewApi(input.MoveKeeper, ctx)
	amount, err := api.AmountToShare([]byte(valAddr.String()), metadata, 150)
	require.NoError(t, err)
	require.Equal(t, uint64(150), amount)
}

func Test_AmountToShareAPI_InvalidAddr(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := valAddrs[0]
	valPubKey := valPubKeys[0]

	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddrs[0])
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))

	sh := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	_, err = sh.CreateValidator(ctx, newTestMsgCreateValidator(valAddrStr, valPubKey, math.NewInt(100_000)))
	require.NoError(t, err)

	metadata, err := types.MetadataAddressFromDenom(bondDenom)
	require.NoError(t, err)

	api := keeper.NewApi(input.MoveKeeper, ctx)
	_, err = api.AmountToShare(valAddr, metadata, 150)
	require.Error(t, err)
}

func Test_ShareToAmountAPI(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := valAddrs[0]
	valPubKey := valPubKeys[0]

	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddrs[0])
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))

	sh := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	_, err = sh.CreateValidator(ctx, newTestMsgCreateValidator(valAddrStr, valPubKey, math.NewInt(100_000)))
	require.NoError(t, err)

	metadata, err := types.MetadataAddressFromDenom(bondDenom)
	require.NoError(t, err)

	api := keeper.NewApi(input.MoveKeeper, ctx)
	amount, err := api.ShareToAmount([]byte(valAddr.String()), metadata, 150)
	require.NoError(t, err)
	require.Equal(t, uint64(150), amount)
}

func Test_ShareToAmountAPI_InvalidAddr(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := valAddrs[0]
	valPubKey := valPubKeys[0]

	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddrs[0])
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))

	sh := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	_, err = sh.CreateValidator(ctx, newTestMsgCreateValidator(valAddrStr, valPubKey, math.NewInt(100_000)))
	require.NoError(t, err)

	metadata, err := types.MetadataAddressFromDenom(bondDenom)
	require.NoError(t, err)

	api := keeper.NewApi(input.MoveKeeper, ctx)
	_, err = api.ShareToAmount(valAddr, metadata, 150)
	require.Error(t, err)
}

func Test_UnbondTimestamp(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// set UnbondingTime
	stakingParams, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)

	stakingParams.UnbondingTime = time.Duration(60 * 60 * 24 * 7)
	input.StakingKeeper.SetParams(ctx, stakingParams)

	now := time.Now()
	api := keeper.NewApi(input.MoveKeeper, ctx.WithBlockTime(now))

	resTimestamp := api.UnbondTimestamp()
	require.Equal(t, uint64(now.Unix()+60*60*24*7), resTimestamp)
}
