package genutil_test

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
	initiaapp "github.com/initia-labs/initia/app"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Bond denom should be set for staking test
const bondDenom = initiaapp.BondDenom

var (
	valPubKeys = simtestutil.CreateTestPubKeys(5)

	priv1 = secp256k1.GenPrivKey()
	addr1 = sdk.AccAddress(priv1.PubKey().Address())
	priv2 = secp256k1.GenPrivKey()
	addr2 = sdk.AccAddress(priv2.PubKey().Address())

	bondCoin = sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1000000))
)

func createApp(t *testing.T) *initiaapp.InitiaApp {
	app := initiaapp.SetupWithGenesisAccounts(nil, authtypes.GenesisAccounts{})
	return app
}

func checkBalance(t *testing.T, app *initiaapp.InitiaApp, addr sdk.AccAddress, balances sdk.Coins) {
	ctxCheck := app.BaseApp.NewContext(true)
	require.True(t, balances.Equal(app.BankKeeper.GetAllBalances(ctxCheck, addr)))
}

func setAccountBalance(t *testing.T, addr sdk.AccAddress, genCoins sdk.Coins) json.RawMessage {
	app := initiaapp.SetupWithGenesisAccounts(nil, authtypes.GenesisAccounts{
		&authtypes.BaseAccount{Address: addr.String()},
	},
		banktypes.Balance{Address: addr.String(), Coins: genCoins},
	)

	checkBalance(t, app, addr, genCoins)

	ctxCheck := app.BaseApp.NewContext(true)

	bankGenesisState := app.BankKeeper.ExportGenesis(ctxCheck)
	bankGenesis, err := initiaapp.MakeEncodingConfig().Amino.MarshalJSON(bankGenesisState) // TODO switch this to use Marshaler
	require.NoError(t, err)

	return bankGenesis
}
