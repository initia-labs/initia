package benchmarks

import (
	"bytes"
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var transferAmount, _ = vmtypes.SerializeUint64(100)

// func BenchmarkTxSending(b *testing.B) {
// 	cases := map[string]struct {
// 		db          func(*testing.B) dbm.DB
// 		txBuilder   func(*testing.B, *AppInfo) []sdk.Tx
// 		numTxs      int
// 		numAccounts int
// 	}{
// 		"basic send - memdb": {
// 			db:          buildMemDB,
// 			numTxs:      20,
// 			txBuilder:   buildTxFromMsg(bankSendMsg, 20),
// 			numAccounts: 50,
// 		},
// 		"native_uinit transfer - memdb": {
// 			db:          buildMemDB,
// 			numTxs:      20,
// 			txBuilder:   buildTxFromMsg(coinTransferMsg, 20),
// 			numAccounts: 50,
// 		},
// 		"basic send - leveldb": {
// 			db:          buildLevelDB,
// 			numTxs:      20,
// 			txBuilder:   buildTxFromMsg(bankSendMsg, 20),
// 			numAccounts: 50,
// 		},
// 		"native_uinit transfer - leveldb": {
// 			db:          buildLevelDB,
// 			numTxs:      20,
// 			txBuilder:   buildTxFromMsg(coinTransferMsg, 20),
// 			numAccounts: 50,
// 		},
// 		"native_uinit transfer - leveldb - 1k accounts": {
// 			db:          buildLevelDB,
// 			numTxs:      20,
// 			txBuilder:   buildTxFromMsg(coinTransferMsg, 20),
// 			numAccounts: 1_000,
// 		},
// 		"native_uinit transfer - leveldb - 1k accounts - huge blocks": {
// 			db:          buildLevelDB,
// 			numTxs:      1_000,
// 			txBuilder:   buildTxFromMsg(coinTransferMsg, 1_000),
// 			numAccounts: 1_000,
// 		},
// 	}

// 	for name, tc := range cases {
// 		b.Run(name, func(b *testing.B) {
// 			db := tc.db(b)
// 			appInfo := InitializeBenchApp(b, &db, tc.numAccounts)
// 			txs := tc.txBuilder(b, &appInfo)

// 			// number of Tx per block for the benchmarks
// 			numTxs := tc.numTxs
// 			txEncoder := appInfo.TxConfig.TxEncoder()
// 			b.ResetTimer()

// 			for i := 0; i < b.N/numTxs; i++ {
// 				height := appInfo.App.LastBlockHeight() + 1
// 				appInfo.App.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: height, Time: time.Now()}})

// 				for j := 0; j < numTxs; j++ {
// 					idx := i*numTxs + j

// 					_, _, err := appInfo.App.SimDeliver(txEncoder, txs[idx])
// 					require.NoError(b, err)
// 				}

// 				appInfo.App.EndBlock(abci.RequestEndBlock{Height: height})
// 				appInfo.App.Commit()
// 			}
// 		})
// 	}
// }

func coinTransferMsg(info *AppInfo, idx int) ([]sdk.Msg, error) {
	rcpt := info.AccKeys[idx%len(info.AccKeys)].PubKey().Address()
	mt, err := movetypes.MetadataAddressFromDenom("uinit")
	if err != nil {
		return nil, err
	}

	msgTransfer := &movetypes.MsgExecute{
		Sender:        info.MinterAddr.String(),
		ModuleAddress: movetypes.StdAddr.String(),
		ModuleName:    movetypes.MoveModuleNameCoin,
		FunctionName:  movetypes.FunctionNameCoinTransfer,
		TypeArgs:      []string{},
		Args:          [][]byte{append(bytes.Repeat([]byte{0}, 12), rcpt...), mt[:], transferAmount},
	}
	return []sdk.Msg{msgTransfer}, nil
}

func buildTxFromMsg(builder func(*AppInfo, int) ([]sdk.Msg, error), numTxs int) func(b *testing.B, info *AppInfo) []sdk.Tx {
	return func(b *testing.B, info *AppInfo) []sdk.Tx {
		return GenSequenceOfTxs(b, info, builder, b.N*numTxs)
	}
}

func buildMemDB(b *testing.B) dbm.DB {
	return dbm.NewMemDB()
}

func buildLevelDB(b *testing.B) dbm.DB {
	levelDB, err := dbm.NewGoLevelDBWithOpts("testing", b.TempDir(), &opt.Options{BlockCacher: opt.NoCacher})
	require.NoError(b, err)
	return levelDB
}

func bankSendMsg(info *AppInfo, _ int) ([]sdk.Msg, error) {
	rcpt := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	coins := sdk.Coins{sdk.NewInt64Coin(info.Denom, 100)}
	sendMsg := banktypes.NewMsgSend(info.MinterAddr, rcpt, coins)
	return []sdk.Msg{sendMsg}, nil
}
