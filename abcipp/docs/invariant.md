# Priority Mempool Invariants

This document defines the behavioral invariants for `abcipp/PriorityMempool`.

## 1) Sender State Model

For each sender, mempool state is split into:

- `active`: txs currently eligible for proposal ordering.
- `queued`: future-nonce txs waiting for promotion.
- `onChainSeq`: latest known on-chain sequence for the sender.

Each sender also tracks cached nonce bounds:

- `activeMin`, `activeMax` for `active`.
- `queuedMin`, `queuedMax` for `queued`.

## 2) Next Expected Nonce

`nextExpectedNonce()` is defined as:

- `onChainSeq` when `active` is empty.
- `max(onChainSeq, activeMax + 1)` when `active` is non-empty.

This value is the sender cursor used for:

- stale checks on insert,
- contiguous queued promotion,
- user-facing `NextExpectedSequence`.

## 3) Insert Routing

Given tx nonce `n` and sender cursor `next`:

- `n < next`:
  - reject as stale unless it is same-nonce replacement of existing active/queued tx.
- `n == next`:
  - candidate for `active`.
  - may trigger queued chain promotion (`next`, `next+1`, ...).
- `n > next`:
  - route to `queued`.

## 4) Active/Queued Range Tracking

`activeMin/activeMax` and `queuedMin/queuedMax` are maintained incrementally.

- Insert updates bounds with min/max comparison.
- Remove updates bounds from boundary movement.
- No full-map recompute is required in normal paths.

## 5) Remove Semantics

`Remove(tx)` is **commit-path removal** only and is equivalent to:

- `RemoveWithReason(tx, RemovalReasonCommittedInBlock)`

Reason-specific behavior:

- `CommittedInBlock`:
  - sets sender `onChainSeq = removedNonce + 1`,
  - applies stale cleanup against updated chain sequence.
- `AnteRejectedInPrepare`:
  - removes target tx locally,
  - may demote higher active suffix as needed,
  - does not imply on-chain progression.
- `CapacityEvicted`:
  - treated as demotion flow (active -> queued) where possible.

## 6) Demotion vs Removal Events

Demotion is not removal from mempool.

- Active tx demoted to queued **must not** emit `EventTxRemoved`.
- Actual deletions (stale cleanup, replacement victim, explicit reasoned removal) emit `EventTxRemoved`.
- Newly active txs (direct insert or promotion) emit `EventTxInserted`.

## 7) Capacity Policy

Active capacity (`MaxTx`) is enforced for active index.

- If incoming/promoted tx cannot fit:
  - low-priority active suffix can be demoted to queued (not dropped) when policy allows.
  - if still not acceptable, insertion/promotion fails without silently corrupting sender nonce chain.

Queued capacity is constrained by:

- `MaxQueuedPerSender`
- `MaxQueuedTotal`

Per-sender queued overflow:

- evict highest queued nonce when inserting lower nonce.
- reject if new nonce is not better than existing policy outcome.

Global queued overflow:

- future-nonce insertion is skipped without hard failure in current policy paths.

## 8) Promotion Safety

Promotion from `queued` happens only for contiguous nonces starting at `nextExpectedNonce()`.

If promotion fails due to capacity mid-chain:

- failed nonce and remaining suffix are requeued,
- nonce continuity is preserved (no gaps, no loss).

## 9) Sender Cleanup

Sender state is removed only when fully drained:

- no active tx,
- no queued tx.

For non-commit local removals, cleanup can happen immediately if sender is empty.

## 10) Observable Guarantees

External callers can rely on:

- `Select()` returns active txs only.
- `Lookup()` can find both active and queued txs.
- `CountTx()` reflects active + queued total.
- `NextExpectedSequence()` reflects `nextExpectedNonce()` for tracked sender.

## 11) Test Contract

Contract coverage is split by concern:

- `abcipp/mempool_test.go`: end-to-end mempool behavior scenarios including insert/promotion/replacement paths.
- `abcipp/mempool_remove_test.go`: stale/removal branch behavior.
- `abcipp/mempool_cleanup_test.go`: cleanup worker and cleanup semantics.
- `abcipp/mempool_event_test.go`: event dispatcher lifecycle semantics.
- `abcipp/mempool_query_test.go`: query/iterator behavior.
- `abcipp/mempool_sender_state_test.go`: sender-state cursor/range invariants.
- `abcipp/mempool_tier_test.go`: tier matching/order/distribution rules.
- `abcipp/mempool_test_utils_test.go`: shared event and keeper test utilities.
