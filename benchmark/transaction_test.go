package benchmarks

import (
	"testing"

	"github.com/stretchr/testify/require"

	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func BenchmarkTPS(b *testing.B) {
	cases := map[string]struct {
		db          func(*testing.B) dbm.DB
		txBuilder   func(*testing.B, *AppInfo) []sdk.Tx
		numTxs      int
		numAccounts int
	}{
		"native_uinit transfer - memdb - 100 accounts - 100 txs": {
			db:          buildMemDB,
			numTxs:      100,
			txBuilder:   buildTxFromMsg(coinTransferMsg, 100),
			numAccounts: 100,
		},
		"native_uinit transfer - leveldb - 100 accounts - 100 txs": {
			db:          buildLevelDB,
			numTxs:      100,
			txBuilder:   buildTxFromMsg(coinTransferMsg, 100),
			numAccounts: 100,
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
