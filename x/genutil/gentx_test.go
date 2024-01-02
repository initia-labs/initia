package genutil_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	initiaapp "github.com/initia-labs/initia/app"
	"github.com/initia-labs/initia/x/genutil"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

func TestSetGenTxsInAppGenesisState(t *testing.T) {
	var (
		txBuilder = initiaapp.MakeEncodingConfig().TxConfig.NewTxBuilder()
		desc      = stakingtypes.NewDescription("testname", "", "", "", "")
		comm      = stakingtypes.CommissionRates{}
		genTxs    []sdk.Tx
	)

	msg1, err := stakingtypes.NewMsgCreateValidator(
		sdk.ValAddress(valPubKeys[0].Address()).String(), valPubKeys[0], sdk.NewCoins(bondCoin), desc, comm)
	require.NoError(t, err)

	msg2, err := stakingtypes.NewMsgCreateValidator(
		sdk.ValAddress(valPubKeys[1].Address()).String(), valPubKeys[1], sdk.NewCoins(bondCoin), desc, comm)
	require.NoError(t, err)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"one genesis transaction",
			func() {
				err := txBuilder.SetMsgs(msg1)
				require.NoError(t, err)
				tx := txBuilder.GetTx()
				genTxs = []sdk.Tx{tx}
			},
			true,
		},
		{
			"two genesis transactions",
			func() {
				err := txBuilder.SetMsgs(msg1, msg2)
				require.NoError(t, err)
				tx := txBuilder.GetTx()
				genTxs = []sdk.Tx{tx}
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.msg), func(t *testing.T) {
			cdc := initiaapp.MakeEncodingConfig().Marshaler
			txJSONEncoder := initiaapp.MakeEncodingConfig().TxConfig.TxJSONEncoder()

			tc.malleate()
			appGenesisState, err := genutil.SetGenTxsInAppGenesisState(cdc, txJSONEncoder, make(map[string]json.RawMessage), genTxs)

			if tc.expPass {
				require.NoError(t, err)
				require.NotNil(t, appGenesisState[cosmostypes.ModuleName])

				var genesisState cosmostypes.GenesisState
				err := cdc.UnmarshalJSON(appGenesisState[cosmostypes.ModuleName], &genesisState)
				require.NoError(t, err)
				require.NotNil(t, genesisState.GenTxs)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestValidateAccountInGenesis(t *testing.T) {
	var (
		appGenesisState = make(map[string]json.RawMessage)
		coins           sdk.Coins
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"no accounts",
			func() {
				coins = sdk.Coins{sdk.NewInt64Coin(bondDenom, 0)}
			},
			false,
		},
		{
			"account without balance in the genesis state",
			func() {
				coins = sdk.Coins{sdk.NewInt64Coin(bondDenom, 0)}
				appGenesisState[banktypes.ModuleName] = setAccountBalance(t, addr2, sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(50))))
			},
			false,
		},
		{
			"account without enough funds of default bond denom",
			func() {
				coins = sdk.Coins{sdk.NewInt64Coin(bondDenom, 50)}
				appGenesisState[banktypes.ModuleName] = setAccountBalance(t, addr1, sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(25))))
			},
			false,
		},
		{
			"account with enough funds of default bond denom",
			func() {
				coins = sdk.Coins{sdk.NewInt64Coin(bondDenom, 10)}
				appGenesisState[banktypes.ModuleName] = setAccountBalance(t, addr1, sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(25))))
			},
			true,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.msg), func(t *testing.T) {
			app := createApp(t)
			ctx := app.BaseApp.NewContext(false)
			cdc := initiaapp.MakeEncodingConfig().Marshaler

			stakingGenesisState := app.StakingKeeper.ExportGenesis(ctx)
			stakingGenesis, err := cdc.MarshalJSON(stakingGenesisState) // TODO switch this to use Marshaler
			require.NoError(t, err)
			appGenesisState[stakingtypes.ModuleName] = stakingGenesis

			tc.malleate()
			err = genutil.ValidateAccountInGenesis(
				appGenesisState, banktypes.GenesisBalancesIterator{},
				addr1, coins, cdc,
			)

			if tc.expPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// func (suite *GenTxTestSuite) TestDeliverGenTxs() {
// 	var (
// 		genTxs    []json.RawMessage
// 		txBuilder = suite.encodingConfig.TxConfig.NewTxBuilder()
// 	)

// 	testCases := []struct {
// 		msg      string
// 		malleate func()
// 		expPass  bool
// 	}{
// 		{
// 			"no signature supplied",
// 			func() {
// 				err := txBuilder.SetMsgs(suite.msg1)
// 				suite.Require().NoError(err)

// 				genTxs = make([]json.RawMessage, 1)
// 				tx, err := suite.encodingConfig.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
// 				suite.Require().NoError(err)
// 				genTxs[0] = tx
// 			},
// 			false,
// 		},
// 		{
// 			"success",
// 			func() {
// 				_ = suite.setAccountBalance(addr1, 50)
// 				_ = suite.setAccountBalance(addr2, 1)

// 				msg := banktypes.NewMsgSend(addr1, addr2, sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)})
// 				tx, err := helpers.GenSignedMockTx(
// 					rand.New(rand.NewSource(time.Now().UnixNano())),
// 					suite.encodingConfig.TxConfig,
// 					[]sdk.Msg{msg},
// 					sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 10)},
// 					helpers.DefaultGenTxGas,
// 					suite.ctx.ChainID(),
// 					[]uint64{0},
// 					[]uint64{0},
// 					priv1,
// 				)
// 				suite.Require().NoError(err)

// 				genTxs = make([]json.RawMessage, 1)
// 				genTx, err := suite.encodingConfig.TxConfig.TxJSONEncoder()(tx)
// 				suite.Require().NoError(err)
// 				genTxs[0] = genTx
// 			},
// 			true,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
// 			suite.SetupTest()

// 			tc.malleate()

// 			if tc.expPass {
// 				suite.Require().NotPanics(func() {
// 					genutil.DeliverGenTxs(
// 						suite.ctx, genTxs, suite.app.StakingKeeper, suite.app.BaseApp.DeliverTx,
// 						suite.encodingConfig.TxConfig,
// 					)
// 				})
// 			} else {
// 				suite.Require().Panics(func() {
// 					genutil.DeliverGenTxs(
// 						suite.ctx, genTxs, suite.app.StakingKeeper, suite.app.BaseApp.DeliverTx,
// 						suite.encodingConfig.TxConfig,
// 					)
// 				})
// 			}
// 		})
// 	}
// }
