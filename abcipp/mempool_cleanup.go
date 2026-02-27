package abcipp

import (
	"sort"
	"time"

	cmtmempool "github.com/cometbft/cometbft/mempool"

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

// cleanUpEntries removes stale and ante-invalid active entries per sender.
func (p *PriorityMempool) cleanUpEntries(bApp BaseApp, ak AccountKeeper) {
	sdkCtx, ok := safeGetContext(bApp)
	if !ok {
		return
	}

	type senderSnapshot struct {
		sender  string
		entries []*txEntry
	}

	p.mtx.RLock()
	snapshots := make([]senderSnapshot, 0, len(p.senders))
	for sender, state := range p.senders {
		if len(state.active) == 0 {
			continue
		}
		entries := make([]*txEntry, 0, len(state.active))
		for _, entry := range state.active {
			entries = append(entries, entry)
		}
		snapshots = append(snapshots, senderSnapshot{sender: sender, entries: entries})
	}
	p.mtx.RUnlock()

	for idx := range snapshots {
		sort.Slice(snapshots[idx].entries, func(a, b int) bool {
			return snapshots[idx].entries[a].sequence < snapshots[idx].entries[b].sequence
		})
	}

	var removed []*txEntry
	for _, snap := range snapshots {
		accountAddr, err := sdk.AccAddressFromBech32(snap.sender)
		if err != nil {
			continue
		}
		accountSeq, err := ak.GetSequence(sdkCtx, accountAddr)
		if err != nil {
			continue
		}

		// Remove entries that are now below on-chain sequence.
		validStart := 0
		for i, entry := range snap.entries {
			if entry.sequence < accountSeq {
				removed = append(removed, entry)
			} else {
				validStart = i
				break
			}
			validStart = i + 1
		}

		// Recheck remaining entries in nonce order and drop suffix after first failure.
		if p.cfg.AnteHandler != nil {
			failed := false
			for _, entry := range snap.entries[validStart:] {
				if failed {
					removed = append(removed, entry)
					continue
				}
				cacheCtx, write := sdkCtx.WithTxBytes(entry.bytes).WithIsReCheckTx(true).CacheContext()
				if _, err := p.cfg.AnteHandler(cacheCtx, entry.tx, false); err != nil {
					failed = true
					removed = append(removed, entry)
					continue
				}
				write()
			}
		}
	}

	if len(removed) == 0 {
		return
	}

	p.mtx.Lock()
	for _, entry := range removed {
		if existing, ok := p.entries[entry.key]; ok {
			p.removeEntryLocked(existing)
			p.enqueueEvent(cmtmempool.EventTxRemoved, existing.bytes)
		}
	}
	p.mtx.Unlock()
}
