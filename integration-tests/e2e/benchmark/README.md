# E2E Benchmark

Performance benchmark for ProxyMempool + PriorityMempool + MemIAVL. Measures throughput, latency, and mempool behavior
across optimization layers using a multi-node cluster with production-realistic settings.

## Cluster topology

8-node cluster: 5 validators + 3 edge (non-validator) nodes on localhost with deterministic port allocation.

- **Block gas limit**: 200,000,000 (matches Initia mainnet `consensus_params.block.max_gas`)
- **Edge-only submission**: All benchmark load is submitted exclusively to non-validator edge nodes (indices 5-7), testing gossip propagation from edge to validator.
- **Timeout commit**: 500ms
- **Queued tx extension**: All tx submissions include `--allow-queued` flag (required for `ExtensionOptionQueuedTx`)

## Comparison matrix

### 1. Mempool comparison: CList vs Proxy+Priority

Three load patterns: sequential tests give a fair TPS comparison (both mempools handle in-order correctly), burst tests demonstrate CList's tx-drop problem.

**Baselines** (run with v1.3.1 binary):

| Test | Load | Config |
|---|---|---|
| `TestBenchmarkBaselineSeq` | Sequential, bank send | 10 accts x 200 txs |
| `TestBenchmarkBaselineBurst` | Burst, bank send | 10 accts x 200 txs |
| `TestBenchmarkBaselineSeqMoveExec` | Sequential, Move exec | 100 accts x 50 txs, 30 writes/tx |

**Comparisons** (3-way: CList vs Proxy+IAVL vs Proxy+MemIAVL):

| Test | Load | Purpose                                                                                                |
|---|---|--------------------------------------------------------------------------------------------------------|
| `TestBenchmarkSeqComparison` | Sequential, bank send | Fair TPS comparison, lightweight workload                                                              |
| `TestBenchmarkSeqComparisonMoveExec` | Sequential, Move exec | Fair TPS comparison under heavy state pressure                                                         |
| `TestBenchmarkBurstComparison` | Burst, bank send | Inclusion rate (CList drops txs)                                                                       |
| `TestBenchmarkBurstComparisonMoveExec` | Burst, Move exec | Proxy+IAVL vs Proxy+MemIAVL under burst + heavy state (no CList baseline since it would just drop txs) |

### 2. State DB comparison: IAVL vs MemIAVL

Both use Proxy+Priority mempool. Isolates state storage impact.

| Test | Workload | Config |
|---|---|---|
| `TestBenchmarkMemIAVLBankSend` | Bank sends | 100 accts x 200 txs |
| `TestBenchmarkMemIAVLMoveExec` | Move exec (BenchHeavyState::write_mixed) | 100 accts x 50 txs, 30 writes/tx |

### 3. Pre-signed HTTP broadcast (saturated chain)

Bypasses CLI bottleneck. Txs are generated+signed offline, then POSTed via HTTP.

| Test | Load | Config |
|---|---|---|
| `TestBenchmarkPreSignedSeqComparison` | Sequential, bank send, HTTP | 20 accts x 100 txs |
| `TestBenchmarkPreSignedBurstComparison` | Burst, bank send, HTTP | 20 accts x 100 txs |
| `TestBenchmarkPreSignedSeqMoveExec` | Sequential, Move exec, HTTP | 20 accts x 100 txs, 60 writes/tx |

### 4. Capability demos

| Test | What | Config |
|---|---|---|
| `TestBenchmarkQueuePromotion` | Out-of-order nonce handling, 100% inclusion | 10 accts x 50 txs |
| `TestBenchmarkGossipPropagation` | Gossip across nodes | 5 accts x 50 txs |
| `TestBenchmarkQueuedFlood` | Future-nonce burst (nonce gaps), queued pool stress + promotion cascade | 10 accts x 50 txs |
| `TestBenchmarkQueuedGapEviction` | Gap TTL eviction under sustained load | 10 accts x 50 txs |

## Expected outcomes

1. **Sequential (fair comparison)**: CList and Proxy+Priority both handle in-order nonces correctly, so sequential submission should show similar TPS. This is the control that proves Proxy+Priority doesn't regress on the happy path.
2. **Burst (stress test)**: Proxy+Priority >> CList. Under burst, CList's `reCheckTx` and cache-based dedup cause it to silently drop txs, while Proxy+Priority's queued pool absorbs out-of-order arrivals and achieves 100% inclusion.
3. **Heavy state writes**: MemIAVL > IAVL. Lightweight workloads (bank send) won't show a difference because the state DB isn't the bottleneck. Heavy Move exec with many writes per tx is needed, and the chain must be saturated (pre-signed HTTP) so the state DB becomes the limiting factor.
4. **Combined (Proxy+Priority+MemIAVL)**: Best overall throughput and latency, the mempool improvement eliminates tx drops, and MemIAVL reduces state commit time under heavy writes.

## Results

### Mempool comparison: CList vs Proxy+Priority (sequential bank send)

8-node cluster (5 val + 3 edge), 10 accounts x 200 txs = 2000 total, sequential submission via edge nodes.

```
Config                     | Variant      |   TPS | vs base |   P50ms | vs base |   P95ms | vs base | Peak Mempool
clist/iavl/seq             | baseline     |  39.5 |       - |      17 |       - |     335 |       - |           47
proxy+priority/iavl/seq    | mempool-only |  39.5 |   +0.0% |      16 |   -5.9% |     339 |   +1.2% |           45
proxy+priority/memiavl/seq | combined     |  38.5 |   -2.5% |      26 |  +52.9% |     342 |   +2.1% |           42
```

Sequential submission is low-pressure, all three variants perform identically (~39 TPS). This confirms a fair baseline where the mempool type doesn't matter.

### Mempool comparison: CList vs Proxy+Priority (burst bank send)

8-node cluster (5 val + 3 edge), 10 accounts x 200 txs = 2000 total, burst submission via edge nodes.

```
Config                      | Variant      |   TPS | vs base | Included  | Peak Mempool
clist/iavl/burst            | baseline     |  18.5 |       - |  935/2000 |           36
proxy+priority/iavl/burst   | mempool-only |  39.8 | +115.1% | 2000/2000 |           44
```

Under burst, CList drops 53% of txs (935/2000 included) this is expected since CList cannot accept out-of-order txs 
while Proxy+Priority includes 100%. **+115% TPS improvement** with full inclusion.

### Mempool comparison: CList vs Proxy+Priority (sequential Move exec with heavy state)

8-node cluster (5 val + 3 edge), 100 accounts x 50 txs = 5000 total, sequential submission.
BenchHeavyState::write_mixed(5 shared, 25 local) 30 state writes per tx, 150K total writes.

```
Config                              | Variant      |   TPS | vs base |   P50ms | vs base |   P95ms | vs base | Included  | Peak Mempool
clist/iavl/seq-move-exec            | baseline     |  23.3 |       - |    5452 |       - |   43683 |       - | 2705/5000 |         1000
proxy+priority/iavl/seq-move-exec   | mempool-only |  36.0 |  +54.5% |    3236 |  -40.6% |    6250 |  -85.7% | 5000/5000 |          569
proxy+priority/memiavl/seq-move-exec| combined     |  36.1 |  +54.9% |    3816 |  -30.0% |    7606 |  -82.6% | 5000/5000 |          629
```

Under heavy state pressure, Proxy+Priority delivers **+55% TPS** and **-83-86% P95 latency** vs CList. CList only included 2705/5000 txs (54%) due to tx drops.

### Burst Move exec: Proxy+IAVL vs Proxy+MemIAVL

8-node cluster (5 val + 3 edge), 100 accounts x 50 txs = 5000 total, burst submission.
BenchHeavyState::write_mixed(5 shared, 25 local) 30 state writes per tx, 150K total writes.

```
Config                                 | Variant      |   TPS |   P50ms |   P95ms |   P99ms | Included  | Peak Mempool
proxy+priority/iavl/burst-move-exec    | mempool-only |  37.7 |   16440 |   34538 |   37604 | 5000/5000 |         1913
proxy+priority/memiavl/burst-move-exec | combined     |  40.2 |   14849 |   30864 |   33882 | 5000/5000 |         1872
                                                         +6.6%    -9.7%    -10.6%     -9.9%
```

Under burst with heavy state, MemIAVL shows **+6.6% TPS** and **~10% lower latencies** across all percentiles. Both variants achieve 100% inclusion.
See below further for even more stressful tests regarding stores.

### State db comparison: IAVL vs MemIAVL (bank sends)

8-node cluster (5 val + 3 edge), 100 accounts x 200 txs = 20000 total, burst mode via edge nodes.

```
Config                           | Variant      |   TPS |   P50ms |   P95ms |   P99ms | Included    | Peak Mempool
memiavl-compare/iavl/bank-send   | mempool-only |  37.4 |    3555 |    7848 |   12576 | 20000/20000 |          955
memiavl-compare/memiavl/bank-send| combined     |  38.1 |    3912 |   11917 |   16170 | 20000/20000 |         1078
```

Bank sends are lightweight, so IAVL vs. MemIAVL shows no meaningful difference. This confirms the heavy-state workload is necessary to expose the state db bottleneck.

### Pre-signed HTTP broadcast (saturated chain)

Pre-signed txs bypass CLI overhead (~40 tx/s bottleneck) and submit via HTTP to
`/broadcast_tx_sync`, saturating the chain at 2000-4000 tx/s submission rate.

**Bank send (sequential)**: 20 accounts x 100 txs = 2000 total

```
Config                    | Variant      |   TPS |   P50ms |   P95ms |   P99ms | Included  | Peak Mempool
presigned/iavl/seq        | mempool-only | 219.1 |    1623 |    4092 |    4152 | 2000/2000 |         2000
presigned/memiavl/seq     | combined     | 210.2 |    2087 |    4228 |    4327 | 2000/2000 |         2000
```

**Bank send (burst)**: 20 accounts x 100 txs = 2000 total

```
Config                    | Variant      |   TPS |   P50ms |   P95ms |   P99ms | Included  | Peak Mempool
presigned/iavl/burst      | mempool-only | 217.5 |    1695 |    3992 |    4016 | 2000/2000 |         2000
presigned/memiavl/burst   | combined     | 218.7 |    1620 |    3884 |    3896 | 2000/2000 |         2000
```

**Move exec (sequential)**: 20 accounts x 100 txs = 2000 total, 60 writes/tx (write_mixed(10, 50))

```
Config                         | Variant      |   TPS |   P50ms |   P95ms |   P99ms | Included  | Peak Mempool
presigned/iavl/seq-move-exec   | mempool-only | 150.2 |    3021 |    5993 |    7389 | 2000/2000 |         1753
presigned/memiavl/seq-move-exec| combined     | 185.6 |    3007 |    5351 |    6253 | 2000/2000 |         1884
                                                +23.6%   -0.5%   -10.7%   -15.4%
```

With the chain fully saturated, real TPS is **5-6x higher** than CLI-bottlenecked tests
(~218 TPS for bank send vs ~39 TPS. ~186 TPS for Move exec vs ~36 TPS). Bank send shows
no IAVL vs MemIAVL difference (lightweight workload), but **Move exec with heavy state
writes reveals a clear +23.6% TPS and -15% P99 latency advantage for MemIAVL**. 100%
inclusion across all tests.

### Capability demos

```
Config                    | Variant      |   TPS | Included  | Peak Mempool
queue-promotion           | mempool-only |  35.0 |   500/500 |           46
gossip/mempool-only       | mempool-only |  23.9 |   250/250 |           35
queued-flood              | mempool-only |  27.9 |   500/500 |          500
```

- **Queue promotion**: 100% inclusion with out-of-order nonces `[seq+2, seq+0, seq+1]`. Confirms the proxy mempool correctly queues and promotes.
- **Gossip**: All 250 txs submitted to node 0 propagated to the full cluster and were included.
- **Queued flood**: 500 txs with future nonces (gap at base+0) all promoted and included after gap fill. Peak mempool = 500 confirms all txs were queued simultaneously.
- **Gap eviction**: Verified that queued txs with unfilled gaps are evicted after the 60s gap TTL expires.

## Run

All commands assume `cd integration-tests/e2e` first. The full workflow has 3 phases:
baselines first, then current-branch benchmarks, then the comparison tests that
load both result sets. Capability demos / queued tests are standalone and can run
any time.

### Phase 1 Collecting baselines (CList mempool)

Build the pre-proxy binary once, then run the three baseline tests.
Results are written to `benchmark/results/` as JSON keyed by label.

```bash
# Build pre-proxy binary
git checkout tags/v1.3.1
make build
cp build/initiad build/initiad-baseline
git checkout -   # return to current branch

cd integration-tests/e2e

# Sequential bank send baseline
E2E_INITIAD_BIN="$(pwd)/../../build/initiad-baseline" \
  go test -v -tags benchmark -run TestBenchmarkBaselineSeq -timeout 30m -count=1 ./benchmark/

# Burst bank send baseline
E2E_INITIAD_BIN="$(pwd)/../../build/initiad-baseline" \
  go test -v -tags benchmark -run TestBenchmarkBaselineBurst -timeout 30m -count=1 ./benchmark/

# Sequential Move exec baseline
E2E_INITIAD_BIN="$(pwd)/../../build/initiad-baseline" \
  go test -v -tags benchmark -run TestBenchmarkBaselineSeqMoveExec -timeout 60m -count=1 ./benchmark/
```

### Phase 2 Running current-branch benchmarks

These use the current binary (auto-built or via `E2E_INITIAD_BIN`).
Each test writes its own result JSON.

```bash
# State db comparison (IAVL vs MemIAVL)
go test -v -tags benchmark -run TestBenchmarkMemIAVLBankSend -timeout 60m -count=1 ./benchmark/
go test -v -tags benchmark -run TestBenchmarkMemIAVLMoveExec -timeout 60m -count=1 ./benchmark/

# Capability demos (standalone no baselines needed)
go test -v -tags benchmark -run TestBenchmarkQueuePromotion -timeout 30m -count=1 ./benchmark/
go test -v -tags benchmark -run TestBenchmarkGossipPropagation -timeout 30m -count=1 ./benchmark/

# Queued mempool behavior (standalone no baselines needed)
go test -v -tags benchmark -run TestBenchmarkQueuedFlood -timeout 30m -count=1 ./benchmark/
go test -v -tags benchmark -run TestBenchmarkQueuedGapEviction -timeout 30m -count=1 ./benchmark/
```

### Pre-signed HTTP broadcast tests (saturated chain)

These use pre-signed txs via HTTP to saturate the chain, bypassing the CLI bottleneck.

```bash
# Sequential bank send (IAVL vs MemIAVL)
go test -v -tags benchmark -run TestBenchmarkPreSignedSeqComparison -timeout 20m -count=1 ./benchmark/

# Burst bank send (IAVL vs MemIAVL)
go test -v -tags benchmark -run TestBenchmarkPreSignedBurstComparison -timeout 20m -count=1 ./benchmark/

# Sequential Move exec (IAVL vs MemIAVL)
go test -v -tags benchmark -run TestBenchmarkPreSignedSeqMoveExec -timeout 30m -count=1 ./benchmark/
```

### Phase 3 Comparing tests (baseline vs current)

These load baseline JSONs from `benchmark/results/` by label and run Proxy+IAVL
and Proxy+MemIAVL variants, then print a side-by-side comparison table with deltas.

```bash
# Sequential bank send: CList vs Proxy+IAVL vs Proxy+MemIAVL
go test -v -tags benchmark -run TestBenchmarkSeqComparison -timeout 30m -count=1 ./benchmark/

# Sequential Move exec: CList vs Proxy+IAVL vs Proxy+MemIAVL
go test -v -tags benchmark -run TestBenchmarkSeqComparisonMoveExec -timeout 60m -count=1 ./benchmark/

# Burst bank send: CList vs Proxy+IAVL vs Proxy+MemIAVL
go test -v -tags benchmark -run TestBenchmarkBurstComparison -timeout 30m -count=1 ./benchmark/

# Burst Move exec: Proxy+IAVL vs Proxy+MemIAVL (no CList since it drops txs under burst)
go test -v -tags benchmark -run TestBenchmarkBurstComparisonMoveExec -timeout 60m -count=1 ./benchmark/
```

Each Phase 3 test prints a comparison table like:

```
Config                    | Variant      |     TPS | vs base |   P50ms | vs base |   P95ms | vs base | Peak Mempool
clist/iavl/seq            | baseline     |   120.5 |       - |    2500 |       - |    4800 |       - |         1950
proxy+priority/iavl/seq   | mempool-only |   245.3 | +103.6% |    1823 |  -27.1% |    3412 |  -28.9% |         1847
proxy+priority/memiavl/seq| combined     |   312.7 | +159.5% |    1401 |  -44.0% |    2845 |  -40.7% |         1823
```

## Configuration

### Ground Rules

1. Baseline requires a separate binary built from the pre-proxy CometBFT tag and pre abcipp changes in initia.
2. Run baseline and current benchmarks on the same machine.
3. Warmup runs before every measured load (5 txs, metadata re-queried after).
4. TPS is derived from block timestamps, not submission wall clock.
5. Latency = `block_time - submit_time` (covers mempool wait, gossip, proposal, execution).
6. Block gas limit matches Initia mainnet (200M) to prevent unrealistic mega-blocks.
7. Load is submitted only to edge (non-validator) nodes to test realistic gossip propagation.

### Configurable mempool limits

These can be tuned in `app.toml` under `[abcipp]` (defaults shown):

| Parameter | Default | Description |
|---|---|---|
| `max-queued-per-sender` | 64 | Max queued txs per sender |
| `max-queued-total` | 1024 | Max queued txs globally |
| `queued-gap-ttl` | 60s | TTL for stalled senders missing head nonce |

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `E2E_INITIAD_BIN` | (auto-build) | Path to prebuilt `initiad` binary |
| `BENCHMARK_RESULTS_DIR` | `results/` | Output directory for JSON results |

## Structure

```
benchmark/
  config.go          Variant definitions, BenchConfig, preset constructors
  load.go            Load generators (BurstLoad, SequentialLoad, OutOfOrderLoad, SingleNodeLoad, MoveExecBurstLoad, MoveExecSequentialLoad, QueuedFloodLoad, PreSignBankTxs, PreSignMoveExecTxs, PreSignedBurstLoad, PreSignedSequentialLoad)
  collector.go       MempoolPoller, CollectResults, latency aggregation
  report.go          JSON output, comparison tables, delta calculations, LoadBaselineResultsByLabel
  benchmark_test.go  Test suite (build-tagged `benchmark`)
  move-bench/        Standalone Move package (BenchHeavyState module)
  results/           JSON output directory
```

### Load generators

All load generators route transactions to edge nodes when `ValidatorCount > 0`.

- **BurstLoad**: All accounts submit concurrently with sequential nonces, round-robin across edge nodes.
- **SequentialLoad**: Accounts run concurrently, but each account sends txs one-at-a-time. Each account pinned to a single edge node.
- **OutOfOrderLoad**: First 3 txs per account use `[seq+2, seq+0, seq+1]` to test queue promotion.
- **SingleNodeLoad**: All txs to a single node for gossip propagation measurement.
- **MoveExecBurstLoad**: Like BurstLoad but calls `SendMoveExecuteJSONWithGas` instead of bank sends.
- **MoveExecSequentialLoad**: Like SequentialLoad but calls `SendMoveExecuteJSONWithGas`. Each account pinned to a single edge node.
- **QueuedFloodLoad**: Sends txs with nonces `[base+1..base+N]` (skipping `base+0`), then after all are submitted, sends the gap-filling `base+0` tx to trigger promotion cascade.
- **PreSignedBurstLoad**: Broadcasts pre-signed txs via HTTP POST to `/broadcast_tx_sync`. All accounts concurrent, round-robin across edge nodes.
- **PreSignedSequentialLoad**: Broadcasts pre-signed txs via HTTP POST. Each account pinned to a single edge node, txs sent sequentially per account.

### Metrics

| Metric | Source |
|---|---|
| **TPS** | `included_tx_count / block_time_span` |
| **Latency** (avg, p50, p95, p99, max) | `block_timestamp - submit_timestamp` per tx |
| **Peak mempool size** | Goroutine polling `/num_unconfirmed_txs` every 500ms |
| **Per-block tx count** | CometBFT RPC `/block?height=N` |

## Move exec workload: BenchHeavyState

The Move exec tests deploy the `BenchHeavyState` module at runtime. Each tx calls `write_mixed(shared_count, local_count)` which performs:

- **shared writes** to a global `SharedState` table at `@Publisher` (contended).
- **local writes** to the caller's own `State` table (non-contended).

CLI-based tests use `write_mixed(5, 25)` = 30 writes/tx. Pre-signed HTTP tests use `write_mixed(10, 50)` = 60 writes/tx.

This mixed contention pattern is more realistic than pure local or pure shared workloads, reflecting real-world patterns where some operations touch shared state and others are account-local.
