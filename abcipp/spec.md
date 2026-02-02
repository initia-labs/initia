# ABCI++ Package

The `abcipp` package wires up the ABCI++ surfaces that Initia needs: a priority-aware mempool, the proposal/CheckTx handlers that keep the pool in sync with CometBFT, and the ABCI++ gRPC queries that surface mempool state to clients.

## Contracts

### Mempool interface extensions

* `Mempool` builds on `sdkmempool.Mempool` by exposing callers to mempool-specific metadata:
  * `Contains` and `Lookup` support existence and sender/nonce lookups without leaking internals.
  * `GetTxDistribution` returns a tier name→count map so telemetry can track whether low/high priority lanes are flowing.
  * `GetTxInfo` reports the `TxInfo` struct (sender, sequence, size, gas limit, bytes, tier) used during proposal creation.
  * `NextExpectedSequence` surfaces the next nonce that would be accepted for a sender, which blocks can use to maintain nonce ordering.
* `AccountKeeper` and `BaseApp` provide the minimal hooks the mempool needs for cleanup (`GetSequence`) and for simulating transactions during ante checks (`GetContextForSimulate`).

## Priority mempool architecture

1. **Configuration**
   * `PriorityMempoolConfig` governs the upper transaction limit (`MaxTx`), the ordered `Tiers` to prefer, and the `AnteHandler` used to revalidate cached transactions.
   * `Tier`/`TierMatcher` pairs are canonicalized by `buildTierMatchers`, which trims empty names and always produces a fallback `default` tier.

2. **Data model**
   * Entries are stored in three structures: a skiplist rooted at `priorityIndex` (ordered by tier, priority, insertion order, sender, nonce), a map keyed by `(sender, nonce)` for fast lookups, and per-sender `userBucket`s that track contiguous nonce ranges and provide hints for cleaning.
   * Each `txEntry` bundles the transaction, priority, sequence, tier index, gas, encoded bytes, and an insertion order to break ties.
   * Tier counts are tracked in `tierDistribution`, keeping per-tier occupancies up to date.

3. **Admission & eviction**
   * `Insert` infers the tx key from `FirstSignature`, encodes the tx, extracts gas from `FeeTx`, matches the tier, and optionally enforces nonce order via `expectedNextSequenceLocked`.
   * Duplicate or lower-priority entries replace existing ones; `canAccept` enforces the `MaxTx` cap by calling `evictLower`, and it rejects transactions that exceed the consensus block gas/byte limits.
   * Evicted transactions trigger `TxEventListener.OnTxRemoved`, while successful inserts trigger `OnTxInserted`.
   * `Remove`, `Contains`, `CountTx`, `Lookup`, `GetTxInfo`, and `NextExpectedSequence` all operate under the same mutex to keep the skiplist, map, and buckets consistent.

4. **Tier mechanics & listeners**
   * `selectTier` walks the configured matchers to assign the correct tier index for a transaction; `tierName` translates indexes back into configured names for distribution tracking.
   * `RegisterEventListener` allows observers to react to insertion/removal events without holding the main mutex.
   * `dispatchInserted`/`dispatchRemoved` methods notify copied listener slices so the mempool state change can ripple through instrumentation or replication layers.

5. **Background cleanup**
   * `StartCleaningWorker` launches a ticker (default interval defined by `DefaultMempoolCleaningInterval`) that replays `safeGetContext` through the `BaseApp` simulation context to inspect the latest committed account sequences.
   * `cleanUpEntries` collects stale transactions whose sequence is behind the on-chain `AccountKeeper` value and invalid transactions discovered by re-running the `AnteHandler` (per sender bucket).
   * Collected entries are removed atomically and listeners are notified, keeping the pool in sync with the application state.

## CheckTx alignment

* `CheckTxHandler` wraps the application's `CheckTx` logic to ensure the application-side mempool stays aligned with the CometBFT-side cache.
* During re-checks (`RequestCheckTx.Type == CheckTxType_Recheck`), the handler confirms the tx still exists in the mempool; a missing entry yields an error that forces CometBFT to drop the tx.
* After executing `baseApp.CheckTx`, any re-check failure also removes the tx from the mempool so CometBFT and the application never diverge.
* The handler is constructed with the logger, `BaseApp`, a concrete `Mempool`, a `txDecoder`, the `CheckTx` function, and the `AnteHandler`'s fee checker for future extensibility.

## Proposal handling

* `ProposalHandler` bundles ABCI++ `PrepareProposal` and `ProcessProposal` logic. Both handlers mirror validation to ensure every validator reaches the same block-body decision:
  * `PrepareProposal` runs on the proposer, walks the mempool iterator in priority order, and greedily packs tx bytes/gas until the block limits are reached.
  * `GetTxInfo` is used to fetch size/gas metadata, and oversized transactions (individually violating block limits) are dropped from the mempool.
  * Every transaction is re-run through the `AnteHandler` (with `CacheContext`) before inclusion; failures or panics cause removal from the mempool.
  * Logs capture the mempool distribution before/after proposal creation to aid observability.
* `ProcessProposal` runs on the non-proposing validators, duplicating the same limits and ante checks to determine whether the incoming proposal is acceptable:
  * It decodes the proposal transactions via `GetDecodedTxs`, tracks cumulative gas/bytes, rejects proposals that breach limits, and revalidates each tx with the `AnteHandler`.
  * Any mismatch (invalid tx, gas violation, size violation) results in `ResponseProcessProposal_REJECT`, keeping consensus deterministic.
  * Successful processing returns `ResponseProcessProposal_ACCEPT` after logging the totals.

## Observability & queries

* `RegisterQueryServer` and `RegisterGRPCGatewayRoutes` expose the gRPC server and gateway endpoints defined in `abcipp/types/query.proto`.
* `MempoolQueryServer` exposes two ABCI++ queries:
  * `QueryTxDistribution` returns the tier distribution map for telemetry.
  * `QueryTxHash` accepts either hex or bech32 sender strings (decoded via `DecodeAddress`) and a decimal sequence; it looks up the hash via `Lookup` and formats the response using `TxHash`.

## Helpers

* `GetDecodedTxs` decodes raw transactions with the provided `txDecoder`, returning `[]sdk.Tx` for proposal verification.
* `TxHash` computes the uppercase hex hash of raw tx bytes as expected by CometBFT.
* `DecodeAddress` accepts either `0x`-prefixed hex or bech32 addresses and returns an `sdk.AccAddress`, enabling human-friendly query paths.
* `FirstSignature` pulls the first signer/public key sequence from a sig-verifiable transaction so we can key entries by `(sender, nonce)`.
