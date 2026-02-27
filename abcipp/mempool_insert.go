package abcipp

import (
	"context"
	"fmt"

	cmtmempool "github.com/cometbft/cometbft/mempool"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

// Insert routes the tx to the active priority index or queued pool based on nonce.
func (p *PriorityMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	priority := sdkCtx.Priority()

	key, err := txKeyFromTx(tx)
	if err != nil {
		return err
	}

	bz := sdkCtx.TxBytes()
	if len(bz) == 0 {
		var err error
		bz, err = p.txEncoder(tx)
		if err != nil {
			return err
		}
	}
	// Keep tier/capacity evaluation context consistent with the encoded tx bytes.
	sdkCtx = sdkCtx.WithTxBytes(bz)
	size := int64(len(bz))

	var gas uint64
	if feeTx, ok := tx.(sdk.FeeTx); ok {
		gas = feeTx.GetGas()
	} else {
		return fmt.Errorf("tx does not implement FeeTx")
	}

	p.mtx.Lock()
	if p.ak == nil {
		p.mtx.Unlock()
		return fmt.Errorf("account keeper is required")
	}
	ss, exists := p.senders[key.sender]
	if !exists {
		ak := p.ak
		p.mtx.Unlock()
		seq, seqOk := fetchSequence(sdkCtx, ak, key.sender)
		if !seqOk {
			return fmt.Errorf("failed to fetch account sequence for sender %s", key.sender)
		}
		p.mtx.Lock()
		// recheck in case another goroutine created sender state while unlocked.
		ss, exists = p.senders[key.sender]
		if !exists {
			ss = p.getOrCreateSenderLocked(key.sender)
			// Initialize sender cursor from on-chain sequence only.
			ss.setOnChainSeqLocked(seq)
		}
	}

	nextExpected := ss.nextExpectedNonce()
	if key.nonce < nextExpected {
		// same-nonce replacement is allowed even if sender cursor has moved forward.
		if _, ok := p.entries[key]; !ok {
			if _, ok := ss.queued[key.nonce]; !ok {
				p.mtx.Unlock()
				return fmt.Errorf("tx nonce %d is stale for sender %s (expected >= %d)", key.nonce, key.sender, nextExpected)
			}
		}
	}

	if key.nonce > nextExpected {
		entry := &txEntry{
			tx:       tx,
			priority: priority,
			size:     size,
			key:      key,
			sequence: key.nonce,
			tier:     queuedTier,
			gas:      gas,
			bytes:    bz,
		}
		inserted, evicted := p.insertQueuedLocked(ss, key, entry)
		if !inserted {
			p.mtx.Unlock()
			return nil
		}
		if evicted != nil {
			p.enqueueEvent(cmtmempool.EventTxRemoved, evicted.bytes)
		}
		p.mtx.Unlock()
		return nil
	}

	entry := &txEntry{
		tx:       tx,
		priority: priority,
		size:     size,
		key:      key,
		sequence: key.nonce,
		order:    p.nextOrder(),
		tier:     p.selectTier(sdkCtx, tx),
		gas:      gas,
		bytes:    bz,
	}

	var removed []*txEntry

	// check if entry already exists in the active pool
	existing, hasExisting := p.entries[entry.key]
	if hasExisting {
		if entry.priority <= existing.priority {
			p.mtx.Unlock()
			return nil
		}
	}
	// If a queued tx exists for the same nonce, check priority but defer
	// deletion until after canAcceptLocked confirms capacity, so the
	// queued entry is preserved when the active insert is rejected.
	var queuedToRemove *txEntry
	if queued, exists := ss.queued[key.nonce]; exists {
		if !hasExisting && entry.priority <= queued.priority {
			p.mtx.Unlock()
			return nil
		}
		queuedToRemove = queued
	}

	if ok, ev := p.canAcceptLocked(sdkCtx, entry.tier, entry.priority, entry.size, entry.gas, existing); ok {
		if queuedToRemove != nil {
			delete(ss.queued, key.nonce)
			p.queuedCount.Add(-1)
			ss.setQueuedRangeOnRemoveLocked(key.nonce)
			removed = append(removed, queuedToRemove)
		}
		if hasExisting {
			p.removeEntryLocked(existing)
			removed = append(removed, existing)
		}
		removed = append(removed, p.removeEntriesByReasonLocked(ev, RemovalReasonCapacityEvicted)...)
	} else {
		p.mtx.Unlock()
		return sdkmempool.ErrMempoolTxMaxCapacity
	}

	p.addEntryLocked(entry)

	// promote continuous queued entries from the current sender cursor.
	var promoted []*txEntry
	toPromote := p.collectPromotableLocked(ss)
	for idx, pe := range toPromote {
		pe.order = p.nextOrder()
		peCtx := sdkCtx.WithTxBytes(pe.bytes)
		pe.tier = p.selectTier(peCtx, pe.tx)
		if accepted, ev := p.canAcceptLocked(peCtx, pe.tier, pe.priority, pe.size, pe.gas, nil); accepted {
			removed = append(removed, p.removeEntriesByReasonLocked(ev, RemovalReasonCapacityEvicted)...)
			p.addEntryLocked(pe)
			promoted = append(promoted, pe)
		} else {
			// we must requeue the failed entry and all remaining entries to prevent nonce gaps
			p.requeueEntriesLocked(ss, toPromote[idx:])
			break
		}
	}

	p.enqueueRemovedEvents(removed)
	p.enqueueEvent(cmtmempool.EventTxInserted, entry.bytes)
	for _, pe := range promoted {
		p.enqueueEvent(cmtmempool.EventTxInserted, pe.bytes)
	}
	p.mtx.Unlock()

	return nil
}

// insertQueuedLocked adds or replaces a tx in the queued pool. When the per sender
// limit is hit, the entry with the highest nonce is evicted (unless the new tx has
// the highest nonce, in which case it is rejected). the caller must hold p.mtx.
func (p *PriorityMempool) insertQueuedLocked(ss *senderState, key txKey, entry *txEntry) (bool, *txEntry) {
	// same nonce replacement, only if higher priority
	if existing, exists := ss.queued[key.nonce]; exists {
		if entry.priority <= existing.priority {
			return false, nil
		}
		ss.queued[key.nonce] = entry
		return true, existing
	}

	// per sender eviction, swap highest nonce for a lower one
	var evicted *txEntry
	if p.maxQueuedPerSender > 0 && len(ss.queued) >= p.maxQueuedPerSender {
		highestNonce := uint64(0)
		for n := range ss.queued {
			if n > highestNonce {
				highestNonce = n
			}
		}
		if key.nonce >= highestNonce {
			return false, nil
		}
		evicted = ss.queued[highestNonce]
		delete(ss.queued, highestNonce)
		p.queuedCount.Add(-1)
		ss.setQueuedRangeOnRemoveLocked(highestNonce)
	} else if p.maxQueuedTotal > 0 && int(p.queuedCount.Load()) >= p.maxQueuedTotal {
		return false, nil
	}

	ss.queued[key.nonce] = entry
	p.queuedCount.Add(1)
	ss.setQueuedRangeOnInsertLocked(key.nonce)

	return true, evicted
}

// collectPromotableLocked removes queued txs with continuous nonces starting from
// nextExpectedNonce and returns them for promotion. the caller must hold p.mtx.
func (p *PriorityMempool) collectPromotableLocked(ss *senderState) []*txEntry {
	if len(ss.queued) == 0 {
		return nil
	}

	var entries []*txEntry
	next := ss.nextExpectedNonce()
	for {
		entry, exists := ss.queued[next]
		if !exists {
			break
		}
		entries = append(entries, entry)
		delete(ss.queued, next)
		p.queuedCount.Add(-1)
		ss.setQueuedRangeOnRemoveLocked(next)
		next++
	}

	return entries
}

// requeueEntriesLocked puts entries back into the sender's queued pool and
// should be used when capacity is exhausted during promotion to avoid
// losing transactions and creating nonce gaps. the caller must hold p.mtx.
func (p *PriorityMempool) requeueEntriesLocked(ss *senderState, entries []*txEntry) {
	if len(entries) == 0 {
		return
	}

	for _, entry := range entries {
		ss.queued[entry.key.nonce] = entry
		p.queuedCount.Add(1)
		ss.setQueuedRangeOnInsertLocked(entry.key.nonce)
	}
}
