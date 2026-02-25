//go:build benchmark

package benchmark

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	e2e "github.com/initia-labs/initia/integration-tests/e2e"
	"github.com/stretchr/testify/require"
)

const (
	clusterReadyTimeout = 120 * time.Second
	mempoolDrainTimeout = 180 * time.Second
	mempoolPollInterval = 500 * time.Millisecond
	warmupSettleTime    = 5 * time.Second
)

func resultsDir(t *testing.T) string {
	t.Helper()
	if d := os.Getenv("BENCHMARK_RESULTS_DIR"); d != "" {
		return d
	}
	return filepath.Join("results")
}

func setupCluster(t *testing.T, ctx context.Context, cfg BenchConfig) *e2e.Cluster {
	t.Helper()

	cluster, err := e2e.NewCluster(ctx, t, e2e.ClusterOptions{
		NodeCount:    cfg.NodeCount,
		AccountCount: cfg.AccountCount,
		ChainID:      "bench-e2e",
		BinaryPath:   os.Getenv("E2E_INITIAD_BIN"),
		MemIAVL:      cfg.MemIAVL,
	})
	require.NoError(t, err)

	require.NoError(t, cluster.Start(ctx))
	require.NoError(t, cluster.WaitForReady(ctx, clusterReadyTimeout))

	return cluster
}

func runBenchmark(t *testing.T, cfg BenchConfig, loadFn func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult) BenchResult {
	t.Helper()
	ctx := context.Background()

	cluster := setupCluster(t, ctx, cfg)
	defer cluster.Close()

	// collect initial account metadata
	metas, err := CollectInitialMetas(ctx, cluster)
	require.NoError(t, err)

	// warmup
	Warmup(ctx, cluster, metas)
	require.NoError(t, cluster.WaitForMempoolEmpty(ctx, 30*time.Second))
	time.Sleep(warmupSettleTime)

	// recollect metas after warmup (sequences changed)
	metas, err = CollectInitialMetas(ctx, cluster)
	require.NoError(t, err)

	startHeight, err := cluster.LatestHeight(ctx, 0)
	require.NoError(t, err)

	// start mempool poller
	poller := NewMempoolPoller(ctx, cluster, mempoolPollInterval)

	// run load
	t.Logf("Starting load: %d accounts x %d txs = %d total", cfg.AccountCount, cfg.TxPerAccount, cfg.TotalTx())
	loadResult := loadFn(ctx, cluster, cfg, metas)
	t.Logf("Load complete: %d submitted, %d errors, duration=%.1fs",
		len(loadResult.Submissions), len(loadResult.Errors),
		loadResult.EndTime.Sub(loadResult.StartTime).Seconds())

	// wait for all txs to be included
	endHeight, err := WaitForAllIncluded(ctx, cluster, mempoolDrainTimeout)
	require.NoError(t, err)

	peakMempool := poller.Stop()

	// collect results
	result, err := CollectResults(ctx, cluster, cfg, loadResult, startHeight, endHeight, peakMempool)
	require.NoError(t, err)

	t.Logf("Results: TPS=%.1f, P50=%.0fms, P95=%.0fms, P99=%.0fms, included=%d/%d, peak_mempool=%d",
		result.TxPerSecond, result.P50LatencyMs, result.P95LatencyMs, result.P99LatencyMs,
		result.TotalIncluded, result.TotalSubmitted, result.PeakMempoolSize)

	// write results
	require.NoError(t, WriteResult(t, result, resultsDir(t)))

	return result
}

// TestBenchmarkBaseline records baseline results using whichever binary is provided.
// For true CListMempool baseline: checkout the pre-proxy cometbft tag, rebuild initiad,
// and pass it via E2E_INITIAD_BIN.
//
// Usage:
//
//	# 1. checkout pre-proxy tag and build
//	git checkout tags/v1.3.1 && make build
//	# 2. run baseline benchmark
//	E2E_INITIAD_BIN=./build/initiad make benchmark-e2e BENCH_RUN=TestBenchmarkBaseline
func TestBenchmarkBaseline(t *testing.T) {
	cfg := BaselineConfig()
	runBenchmark(t, cfg, BurstLoad)
}

// TestBenchmarkThroughput measures throughput with burst mode and sequential nonces.
// Uses the mempool-only variant (ProxyMempool+PriorityMempool + IAVL).
func TestBenchmarkThroughput(t *testing.T) {
	cfg := MempoolOnlyConfig()
	cfg.Label = "throughput/mempool-only"
	runBenchmark(t, cfg, BurstLoad)
}

// TestBenchmarkLatency measures latency distribution with burst mode.
func TestBenchmarkLatency(t *testing.T) {
	cfg := MempoolOnlyConfig()
	cfg.Label = "latency/mempool-only"
	result := runBenchmark(t, cfg, BurstLoad)

	require.Greater(t, result.TotalIncluded, 0, "no transactions were included")
	t.Logf("Latency distribution: avg=%.0fms p50=%.0fms p95=%.0fms p99=%.0fms max=%.0fms",
		result.AvgLatencyMs, result.P50LatencyMs, result.P95LatencyMs, result.P99LatencyMs, result.MaxLatencyMs)
}

// TestBenchmarkQueuePromotion tests out-of-order nonce handling and verifies all txs are included.
func TestBenchmarkQueuePromotion(t *testing.T) {
	cfg := MempoolOnlyConfig()
	cfg.TxPerAccount = 50
	cfg.Label = "queue-promotion/mempool-only"
	result := runBenchmark(t, cfg, OutOfOrderLoad)

	require.Equal(t, result.TotalSubmitted, result.TotalIncluded,
		"not all out-of-order transactions were included: submitted=%d included=%d",
		result.TotalSubmitted, result.TotalIncluded)
}

// TestBenchmarkFullComparison runs the three-way comparison:
//
//  1. mempool-only: ProxyMempool+PriorityMempool + standard IAVL
//  2. combined:     ProxyMempool+PriorityMempool + MemIAVL
//  3. baseline:     loaded from prior results if available (run TestBenchmarkBaseline separately
//     with the pre-proxy binary)
//
// This measures the incremental improvement from each optimization layer.
func TestBenchmarkFullComparison(t *testing.T) {
	var results []BenchResult

	// Try to load baseline results from a prior run
	baselineResults := LoadBaselineResults(resultsDir(t))
	if len(baselineResults) > 0 {
		t.Logf("Loaded %d baseline result(s) from prior run", len(baselineResults))
		results = append(results, baselineResults[0])
	} else {
		t.Log("No baseline results found. Run TestBenchmarkBaseline with pre-proxy binary for full 3-way comparison.")
	}

	// Run mempool-only (ProxyMempool+PriorityMempool + IAVL)
	t.Run("MempoolOnly", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	// Run combined (ProxyMempool+PriorityMempool + MemIAVL)
	t.Run("Combined", func(t *testing.T) {
		cfg := CombinedConfig()
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	// Print comparison
	if len(results) >= 2 {
		PrintComparisonTable(t, results)
		PrintImprovementTable(t, results)
	}
}

// TestBenchmarkSaturation tests behavior under mempool pressure.
func TestBenchmarkSaturation(t *testing.T) {
	cfg := MempoolOnlyConfig()
	cfg.AccountCount = 5
	cfg.TxPerAccount = 100
	cfg.Label = "saturation/mempool-only"

	result := runBenchmark(t, cfg, BurstLoad)
	t.Logf("Saturation: submitted=%d included=%d peak_mempool=%d",
		result.TotalSubmitted, result.TotalIncluded, result.PeakMempoolSize)
}

// TestBenchmarkWideState runs IAVL vs MemIAVL with 50 accounts to stress the state tree.
// More unique account leaves usually result in deeper IAVL traversals where MemIAVL should differentiate.
func TestBenchmarkWideState(t *testing.T) {
	var results []BenchResult

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 50
		cfg.Label = "wide-state/iavl"
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	t.Run("MemIAVL", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 50
		cfg.Label = "wide-state/memiavl"
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	if len(results) == 2 {
		PrintComparisonTable(t, results)
	}
}

// TestBenchmarkGossipPropagation submits all txs to node 0 and monitors propagation.
func TestBenchmarkGossipPropagation(t *testing.T) {
	cfg := MempoolOnlyConfig()
	cfg.AccountCount = 5
	cfg.TxPerAccount = 50
	cfg.Label = "gossip/mempool-only"

	ctx := context.Background()
	cluster := setupCluster(t, ctx, cfg)
	defer cluster.Close()

	metas, err := CollectInitialMetas(ctx, cluster)
	require.NoError(t, err)

	// warmup
	Warmup(ctx, cluster, metas)
	require.NoError(t, cluster.WaitForMempoolEmpty(ctx, 30*time.Second))
	time.Sleep(warmupSettleTime)

	metas, err = CollectInitialMetas(ctx, cluster)
	require.NoError(t, err)

	startHeight, err := cluster.LatestHeight(ctx, 0)
	require.NoError(t, err)

	// start mempool pollers for each node
	pollers := make([]*MempoolPoller, cfg.NodeCount)
	for i := 0; i < cfg.NodeCount; i++ {
		pollers[i] = NewMempoolPoller(ctx, cluster, mempoolPollInterval)
	}

	// submit all txs to node 0 only
	loadResult := SingleNodeLoad(ctx, cluster, cfg, metas, 0)
	t.Logf("Submitted %d txs to node 0", len(loadResult.Submissions))

	// wait for inclusion
	endHeight, err := WaitForAllIncluded(ctx, cluster, mempoolDrainTimeout)
	require.NoError(t, err)

	for i, p := range pollers {
		peak := p.Stop()
		t.Logf("Node %d peak mempool size: %d", i, peak)
	}

	result, err := CollectResults(ctx, cluster, cfg, loadResult, startHeight, endHeight, 0)
	require.NoError(t, err)

	t.Logf("Gossip test: TPS=%.1f, included=%d/%d",
		result.TxPerSecond, result.TotalIncluded, result.TotalSubmitted)
	require.NoError(t, WriteResult(t, result, resultsDir(t)))
}
