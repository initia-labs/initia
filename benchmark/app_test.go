package benchmarks

import (
	"math/rand"
	"testing"
	"time"

	//"github.com/cometbft/cometbft/libs/rand"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/require"

	simappparams "cosmossdk.io/simapp/params"
	vmtypes "github.com/initia-labs/initiavm/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	initiaapp "github.com/initia-labs/initia/app"
)

func init() {
	sdkConfig := sdk.GetConfig()
	sdkConfig.SetCoinType(initiaapp.CoinType)

	accountPubKeyPrefix := initiaapp.AccountAddressPrefix + "pub"
	validatorAddressPrefix := initiaapp.AccountAddressPrefix + "valoper"
	validatorPubKeyPrefix := initiaapp.AccountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := initiaapp.AccountAddressPrefix + "valcons"
	consNodePubKeyPrefix := initiaapp.AccountAddressPrefix + "valconspub"

	sdkConfig.SetBech32PrefixForAccount(initiaapp.AccountAddressPrefix, accountPubKeyPrefix)
	sdkConfig.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	sdkConfig.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	sdkConfig.SetAddressVerifier(initiaapp.VerifyAddressLen())
	sdkConfig.Seal()
}

type AppInfo struct {
	App           *initiaapp.InitiaApp
	MinterAddr    sdk.AccAddress
	MinterHexAddr vmtypes.AccountAddress
	MinterKey     *secp256k1.PrivKey
	ModuleName    string
	Denom         string
	AccNum        uint64
	Sequence      uint64
	TxConfig      client.TxConfig
	AccKeys       []secp256k1.PrivKey
}

func InitializeBenchApp(b *testing.B, db *dbm.DB, numAccounts int) AppInfo {
	// constants
	denom := initiaapp.BondDenom
	moduleName := "coin"

	// initia18ndwzuhkcyzrkrkada9n7un0gauq6tmjc9y2mm <> 0x3cdae172f6c1043b0edd6f4b3f726f47780d2f72
	minterKey := secp256k1.GenPrivKeyFromSecret([]byte("__KEY__SECRET__FOR__BENCHMARK__"))
	minterAddr := sdk.AccAddress(minterKey.PubKey().Address())
	minterHexAddr, err := vmtypes.NewAccountAddressFromBytes(minterAddr.Bytes())
	require.NoError(b, err)

	// genesis setup (with a bunch of random accounts)
	accKeys := make([]secp256k1.PrivKey, numAccounts)
	bals := make([]banktypes.Balance, numAccounts+1)
	genAccs := make([]authtypes.GenesisAccount, numAccounts+1)

	// set minter as first genesis account with balance
	genAccs[0] = &authtypes.BaseAccount{Address: minterAddr.String()}
	bals[0] = banktypes.Balance{Address: minterAddr.String(), Coins: sdk.NewCoins(sdk.NewInt64Coin(denom, 1_000_000_000_000_000))}

	for i := 0; i < numAccounts; i++ {
		pk := secp256k1.GenPrivKey()

		accKeys[i] = *pk
		addr := sdk.AccAddress(pk.PubKey().Address()).String()

		genAccs[i+1] = &authtypes.BaseAccount{Address: addr}
		bals[i+1] = banktypes.Balance{Address: addr, Coins: sdk.NewCoins(sdk.NewInt64Coin(denom, 1_000_000_000))}
	}

	initiaApp := initiaapp.SetupWithGenesisAccounts(nil, genAccs, bals...)
	config := simappparams.MakeTestEncodingConfig().TxConfig

	height := initiaApp.LastBlockHeight() + 1
	initiaApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: time.Now()})
	initiaApp.Commit()

	appInfo := AppInfo{
		App:           initiaApp,
		MinterKey:     minterKey,
		MinterAddr:    minterAddr,
		MinterHexAddr: minterHexAddr,
		ModuleName:    moduleName,
		Denom:         denom,
		AccNum:        0,
		Sequence:      0,
		TxConfig:      config,
		AccKeys:       accKeys[:],
	}

	return appInfo
}

func GenSequenceOfTxs(b *testing.B, info *AppInfo, msgGen func(*AppInfo, int) ([]sdk.Msg, error), numToGenerate int) []sdk.Tx {
	fees := sdk.NewCoins(sdk.NewInt64Coin(info.Denom, 1_000_000))
	txs := make([]sdk.Tx, numToGenerate)

	for i := 0; i < (numToGenerate); i++ {
		msgs, err := msgGen(info, i)
		require.NoError(b, err)
		txs[i], err = simtestutil.GenSignedMockTx(
			rand.New(rand.NewSource(time.Now().UnixNano())),
			info.TxConfig,
			msgs,
			fees,
			1_000_000,
			"",
			[]uint64{info.AccNum},
			[]uint64{info.Sequence},
			info.MinterKey,
		)
		require.NoError(b, err)
		info.Sequence += 1
	}
	return txs
}
