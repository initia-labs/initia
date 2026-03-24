package abcipp

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultMempoolCleaningInterval is the default interval for the mempool cleaning worker.
const DefaultMempoolCleaningInterval = time.Second * 5

// StartCleaningWorker starts a background worker that periodically cleans stale txs.
func (p *PriorityMempool) StartCleaningWorker(baseApp BaseApp, interval time.Duration) {
	if interval <= 0 {
		interval = DefaultMempoolCleaningInterval
	}

	p.mtx.Lock()
	if p.cleaningStopCh != nil {
		p.mtx.Unlock()
		return
	}
	p.cleaningStopCh = make(chan struct{})
	p.cleaningDoneCh = make(chan struct{})
	stopCh := p.cleaningStopCh
	doneCh := p.cleaningDoneCh
	p.mtx.Unlock()

	go func() {
		defer close(doneCh)
		timer := time.NewTicker(interval)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				p.cleanUpEntries(baseApp, p.ak)
			case <-stopCh:
				return
			}
		}
	}()
}

// StopCleaningWorker signals the background cleaning worker to exit.
func (p *PriorityMempool) StopCleaningWorker() {
	p.mtx.Lock()
	stopCh := p.cleaningStopCh
	doneCh := p.cleaningDoneCh
	p.cleaningStopCh = nil
	p.cleaningDoneCh = nil
	p.mtx.Unlock()

	if stopCh == nil {
		return
	}

	close(stopCh)
	if doneCh != nil {
		<-doneCh
	}
}

// safeGetContext tries to get a non-panicking context from BaseApp.
func safeGetContext(bApp BaseApp) (ctx sdk.Context, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()

	// Use simulate context to avoid state mutation during cleanup.
	ctx = bApp.GetContextForSimulate(nil)
	ok = true
	return
}

// cleanUpEntries removes stale entries per sender.
func (p *PriorityMempool) cleanUpEntries(bApp BaseApp, ak AccountKeeper) {
	sdkCtx, ok := safeGetContext(bApp)
	if !ok {
		return
	}
	now := time.Now()

	p.mtx.Lock()
	senders := make([]string, 0, len(p.senders))
	for sender, state := range p.senders {
		if len(state.active) == 0 && len(state.queued) == 0 {
			delete(p.senders, sender)
			continue
		}
		senders = append(senders, sender)
	}
	p.mtx.Unlock()

	onChainSequences := make(map[string]uint64, len(senders))
	for _, sender := range senders {
		accountAddr, err := sdk.AccAddressFromBech32(sender)
		if err != nil {
			continue
		}
		accountSeq, err := ak.GetSequence(sdkCtx, accountAddr)
		if err != nil {
			continue
		}
		onChainSequences[sender] = accountSeq
	}

	if len(onChainSequences) == 0 {
		return
	}

	p.mtx.Lock()
	for sender, onChainSeq := range onChainSequences {
		ss := p.senders[sender]
		if ss == nil {
			continue
		}
		ss.setOnChainSeqLocked(onChainSeq)
		if staled := p.removeStaleLocked(ss, onChainSeq); len(staled) > 0 {
			p.enqueueRemovedEvents(staled)
		}
		if expired := p.expireQueuedGapLocked(ss, now); len(expired) > 0 {
			p.enqueueRemovedEvents(expired)
		}
		p.cleanupSenderLocked(sender)
	}
	p.mtx.Unlock()

	// Ante cleanup for sender-head txs.
	// Non-validator nodes do not run PrepareProposal, so without this recheck
	// ante-invalid txs can stay in local mempool indefinitely and cause
	// unbounded memory growth.

	if p.cfg.AnteHandler == nil {
		return
	}

	p.mtx.Lock()
	lenSenders := len(p.senders)
	headTxEntries := make([]*txEntry, 0, lenSenders)
	for _, sender := range senders {
		ss := p.senders[sender]
		if ss == nil {
			continue
		}

		headTxEntry := ss.active[ss.activeMin]
		if headTxEntry != nil {
			headTxEntries = append(headTxEntries, headTxEntry)
		}
	}
	p.mtx.Unlock()

	removed := make([]*txEntry, 0, lenSenders)
	for _, txEntry := range headTxEntries {
		cacheCtx, write := sdkCtx.WithTxBytes(txEntry.bytes).WithIsReCheckTx(true).CacheContext()
		if _, err := p.cfg.AnteHandler(cacheCtx, txEntry.tx, false); err != nil {
			removed = append(removed, txEntry)
			continue
		}
		write()
	}
	for _, entry := range removed {
		if err := p.RemoveWithReason(entry.tx, RemovalReasonAnteRejectedInPrepare); err != nil {
			p.logger.Debug(
				"failed to remove tx from app-side mempool when purging for re-check failure",
				"removal-err", err,
			)
		}
	}

}

// expireQueuedGapLocked evicts all queued txs for a sender when the sender has
// no active tx and remains blocked on a missing head nonce for longer than ttl.
func (p *PriorityMempool) expireQueuedGapLocked(ss *senderState, now time.Time) []*txEntry {
	if ss == nil {
		return nil
	}
	if len(ss.active) == 0 && len(ss.queued) > 0 && ss.queuedMin > ss.onChainSeq {
		if ss.gapSince.IsZero() {
			ss.gapSince = now
			return nil
		}
		if now.Sub(ss.gapSince) >= p.queuedGapTTL {
			ss.gapSince = time.Time{}
			return p.removeAllQueuedLocked(ss)
		}
		return nil
	}

	ss.gapSince = time.Time{}
	return nil
}
