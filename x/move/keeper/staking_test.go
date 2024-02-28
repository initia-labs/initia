package keeper_test

import (
	"testing"

	"cosmossdk.io/math"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	distrkeeper "github.com/initia-labs/initia/x/distribution/keeper"
	stakingkeeper "github.com/initia-labs/initia/x/mstaking/keeper"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

// newTestMsgCreateValidator test msg creator
func newTestMsgCreateValidator(valAddr string, pubKey cryptotypes.PubKey, amt math.Int) *stakingtypes.MsgCreateValidator {
	commission := stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDec(0))
	msg, _ := stakingtypes.NewMsgCreateValidator(
		valAddr, pubKey, sdk.NewCoins(sdk.NewCoin(bondDenom, amt)),
		stakingtypes.NewDescription("homeDir", "", "", "", ""), commission,
	)
	return msg
}

func createValidatorWithBalance(
	t *testing.T,
	ctx sdk.Context,
	input TestKeepers,
	balance int64,
	delBalance int64,
) sdk.ValAddress {
	valAddr := valAddrs[0]
	valPubKey := valPubKeys[0]

	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddrs[0])
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, math.NewInt(balance)))

	sh := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	_, err = sh.CreateValidator(ctx, newTestMsgCreateValidator(valAddrStr, valPubKey, math.NewInt(delBalance)))
	if err != nil {
		require.NoError(t, err)
	}

	// power update
	_, err = input.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	if err != nil {
		require.NoError(t, err)
	}

	return valAddr
}

// mint coins and supply the coins to distribution module account
// also allocate that coins to validator rewards pool
func setValidatorRewards(
	t *testing.T,
	ctx sdk.Context,
	faucet *TestFaucet,
	stakingKeeper stakingkeeper.Keeper,
	distKeeper distrkeeper.Keeper,
	valAddr sdk.ValAddress, rewards ...sdk.Coin) {

	// allocate some rewards
	validator, err := stakingKeeper.Validator(ctx, valAddr)
	require.NoError(t, err)

	payout := sdk.NewDecCoinsFromCoins(rewards...)
	distKeeper.AllocateTokensToValidatorPool(ctx, validator, bondDenom, payout)

	// allocate rewards to validator by minting tokens to distr module balance
	faucet.Fund(ctx, authtypes.NewModuleAddress(distrtypes.ModuleName), rewards...)
}

func TestDelegateToValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr := createValidatorWithBalance(t, ctx, input, 100_000_000, 100_000)

	input.Faucet.Fund(ctx, types.MoveStakingModuleAddress, sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))
	moveBalance := input.BankKeeper.GetBalance(ctx, types.MoveStakingModuleAddress, bondDenom).Amount.Uint64()
	require.Equal(t, uint64(100_000_000), moveBalance)

	_, err := input.MoveKeeper.DelegateToValidator(ctx, valAddr, sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100))))
	require.NoError(t, err)

	moveBalance = input.BankKeeper.GetBalance(ctx, types.MoveStakingModuleAddress, bondDenom).Amount.Uint64()
	require.Equal(t, uint64(99_999_900), moveBalance)

	_, err = input.MoveKeeper.DelegateToValidator(ctx, valAddr, sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100_000_000_000))))
	require.Error(t, err)
}

func TestAmountToShare(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr := createValidatorWithBalance(t, ctx, input, 100_000_000, 100_000)

	amount := sdk.NewCoin(bondDenom, math.NewInt(150))
	share, err := input.MoveKeeper.AmountToShare(ctx, valAddr, amount)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(150), share)
}

func TestShareToAmount(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr := createValidatorWithBalance(t, ctx, input, 100_000_000, 100_000)

	share := sdk.NewDecCoin(bondDenom, math.NewInt(150))
	amount, err := input.MoveKeeper.ShareToAmount(ctx, valAddr, share)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(150), amount)

}

func TestWithdrawRewards(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr := createValidatorWithBalance(t, ctx, input, 100_000_000, 1_000_000)

	// mint coins to move staking module
	input.Faucet.Fund(ctx, types.MoveStakingModuleAddress, sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))

	// delegate staking coins to validator
	delegationCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100_000)))
	_, err := input.MoveKeeper.DelegateToValidator(ctx, valAddr, delegationCoins)
	require.NoError(t, err)

	// withdraw zero rewards
	_, err = input.MoveKeeper.WithdrawRewards(ctx, valAddr)
	require.NoError(t, err)

	moveAccOriginBalance := input.BankKeeper.GetAllBalances(ctx, types.MoveStakingModuleAddress)

	var accRewards sdk.Coins
	for i := 0; i < 10; i++ {
		setValidatorRewards(t, ctx, input.Faucet, input.StakingKeeper, input.DistKeeper, valAddr, sdk.NewCoin(bondDenom, math.NewInt(100)))
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
		rewards, err := input.MoveKeeper.WithdrawRewards(ctx, valAddr)
		require.NoError(t, err)
		accRewards = accRewards.Add(rewards.Sum()...)
	}

	moveAccRewardedBalance := input.BankKeeper.GetAllBalances(ctx, types.MoveStakingModuleAddress)
	require.NotEqual(t, moveAccRewardedBalance, moveAccOriginBalance)
	require.Equal(t, moveAccRewardedBalance, moveAccOriginBalance.Add(accRewards...))
}

func TestInstantUnbondFromValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr := createValidatorWithBalance(t, ctx, input, 100_000_000, 100_000)

	input.Faucet.Fund(ctx, types.MoveStakingModuleAddress, sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))

	moveBalance := input.BankKeeper.GetBalance(ctx, types.MoveStakingModuleAddress, bondDenom).Amount.Uint64()
	require.Equal(t, uint64(100_000_000), moveBalance)

	delegationCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))
	_, err := input.MoveKeeper.DelegateToValidator(ctx, valAddr, delegationCoins)
	require.NoError(t, err)

	moveBalance = input.BankKeeper.GetBalance(ctx, types.MoveStakingModuleAddress, bondDenom).Amount.Uint64()
	require.Equal(t, uint64(0), moveBalance)

	undelegationShares := sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(100_000_000)))
	_, err = input.MoveKeeper.InstantUnbondFromValidator(ctx, valAddr, undelegationShares)
	require.NoError(t, err)

	moveBalance = input.BankKeeper.GetBalance(ctx, types.MoveStakingModuleAddress, bondDenom).Amount.Uint64()
	require.Equal(t, uint64(100_000_000), moveBalance)

	// not enough delegation balance
	_, err = input.MoveKeeper.InstantUnbondFromValidator(ctx, valAddr, undelegationShares)
	require.Error(t, err)
}

func TestInstantUnbondFromBondedValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr := createValidatorWithBalance(t, ctx, input, 100_000_000, 100_000)

	input.Faucet.Fund(ctx, types.MoveStakingModuleAddress, sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))

	moveBalance := input.BankKeeper.GetBalance(ctx, types.MoveStakingModuleAddress, bondDenom).Amount.Uint64()
	require.Equal(t, uint64(100_000_000), moveBalance)

	delegationCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))
	_, err := input.MoveKeeper.DelegateToValidator(ctx, valAddr, delegationCoins)
	require.NoError(t, err)

	undelegationShares := sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(50_000_000)))
	_, err = input.MoveKeeper.InstantUnbondFromValidator(ctx, valAddr, undelegationShares)
	require.NoError(t, err)

	_, err = input.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	val, _ := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, val.IsBonded())

	_, err = input.MoveKeeper.InstantUnbondFromValidator(ctx, valAddr, undelegationShares)
	require.NoError(t, err)

	_, err = input.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	val, _ = input.StakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, val.IsUnbonding())
}

func TestApplyStakingDeltas(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// create dex
	baseAmount := math.NewInt(1_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := math.NewInt(2_500_000_000_000)

	metadataLP := createDexPool(
		t, ctx, input,
		sdk.NewCoin(bondDenom, baseAmount), sdk.NewCoin(denomQuote, quoteAmount),
		math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1),
	)

	secondBondDenom, err := types.DenomFromMetadataAddress(ctx, keeper.NewMoveBankKeeper(&input.MoveKeeper), metadataLP)
	require.NoError(t, err)

	// add second BondDenom to staking keeper
	input.StakingKeeper.SetBondDenoms(ctx, []string{bondDenom, secondBondDenom})

	// initialize staking
	err = input.MoveKeeper.InitializeStaking(ctx, secondBondDenom)
	require.NoError(t, err)

	valAddr := createValidatorWithBalance(t, ctx, input, 100_000_000, 100_000)

	// mint not possible for lp coin, so transfer from the 0x2
	_, _, addr := keyPubAddr()
	require.NoError(t, input.BankKeeper.SendCoins(ctx, types.TestAddr, addr, sdk.NewCoins(sdk.NewCoin(secondBondDenom, math.NewInt(100_000_000)))))

	// delegate coins via move staking module
	valAddrArg, err := vmtypes.SerializeString(valAddr.String())
	require.NoError(t, err)

	amountArg, err := vmtypes.SerializeUint64(50_000_000)
	require.NoError(t, err)

	vmAddr, err := vmtypes.NewAccountAddressFromBytes(addr)
	require.NoError(t, err)
	err = input.MoveKeeper.ExecuteEntryFunction(
		ctx,
		vmAddr,
		vmtypes.StdAddress,
		types.MoveModuleNameStaking,
		types.FunctionNameStakingDelegate,
		[]vmtypes.TypeTag{},
		[][]byte{metadataLP[:], valAddrArg, amountArg},
	)
	require.NoError(t, err)

	delModuleAddr := types.GetDelegatorModuleAddress(valAddr)
	delegation, err := input.StakingKeeper.GetDelegation(ctx, delModuleAddr, valAddr)
	require.NoError(t, err)
	require.Equal(t, delegation.Shares, sdk.NewDecCoins(sdk.NewDecCoin(secondBondDenom, math.NewInt(50_000_000))))

	// undelegate half
	halfAmountArg, err := vmtypes.SerializeUint64(25_000_000)
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunction(
		ctx,
		vmAddr,
		vmtypes.StdAddress,
		types.MoveModuleNameStaking,
		types.FunctionNameStakingUndelegate,
		[]vmtypes.TypeTag{},
		[][]byte{metadataLP[:], valAddrArg, halfAmountArg},
	)
	require.NoError(t, err)

	moveBalance := input.BankKeeper.GetBalance(ctx, types.MoveStakingModuleAddress, secondBondDenom).Amount.Uint64()
	require.Equal(t, uint64(0), moveBalance)

	delegation, err = input.StakingKeeper.GetDelegation(ctx, delModuleAddr, valAddr)
	require.NoError(t, err)
	require.Equal(t, delegation.Shares, sdk.NewDecCoins(sdk.NewDecCoin(secondBondDenom, math.NewInt(25_000_000))))

	// check staking state
	tableHandle, err := input.MoveKeeper.GetStakingStatesTableHandle(ctx)
	require.NoError(t, err)

	// read metadata entry
	tableEntry, err := input.MoveKeeper.GetTableEntryBytes(ctx, tableHandle, metadataLP[:])
	require.NoError(t, err)
	metadataTableHandle, err := types.ReadTableHandleFromTable(tableEntry.ValueBytes)
	require.NoError(t, err)

	// read validator entry
	keyBz, err := vmtypes.SerializeString(valAddr.String())
	require.NoError(t, err)
	tableEntry, err = input.MoveKeeper.GetTableEntry(ctx, metadataTableHandle, keyBz)
	require.NoError(t, err)

	unbondingShare, unbondingCoinStore, err := types.ReadUnbondingInfosFromStakingState(tableEntry.ValueBytes)
	require.NoError(t, err)
	require.Equal(t, unbondingShare, math.NewInt(25_000_000))

	_, unbondingAmount, err := keeper.NewMoveBankKeeper(&input.MoveKeeper).Balance(ctx, unbondingCoinStore)
	require.NoError(t, err)
	require.Equal(t, unbondingAmount, math.NewInt(25_000_000))
}

func Test_SlashUnbondingDelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// create dex
	baseAmount := math.NewInt(1_000_000_000_000)

	denomQuote := "uusdc"
	quoteAmount := math.NewInt(2_500_000_000_000)

	metadataLP := createDexPool(
		t, ctx, input,
		sdk.NewCoin(bondDenom, baseAmount), sdk.NewCoin(denomQuote, quoteAmount),
		math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1),
	)

	secondBondDenom, err := types.DenomFromMetadataAddress(ctx, keeper.NewMoveBankKeeper(&input.MoveKeeper), metadataLP)
	require.NoError(t, err)

	// add second BondDenom to staking keeper
	input.StakingKeeper.SetBondDenoms(ctx, []string{bondDenom, secondBondDenom})

	// initialize staking
	err = input.MoveKeeper.InitializeStaking(ctx, secondBondDenom)
	require.NoError(t, err)

	valAddr := createValidatorWithBalance(t, ctx, input, 100_000_000, 100_000)

	// mint not possible for lp coin, so transfer from the 0x1
	_, _, addr := keyPubAddr()
	require.NoError(t, input.BankKeeper.SendCoins(ctx, types.TestAddr, addr, sdk.NewCoins(sdk.NewCoin(secondBondDenom, math.NewInt(100_000_000)))))

	// delegate coins through move staking module
	valAddrArg, err := vmtypes.SerializeString(valAddr.String())
	require.NoError(t, err)

	amountArg, err := vmtypes.SerializeUint64(50_000_000)
	require.NoError(t, err)

	vmAddr, err := vmtypes.NewAccountAddressFromBytes(addr)
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunction(
		ctx,
		vmAddr,
		vmtypes.StdAddress,
		types.MoveModuleNameStaking,
		types.FunctionNameStakingDelegate,
		[]vmtypes.TypeTag{},
		[][]byte{metadataLP[:], valAddrArg, amountArg},
	)
	require.NoError(t, err)

	delModuleAddr := types.GetDelegatorModuleAddress(valAddr)
	delegation, err := input.StakingKeeper.GetDelegation(ctx, delModuleAddr, valAddr)
	require.NoError(t, err)
	require.Equal(t, delegation.Shares, sdk.NewDecCoins(sdk.NewDecCoin(secondBondDenom, math.NewInt(50_000_000))))

	// undelegate half
	halfAmountArg, err := vmtypes.SerializeUint64(25_000_000)
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunction(
		ctx,
		vmAddr,
		vmtypes.StdAddress,
		types.MoveModuleNameStaking,
		types.FunctionNameStakingUndelegate,
		[]vmtypes.TypeTag{},
		[][]byte{metadataLP[:], valAddrArg, halfAmountArg},
	)
	require.NoError(t, err)

	moveBalance := input.BankKeeper.GetBalance(ctx, types.MoveStakingModuleAddress, secondBondDenom).Amount.Uint64()
	require.Equal(t, uint64(0), moveBalance)

	delegation, err = input.StakingKeeper.GetDelegation(ctx, delModuleAddr, valAddr)
	require.NoError(t, err)
	require.Equal(t, delegation.Shares, sdk.NewDecCoins(sdk.NewDecCoin(secondBondDenom, math.NewInt(25_000_000))))

	// slash 5%
	input.MoveKeeper.Hooks().SlashUnbondingDelegations(ctx, valAddr, math.LegacyNewDecWithPrec(5, 2))

	// check staking state
	tableHandle, err := input.MoveKeeper.GetStakingStatesTableHandle(ctx)
	require.NoError(t, err)

	// read metadata entry
	tableEntry, err := input.MoveKeeper.GetTableEntryBytes(ctx, tableHandle, metadataLP[:])
	require.NoError(t, err)
	metadataTableHandle, err := types.ReadTableHandleFromTable(tableEntry.ValueBytes)
	require.NoError(t, err)

	// read validator entry
	keyBz, err := vmtypes.SerializeString(valAddr.String())
	require.NoError(t, err)
	tableEntry, err = input.MoveKeeper.GetTableEntry(ctx, metadataTableHandle, keyBz)
	require.NoError(t, err)

	unbondingShare, unbondingCoinStore, err := types.ReadUnbondingInfosFromStakingState(tableEntry.ValueBytes)
	require.NoError(t, err)

	_, unbondingAmount, err := keeper.NewMoveBankKeeper(&input.MoveKeeper).Balance(ctx, unbondingCoinStore)
	require.NoError(t, err)
	require.Equal(t, unbondingAmount, math.NewInt(23_750_000))
	require.Equal(t, unbondingShare, math.NewInt(25_000_000))
}
