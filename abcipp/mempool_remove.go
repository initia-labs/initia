package abcipp

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

// Remove removes the tx from the active pool or queued pool.
func (p *PriorityMempool) Remove(tx sdk.Tx) error {
	return p.RemoveWithReason(tx, RemovalReasonCommittedInBlock)
}

// RemoveWithReason removes a tx from the pool while applying reason-specific
// sender state reconciliation.
func (p *PriorityMempool) RemoveWithReason(tx sdk.Tx, reason RemovalReason) error {
	key, err := txKeyFromTx(tx)
	if err != nil {
		return err
	}

	p.mtx.Lock()
	onChainSeq, hasOnChainSeq := uint64(0), false
	if reason == RemovalReasonCommittedInBlock {
		onChainSeq = key.nonce + 1
		hasOnChainSeq = true
	}
	removed := p.removeByReasonLocked(key.sender, key.nonce, reason, onChainSeq, hasOnChainSeq)
	if len(removed) > 0 {
		p.enqueueRemovedEvents(removed)
		p.mtx.Unlock()
		return nil
	}
	p.mtx.Unlock()

	return sdkmempool.ErrTxNotFound
}

// removeEntriesByReasonLocked removes multiple active entries while applying
// reason-specific sender reconciliation where applicable.
func (p *PriorityMempool) removeEntriesByReasonLocked(entries []*txEntry, reason RemovalReason) []*txEntry {
	var removed []*txEntry
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if _, ok := p.entries[entry.key]; !ok {
			continue
		}
		removed = append(removed, p.removeByReasonLocked(entry.key.sender, entry.key.nonce, reason, 0, false)...)
	}
	return removed
}

// removeQueuedStaleLocked removes queued txs with nonce lower than on-chain sequence.
func (p *PriorityMempool) removeQueuedStaleLocked(ss *senderState, onChainSeq uint64) []*txEntry {
	var removed []*txEntry
	if len(ss.queued) == 0 || onChainSeq == 0 || ss.queuedMin >= onChainSeq {
		return removed
	}

	end := onChainSeq - 1
	if end > ss.queuedMax {
		end = ss.queuedMax
	}

	// Intentional tradeoff: iterate by nonce range (queuedMin..end) for simple
	// boundary-based state updates. This is bounded in practice by small queued
	// limits (notably maxQueuedPerSender), so sender-local cleanup remains cheap.
	for nonce := ss.queuedMin; nonce <= end; nonce++ {
		entry, ok := ss.queued[nonce]
		if !ok {
			continue
		}
		delete(ss.queued, nonce)
		p.queuedCount.Add(-1)
		ss.setQueuedRangeOnRemoveLocked(nonce)
		removed = append(removed, entry)
	}
	return removed
}

// removeNonceLocked removes a single nonce from active first, then queued.
func (p *PriorityMempool) removeNonceLocked(ss *senderState, nonce uint64) (*txEntry, bool) {
	if entry, ok := ss.active[nonce]; ok {
		p.removeEntryLocked(entry)
		return entry, true
	}
	if entry, ok := ss.queued[nonce]; ok {
		delete(ss.queued, nonce)
		p.queuedCount.Add(-1)
		ss.setQueuedRangeOnRemoveLocked(nonce)
		return entry, true
	}
	return nil, false
}

// removeActiveStaleLocked removes stale active txs with nonce lower than on-chain sequence.
func (p *PriorityMempool) removeActiveStaleLocked(ss *senderState, onChainSeq uint64) []*txEntry {
	var removed []*txEntry
	if len(ss.active) == 0 || onChainSeq == 0 || ss.activeMin >= onChainSeq {
		return removed
	}

	end := min(onChainSeq-1, ss.activeMax)

	for nonce := ss.activeMin; nonce <= end; nonce++ {
		entry, ok := ss.active[nonce]
		if !ok {
			continue
		}
		p.removeEntryLocked(entry)
		removed = append(removed, entry)
	}
	return removed
}

// removeStaleLocked removes both active and queued stale txs for a sender.
func (p *PriorityMempool) removeStaleLocked(ss *senderState, onChainSeq uint64) []*txEntry {
	removed := p.removeActiveStaleLocked(ss, onChainSeq)
	removed = append(removed, p.removeQueuedStaleLocked(ss, onChainSeq)...)
	return removed
}

// demoteActiveRangeLocked moves a suffix of active txs back to queued to restore
// sender prefix continuity after reason-based removals.
func (p *PriorityMempool) demoteActiveRangeLocked(
	ss *senderState,
	startNonce uint64,
	inclusive bool,
	onChainSeq uint64,
	hasOnChainSeq bool,
) []*txEntry {
	if len(ss.active) == 0 {
		return nil
	}

	start := startNonce
	if !inclusive {
		if startNonce == ^uint64(0) {
			return nil
		}
		start = startNonce + 1
	}
	if start > ss.activeMax {
		return nil
	}

	end := ss.activeMax

	var removed []*txEntry
	for nonce := start; nonce <= end; nonce++ {
		entry, ok := ss.active[nonce]
		if !ok {
			continue
		}
		p.removeEntryLocked(entry)
		if hasOnChainSeq && nonce < onChainSeq {
			removed = append(removed, entry)
			continue
		}

		entry.tier = queuedTier
		inserted, evicted := p.insertQueuedLocked(ss, entry.key, entry)
		if evicted != nil {
			removed = append(removed, evicted)
		}
		if !inserted {
			removed = append(removed, entry)
		}
	}
	return removed
}

// removeByReasonLocked is the central removal policy entrypoint. It applies
// reason-specific active/queued mutations and keeps sender bookkeeping consistent.
func (p *PriorityMempool) removeByReasonLocked(
	sender string,
	nonce uint64,
	reason RemovalReason,
	onChainSeq uint64,
	hasOnChainSeq bool,
) []*txEntry {
	ss := p.senders[sender]
	if ss == nil {
		return nil
	}

	var removed []*txEntry
	switch reason {
	case RemovalReasonCapacityEvicted:
		removed = append(removed, p.demoteActiveRangeLocked(ss, nonce, true, onChainSeq, hasOnChainSeq)...)
	case RemovalReasonAnteRejectedInPrepare:
		if entry, ok := p.removeNonceLocked(ss, nonce); ok {
			removed = append(removed, entry)
		}
		removed = append(removed, p.demoteActiveRangeLocked(ss, nonce, false, onChainSeq, hasOnChainSeq)...)
	default:
		if entry, ok := p.removeNonceLocked(ss, nonce); ok {
			removed = append(removed, entry)
		}
	}

	if hasOnChainSeq {
		ss.setOnChainSeqLocked(onChainSeq)
		removed = append(removed, p.removeStaleLocked(ss, onChainSeq)...)
	}

	if reason == RemovalReasonCommittedInBlock {
		return removed
	}

	p.cleanupSenderLocked(sender)
	return removed
}
