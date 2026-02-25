# E2E Benchmark

Performance benchmark for ProxyMempool + PriorityMempool + MemIAVL. Measures throughput, latency, and mempool behavior
across optimization layers using a multi-node cluster.

## Comparison matrix

### 1. Mempool comparison: CList vs Proxy+Priority

Sequential and burst load patterns with bank sends.

| Test | Load | Config |
|---|---|---|
| `TestBenchmarkBaselineSeq` | Sequential | 10 accts x 200 txs, label `clist/iavl/seq` |
| `TestBenchmarkBaselineBurst` | Burst | 10 accts x 200 txs, label `clist/iavl/burst` |
| `TestBenchmarkSeqComparison` | Sequential | Loads baseline-seq, runs Proxy+IAVL + Proxy+MemIAVL |
| `TestBenchmarkBurstComparison` | Burst | Loads baseline-burst, runs Proxy+IAVL + Proxy+MemIAVL |

### 2. State DB comparison: IAVL vs MemIAVL

Both use Proxy+Priority mempool. Isolates state storage impact.

| Test | Workload | Config              |
|---|---|---------------------|
| `TestBenchmarkMemIAVLBankSend` | Bank sends | 100 accts x 300 txs |
| `TestBenchmarkMemIAVLMoveExec` | Move exec (Counter::increase) | 100 accts x 300 txs |

### 3. Capability demos

| Test | What | Config |
|---|---|---|
| `TestBenchmarkQueuePromotion` | Out-of-order nonce handling, 100% inclusion | 10 accts x 50 txs |
| `TestBenchmarkGossipPropagation` | Gossip across nodes | 5 accts x 50 txs |

## Baseline collection

Two baselines are needed (sequential + burst). Build the pre-proxy binary and run each:

```bash
# 1. Build pre-proxy binary
git checkout tags/v1.3.1
make build

# 2. Run sequential baseline
cd integration-tests/e2e && \
  E2E_INITIAD_BIN=../../build/initiad \
  go test -v -tags benchmark -run TestBenchmarkBaselineSeq -timeout 30m -count=1 ./benchmark/

# 3. Run burst baseline
cd integration-tests/e2e && \
  E2E_INITIAD_BIN=../../build/initiad \
  go test -v -tags benchmark -run TestBenchmarkBaselineBurst -timeout 30m -count=1 ./benchmark/

# 4. Return to current branch
git checkout -
```

Results land in `benchmark/results/` as JSON. Comparison tests load them automatically by label.

## Move exec workload

The `TestBenchmarkMemIAVLMoveExec` test deploys the Counter module at runtime:

1. Builds `x/move/keeper/contracts/sources/Counter.move`
2. Publishes via acc1
3. Estimates gas once for `Counter::increase()`
4. Uses `MoveExecBurstLoad` with the estimated gas for all subsequent txs

No pre-deployment or external setup needed.

## Run

Full benchmark suite:

```bash
make benchmark-e2e
```

Single test:

```bash
cd integration-tests/e2e && \
  go test -v -tags benchmark -run TestBenchmarkSeqComparison -timeout 30m -count=1 ./benchmark/
```

Comparison tests (after baselines exist):

```bash
cd integration-tests/e2e && \
  go test -v -tags benchmark -run "TestBenchmarkSeqComparison|TestBenchmarkBurstComparison" -timeout 30m -count=1 ./benchmark/
```

## Rules

1. Baseline requires a separate binary built from the pre-proxy CometBFT tag.
2. Run baseline and current benchmarks on the same machine.
3. Warmup runs before every measured load (5 txs, metas re-queried after).
4. TPS is derived from block timestamps, not submission wall clock.
5. Latency = `block_time - submit_time` (covers mempool wait, gossip, proposal, execution).

## Structure

```
benchmark/
  config.go          Variant definitions, BenchConfig, preset constructors
  load.go            Load generators (BurstLoad, SequentialLoad, OutOfOrderLoad, SingleNodeLoad, MoveExecBurstLoad)
  collector.go       MempoolPoller, CollectResults, latency aggregation
  report.go          JSON output, comparison tables, delta calculations, LoadBaselineResultsByLabel
  benchmark_test.go  Test suite (build-tagged `benchmark`)
  results/           JSON output directory
```

### Cluster topology

3-node cluster (1 validator + 2 full nodes) on localhost with deterministic port allocation.

### Load generators

- **BurstLoad**: All accounts submit concurrently with sequential nonces, round-robin across nodes.
- **SequentialLoad**: Accounts run concurrently, but each account sends txs one-at-a-time. Each account pinned to a single node.
- **OutOfOrderLoad**: First 3 txs per account use `[seq+2, seq+0, seq+1]` to test queue promotion.
- **SingleNodeLoad**: All txs to a single node for gossip propagation measurement.
- **MoveExecBurstLoad**: Like BurstLoad but calls `SendMoveExecuteJSONWithGas` instead of bank sends.

### Metrics

| Metric | Source |
|---|---|
| **TPS** | `included_tx_count / block_time_span` |
| **Latency** (avg, p50, p95, p99, max) | `block_timestamp - submit_timestamp` per tx |
| **Peak mempool size** | Goroutine polling `/num_unconfirmed_txs` every 500ms |
| **Per-block tx count** | CometBFT RPC `/block?height=N` |

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `E2E_INITIAD_BIN` | (auto-build) | Path to prebuilt `initiad` binary |
| `BENCHMARK_RESULTS_DIR` | `results/` | Output directory for JSON results |
