# ABCI++ Package

The `abcipp` package wires up the ABCI++ surfaces that Initia needs: a priority-aware mempool, the proposal/CheckTx handlers that keep the pool in sync with CometBFT, and the ABCI++ gRPC queries that surface mempool state to clients.

## Contracts

### Mempool interface extensions

* `Mempool` builds on `sdkmempool.Mempool` by exposing callers to mempool-specific metadata:
  * `Contains` and `Lookup` support existence and sender/nonce lookups without exposing internals.
  * `GetTxDistribution` returns a tier name→count map so telemetry can track whether low- and high-priority lanes are flowing.
  * `GetTxInfo` reports the `TxInfo` struct (sender, sequence, size, gas limit, bytes, tier) used during proposal creation.
* `AccountKeeper` and `BaseApp` provide the minimal hooks the mempool needs for cleanup (`GetSequence`) and for simulating transactions during ante checks (`GetContextForSimulate`).

## Priority mempool architecture

1. **Configuration**
   * `PriorityMempoolConfig` governs the upper active transaction limit (`MaxTx`), queued limits (`MaxQueuedPerSender`, `MaxQueuedTotal`), the ordered `Tiers` to prefer, and the `AnteHandler` used to revalidate cached transactions.
   * `Tier`/`TierMatcher` pairs are canonicalized by `buildTierMatchers`, which trims empty names, drops nil matchers, and always produces a fallback `default` tier.

2. **Data model**
   * Active entries are stored in a skiplist rooted at `priorityIndex` (ordered by tier, priority, insertion order, sender, nonce) and a global map keyed by `(sender, nonce)` for quick O(1) lookups.
   * Per sender state is unified in `senderState` structs (held in `senders map[string]*senderState`), with each containing:
     * `active` entries (same pointers as the global map),
     * `queued` future-nonce entries,
     * cached `onChainSeq`,
     * cached nonce bounds (`activeMin/activeMax`, `queuedMin/queuedMax`).
   * Sender cursor is derived by `nextExpectedNonce()`:
     * `onChainSeq` when `active` is empty,
     * `max(onChainSeq, activeMax+1)` when `active` is non-empty.
   * Each `txEntry` bundles the transaction, priority, sequence, tier index, gas, encoded bytes, and an insertion order to break ties.
   * Tier counts are tracked in `tierDistribution`, keeping per-tier occupancies up to date. `GetTxDistribution` appends a `"queued"` entry when queued txs exist.
   * Implementation is split by responsibility:
     * `mempool_sender_state.go`: sender cursor and nonce-range helpers.
     * `mempool_tier.go`: tier matching and active ordering comparator.
     * `mempool_insert.go`: insert routing and promotion logic.
     * `mempool_remove.go`: reason-based remove/demotion/stale logic.
     * `mempool_query.go`: query and iteration APIs.
     * `mempool_event.go`: async CometBFT event dispatch.
     * `mempool_cleanup.go`: background stale/ante cleanup worker.
     * `mempool_invariant.go`: runtime invariant assertions.

3. **Insert routing**
   * `Insert` infers the tx key from `FirstSignature`, encodes the tx, extracts gas from `FeeTx`, and routes based on nonce vs `nextExpectedNonce()`:
     * `nonce < nextExpected` -> rejected as stale unless it is same-nonce replacement of existing active/queued tx.
     * `nonce > nextExpected` -> added to the queued pool. When the per-sender limit is hit, the entry with the highest nonce is evicted to prefer lower (closer to promotable) nonces. Same nonce replacement is allowed but requires strictly higher priority.
     * `nonce == nextExpected` -> inserted into the priority index. Continuous queued nonce chain is promoted in the same call when capacity permits.
   * For active entries: if a duplicate `(sender, nonce)` already exists, a higher-priority replacement evicts the old entry; lower-priority replacements are ignored.
   * `canAcceptLocked` enforces the `MaxTx` cap by computing an eviction set from the active index and also rejects transactions that exceed consensus block gas/byte limits.

4. **Promotion**
   * `PromoteQueued` runs after each block commit via `PrepareCheckStater`. It partitions tracked senders into those with queued entries (requiring an on-chain sequence fetch) and active only senders (cheap in-memory check).
   * For queued senders: cached `onChainSeq` is refreshed from `AccountKeeper.GetSequence`, stale entries (`nonce < onChainSeq`) are removed, then continuous entries starting from `nextExpectedNonce()` are collected and promoted.
   * If promotion fails mid-chain due to capacity, the failed nonce and suffix are requeued to preserve nonce continuity.
   * For active-only senders: if the sender has no remaining pool entries, sender state is cleaned up.

5. **Removal semantics**
   * `Remove(tx)` is commit-path removal and delegates to `RemoveWithReason(tx, RemovalReasonCommittedInBlock)`.
   * `RemoveWithReason` applies reason-specific policy:
     * `CommittedInBlock`: sets sender `onChainSeq = removedNonce + 1`, then removes stale entries.
     * `AnteRejectedInPrepare`: removes target entry locally and can demote higher active suffix when needed.
     * `CapacityEvicted`: uses demotion flow for active suffix where possible.
   * Sender cleanup runs for non-commit paths when both active and queued sets become empty.

6. **Event dispatch**
   * Events are enqueued into an internal FIFO (`eventQueue`) and delivered asynchronously by `eventDispatchLoop` to CometBFT `AppMempoolEvent` channel.
   * Active insertions fire `EventTxInserted`.
   * Actual deletions (active/queued removal, stale eviction, replacement victims) fire `EventTxRemoved`.
   * Capacity demotion from active to queued is **not** a deletion and does **not** fire `EventTxRemoved`.
   * Queued insertions do not fire events since CometBFT handles `EventTxQueued` from CheckTx.
   * `SetEventCh` wires the CometBFT `AppMempoolEvent` channel so the proxy mempool in CometBFT reacts to app-side state changes.

7. **Tier mechanics**
   * `selectTier` walks the configured matchers to assign the correct tier index for a transaction; `tierName` translates indexes back into configured names for distribution tracking.

8. **Background cleanup**
   * `StartCleaningWorker` launches a ticker (default interval defined by `DefaultMempoolCleaningInterval`) that replays `safeGetContext` through the `BaseApp` simulation context to inspect the latest committed account sequences.
   * `cleanUpEntries` groups active entries by sender on the fly, sorts each group by nonce, then collects stale transactions (sequence behind the on-chain `AccountKeeper` value) and invalid transactions discovered by re-running the `AnteHandler` sequentially per sender.
   * Collected entries are removed atomically, and events are dispatched, keeping the pool in sync with the app state.

9. **Query methods**
   * `Contains`, `Lookup`, `GetTxInfo`, `Remove`, and `RemoveWithReason` check active pool first, then queued pool.
   * `CountTx` sums the active pool count and `queuedCount`. `Select` returns only active entries.
   * `IteratePendingTxs` and `IterateQueuedTxs` expose deterministic sender/nonce iteration over active/queued sets.

## CheckTx alignment

* `CheckTxHandler` wraps the application's `CheckTx` logic to ensure the application-side mempool stays aligned with the CometBFT-side cache.
* During re-checks (`RequestCheckTx.Type == CheckTxType_Recheck`), the handler confirms the tx still exists in the mempool; a missing entry yields an error that forces CometBFT to drop the tx.
* After executing `baseApp.CheckTx`, any re-check failure also removes the tx from the mempool so CometBFT and the application never diverge.
* The handler is constructed with the logger, `BaseApp`, a concrete `Mempool`, a `txDecoder`, the `CheckTx` function, and the fee checker (currently unused in the handler but kept for parity with the app wiring).

## Proposal handling

* `ProposalHandler` bundles ABCI++ `PrepareProposal` and `ProcessProposal` logic. Both handlers mirror validation to ensure every validator reaches the same block-body decision:
  * `PrepareProposal` runs on the proposer, walks the mempool iterator in priority order, and greedily packs tx bytes/gas until the block limits are reached.
  * `GetTxInfo` is used to fetch size/gas metadata. If a tx individually exceeds block max bytes/gas it is removed from the mempool; if it only exceeds remaining capacity, it is skipped for the proposal.
  * Every transaction is re-run through the `AnteHandler` (with `CacheContext`) before inclusion; failures cause removal from the mempool.
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
