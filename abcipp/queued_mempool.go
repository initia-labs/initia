package abcipp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	cmtmempool "github.com/cometbft/cometbft/mempool"
	cmttypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var _ Mempool = (*QueuedMempool)(nil)

const (
	// DefaultMaxQueuedPerSender is the default per-sender queued tx limit.
	DefaultMaxQueuedPerSender = 16
	// DefaultMaxQueuedTotal is the default total queued tx limit.
	DefaultMaxQueuedTotal = 1000

	// the queuedTier is the tier for txs still waiting in the queue.
	queuedTier = -1
)

// QueuedMempool wraps a PriorityMempool and adds queued-tx routing and promotion.
// Active, or next in sequence, txs are delegated to the inner txpool.
// Future nonce txs are held in a queued pool until their predecessors arrive
// or the on-chain sequence catches up.
type QueuedMempool struct {
	mtx                sync.Mutex
	txpool             *PriorityMempool
	ak                 AccountKeeper
	txEncoder          sdk.TxEncoder
	queued             map[string]map[uint64]*txEntry // tracks sender -> nonce -> entry
	activeNext         map[string]uint64              // tracks sender -> next expected active nonce
	queuedCount        atomic.Int64
	maxQueuedPerSender int
	maxQueuedTotal     int
	eventCh            chan<- cmtmempool.AppMempoolEvent
}

// queuedEventBridge forwards PriorityMempool events to QueuedMempool listeners.
type queuedEventBridge struct {
	qm *QueuedMempool
}

func (b *queuedEventBridge) OnTxInserted(tx sdk.Tx) { b.qm.forwardInserted(tx) }
func (b *queuedEventBridge) OnTxRemoved(tx sdk.Tx)  { b.qm.forwardRemoved(tx) }

// NewQueuedMempool creates a QueuedMempool wrapping the given PriorityMempool.
func NewQueuedMempool(txpool *PriorityMempool, txEncoder sdk.TxEncoder) *QueuedMempool {
	qm := &QueuedMempool{
		txpool:             txpool,
		txEncoder:          txEncoder,
		queued:             make(map[string]map[uint64]*txEntry),
		activeNext:         make(map[string]uint64),
		maxQueuedPerSender: DefaultMaxQueuedPerSender,
		maxQueuedTotal:     DefaultMaxQueuedTotal,
	}
	txpool.RegisterEventListener(&queuedEventBridge{qm: qm})

	return qm
}

// SetAccountKeeper sets the account keeper used for querying on-chain sequences.
func (qm *QueuedMempool) SetAccountKeeper(ak AccountKeeper) {
	qm.mtx.Lock()
	defer qm.mtx.Unlock()

	qm.ak = ak
}

// SetMaxQueuedPerSender overrides the default per-sender queued tx limit.
func (qm *QueuedMempool) SetMaxQueuedPerSender(n int) {
	qm.mtx.Lock()
	defer qm.mtx.Unlock()

	qm.maxQueuedPerSender = n
}

// SetMaxQueuedTotal overrides the default total queued tx limit.
func (qm *QueuedMempool) SetMaxQueuedTotal(n int) {
	qm.mtx.Lock()
	defer qm.mtx.Unlock()

	qm.maxQueuedTotal = n
}

// StartCleaningWorker delegates to the inner txpool.
func (qm *QueuedMempool) StartCleaningWorker(baseApp BaseApp, ak AccountKeeper, interval time.Duration) {
	qm.txpool.StartCleaningWorker(baseApp, ak, interval)
}

// StopCleaningWorker delegates to the inner txpool.
func (qm *QueuedMempool) StopCleaningWorker() {
	qm.txpool.StopCleaningWorker()
}

// Insert routes the tx to the active txpool or queued pool based on nonce.
func (qm *QueuedMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	key, err := txKeyFromTx(tx)
	if err != nil {
		return err
	}

	priority := sdkCtx.Priority()

	// get or initialize activeNext for this sender.
	// if unseen, release the lock for the (potentially) slower GetSequence call, then recheck under lock.
	qm.mtx.Lock()
	activeNext, hasActiveNext := qm.activeNext[key.sender]
	if !hasActiveNext && qm.ak != nil {
		qm.mtx.Unlock()
		seq, seqOk := qm.fetchSequence(sdkCtx, key.sender)
		qm.mtx.Lock()

		if current, exists := qm.activeNext[key.sender]; exists {
			activeNext = current
			hasActiveNext = true
		} else if seqOk {
			activeNext = seq
			qm.activeNext[key.sender] = seq
			hasActiveNext = true
		}
	}

	switch {
	case hasActiveNext && key.nonce < activeNext:
		qm.mtx.Unlock()
		return fmt.Errorf("tx nonce %d is stale for sender %s (expected >= %d)", key.nonce, key.sender, activeNext)

	case hasActiveNext && key.nonce > activeNext:
		bz, err := qm.txEncoder(tx)
		if err != nil {
			qm.mtx.Unlock()
			return err
		}
		var gas uint64
		if feeTx, ok := tx.(sdk.FeeTx); ok {
			gas = feeTx.GetGas()
		}
		entry := &txEntry{
			tx:       tx,
			priority: priority,
			size:     int64(len(bz)),
			key:      key,
			sequence: key.nonce,
			tier:     queuedTier,
			gas:      gas,
			bytes:    bz,
		}
		inserted, evicted := qm.insertQueuedLocked(key, entry)
		if !inserted {
			qm.mtx.Unlock()
			return nil
		}
		qm.mtx.Unlock()

		if evicted != nil {
			qm.pushEvent(cmtmempool.EventTxRemoved, evicted.bytes)
		}
		return nil

	default:
		qm.mtx.Unlock()

		if err := qm.txpool.Insert(ctx, tx); err != nil {
			return err
		}

		qm.mtx.Lock()
		var toPromote []*txEntry
		if hasActiveNext {
			qm.activeNext[key.sender] = max(qm.activeNext[key.sender], activeNext+1)
			toPromote = qm.collectPromotableLocked(key.sender)
		}
		qm.mtx.Unlock()

		for _, entry := range toPromote {
			promoteCtx := sdkCtx.WithPriority(entry.priority)
			_ = qm.txpool.Insert(promoteCtx, entry.tx)
		}
		return nil
	}
}

// fetchSequence queries the on-chain sequence for a sender.
func (qm *QueuedMempool) fetchSequence(ctx sdk.Context, sender string) (uint64, bool) {
	addr, err := sdk.AccAddressFromBech32(sender)
	if err != nil {
		return 0, false
	}

	seq, err := qm.ak.GetSequence(ctx, addr)
	if err != nil {
		return 0, false
	}

	return seq, true
}

// insertQueuedLocked adds or replaces a tx in the queued pool, along with enforcing per-sender
// and global limits. returns (true, evicted) on success. when the per-sender limit is hit,
// an entry with the highest nonce is evicted to make room (unless the new tx has
// the highest nonce, in which case it is rejected). the caller must hold qm.mtx.
func (qm *QueuedMempool) insertQueuedLocked(key txKey, entry *txEntry) (bool, *txEntry) {
	senderQ, ok := qm.queued[key.sender]
	if !ok {
		senderQ = make(map[uint64]*txEntry)
		qm.queued[key.sender] = senderQ
	}

	// same nonce replacement, only if higher priority
	if existing, exists := senderQ[key.nonce]; exists {
		if entry.priority <= existing.priority {
			return false, nil
		}
		senderQ[key.nonce] = entry
		return true, nil
	}

	// per sender eviction, try to swap the highest nonce for a lower one
	var evicted *txEntry
	if qm.maxQueuedPerSender > 0 && len(senderQ) >= qm.maxQueuedPerSender {
		highestNonce := uint64(0)
		for n := range senderQ {
			if n > highestNonce {
				highestNonce = n
			}
		}
		if key.nonce >= highestNonce {
			return false, nil
		}
		evicted = senderQ[highestNonce]
		delete(senderQ, highestNonce)
		qm.queuedCount.Add(-1)
	} else if qm.maxQueuedTotal > 0 && int(qm.queuedCount.Load()) >= qm.maxQueuedTotal {
		return false, nil
	}

	senderQ[key.nonce] = entry
	qm.queuedCount.Add(1)

	return true, evicted
}

// collectPromotableLocked removes queued txs with continuous nonces starting from
// activeNext and returns them for promotion into the txpool. the caller must hold qm.mtx.
func (qm *QueuedMempool) collectPromotableLocked(sender string) []*txEntry {
	senderQ := qm.queued[sender]
	if senderQ == nil {
		return nil
	}

	var entries []*txEntry
	next := qm.activeNext[sender]
	for {
		entry, exists := senderQ[next]
		if !exists {
			break
		}
		entries = append(entries, entry)
		delete(senderQ, next)
		qm.queuedCount.Add(-1)
		next++
	}

	if len(entries) > 0 {
		qm.activeNext[sender] = next
	}
	qm.cleanupSenderQueueLocked(sender)

	return entries
}

// cleanupSenderQueueLocked removes the sender's queue map if empty. the caller must hold qm.mtx.
func (qm *QueuedMempool) cleanupSenderQueueLocked(sender string) {
	if senderQ := qm.queued[sender]; senderQ != nil && len(senderQ) == 0 {
		delete(qm.queued, sender)
	}
}

// Remove tries the txpool first, then the queued pool.
func (qm *QueuedMempool) Remove(tx sdk.Tx) error {
	if err := qm.txpool.Remove(tx); err == nil {
		return nil
	}

	key, err := txKeyFromTx(tx)
	if err != nil {
		return err
	}

	qm.mtx.Lock()
	senderQ := qm.queued[key.sender]
	if senderQ == nil {
		qm.mtx.Unlock()
		return sdkmempool.ErrTxNotFound
	}
	entry, exists := senderQ[key.nonce]
	if !exists {
		qm.mtx.Unlock()
		return sdkmempool.ErrTxNotFound
	}

	delete(senderQ, key.nonce)
	qm.queuedCount.Add(-1)
	qm.cleanupSenderQueueLocked(key.sender)
	removedBytes := entry.bytes
	qm.mtx.Unlock()

	qm.pushEvent(cmtmempool.EventTxRemoved, removedBytes)

	return nil
}

// Select delegates to the inner txpool.
func (qm *QueuedMempool) Select(ctx context.Context, keys [][]byte) sdkmempool.Iterator {
	return qm.txpool.Select(ctx, keys)
}

// CountTx returns active + queued count.
func (qm *QueuedMempool) CountTx() int {
	return qm.txpool.CountTx() + int(qm.queuedCount.Load())
}

// Contains checks both txpool and queued pool.
func (qm *QueuedMempool) Contains(tx sdk.Tx) bool {
	if qm.txpool.Contains(tx) {
		return true
	}

	key, err := txKeyFromTx(tx)
	if err != nil {
		return false
	}

	qm.mtx.Lock()
	defer qm.mtx.Unlock()

	if senderQ := qm.queued[key.sender]; senderQ != nil {
		_, exists := senderQ[key.nonce]
		return exists
	}

	return false
}

// Lookup checks txpool first, then queued pool.
func (qm *QueuedMempool) Lookup(sender string, nonce uint64) (string, bool) {
	if hash, ok := qm.txpool.Lookup(sender, nonce); ok {
		return hash, true
	}

	qm.mtx.Lock()
	defer qm.mtx.Unlock()

	if senderQ := qm.queued[sender]; senderQ != nil {
		if entry, exists := senderQ[nonce]; exists {
			return TxHash(entry.bytes), true
		}
	}

	return "", false
}

// GetTxInfo checks txpool first, then queued pool.
func (qm *QueuedMempool) GetTxInfo(ctx sdk.Context, tx sdk.Tx) (TxInfo, error) {
	if info, err := qm.txpool.GetTxInfo(ctx, tx); err == nil {
		return info, nil
	}

	key, err := txKeyFromTx(tx)
	if err != nil {
		return TxInfo{}, err
	}

	qm.mtx.Lock()
	defer qm.mtx.Unlock()

	if senderQ := qm.queued[key.sender]; senderQ != nil {
		if entry, exists := senderQ[key.nonce]; exists {
			return TxInfo{
				Size:     entry.size,
				GasLimit: entry.gas,
				Sender:   entry.key.sender,
				Sequence: entry.sequence,
				TxBytes:  entry.bytes,
				Tier:     "queued",
			}, nil
		}
	}

	return TxInfo{}, sdkmempool.ErrTxNotFound
}

// GetTxDistribution returns the tier distribution including queued txs.
func (qm *QueuedMempool) GetTxDistribution() map[string]uint64 {
	out := qm.txpool.GetTxDistribution()
	if n := qm.queuedCount.Load(); n > 0 {
		out["queued"] = uint64(n)
	}

	return out
}

// NextExpectedSequence returns the activeNext for a sender.
func (qm *QueuedMempool) NextExpectedSequence(ctx sdk.Context, sender string) (uint64, bool, error) {
	qm.mtx.Lock()
	next, ok := qm.activeNext[sender]
	qm.mtx.Unlock()
	return next, ok, nil
}

// PromoteQueued evicts stale entries, promotes sequential queued entries,
// and refreshes activeNext for all tracked senders.
// Called after each block commit via the app PrepareCheckStater.
func (qm *QueuedMempool) PromoteQueued(ctx context.Context) {
	if qm.ak == nil {
		return
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// snapshot senders, partitioned by whether they have queued entries.
	// senders with queued entries need an on-chain sequence fetch, for stale eviction.
	// active-only senders just need a pool existence check.
	qm.mtx.Lock()
	var queuedSenders, activeOnlySenders []string
	for sender := range qm.activeNext {
		if _, hasQueued := qm.queued[sender]; hasQueued {
			queuedSenders = append(queuedSenders, sender)
		} else {
			activeOnlySenders = append(activeOnlySenders, sender)
		}
	}
	qm.mtx.Unlock()

	if len(queuedSenders) == 0 && len(activeOnlySenders) == 0 {
		return
	}

	// queued senders, fetch on-chain sequence + pool membership outside the lock
	type senderState struct {
		onChainSeq     uint64
		hasPoolEntries bool
	}
	states := make(map[string]*senderState, len(queuedSenders))
	for _, sender := range queuedSenders {
		seq, ok := qm.fetchSequence(sdkCtx, sender)
		if !ok {
			continue
		}
		states[sender] = &senderState{
			onChainSeq:     seq,
			hasPoolEntries: qm.txpool.HasSenderEntries(sender),
		}
	}

	// active only senders, just check pool membership
	activeOnlyHasPool := make(map[string]bool, len(activeOnlySenders))
	for _, sender := range activeOnlySenders {
		activeOnlyHasPool[sender] = qm.txpool.HasSenderEntries(sender)
	}

	qm.mtx.Lock()
	var staleEntries, promoteEntries []*txEntry
	for sender, state := range states {
		currentActive := qm.activeNext[sender]
		newActive := max(state.onChainSeq, currentActive)

		// evict stale queued entries
		if senderQ := qm.queued[sender]; senderQ != nil {
			for nonce, entry := range senderQ {
				if nonce < state.onChainSeq {
					staleEntries = append(staleEntries, entry)
					delete(senderQ, nonce)
					qm.queuedCount.Add(-1)
				}
			}
		}

		// advance activeNext past on-chain seq, then collect promotable entries
		qm.activeNext[sender] = newActive
		promoteEntries = append(promoteEntries, qm.collectPromotableLocked(sender)...)

		hasQueued := qm.queued[sender] != nil && len(qm.queued[sender]) > 0
		if !hasQueued && (state.onChainSeq >= qm.activeNext[sender] || !state.hasPoolEntries) {
			delete(qm.activeNext, sender)
		}
	}
	for sender, hasPool := range activeOnlyHasPool {
		if !hasPool {
			delete(qm.activeNext, sender)
		}
	}
	qm.mtx.Unlock()

	for _, entry := range staleEntries {
		qm.pushEvent(cmtmempool.EventTxRemoved, entry.bytes)
	}

	for _, entry := range promoteEntries {
		_ = qm.txpool.Insert(sdkCtx.WithPriority(entry.priority), entry.tx)
	}
}

// SetEventCh stores the cometbft event channel for event dispatch.
func (qm *QueuedMempool) SetEventCh(ch chan<- cmtmempool.AppMempoolEvent) {
	qm.mtx.Lock()
	defer qm.mtx.Unlock()

	qm.eventCh = ch
}

// pushEvent sends an event to the cometbft event channel if wired.
func (qm *QueuedMempool) pushEvent(eventType cmtmempool.AppMempoolEventType, txBytes []byte) {
	qm.mtx.Lock()
	ch := qm.eventCh
	qm.mtx.Unlock()

	if ch == nil {
		return
	}

	cmtTx := cmttypes.Tx(txBytes)
	select {
	case ch <- cmtmempool.AppMempoolEvent{
		Type:  eventType,
		TxKey: cmtTx.Key(),
		Tx:    cmtTx,
	}:
	default:
	}
}

// forwardInserted is called by the bridge when PriorityMempool inserts a tx.
func (qm *QueuedMempool) forwardInserted(tx sdk.Tx) {
	if txBytes, err := qm.txEncoder(tx); err == nil {
		qm.pushEvent(cmtmempool.EventTxInserted, txBytes)
	}
}

// forwardRemoved is called by the bridge when PriorityMempool removes a tx.
func (qm *QueuedMempool) forwardRemoved(tx sdk.Tx) {
	if txBytes, err := qm.txEncoder(tx); err == nil {
		qm.pushEvent(cmtmempool.EventTxRemoved, txBytes)
	}
}
