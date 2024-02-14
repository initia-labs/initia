package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

func TestExportGenesis(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	expectedMetadata := getTestMetadata()
	expectedBalances, totalSupply := getTestBalancesAndSupply(input.Faucet.sender)
	for i := range []int{1, 2} {
		input.BankKeeper.SetDenomMetaData(ctx, expectedMetadata[i])
		accAddr, err1 := sdk.AccAddressFromBech32(expectedBalances[i].Address)
		if err1 != nil {
			panic(err1)
		}
		// set balances via mint and send
		require.NoError(t, input.BankKeeper.MintCoins(ctx, authtypes.Minter, expectedBalances[i].Coins))
		require.NoError(t, input.BankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.Minter, accAddr, expectedBalances[i].Coins))
	}
	input.BankKeeper.SetParams(ctx, types.DefaultParams())

	exportGenesis := input.BankKeeper.ExportGenesis(ctx)

	require.Len(t, exportGenesis.Params.SendEnabled, 0)
	require.Equal(t, types.DefaultParams().DefaultSendEnabled, exportGenesis.Params.DefaultSendEnabled)
	require.Equal(t, totalSupply, exportGenesis.Supply)
	require.Equal(t, expectedBalances, exportGenesis.Balances)
	require.Equal(t, expectedMetadata, exportGenesis.DenomMetadata)
}

func getTestBalancesAndSupply(faucetAddr sdk.AccAddress) ([]types.Balance, sdk.Coins) {
	addr1 := sdk.AccAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	addr2 := sdk.AccAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2})
	addr1Balance := sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 10))
	addr2Balance := sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 32), sdk.NewInt64Coin(testDenoms[0], 34))
	faucetBalance := initialTotalSupply()

	totalSupply := addr1Balance
	totalSupply = totalSupply.Add(addr2Balance...)
	totalSupply = totalSupply.Add(faucetBalance...)

	return []types.Balance{
		{Address: addr1.String(), Coins: addr1Balance},
		{Address: addr2.String(), Coins: addr2Balance},
		{Address: faucetAddr.String(), Coins: faucetBalance},
	}, totalSupply
}

func Test_InitGenesis(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	m := types.Metadata{Description: bondDenom, Base: bondDenom, Display: bondDenom}
	g := types.DefaultGenesisState()
	g.DenomMetadata = []types.Metadata{m}
	bk := input.BankKeeper
	bk.InitGenesis(ctx, g)

	m2, found := bk.GetDenomMetaData(ctx, m.Base)
	require.True(t, found)
	require.Equal(t, m, m2)
}

func TestTotalSupply(t *testing.T) {
	// Prepare some test data.
	defaultGenesis := types.DefaultGenesisState()
	balances := []types.Balance{
		{Coins: sdk.NewCoins(sdk.NewCoin("foocoin", math.NewInt(1))), Address: "cosmos1f9xjhxm0plzrh9cskf4qee4pc2xwp0n0556gh0"},
		{Coins: sdk.NewCoins(sdk.NewCoin("barcoin", math.NewInt(1))), Address: "cosmos1t5u0jfg3ljsjrh2m9e47d4ny2hea7eehxrzdgd"},
		{Coins: sdk.NewCoins(sdk.NewCoin("foocoin", math.NewInt(10)), sdk.NewCoin("barcoin", math.NewInt(20))), Address: "cosmos1m3h30wlvsf8llruxtpukdvsy0km2kum8g38c8q"},
	}
	totalSupply := sdk.NewCoins(sdk.NewCoin("foocoin", math.NewInt(11)), sdk.NewCoin("barcoin", math.NewInt(21)))

	testcases := map[string]struct {
		genesis   *types.GenesisState
		expSupply sdk.Coins
	}{
		"calculation matches genesis Supply field": {
			types.NewGenesisState(defaultGenesis.Params, balances, totalSupply, defaultGenesis.DenomMetadata, defaultGenesis.SendEnabled),
			totalSupply,
		},
		"calculation is correct, empty genesis Supply field": {
			types.NewGenesisState(defaultGenesis.Params, balances, nil, defaultGenesis.DenomMetadata, defaultGenesis.SendEnabled),
			totalSupply,
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			ctx, input := createDefaultTestInput(t)
			input.BankKeeper.InitGenesis(ctx, tc.genesis)
			totalSupply, _, err := input.BankKeeper.GetPaginatedTotalSupply(ctx, &query.PageRequest{Limit: query.PaginationMaxLimit})
			assert.NoError(t, err)

			// we need to exclude uinit and ustake due to faucet initial balance
			for _, coin := range tc.expSupply {
				assert.Equal(t, coin.Amount, totalSupply.AmountOf(coin.Denom))
			}
		})
	}
}
