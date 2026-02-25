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

// runBenchmarkWithCluster runs the benchmark pipeline on a pre-created cluster.
// This is used when the cluster needs setup before the measured load (e.g., deploying a Move module).
func runBenchmarkWithCluster(t *testing.T, ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, loadFn func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult) BenchResult {
	t.Helper()

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

func runBenchmark(t *testing.T, cfg BenchConfig, loadFn func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult) BenchResult {
	t.Helper()
	ctx := context.Background()

	cluster := setupCluster(t, ctx, cfg)
	defer cluster.Close()

	return runBenchmarkWithCluster(t, ctx, cluster, cfg, loadFn)
}

// setupMoveExecLoad deploys the Counter module and estimates gas once.
// Returns a LoadFn closure that uses MoveExecBurstLoad.
func setupMoveExecLoad(t *testing.T, ctx context.Context, cluster *e2e.Cluster) func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
	t.Helper()

	// 1. build Counter module
	modulePath, err := cluster.BuildMoveModule(ctx,
		cluster.RepoPath("x", "move", "keeper", "contracts"),
		"Counter", nil)
	require.NoError(t, err)
	t.Logf("Built Counter module: %s", modulePath)

	// 2. publish via acc1
	publisherName := cluster.AccountNames()[0]
	res := cluster.MovePublish(ctx, publisherName, []string{modulePath}, 0)
	require.NoError(t, res.Err)
	require.Equal(t, int64(0), res.Code, "publish failed: %s", res.RawLog)

	// 3. wait for inclusion
	require.NoError(t, cluster.WaitForMempoolEmpty(ctx, 30*time.Second))
	time.Sleep(3 * time.Second)

	// 4. estimate gas once for increase()
	publisherAddr, err := cluster.AccountAddress(publisherName)
	require.NoError(t, err)

	meta, err := cluster.QueryAccountMeta(ctx, 0, publisherAddr)
	require.NoError(t, err)

	estimatedGas, err := cluster.MoveEstimateExecuteJSONGasWithSequence(
		ctx,
		publisherName,
		publisherAddr,
		"Counter",
		"increase",
		nil, nil,
		meta.AccountNumber, meta.Sequence,
		0,
	)
	require.NoError(t, err)
	t.Logf("Estimated gas for Counter::increase: %d", estimatedGas)

	// 5. return MoveExecBurstLoad with captured parameters
	return MoveExecBurstLoad(publisherAddr, "Counter", "increase", nil, nil, estimatedGas)
}

// ---------------------------------------------------------------------------
// Mempool comparison: CList vs. Proxy+Priority
// ---------------------------------------------------------------------------

// TestBenchmarkBaselineSeq records CList baseline with a sequential load.
// Build the pre-proxy binary and pass via E2E_INITIAD_BIN.
func TestBenchmarkBaselineSeq(t *testing.T) {
	cfg := BaselineConfig()
	cfg.Label = "clist/iavl/seq"
	runBenchmark(t, cfg, SequentialLoad)
}

// TestBenchmarkBaselineBurst records CList baseline with a burst load.
// Build the pre-proxy binary and pass via E2E_INITIAD_BIN.
func TestBenchmarkBaselineBurst(t *testing.T) {
	cfg := BaselineConfig()
	cfg.Label = "clist/iavl/burst"
	runBenchmark(t, cfg, BurstLoad)
}

// TestBenchmarkSeqComparison loads the sequential baseline, then runs
// Proxy+IAVL and Proxy+MemIAVL with a sequential load for comparison.
func TestBenchmarkSeqComparison(t *testing.T) {
	var results []BenchResult

	// load baseline from JSON (label="clist/iavl/seq")
	baselines := LoadBaselineResultsByLabel(resultsDir(t), "clist/iavl/seq")
	if len(baselines) > 0 {
		t.Logf("Loaded baseline result: %s", baselines[0].Config.Label)
		results = append(results, baselines[0])
	} else {
		t.Log("No baseline results found. Run TestBenchmarkBaselineSeq with pre-proxy binary for full comparison.")
	}

	t.Run("MempoolOnly", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.Label = "proxy+priority/iavl/seq"
		result := runBenchmark(t, cfg, SequentialLoad)
		results = append(results, result)
	})

	t.Run("Combined", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.Label = "proxy+priority/memiavl/seq"
		result := runBenchmark(t, cfg, SequentialLoad)
		results = append(results, result)
	})

	if len(results) >= 2 {
		PrintComparisonTable(t, results)
		PrintImprovementTable(t, results)
	}
}

// TestBenchmarkBurstComparison loads the burst baseline, then runs
// Proxy+IAVL and Proxy+MemIAVL with a burst load for comparison.
func TestBenchmarkBurstComparison(t *testing.T) {
	var results []BenchResult

	// load baseline from JSON (label="clist/iavl/burst")
	baselines := LoadBaselineResultsByLabel(resultsDir(t), "clist/iavl/burst")
	if len(baselines) > 0 {
		t.Logf("Loaded baseline result: %s", baselines[0].Config.Label)
		results = append(results, baselines[0])
	} else {
		t.Log("No baseline results found. Run TestBenchmarkBaselineBurst with pre-proxy binary for full comparison.")
	}

	t.Run("MempoolOnly", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.Label = "proxy+priority/iavl/burst"
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	t.Run("Combined", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.Label = "proxy+priority/memiavl/burst"
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	if len(results) >= 2 {
		PrintComparisonTable(t, results)
		PrintImprovementTable(t, results)
	}
}

// ---------------------------------------------------------------------------
// State DB comparison: IAVL vs MemIAVL
// ---------------------------------------------------------------------------

// TestBenchmarkMemIAVLBankSend compares IAVL vs MemIAVL with bank send workload.
// Uses 100 accounts x 300 txs to stress the state tree.
func TestBenchmarkMemIAVLBankSend(t *testing.T) {
	var results []BenchResult

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 300
		cfg.Label = "memiavl-compare/iavl/bank-send"
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	t.Run("MemIAVL", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 300
		cfg.Label = "memiavl-compare/memiavl/bank-send"
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	if len(results) == 2 {
		PrintComparisonTable(t, results)
	}
}

// TestBenchmarkMemIAVLMoveExec compares IAVL vs. MemIAVL with Move exec workload.
// Deploys the Counter module and runs increase() calls.
func TestBenchmarkMemIAVLMoveExec(t *testing.T) {
	var results []BenchResult

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 300
		cfg.Label = "memiavl-compare/iavl/move-exec"

		ctx := context.Background()
		cluster := setupCluster(t, ctx, cfg)
		defer cluster.Close()

		moveLoadFn := setupMoveExecLoad(t, ctx, cluster)
		result := runBenchmarkWithCluster(t, ctx, cluster, cfg, moveLoadFn)
		results = append(results, result)
	})

	t.Run("MemIAVL", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 300
		cfg.Label = "memiavl-compare/memiavl/move-exec"

		ctx := context.Background()
		cluster := setupCluster(t, ctx, cfg)
		defer cluster.Close()

		moveLoadFn := setupMoveExecLoad(t, ctx, cluster)
		result := runBenchmarkWithCluster(t, ctx, cluster, cfg, moveLoadFn)
		results = append(results, result)
	})

	if len(results) == 2 {
		PrintComparisonTable(t, results)
	}
}

// ---------------------------------------------------------------------------
// Capability demos
// ---------------------------------------------------------------------------

// TestBenchmarkQueuePromotion tests out-of-order nonce handling and verifies 100% inclusion.
func TestBenchmarkQueuePromotion(t *testing.T) {
	cfg := MempoolOnlyConfig()
	cfg.TxPerAccount = 50
	cfg.Label = "queue-promotion/mempool-only"
	result := runBenchmark(t, cfg, OutOfOrderLoad)

	require.Equal(t, result.TotalSubmitted, result.TotalIncluded,
		"not all out-of-order transactions were included: submitted=%d included=%d",
		result.TotalSubmitted, result.TotalIncluded)
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

	// start cluster-wide mempool poller
	poller := NewMempoolPoller(ctx, cluster, mempoolPollInterval)

	// submit all txs to node 0 only
	loadResult := SingleNodeLoad(ctx, cluster, cfg, metas, 0)
	t.Logf("Submitted %d txs to node 0", len(loadResult.Submissions))

	// wait for inclusion
	endHeight, err := WaitForAllIncluded(ctx, cluster, mempoolDrainTimeout)
	require.NoError(t, err)

	peakMempool := poller.Stop()
	t.Logf("Cluster peak mempool size: %d", peakMempool)

	result, err := CollectResults(ctx, cluster, cfg, loadResult, startHeight, endHeight, peakMempool)
	require.NoError(t, err)

	t.Logf("Gossip test: TPS=%.1f, included=%d/%d",
		result.TxPerSecond, result.TotalIncluded, result.TotalSubmitted)
	require.NoError(t, WriteResult(t, result, resultsDir(t)))
}
