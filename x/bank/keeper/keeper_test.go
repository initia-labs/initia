package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/stretchr/testify/require"
)

func getCoinsByName(ctx sdk.Context, bk keeper.Keeper, ak types.AccountKeeper, moduleName string) sdk.Coins {
	moduleAddress := ak.GetModuleAddress(moduleName)
	macc := ak.GetAccount(ctx, moduleAddress)
	if macc == nil {
		return sdk.Coins(nil)
	}

	return bk.GetAllBalances(ctx, macc.GetAddress())
}

func getTestMetadata() []types.Metadata {
	return []types.Metadata{
		{
			Name:        "Initia INIT",
			Symbol:      "INIT",
			Description: "The native reward token of the Initia Network.",
			DenomUnits: []*types.DenomUnit{
				{Denom: "uinit", Exponent: uint32(0), Aliases: []string{"microinit"}},
				{Denom: "minit", Exponent: uint32(3), Aliases: []string{"milliinit"}},
				{Denom: "init", Exponent: uint32(6), Aliases: nil},
			},
			Base:    "uinit",
			Display: "init",
		},
		{
			Name:        "Initia LP",
			Symbol:      "LP",
			Description: "The native staking token of the Initia Network.",
			DenomUnits: []*types.DenomUnit{
				{Denom: "ulp", Exponent: uint32(0), Aliases: []string{"microlp"}},
				{Denom: "mlp", Exponent: uint32(3), Aliases: []string{"milliop"}},
				{Denom: "lp", Exponent: uint32(6), Aliases: nil},
			},
			Base:    "ulp",
			Display: "lp",
		},
	}
}

func TestSupply(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	totalSupply := initialTotalSupply()
	require.NoError(t, input.BankKeeper.MintCoins(ctx, authtypes.Minter, totalSupply))

	total, _, err := input.BankKeeper.GetPaginatedTotalSupply(ctx, &query.PageRequest{})
	require.NoError(t, err)
	require.Equal(t, totalSupply.Add(totalSupply...), total)

	// burning all supplied tokens
	// BurnCoins is not actually burning the token, but transfer to community pool
	err = input.BankKeeper.BurnCoins(ctx, authtypes.Minter, totalSupply)
	require.NoError(t, err)

	total, _, err = input.BankKeeper.GetPaginatedTotalSupply(ctx, &query.PageRequest{})
	require.NoError(t, err)
	require.Equal(t, totalSupply, total)
}

func TestSendCoinsFromModuleToAccount_Blacklist(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	testDenom := testDenoms[0]
	coins := sdk.NewCoins(sdk.NewCoin(testDenom, math.NewInt(100)))
	require.NoError(t, input.BankKeeper.MintCoins(ctx, authtypes.Minter, coins))
	require.Error(t, input.BankKeeper.SendCoinsFromModuleToAccount(
		ctx, authtypes.Minter, authtypes.NewModuleAddress(authtypes.FeeCollectorName), coins,
	))
}

func TestSupply_SendCoins(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	testDenom := testDenoms[0]
	coins := sdk.NewCoins(sdk.NewCoin(testDenom, math.NewInt(100)))
	input.BankKeeper.MintCoins(ctx, authtypes.Minter, coins)

	require.Panics(t, func() {
		_ = input.BankKeeper.SendCoinsFromModuleToModule(ctx, "", authtypes.Minter, coins) // nolint:errcheck
	})

	require.Panics(t, func() {
		_ = input.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, "", coins) // nolint:errcheck
	})

	require.Panics(t, func() {
		_ = input.BankKeeper.SendCoinsFromModuleToAccount(ctx, "", addrs[0], coins) // nolint:errcheck
	})

	// not enough balance
	require.Error(t, input.BankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.Minter, addrs[0], coins.Add(coins...)))

	require.NoError(t, input.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, authtypes.FeeCollectorName, coins))
	require.Equal(t, sdk.NewCoins().String(), getCoinsByName(ctx, input.BankKeeper, input.AccountKeeper, authtypes.Minter).String())
	require.Equal(t, coins, getCoinsByName(ctx, input.BankKeeper, input.AccountKeeper, authtypes.FeeCollectorName))

	require.NoError(t, input.BankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.FeeCollectorName, addrs[0], coins))
	require.Equal(t, sdk.NewCoins().String(), getCoinsByName(ctx, input.BankKeeper, input.AccountKeeper, authtypes.FeeCollectorName).String())
	require.Equal(t, coins, input.BankKeeper.GetAllBalances(ctx, addrs[0]))

	require.NoError(t, input.BankKeeper.SendCoinsFromAccountToModule(ctx, addrs[0], authtypes.FeeCollectorName, coins))
	require.Equal(t, sdk.NewCoins().String(), input.BankKeeper.GetAllBalances(ctx, addrs[0]).String())
	require.Equal(t, coins, getCoinsByName(ctx, input.BankKeeper, input.AccountKeeper, authtypes.FeeCollectorName))
}

func TestSupply_MintCoins(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	initialSupply, _, err := input.BankKeeper.GetPaginatedTotalSupply(ctx, &query.PageRequest{})
	require.NoError(t, err)

	testDenom := testDenoms[0]
	coins := sdk.NewCoins(sdk.NewCoin(testDenom, math.NewInt(100)))
	require.Panics(t, func() { input.BankKeeper.MintCoins(ctx, "", coins) }, "no module account")                          // nolint:errcheck
	require.Panics(t, func() { input.BankKeeper.MintCoins(ctx, authtypes.FeeCollectorName, coins) }, "invalid permission") // nolint:errcheck
	require.Panics(t, func() {
		input.BankKeeper.MintCoins(ctx, authtypes.Minter, sdk.Coins{sdk.Coin{Denom: "denom", Amount: math.NewInt(-10)}})
	}) // nolint:errcheck

	err = input.BankKeeper.MintCoins(ctx, authtypes.Minter, coins)
	require.NoError(t, err)

	require.Equal(t, coins, getCoinsByName(ctx, input.BankKeeper, input.AccountKeeper, authtypes.Minter))
	totalSupply, _, err := input.BankKeeper.GetPaginatedTotalSupply(ctx, &query.PageRequest{})
	require.NoError(t, err)

	require.Equal(t, initialSupply.Add(coins...), totalSupply)
}

func TestSupply_BurnCoins(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	testDenom := testDenoms[0]
	coins := sdk.NewCoins(sdk.NewCoin(testDenom, math.NewInt(100)))

	// inflate supply
	require.
		NoError(t, input.BankKeeper.MintCoins(ctx, authtypes.Minter, coins))
	supplyAfterInflation, _, err := input.BankKeeper.GetPaginatedTotalSupply(ctx, &query.PageRequest{})
	require.NoError(t, err)

	require.Panics(t, func() { input.BankKeeper.BurnCoins(ctx, "", coins) }, "no module account")                          // nolint:errcheck
	require.Panics(t, func() { input.BankKeeper.BurnCoins(ctx, authtypes.FeeCollectorName, coins) }, "invalid permission") // nolint:errcheck
	err = input.BankKeeper.BurnCoins(ctx, authtypes.Minter, supplyAfterInflation)
	require.Error(t, err, "insufficient coins")

	err = input.BankKeeper.BurnCoins(ctx, authtypes.Minter, coins)
	require.NoError(t, err)
	supplyAfterBurn, _, err := input.BankKeeper.GetPaginatedTotalSupply(ctx, &query.PageRequest{})
	require.NoError(t, err)
	require.Equal(t, sdk.NewCoins().String(), getCoinsByName(ctx, input.BankKeeper, input.AccountKeeper, authtypes.Minter).String())
	require.Equal(t, supplyAfterInflation, supplyAfterBurn.Add(coins...))
}

func TestSendCoinsNewAccount(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	testDenom := testDenoms[0]
	balances := sdk.NewCoins(sdk.NewCoin(testDenom, math.NewInt(100)))

	addr1 := sdk.AccAddress([]byte("addr1_______________"))
	acc1 := input.AccountKeeper.NewAccountWithAddress(ctx, addr1)
	input.AccountKeeper.SetAccount(ctx, acc1)
	input.Faucet.Fund(ctx, addr1, balances...)

	acc1Balances := input.BankKeeper.GetAllBalances(ctx, addr1)
	require.Equal(t, balances, acc1Balances)

	addr2 := sdk.AccAddress([]byte("addr2_______________"))

	require.Nil(t, input.AccountKeeper.GetAccount(ctx, addr2))
	input.BankKeeper.GetAllBalances(ctx, addr2)
	require.Empty(t, input.BankKeeper.GetAllBalances(ctx, addr2))

	sendAmt := sdk.NewCoins(sdk.NewCoin(testDenom, math.NewInt(50)))
	require.NoError(t, input.BankKeeper.SendCoins(ctx, addr1, addr2, sendAmt))

	acc2Balances := input.BankKeeper.GetAllBalances(ctx, addr2)
	acc1Balances = input.BankKeeper.GetAllBalances(ctx, addr1)
	require.Equal(t, sendAmt, acc2Balances)
	updatedAcc1Bal := balances.Sub(sendAmt...)
	require.Len(t, acc1Balances, len(updatedAcc1Bal))
	require.Equal(t, acc1Balances, updatedAcc1Bal)
	require.NotNil(t, input.AccountKeeper.GetAccount(ctx, addr2))
}

func TestSendCoins(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	testDenom := testDenoms[0]
	balances := sdk.NewCoins(sdk.NewCoin(testDenom, math.NewInt(100)), sdk.NewCoin(bondDenom, math.NewInt(100)))

	addr1 := sdk.AccAddress("addr1_______________")
	acc1 := input.AccountKeeper.NewAccountWithAddress(ctx, addr1)
	input.AccountKeeper.SetAccount(ctx, acc1)

	addr2 := sdk.AccAddress("addr2_______________")
	acc2 := input.AccountKeeper.NewAccountWithAddress(ctx, addr2)
	input.AccountKeeper.SetAccount(ctx, acc2)
	input.Faucet.Fund(ctx, addr2, balances...)

	sendAmt := sdk.NewCoins(sdk.NewCoin(testDenom, math.NewInt(50)), sdk.NewCoin(bondDenom, math.NewInt(25)))
	require.Error(t, input.BankKeeper.SendCoins(ctx, addr1, addr2, sendAmt))

	input.Faucet.Fund(ctx, addr1, balances...)
	require.NoError(t, input.BankKeeper.SendCoins(ctx, addr1, addr2, sendAmt))

	acc1Balances := input.BankKeeper.GetAllBalances(ctx, addr1)
	expected := sdk.NewCoins(sdk.NewCoin(testDenom, math.NewInt(50)), sdk.NewCoin(bondDenom, math.NewInt(75)))
	require.Equal(t, expected, acc1Balances)

	acc2Balances := input.BankKeeper.GetAllBalances(ctx, addr2)
	expected = sdk.NewCoins(sdk.NewCoin(testDenom, math.NewInt(150)), sdk.NewCoin(bondDenom, math.NewInt(125)))
	require.Equal(t, expected, acc2Balances)

	var coins sdk.Coins
	input.BankKeeper.IterateAccountBalances(ctx, addr1, func(c sdk.Coin) (stop bool) {
		coins = append(coins, c)
		return false
	})

	require.Len(t, coins, 2)
	require.Equal(t, acc1Balances, coins.Sort(), "expected only bar coins in the account balance, got: %v", coins)
}

func TestValidateBalance(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	addr1 := sdk.AccAddress([]byte("addr1_______________"))
	require.NoError(t, input.BankKeeper.ValidateBalance(ctx, addr1))

	acc := input.AccountKeeper.NewAccountWithAddress(ctx, addr1)
	input.AccountKeeper.SetAccount(ctx, acc)

	balances := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	input.Faucet.Fund(ctx, addr1, balances...)
	require.NoError(t, input.BankKeeper.ValidateBalance(ctx, addr1))
}

func TestSendEnabled(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	enabled := true
	params := types.DefaultParams()
	require.Equal(t, enabled, params.DefaultSendEnabled)

	input.BankKeeper.SetParams(ctx, params)

	bondCoin := sdk.NewCoin(bondDenom, sdk.OneInt())

	testDenom := testDenoms[0]
	testCoin := sdk.NewCoin(testDenom, sdk.OneInt())

	// assert with default (all denom) send enabled both Bar and Bond Denom are enabled
	require.Equal(t, enabled, input.BankKeeper.IsSendEnabledCoin(ctx, testCoin))
	require.Equal(t, enabled, input.BankKeeper.IsSendEnabledCoin(ctx, bondCoin))

	// Both coins should be send enabled.
	err := input.BankKeeper.IsSendEnabledCoins(ctx, testCoin, bondCoin)
	require.NoError(t, err)

	// Set default send_enabled to !enabled, add a reward denom that overrides default as enabled
	params.DefaultSendEnabled = !enabled
	require.NoError(t, input.BankKeeper.SetParams(ctx, params))
	input.BankKeeper.SetSendEnabled(ctx, testCoin.Denom, enabled)

	// Expect our specific override to be enabled, others to be !enabled.
	require.Equal(t, enabled, input.BankKeeper.IsSendEnabledCoin(ctx, testCoin))
	require.Equal(t, !enabled, input.BankKeeper.IsSendEnabledCoin(ctx, bondCoin))

	// Foo coin should be send enabled.
	err = input.BankKeeper.IsSendEnabledCoins(ctx, testCoin)
	require.NoError(t, err)

	// Expect an error when one coin is not send enabled.
	err = input.BankKeeper.IsSendEnabledCoins(ctx, testCoin, bondCoin)
	require.Error(t, err)

	// Expect an error when all coins are not send enabled.
	err = input.BankKeeper.IsSendEnabledCoins(ctx, bondCoin)
	require.Error(t, err)
}

func TestHasBalance(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	addr := sdk.AccAddress([]byte("addr1_______________"))

	acc := input.AccountKeeper.NewAccountWithAddress(ctx, addr)
	input.AccountKeeper.SetAccount(ctx, acc)

	testDenom := testDenoms[0]
	balances := sdk.NewCoins(sdk.NewCoin(testDenom, math.NewInt(100)))
	require.False(t, input.BankKeeper.HasBalance(ctx, addr, sdk.NewCoin(testDenom, math.NewInt(99))))

	input.Faucet.Fund(ctx, addr, balances...)
	require.False(t, input.BankKeeper.HasBalance(ctx, addr, sdk.NewCoin(testDenom, math.NewInt(101))))
	require.True(t, input.BankKeeper.HasBalance(ctx, addr, sdk.NewCoin(testDenom, math.NewInt(100))))
	require.True(t, input.BankKeeper.HasBalance(ctx, addr, sdk.NewCoin(testDenom, math.NewInt(1))))
}

func TestMsgSendEvents(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	addr := sdk.AccAddress([]byte("addr1_______________"))
	addr2 := sdk.AccAddress([]byte("addr2_______________"))
	acc := input.AccountKeeper.NewAccountWithAddress(ctx, addr)

	input.AccountKeeper.SetAccount(ctx, acc)

	testDenom := testDenoms[0]
	newCoins := sdk.NewCoins(sdk.NewInt64Coin(testDenom, 50))
	input.Faucet.Fund(ctx, addr, newCoins...)

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, input.BankKeeper.SendCoins(ctx, addr, addr2, newCoins))
	event1 := sdk.Event{
		Type:       types.EventTypeTransfer,
		Attributes: []abci.EventAttribute{},
	}
	event1.Attributes = append(
		event1.Attributes,
		abci.EventAttribute{Key: types.AttributeKeyRecipient, Value: addr2.String()},
	)
	event1.Attributes = append(
		event1.Attributes,
		abci.EventAttribute{Key: types.AttributeKeySender, Value: addr.String()},
	)
	event1.Attributes = append(
		event1.Attributes,
		abci.EventAttribute{Key: sdk.AttributeKeyAmount, Value: newCoins.String()},
	)

	event2 := sdk.Event{
		Type:       sdk.EventTypeMessage,
		Attributes: []abci.EventAttribute{},
	}
	event2.Attributes = append(
		event2.Attributes,
		abci.EventAttribute{Key: types.AttributeKeySender, Value: addr.String()},
	)

	events := ctx.EventManager().ABCIEvents()
	require.Equal(t, abci.Event(event1), events[len(events)-2])
	require.Equal(t, abci.Event(event2), events[len(events)-1])
}

func TestSetDenomMetaData(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	metadata := getTestMetadata()

	for i := range []int{1, 2} {
		input.BankKeeper.SetDenomMetaData(ctx, metadata[i])
	}

	actualMetadata, found := input.BankKeeper.GetDenomMetaData(ctx, metadata[1].Base)
	require.True(t, found)
	require.Equal(t, metadata[1].GetBase(), actualMetadata.GetBase())
	require.Equal(t, metadata[1].GetDisplay(), actualMetadata.GetDisplay())
	require.Equal(t, metadata[1].GetDescription(), actualMetadata.GetDescription())
	require.Equal(t, metadata[1].GetDenomUnits()[1].GetDenom(), actualMetadata.GetDenomUnits()[1].GetDenom())
	require.Equal(t, metadata[1].GetDenomUnits()[1].GetExponent(), actualMetadata.GetDenomUnits()[1].GetExponent())
	require.Equal(t, metadata[1].GetDenomUnits()[1].GetAliases(), actualMetadata.GetDenomUnits()[1].GetAliases())
}

func TestIterateAllDenomMetaData(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	expectedMetadata := getTestMetadata()
	// set metadata
	for i := range []int{1, 2} {
		input.BankKeeper.SetDenomMetaData(ctx, expectedMetadata[i])
	}
	// retrieve metadata
	actualMetadata := make([]types.Metadata, 0)
	input.BankKeeper.IterateAllDenomMetaData(ctx, func(metadata types.Metadata) bool {
		actualMetadata = append(actualMetadata, metadata)
		return false
	})
	// execute checks
	for i := range []int{1, 2} {
		require.Equal(t, expectedMetadata[i].GetBase(), actualMetadata[i].GetBase())
		require.Equal(t, expectedMetadata[i].GetDisplay(), actualMetadata[i].GetDisplay())
		require.Equal(t, expectedMetadata[i].GetDescription(), actualMetadata[i].GetDescription())
		require.Equal(t, expectedMetadata[i].GetDenomUnits()[1].GetDenom(), actualMetadata[i].GetDenomUnits()[1].GetDenom())
		require.Equal(t, expectedMetadata[i].GetDenomUnits()[1].GetExponent(), actualMetadata[i].GetDenomUnits()[1].GetExponent())
		require.Equal(t, expectedMetadata[i].GetDenomUnits()[1].GetAliases(), actualMetadata[i].GetDenomUnits()[1].GetAliases())
	}
}
