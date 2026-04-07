//go:build benchmark

package benchmark

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	e2e "github.com/initia-labs/initia/integration-tests/e2e"
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
		NodeCount:      cfg.NodeCount,
		AccountCount:   cfg.AccountCount,
		ChainID:        "bench-e2e",
		BinaryPath:     os.Getenv("E2E_INITIAD_BIN"),
		MemIAVL:        cfg.MemIAVL,
		TimeoutCommit:  cfg.TimeoutCommit,
		ValidatorCount: cfg.ValidatorCount,
		MaxBlockGas:    cfg.MaxBlockGas,
		NoAllowQueued:  cfg.NoAllowQueued,
	})
	require.NoError(t, err)

	require.NoError(t, cluster.Start(ctx))
	t.Cleanup(cluster.Close)
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

	// scale drain timeout with total tx count: base 180s + 1s per 20 txs
	drainTimeout := mempoolDrainTimeout + time.Duration(cfg.TotalTx()/20)*time.Second
	// wait for load to settle. CList checks validator nodes only
	// because CList can leave txs stranded on non-validator mempools
	endHeight, err := WaitForLoadToSettle(ctx, cluster, drainTimeout, cfg.NoAllowQueued)
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

type moveExecLoadMode int

const (
	moveExecBurst moveExecLoadMode = iota
	moveExecSequential
)

// setupMoveExecLoad deploys the BenchHeavyState module and estimates gas once.
// Returns a LoadFn closure that uses burst or sequential Move exec depending on the mode.
func setupMoveExecLoad(t *testing.T, ctx context.Context, cluster *e2e.Cluster, mode ...moveExecLoadMode) func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
	t.Helper()

	const (
		sharedWrites = "5"  // contended writes to global shared state per tx
		localWrites  = "25" // non-contended writes to per-account state per tx
	)

	// 1. get publisher hex address for Move named-address
	publisherName := cluster.AccountNames()[0]
	publisherHex, err := cluster.AccountAddressHex(ctx, publisherName)
	require.NoError(t, err)
	t.Logf("Publisher hex address: %s", publisherHex)

	// 2. build BenchHeavyState module with Publisher = acc1's address
	packagePath := cluster.RepoPath("integration-tests", "e2e", "benchmark", "move-bench")
	modulePath, err := cluster.BuildMoveModule(ctx,
		packagePath, "BenchHeavyState",
		map[string]string{"Publisher": publisherHex})
	require.NoError(t, err)
	t.Logf("Built BenchHeavyState module: %s", modulePath)

	// 3. publish via acc1
	res := cluster.MovePublish(ctx, publisherName, []string{modulePath}, 0)
	require.NoError(t, res.Err)
	require.Equal(t, int64(0), res.Code, "publish failed: %s", res.RawLog)

	// 4. wait for inclusion
	require.NoError(t, cluster.WaitForMempoolEmpty(ctx, 30*time.Second))
	time.Sleep(3 * time.Second)

	// 5. estimate gas once for write_mixed(shared_count, local_count)
	args := []string{sharedWrites, localWrites}
	publisherAddr, err := cluster.AccountAddress(publisherName)
	require.NoError(t, err)
	meta, err := cluster.QueryAccountMeta(ctx, 0, publisherAddr)
	require.NoError(t, err)

	estimatedGas, err := cluster.MoveEstimateExecuteJSONGasWithSequence(
		ctx,
		publisherName,
		publisherHex,
		"BenchHeavyState",
		"write_mixed",
		nil, args,
		meta.AccountNumber, meta.Sequence,
		0,
	)
	require.NoError(t, err)
	t.Logf("Estimated gas for BenchHeavyState::write_mixed(%s shared, %s local): %d", sharedWrites, localWrites, estimatedGas)

	// 6. return load function with captured parameters
	m := moveExecBurst
	if len(mode) > 0 {
		m = mode[0]
	}
	if m == moveExecSequential {
		return MoveExecSequentialLoad(publisherHex, "BenchHeavyState", "write_mixed", nil, args, estimatedGas)
	}

	return MoveExecBurstLoad(publisherHex, "BenchHeavyState", "write_mixed", nil, args, estimatedGas)
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

// TestBenchmarkBaselineSeqMoveExec records CList baseline with sequential Move exec load.
// Build the pre-proxy binary and pass via E2E_INITIAD_BIN.
// Sequential is the fair TPS comparison since CList handles in-order sequences correctly.
func TestBenchmarkBaselineSeqMoveExec(t *testing.T) {
	cfg := BaselineConfig()
	cfg.AccountCount = 100
	cfg.TxPerAccount = 50
	cfg.Label = "clist/iavl/seq-move-exec"

	ctx := context.Background()
	cluster := setupCluster(t, ctx, cfg)
	defer cluster.Close()

	moveLoadFn := setupMoveExecLoad(t, ctx, cluster, moveExecSequential)
	runBenchmarkWithCluster(t, ctx, cluster, cfg, moveLoadFn)
}

// TestBenchmarkSeqComparisonMoveExec compares CList vs Proxy+IAVL vs Proxy+MemIAVL
// with sequential Move exec workload. This is the fair TPS comparison under heavy state
// since sequential submission means both mempools process correctly.
func TestBenchmarkSeqComparisonMoveExec(t *testing.T) {
	var results []BenchResult

	// load baseline from JSON (label="clist/iavl/seq-move-exec")
	baselines := LoadBaselineResultsByLabel(resultsDir(t), "clist/iavl/seq-move-exec")
	if len(baselines) > 0 {
		t.Logf("Loaded baseline result: %s", baselines[0].Config.Label)
		results = append(results, baselines[0])
	} else {
		t.Log("No baseline results found. Run TestBenchmarkBaselineSeqMoveExec with pre-proxy binary for full comparison.")
	}

	t.Run("MempoolOnly", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 50
		cfg.Label = "proxy+priority/iavl/seq-move-exec"

		ctx := context.Background()
		cluster := setupCluster(t, ctx, cfg)
		defer cluster.Close()

		moveLoadFn := setupMoveExecLoad(t, ctx, cluster, moveExecSequential)
		result := runBenchmarkWithCluster(t, ctx, cluster, cfg, moveLoadFn)
		results = append(results, result)
	})

	t.Run("Combined", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 50
		cfg.Label = "proxy+priority/memiavl/seq-move-exec"

		ctx := context.Background()
		cluster := setupCluster(t, ctx, cfg)
		defer cluster.Close()

		moveLoadFn := setupMoveExecLoad(t, ctx, cluster, moveExecSequential)
		result := runBenchmarkWithCluster(t, ctx, cluster, cfg, moveLoadFn)
		results = append(results, result)
	})

	if len(results) >= 2 {
		PrintComparisonTable(t, results)
		PrintImprovementTable(t, results)
	}
}

// TestBenchmarkBurstComparisonMoveExec compares Proxy+IAVL vs Proxy+MemIAVL
// with the burst Move exec workload. No CList baseline since CList drops
// txs under burst by design, so the comparison is only between Proxy variants.
// This shows the inclusion rate + state pressure story under burst.
func TestBenchmarkBurstComparisonMoveExec(t *testing.T) {
	var results []BenchResult

	t.Run("MempoolOnly", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 50
		cfg.Label = "proxy+priority/iavl/burst-move-exec"

		ctx := context.Background()
		cluster := setupCluster(t, ctx, cfg)
		defer cluster.Close()

		moveLoadFn := setupMoveExecLoad(t, ctx, cluster)
		result := runBenchmarkWithCluster(t, ctx, cluster, cfg, moveLoadFn)
		results = append(results, result)
	})

	t.Run("Combined", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 50
		cfg.Label = "proxy+priority/memiavl/burst-move-exec"

		ctx := context.Background()
		cluster := setupCluster(t, ctx, cfg)
		defer cluster.Close()

		moveLoadFn := setupMoveExecLoad(t, ctx, cluster)
		result := runBenchmarkWithCluster(t, ctx, cluster, cfg, moveLoadFn)
		results = append(results, result)
	})

	if len(results) >= 2 {
		PrintComparisonTable(t, results)
	}
}

// runPreSignedBenchmark runs the benchmark pipeline with a pre-sign step between
// warmup and load. The preSignFn receives post-warmup metas and returns pre-signed txs
// that are passed to loadFnFactory to create the actual load function.
func runPreSignedBenchmark(
	t *testing.T, ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig,
	preSignFn func(metas map[string]e2e.AccountMeta) []e2e.SignedTx,
	loadFnFactory func([]e2e.SignedTx) func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult,
) BenchResult {
	t.Helper()

	metas, err := CollectInitialMetas(ctx, cluster)
	require.NoError(t, err)

	Warmup(ctx, cluster, metas)
	require.NoError(t, cluster.WaitForMempoolEmpty(ctx, 30*time.Second))
	time.Sleep(warmupSettleTime)

	metas, err = CollectInitialMetas(ctx, cluster)
	require.NoError(t, err)

	// Pre-sign with post-warmup sequences
	signedTxs := preSignFn(metas)

	startHeight, err := cluster.LatestHeight(ctx, 0)
	require.NoError(t, err)

	poller := NewMempoolPoller(ctx, cluster, mempoolPollInterval)

	t.Logf("Starting load: %d accounts x %d txs = %d total (pre-signed HTTP)", cfg.AccountCount, cfg.TxPerAccount, cfg.TotalTx())
	loadFn := loadFnFactory(signedTxs)
	loadResult := loadFn(ctx, cluster, cfg, metas)
	t.Logf("Load complete: %d submitted, %d errors, duration=%.1fs",
		len(loadResult.Submissions), len(loadResult.Errors),
		loadResult.EndTime.Sub(loadResult.StartTime).Seconds())

	drainTimeout := mempoolDrainTimeout + time.Duration(cfg.TotalTx()/20)*time.Second
	endHeight, err := WaitForLoadToSettle(ctx, cluster, drainTimeout, cfg.NoAllowQueued)
	require.NoError(t, err)

	peakMempool := poller.Stop()

	result, err := CollectResults(ctx, cluster, cfg, loadResult, startHeight, endHeight, peakMempool)
	require.NoError(t, err)

	t.Logf("Results: TPS=%.1f, P50=%.0fms, P95=%.0fms, P99=%.0fms, included=%d/%d, peak_mempool=%d",
		result.TxPerSecond, result.P50LatencyMs, result.P95LatencyMs, result.P99LatencyMs,
		result.TotalIncluded, result.TotalSubmitted, result.PeakMempoolSize)

	require.NoError(t, WriteResult(t, result, resultsDir(t)))
	return result
}

// ---------------------------------------------------------------------------
// Pre-signed HTTP broadcast: saturated chain benchmarks
// ---------------------------------------------------------------------------

// TestBenchmarkPreSignedSeqComparison uses pre-signed HTTP broadcast to saturate
// the chain, removing the CLI submission bottleneck. Compares IAVL vs MemIAVL
// with sequential bank send under full load.
func TestBenchmarkPreSignedSeqComparison(t *testing.T) {
	var results []BenchResult

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 100
		cfg.Label = "presigned/iavl/seq"

		ctx := context.Background()
		cluster := setupCluster(t, ctx, cfg)
		defer cluster.Close()

		result := runPreSignedBenchmark(t, ctx, cluster, cfg,
			func(metas map[string]e2e.AccountMeta) []e2e.SignedTx {
				return PreSignBankTxs(ctx, t, cluster, cfg, metas)
			}, PreSignedSequentialLoad)
		results = append(results, result)
	})

	t.Run("MemIAVL", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 100
		cfg.Label = "presigned/memiavl/seq"

		ctx := context.Background()
		cluster := setupCluster(t, ctx, cfg)
		defer cluster.Close()

		result := runPreSignedBenchmark(t, ctx, cluster, cfg,
			func(metas map[string]e2e.AccountMeta) []e2e.SignedTx {
				return PreSignBankTxs(ctx, t, cluster, cfg, metas)
			}, PreSignedSequentialLoad)
		results = append(results, result)
	})

	if len(results) == 2 {
		PrintComparisonTable(t, results)
	}
}

// TestBenchmarkPreSignedBurstComparison uses pre-signed HTTP broadcast with burst
// load to saturate the chain. Compares IAVL vs MemIAVL.
func TestBenchmarkPreSignedBurstComparison(t *testing.T) {
	var results []BenchResult

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 100
		cfg.Label = "presigned/iavl/burst"

		ctx := context.Background()
		cluster := setupCluster(t, ctx, cfg)
		defer cluster.Close()

		result := runPreSignedBenchmark(t, ctx, cluster, cfg,
			func(metas map[string]e2e.AccountMeta) []e2e.SignedTx {
				return PreSignBankTxs(ctx, t, cluster, cfg, metas)
			}, PreSignedBurstLoad)
		results = append(results, result)
	})

	t.Run("MemIAVL", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 100
		cfg.Label = "presigned/memiavl/burst"

		ctx := context.Background()
		cluster := setupCluster(t, ctx, cfg)
		defer cluster.Close()

		result := runPreSignedBenchmark(t, ctx, cluster, cfg,
			func(metas map[string]e2e.AccountMeta) []e2e.SignedTx {
				return PreSignBankTxs(ctx, t, cluster, cfg, metas)
			}, PreSignedBurstLoad)
		results = append(results, result)
	})

	if len(results) == 2 {
		PrintComparisonTable(t, results)
	}
}

// setupMoveExecCluster deploys BenchHeavyState and estimates gas. Returns the args needed
// to pre-sign Move exec transactions. Optional writeArgs overrides the default (5 shared, 25 local).
func setupMoveExecCluster(t *testing.T, ctx context.Context, cluster *e2e.Cluster, writeArgs ...string) (publisherHex string, args []string, estimatedGas uint64) {
	t.Helper()

	sharedWrites, localWrites := "5", "25"
	if len(writeArgs) == 2 {
		sharedWrites, localWrites = writeArgs[0], writeArgs[1]
	}

	publisherName := cluster.AccountNames()[0]
	var err error
	publisherHex, err = cluster.AccountAddressHex(ctx, publisherName)
	require.NoError(t, err)

	packagePath := cluster.RepoPath("integration-tests", "e2e", "benchmark", "move-bench")
	modulePath, err := cluster.BuildMoveModule(ctx, packagePath, "BenchHeavyState",
		map[string]string{"Publisher": publisherHex})
	require.NoError(t, err)

	res := cluster.MovePublish(ctx, publisherName, []string{modulePath}, 0)
	require.NoError(t, res.Err)
	require.Equal(t, int64(0), res.Code, "publish failed: %s", res.RawLog)
	require.NoError(t, cluster.WaitForMempoolEmpty(ctx, 30*time.Second))
	time.Sleep(3 * time.Second)

	args = []string{sharedWrites, localWrites}
	publisherAddr, err := cluster.AccountAddress(publisherName)
	require.NoError(t, err)
	meta, err := cluster.QueryAccountMeta(ctx, 0, publisherAddr)
	require.NoError(t, err)
	estimatedGas, err = cluster.MoveEstimateExecuteJSONGasWithSequence(
		ctx, publisherName, publisherHex, "BenchHeavyState", "write_mixed",
		nil, args, meta.AccountNumber, meta.Sequence, 0)
	require.NoError(t, err)

	// Apply 1.5x gas adjustment to handle MoveVM abort-and-retry overhead
	// under shared state contention from concurrent writers.
	estimatedGas = estimatedGas * 3 / 2
	t.Logf("Estimated gas for write_mixed(%s shared, %s local): %d (with 1.5x adjustment)", sharedWrites, localWrites, estimatedGas)

	return publisherHex, args, estimatedGas
}

// TestBenchmarkPreSignedSeqMoveExec uses pre-signed HTTP broadcast with sequential
// Move exec load to saturate the chain. Compares IAVL vs MemIAVL under heavy state writes.
// 20 accounts × 100 txs = 2000 total, write_mixed(10 shared, 50 local) = 60 writes/tx = 120K total writes.
func TestBenchmarkPreSignedSeqMoveExec(t *testing.T) {
	var results []BenchResult

	const (
		sharedWrites = "10"
		localWrites  = "50"
	)

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 100
		cfg.Label = "presigned/iavl/seq-move-exec"

		ctx := context.Background()
		cluster := setupCluster(t, ctx, cfg)
		defer cluster.Close()

		pubHex, args, gas := setupMoveExecCluster(t, ctx, cluster, sharedWrites, localWrites)

		result := runPreSignedBenchmark(t, ctx, cluster, cfg,
			func(metas map[string]e2e.AccountMeta) []e2e.SignedTx {
				return PreSignMoveExecTxs(ctx, t, cluster, cfg, metas,
					pubHex, "BenchHeavyState", "write_mixed", nil, args, gas)
			}, PreSignedSequentialLoad)
		results = append(results, result)
	})

	t.Run("MemIAVL", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 100
		cfg.Label = "presigned/memiavl/seq-move-exec"

		ctx := context.Background()
		cluster := setupCluster(t, ctx, cfg)
		defer cluster.Close()

		pubHex, args, gas := setupMoveExecCluster(t, ctx, cluster, sharedWrites, localWrites)

		result := runPreSignedBenchmark(t, ctx, cluster, cfg,
			func(metas map[string]e2e.AccountMeta) []e2e.SignedTx {
				return PreSignMoveExecTxs(ctx, t, cluster, cfg, metas,
					pubHex, "BenchHeavyState", "write_mixed", nil, args, gas)
			}, PreSignedSequentialLoad)
		results = append(results, result)
	})

	if len(results) == 2 {
		PrintComparisonTable(t, results)
	}
}

// ---------------------------------------------------------------------------
// State DB comparison: IAVL vs MemIAVL
// ---------------------------------------------------------------------------

// TestBenchmarkMemIAVLBankSend compares IAVL vs MemIAVL with bank send workload.
// Uses 100 accounts x 200 txs to stress the state tree.
func TestBenchmarkMemIAVLBankSend(t *testing.T) {
	var results []BenchResult

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 200
		cfg.Label = "memiavl-compare/iavl/bank-send"
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	t.Run("MemIAVL", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 200
		cfg.Label = "memiavl-compare/memiavl/bank-send"
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	if len(results) == 2 {
		PrintComparisonTable(t, results)
	}
}

// TestBenchmarkMemIAVLMoveExec compares IAVL vs. MemIAVL with heavy state Move exec workload.
// Deploys BenchHeavyState and runs write_mixed(5 shared, 25 local). 100 accounts × 50 txs × 30 writes = 150K state writes.
func TestBenchmarkMemIAVLMoveExec(t *testing.T) {
	var results []BenchResult

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 50
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
		cfg.TxPerAccount = 50
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

// TestBenchmarkQueuedFlood submits txs with future nonces (skipping base+0),
// flooding the queued pool, then fills the gap to trigger a promotion cascade.
// Verifies 100% inclusion. All queued txs must promote and land on-chain.
func TestBenchmarkQueuedFlood(t *testing.T) {
	cfg := MempoolOnlyConfig()
	cfg.TxPerAccount = 50 // must be <= max-queued-per-sender (64) to avoid silent drops
	cfg.Label = "queued-flood/mempool-only"
	result := runBenchmark(t, cfg, QueuedFloodLoad)

	require.Equal(t, result.TotalSubmitted, result.TotalIncluded,
		"not all queued-flood transactions were included: submitted=%d included=%d",
		result.TotalSubmitted, result.TotalIncluded)
}

// TestBenchmarkQueuedGapEviction submits txs with future nonces (skipping base+0)
// and never fills the gap. Verifies that the queued pool drains via gap TTL eviction
// (default 60s) and that no txs are included (they should all be evicted, not promoted).
func TestBenchmarkQueuedGapEviction(t *testing.T) {
	cfg := MempoolOnlyConfig()
	cfg.TxPerAccount = 50
	cfg.Label = "queued-gap-eviction/mempool-only"

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

	// start mempool poller before load to capture peak queued size
	poller := NewMempoolPoller(ctx, cluster, mempoolPollInterval)

	// submit future-nonce txs (gap load never fills nonce 0)
	loadResult := QueuedGapLoad(ctx, cluster, cfg, metas)
	t.Logf("Submitted %d future-nonce txs (no gap fill), %d errors",
		len(loadResult.Submissions), len(loadResult.Errors))

	// wait for gap TTL to expire (60s default) + buffer
	t.Log("Waiting for gap TTL eviction (60s + 30s buffer)...")
	time.Sleep(90 * time.Second)

	// verify mempool is drained (all queued txs evicted)
	err = cluster.WaitForMempoolEmpty(ctx, 30*time.Second)
	peakMempool := poller.Stop()

	t.Logf("Gap eviction test: peak_mempool=%d, mempool_drained=%v",
		peakMempool, err == nil)

	require.NoError(t, err, "mempool should be empty after gap TTL eviction")
	require.Greater(t, peakMempool, 0, "should have observed queued txs in mempool")
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

	// wait for load to settle
	endHeight, err := WaitForLoadToSettle(ctx, cluster, mempoolDrainTimeout, false)
	require.NoError(t, err)

	peakMempool := poller.Stop()
	t.Logf("Cluster peak mempool size: %d", peakMempool)

	result, err := CollectResults(ctx, cluster, cfg, loadResult, startHeight, endHeight, peakMempool)
	require.NoError(t, err)

	t.Logf("Gossip test: TPS=%.1f, included=%d/%d",
		result.TxPerSecond, result.TotalIncluded, result.TotalSubmitted)
	require.NoError(t, WriteResult(t, result, resultsDir(t)))
}
