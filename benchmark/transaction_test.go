package benchmarks

import (
	"testing"

	// "fmt"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	dbm "github.com/cometbft/cometbft-db"
)

func BenchmarkTPS(b *testing.B) {
	cases := map[string]struct {
		db          func(*testing.B) dbm.DB
		txBuilder   func(*testing.B, *AppInfo) []sdk.Tx
		numTxs      int
		numAccounts int
	}{
		"native_uinit transfer - memdb - 1k accounts - 1_000 txs": {
			db:          buildMemDB,
			numTxs:      1_000,
			txBuilder:   buildTxFromMsg(coinTransferMsg, 1_000),
			numAccounts: 1_000,
		},
		"native_uinit transfer - leveldb - 1k accounts - 1_000 txs": {
			db:          buildLevelDB,
			numTxs:      1_000,
			txBuilder:   buildTxFromMsg(coinTransferMsg, 1_000),
			numAccounts: 1_000,
		},
		"native_uinit transfer - memdb - 1k accounts - 2_000 txs": {
			db:          buildMemDB,
			numTxs:      2_000,
			txBuilder:   buildTxFromMsg(coinTransferMsg, 2_000),
			numAccounts: 1_000,
		},
		"native_uinit transfer - leveldb - 1k accounts - 2_000 txs": {
			db:          buildLevelDB,
			numTxs:      2_000,
			txBuilder:   buildTxFromMsg(coinTransferMsg, 2_000),
			numAccounts: 1_000,
		},
		"native_uinit transfer - memdb - 1k accounts - 3_000 txs": {
			db:          buildMemDB,
			numTxs:      3_000,
			txBuilder:   buildTxFromMsg(coinTransferMsg, 3_000),
			numAccounts: 1_000,
		},
		"native_uinit transfer - leveldb - 1k accounts - 3_000 txs": {
			db:          buildLevelDB,
			numTxs:      3_000,
			txBuilder:   buildTxFromMsg(coinTransferMsg, 3_000),
			numAccounts: 1_000,
		},
	}

	for name, tc := range cases {
		b.Run(name, func(b *testing.B) {
			db := tc.db(b)
			appInfo := InitializeBenchApp(b, &db, tc.numAccounts)
			txs := tc.txBuilder(b, &appInfo)

			// number of Tx per block for the benchmarks
			numTxs := tc.numTxs
			txEncoder := appInfo.TxConfig.TxEncoder()
			b.ResetTimer()

			for i := 0; i < b.N/numTxs; i++ {
				for j := 0; j < numTxs; j++ {
					idx := i*numTxs + j

					_, _, err := appInfo.App.SimDeliver(txEncoder, txs[idx])
					require.NoError(b, err)
				}

			}
		})
	}
}
