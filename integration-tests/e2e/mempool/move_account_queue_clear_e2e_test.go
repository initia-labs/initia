//go:build e2e

package mempool

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	e2e "github.com/initia-labs/initia/integration-tests/e2e"
	"github.com/stretchr/testify/require"
)

func TestMoveStdAccountCreateQueueClearScenario(t *testing.T) {
	ctx := context.Background()

	cluster, err := e2e.NewCluster(ctx, t, e2e.ClusterOptions{
		NodeCount:    3,
		AccountCount: 3,
		ChainID:      "testnet-move-account-queue",
		BasePort:     26200,
		PortStride:   20,
	})
	require.NoError(t, err)
	defer cluster.Close()

	require.NoError(t, cluster.Start(ctx))
	require.NoError(t, cluster.WaitForReady(ctx, 90*time.Second))

	pickNode := RandomNodePicker(cluster.NodeCount(), rand.NewSource(time.Now().UnixNano()))
	txPerAccount := 3

	initial := map[string]e2e.AccountMeta{}
	for _, name := range cluster.AccountNames() {
		addr, err := cluster.AccountAddress(name)
		require.NoError(t, err)
		meta, err := cluster.QueryAccountMeta(ctx, pickNode(), addr)
		require.NoError(t, err)
		initial[name] = meta
	}

	results := map[string][]e2e.TxResult{}
	for idx, name := range cluster.AccountNames() {
		meta := initial[name]
		seqs := SequencePattern(meta.Sequence, txPerAccount)

		local := make([]e2e.TxResult, 0, len(seqs))
		for _, seq := range seqs {
			newVMAddr := fmt.Sprintf("0x%064x", uint64((idx+1)*1_000_000)+seq)
			res := cluster.MoveExecuteJSONWithSequence(
				ctx,
				name,
				"0x1",
				"account",
				"create_account_script",
				nil,
				[]string{newVMAddr},
				meta.AccountNumber,
				seq,
				pickNode(),
			)
			local = append(local, res)
		}
		results[name] = local
	}

	for name, txs := range results {
		for i, txr := range txs {
			require.NoError(t, txr.Err, "%s tx[%d] err", name, i)
			require.EqualValues(t, 0, txr.Code, "%s tx[%d] code=%d txhash=%s raw_log=%s", name, i, txr.Code, txr.TxHash, txr.RawLog)
		}
	}

	require.NoError(t, cluster.WaitForMempoolEmpty(ctx, 60*time.Second))

	for _, name := range cluster.AccountNames() {
		addr, err := cluster.AccountAddress(name)
		require.NoError(t, err)
		final, err := cluster.QueryAccountMeta(ctx, pickNode(), addr)
		require.NoError(t, err)
		expected := initial[name].Sequence + uint64(txPerAccount)
		require.Equalf(t, expected, final.Sequence, "%s final sequence mismatch", name)
	}

	sample := fmt.Sprintf("0x%064x", uint64(1*1_000_000)+initial["acc1"].Sequence)
	viewOut, err := cluster.MoveQueryViewJSON(ctx, "0x1", "account", "exists_at", nil, []string{sample}, pickNode())
	require.NoError(t, err)
	require.True(t, strings.Contains(string(viewOut), "true") || strings.Contains(string(viewOut), "\"true\""))
}
