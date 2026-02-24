//go:build e2e

package mempool

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	e2e "github.com/initia-labs/initia/integration-tests/e2e"
	"github.com/stretchr/testify/require"
)

const maxNodeCount = 10

func TestQueueClearOrdering(t *testing.T) {
	ctx := context.Background()

	nodeCount := readEnvInt("E2E_NODE_COUNT", 5)
	accountCount := readEnvInt("E2E_ACCOUNT_COUNT", 5)
	txPerAccount := readEnvInt("E2E_TX_PER_ACCOUNT", 10)
	if nodeCount > maxNodeCount {
		t.Fatalf("E2E_NODE_COUNT exceeds max supported (%d): %d", maxNodeCount, nodeCount)
	}

	cluster, err := e2e.NewCluster(ctx, t, e2e.ClusterOptions{
		NodeCount:    nodeCount,
		AccountCount: accountCount,
		ChainID:      "testnet-e2e",
		BasePort:     26000,
		PortStride:   20,
		BinaryPath:   os.Getenv("E2E_INITIAD_BIN"),
	})
	require.NoError(t, err)
	defer cluster.Close()

	require.NoError(t, cluster.Start(ctx))
	require.NoError(t, cluster.WaitForReady(ctx, 90*time.Second))

	initial := make(map[string]e2e.AccountMeta)
	for _, name := range cluster.AccountNames() {
		addr, err := cluster.AccountAddress(name)
		require.NoError(t, err)
		meta, err := cluster.QueryAccountMeta(ctx, 0, addr)
		require.NoError(t, err)
		initial[name] = meta
	}

	txResults := make(map[string][]e2e.TxResult)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, name := range cluster.AccountNames() {
		name := name
		meta := initial[name]
		seqs := SequencePattern(meta.Sequence, txPerAccount)
		cluster.Logf("[queue-clear] account=%s account_number=%d initial_sequence=%d seqs=%v", name, meta.AccountNumber, meta.Sequence, seqs)

		wg.Add(1)
		go func() {
			defer wg.Done()
			results := make([]e2e.TxResult, 0, len(seqs))
			for _, seq := range seqs {
				viaNode := 0
				if cluster.NodeCount() > 1 {
					viaNode = rand.Intn(cluster.NodeCount())
				}
				res := cluster.SendBankTxWithSequence(ctx, name, cluster.ValidatorAddress(), "1uinit", meta.AccountNumber, seq, 500_000, viaNode)
				results = append(results, res)
			}
			mu.Lock()
			txResults[name] = results
			mu.Unlock()
		}()
	}
	wg.Wait()

	for name, results := range txResults {
		for i, res := range results {
			require.NoError(t, res.Err, "%s tx[%d] failed to broadcast", name, i)
			require.EqualValues(t, 0, res.Code, "%s tx[%d] rejected code=%d raw_log=%s", name, i, res.Code, res.RawLog)
		}
	}

	require.NoError(t, cluster.WaitForMempoolEmpty(ctx, 120*time.Second))
	final, err := CollectFinalAccountMeta(ctx, cluster, 0)
	require.NoError(t, err)
	for name, initialMeta := range initial {
		finalMeta, ok := final[name]
		require.True(t, ok, "missing final account for %s", name)
		expected := initialMeta.Sequence + uint64(txPerAccount)
		require.Equalf(t, expected, finalMeta.Sequence, "account %s sequence mismatch", name)
	}

	for name, meta := range final {
		t.Logf("account=%s final_sequence=%d", name, meta.Sequence)
	}
}

func readEnvInt(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		panic(fmt.Sprintf("invalid %s=%q: %v", name, raw, err))
	}
	return n
}

func CollectFinalAccountMeta(ctx context.Context, c *e2e.Cluster, viaNode int) (map[string]e2e.AccountMeta, error) {
	final := make(map[string]e2e.AccountMeta)
	for _, name := range c.AccountNames() {
		addr, _ := c.AccountAddress(name)
		meta, err := c.QueryAccountMeta(ctx, viaNode, addr)
		if err != nil {
			return nil, err
		}
		final[name] = meta
	}
	return final, nil
}

func SequencePattern(base uint64, count int) []uint64 {
	if count <= 3 {
		seqs := []uint64{base + 2, base, base + 1}
		return seqs[:count]
	}

	seqs := []uint64{base + 2, base, base + 1}
	for i := 3; i < count; i++ {
		seqs = append(seqs, base+uint64(i))
	}
	return seqs
}

func RandomNodePicker(nodeCount int, source rand.Source) func() int {
	r := rand.New(source)
	return func() int {
		if nodeCount <= 1 {
			return 0
		}
		return r.Intn(nodeCount)
	}
}
