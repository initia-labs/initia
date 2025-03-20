package move_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func Test_BeginBlocker(t *testing.T) {
	app := createApp(t)

	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)

	// initialize staking for secondBondDenom
	ctx := app.BaseApp.NewUncachedContext(false, tmproto.Header{})
	err = app.MoveKeeper.InitializeStaking(ctx, secondBondDenom)
	require.NoError(t, err)

	// fund addr2
	app.BankKeeper.SendCoins(ctx, types.StdAddr, addr2, sdk.NewCoins(secondBondCoin))

	_, err = app.Commit()
	require.NoError(t, err)

	// delegate coins via move staking module
	denomLP := "ulp"
	metadataLP := types.NamedObjectAddress(vmtypes.StdAddress, denomLP)

	valAddrArg, err := vmtypes.SerializeString(sdk.ValAddress(addr1).String())
	require.NoError(t, err)

	amountArg, err := vmtypes.SerializeUint64(secondBondCoin.Amount.Uint64())
	require.NoError(t, err)

	delegateMsg := types.MsgExecute{
		Sender:        addr2.String(),
		ModuleAddress: types.StdAddr.String(),
		ModuleName:    types.MoveModuleNameStaking,
		FunctionName:  types.FunctionNameStakingDelegateScript,
		TypeArgs:      []string{},
		Args:          [][]byte{metadataLP[:], valAddrArg, amountArg},
	}

	err = executeMsgs(t, app, []sdk.Msg{&delegateMsg}, []uint64{1}, []uint64{0}, priv2)
	require.NoError(t, err)

	// check balance
	checkBalance(t, app, types.MoveStakingModuleAddress, sdk.Coins{})

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)

	// generate rewards
	ctx = app.BaseApp.NewUncachedContext(false, tmproto.Header{})
	validator, err := app.StakingKeeper.Validator(ctx, sdk.ValAddress(addr1))
	require.NoError(t, err)

	rewardCoins := sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 1_000_000))
	delegatorRewardCoins := sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 500_000))

	err = app.BankKeeper.MintCoins(ctx, authtypes.Minter, rewardCoins)
	require.NoError(t, err)
	err = app.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, distrtypes.ModuleName, rewardCoins)
	require.NoError(t, err)
	app.DistrKeeper.AllocateTokensToValidatorPool(
		ctx,
		validator,
		secondBondDenom,
		sdk.NewDecCoinsFromCoins(rewardCoins...))

	_, err = app.Commit()
	require.NoError(t, err)

	// rewards distributed
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)

	_, err = app.Commit()
	require.NoError(t, err)

	// withdraw rewards to move module
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)

	// undelegate coins
	undelegateMsg := types.MsgExecute{
		Sender:        addr2.String(),
		ModuleAddress: types.StdAddr.String(),
		ModuleName:    types.MoveModuleNameStaking,
		FunctionName:  types.FunctionNameStakingUndelegateScript,
		TypeArgs:      []string{},
		Args:          [][]byte{metadataLP[:], valAddrArg, amountArg},
	}

	err = executeMsgs(t, app, []sdk.Msg{&undelegateMsg}, []uint64{1}, []uint64{1}, priv2)
	require.NoError(t, err)

	// half rewards and undelegated coins
	checkBalance(t, app, addr2, genCoins.Add(delegatorRewardCoins...))
}

func Test_EndBlocker(t *testing.T) {
	app := createApp(t)

	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)

	ctx := app.BaseApp.NewUncachedContext(false, tmproto.Header{})
	err = app.MoveKeeper.EIP1559FeeKeeper().SetParams(ctx, types.EIP1559FeeParams{
		BaseFee:       10_000,
		MinBaseFee:    1_000,
		MaxBaseFee:    10_000_000,
		TargetGas:     1_000_000,
		MaxChangeRate: math.LegacyNewDecWithPrec(1, 1),
	})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)

	// initialize staking for secondBondDenom
	ctx = app.BaseApp.NewUncachedContext(false, tmproto.Header{})
	err = app.MoveKeeper.InitializeStaking(ctx, secondBondDenom)
	require.NoError(t, err)

	// fund addr2
	app.BankKeeper.SendCoins(ctx, types.StdAddr, addr2, sdk.NewCoins(secondBondCoin))

	_, err = app.Commit()
	require.NoError(t, err)

	// delegate coins via move staking module
	denomLP := "ulp"
	metadataLP := types.NamedObjectAddress(vmtypes.StdAddress, denomLP)

	valAddrArg, err := vmtypes.SerializeString(sdk.ValAddress(addr1).String())
	require.NoError(t, err)

	amountArg, err := vmtypes.SerializeUint64(10)
	require.NoError(t, err)

	delegateMsg := types.MsgExecute{
		Sender:        addr2.String(),
		ModuleAddress: types.StdAddr.String(),
		ModuleName:    types.MoveModuleNameStaking,
		FunctionName:  types.FunctionNameStakingDelegateScript,
		TypeArgs:      []string{},
		Args:          [][]byte{metadataLP[:], valAddrArg, amountArg},
	}

	_, err = executeMsgsWithGasInfo(t, app, []sdk.Msg{&delegateMsg}, []uint64{1}, []uint64{0}, priv2)
	require.NoError(t, err)

	ctx = app.BaseApp.NewUncachedContext(false, tmproto.Header{})
	lessBaseFee, err := app.MoveKeeper.EIP1559FeeKeeper().GetBaseFee(ctx)
	require.NoError(t, err)
	require.Less(t, lessBaseFee, types.DefaultBaseFee)

	msgs := []sdk.Msg{}
	for i := 0; i < 100; i++ {
		msgs = append(msgs, &banktypes.MsgSend{
			FromAddress: addr2.String(),
			ToAddress:   addr1.String(),
			Amount:      sdk.NewCoins(sdk.NewInt64Coin(secondBondDenom, 10)),
		})
	}

	_, err = executeMsgsWithGasInfo(t, app, msgs, []uint64{1}, []uint64{1}, priv2)
	require.NoError(t, err)

	ctx = app.BaseApp.NewUncachedContext(false, tmproto.Header{})
	baseFee, err := app.MoveKeeper.EIP1559FeeKeeper().GetBaseFee(ctx)
	require.NoError(t, err)
	require.Greater(t, baseFee, lessBaseFee)
}
