# E2E Benchmark

Performance benchmark for ProxyMempool + PriorityMempool + MemIAVL. Measures throughput, latency, and mempool behavior 
across layers using a multi-node cluster.

## Background

Two independent optimizations target different bottlenecks:

| Optimization | Bottleneck |
|---|---|
| **ProxyMempool + PriorityMempool** | Mempool gossip, tx ordering, queue promotion |
| **MemIAVL** | State storage read/write during block execution |

These compounds. Faster state commits let the mempool drain sooner, reducing gossip pressure and 
improving end-to-end latency. The benchmark isolates each layer and measures them together.

## Three-way comparison

| Variant | Mempool | State DB | Label |
|---|---|---|---|
| **baseline** | CListMempool (CometBFT default) | Standard IAVL | `clist/iavl` |
| **mempool-only** | ProxyMempool + PriorityMempool | Standard IAVL | `proxy+priority/iavl` |
| **combined** | ProxyMempool + PriorityMempool | MemIAVL | `proxy+priority/memiavl` |

What each delta means:

- **baseline -> mempool-only**: mempool improvement alone
- **mempool-only -> combined**: incremental gain from MemIAVL on top
- **baseline -> combined**: total end-to-end improvement

## Rules

1. Baseline requires a separate binary.

   - Baseline results come from the pre-proxy CometBFT tag. Build that binary, pass it via `E2E_INITIAD_BIN`, and run `TestBenchmarkBaseline`.
   - The JSON results persist in `results/`. Subsequent `TestBenchmarkFullComparison` runs load them automatically.

2. Run baseline and current benchmarks on the same machine.

   - Hardware variance will dominate if you compare across machines.

3. Warmup runs before every measured load.

   - 5 txs are sent before the actual burst to ensure the cluster is producing blocks and accounts are initialized on-chain.
   - Metas are re-queried after warmup since sequences change.

4. TPS is derived from block timestamps, not submission wall clock.

   - This measures actual chain throughput, not how fast the client can broadcast.

5. Latency includes the full pipeline.

   - `block_time - submit_time` covers mempool wait, gossip, proposal, and execution.

## Run

Full benchmark suite (mempool-only + combined):

```bash
make benchmark-e2e
```

Single test:

```bash
cd integration-tests/e2e && \
  go test -v -tags benchmark -run TestBenchmarkThroughput -timeout 30m -count=1 ./benchmark/
```

## Collecting baseline

```bash
# 1. Build pre-proxy binary
git checkout tags/v1.3.1
make build

# 2. Run baseline
E2E_INITIAD_BIN=./build/initiad \
  cd integration-tests/e2e && \
  go test -v -tags benchmark -run TestBenchmarkBaseline -timeout 30m -count=1 ./benchmark/

# 3. Return to current branch
git checkout -
```

Results land in `benchmark/results/` as JSON.

## Full three-way comparison

After baseline results exist in `results/`:

```bash
cd integration-tests/e2e && \
  go test -v -tags benchmark -run TestBenchmarkFullComparison -timeout 30m -count=1 ./benchmark/
```

Runs mempool-only and combined, loads the baseline from JSON, and prints:

```
Config                    | Variant      |     TPS | vs base |   P50ms | vs base |   P95ms | vs base | Peak Mempool
clist/iavl                | baseline     |   120.5 |       - |    2500 |       - |    4800 |       - |         1950
proxy+priority/iavl       | mempool-only |   245.3 | +103.6% |    1823 |  -27.1% |    3412 |  -28.9% |         1847
proxy+priority/memiavl    | combined     |   312.7 | +159.5% |    1401 |  -44.0% |    2845 |  -40.7% |         1823
```

Positive `vs base` for TPS = improvement. Negative `vs base` for latency = improvement (lower is better).

## Test suite

All tests are build-tagged `//go:build benchmark`. Prefix: `TestBenchmark`.

| Test | What it measures | Load pattern | Config |
|---|---|---|---|
| `TestBenchmarkBaseline` | Baseline throughput (CListMempool + IAVL) | Burst, sequential nonces | 10 accts x 200 txs |
| `TestBenchmarkThroughput` | Throughput with mempool-only | Burst, sequential nonces | 10 accts x 200 txs |
| `TestBenchmarkLatency` | Latency distribution (avg/p50/p95/p99/max) | Burst, sequential nonces | 10 accts x 200 txs |
| `TestBenchmarkQueuePromotion` | Out-of-order nonce handling, 100% inclusion | Out-of-order first 3, then sequential | 10 accts x 50 txs |
| `TestBenchmarkFullComparison` | Three-way comparison with deltas | Burst, sequential nonces | 10 accts x 200 txs |
| `TestBenchmarkSaturation` | Mempool under pressure | Burst, sequential nonces | 5 accts x 100 txs |
| `TestBenchmarkGossipPropagation` | Gossip across nodes | All txs to node 0 | 5 accts x 50 txs |

## Structure

```
benchmark/
  config.go          Variant definitions, BenchConfig, preset constructors
  load.go            Load generators (BurstLoad, OutOfOrderLoad, SingleNodeLoad)
  collector.go       MempoolPoller, CollectResults, latency aggregation
  report.go          JSON output, comparison tables, delta calculations
  benchmark_test.go  Test suite (build-tagged `benchmark`)
  results/           JSON output directory
```

### Cluster topology

3-node cluster (1 validator + 2 full nodes) on localhost with deterministic port allocation. Default load: 10 accounts x 200 txs = 2000 total.

### Load generators

- **BurstLoad**: All accounts submit concurrently with sequential nonces, round-robin across nodes.
- **OutOfOrderLoad**: First 3 txs per account use `[seq+2, seq+0, seq+1]` to test queue promotion, rest sequential.
- **SingleNodeLoad**: All txs to a single node for gossip propagation measurement.

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

## Output format

Each test writes a JSON result file:

```json
{
  "config": {
    "memiavl": false,
    "node_count": 3,
    "account_count": 10,
    "tx_per_account": 200,
    "label": "proxy+priority/iavl",
    "variant": "mempool-only"
  },
  "total_submitted": 2000,
  "total_included": 2000,
  "duration_sec": 8.15,
  "tx_per_second": 245.3,
  "avg_latency_ms": 2103.4,
  "p50_latency_ms": 1823.0,
  "p95_latency_ms": 3412.0,
  "p99_latency_ms": 5102.0,
  "max_latency_ms": 6230.0,
  "peak_mempool_size": 1847,
  "block_stats": [
    {"height": 5, "tx_count": 312, "time": "..."},
    {"height": 6, "tx_count": 298, "time": "..."}
  ]
}
```
