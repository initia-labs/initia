package abcipp

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cmtmempool "github.com/cometbft/cometbft/mempool"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/huandu/skiplist"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var _ Mempool = (*PriorityMempool)(nil)

// PriorityMempoolConfig configures the limits enforced by the priority mempool.
// A MaxTx value less than or equal to zero is treated as "unlimited."
type PriorityMempoolConfig struct {
	MaxTx              int // total active transaction limit
	MaxQueuedPerSender int // per sender queued tx limit (0 = default)
	MaxQueuedTotal     int // total queued tx limit (0 = default)
	Tiers              []Tier

	// AnteHandler to filter out invalid transactions from the mempool
	AnteHandler sdk.AnteHandler
}

type TierMatcher func(ctx sdk.Context, tx sdk.Tx) bool

type Tier struct {
	Name    string
	Matcher TierMatcher
}

type txKey struct {
	sender string
	nonce  uint64
}

const (
	// DefaultMaxQueuedPerSender is the default per-sender queued tx limit.
	DefaultMaxQueuedPerSender = 16
	// DefaultMaxQueuedTotal is the default total queued tx limit.
	DefaultMaxQueuedTotal = 1000

	// queuedTier marks entries in the queued pool.
	queuedTier = -1
)

// senderState tracks all sender mempool state, active entries in the priority
// index, queued future nonce entries, and the next expected insertion nonce.
type senderState struct {
	activeNext    uint64
	hasActiveNext bool
	active        map[uint64]*txEntry
	queued        map[uint64]*txEntry
}

func (s *senderState) isEmpty() bool {
	return len(s.active) == 0 && len(s.queued) == 0
}

// PriorityMempool is a transaction pool that keeps high-priority submissions
// flowing with low latency while still making progress on lower-priority ones.
// It supports queued tx routing with active (next in sequence) txs are inserted into
// the priority index for consensus ordering, while future nonce txs are held in
// a queued pool until their predecessors arrive or the on-chain sequence catches up.
type PriorityMempool struct {
	mtx              sync.RWMutex
	cfg              PriorityMempoolConfig
	txEncoder        sdk.TxEncoder
	priorityIndex    *skiplist.SkipList
	entries          map[txKey]*txEntry
	senders          map[string]*senderState
	orderSeq         int64
	tiers            []tierMatcher
	tierDistribution map[string]uint64
	cleaningStopCh   chan struct{}
	cleaningDoneCh   chan struct{}

	ak                 AccountKeeper
	queuedCount        atomic.Int64
	maxQueuedPerSender int
	maxQueuedTotal     int
	eventCh            atomic.Pointer[chan<- cmtmempool.AppMempoolEvent]
}

// NewPriorityMempool creates a new PriorityMempool with the provided limits.
func NewPriorityMempool(cfg PriorityMempoolConfig, txEncoder sdk.TxEncoder) *PriorityMempool {
	if txEncoder == nil {
		panic("tx encoder is required")
	}
	tiers := buildTierMatchers(cfg)
	dist := initTierDistribution(tiers)

	maxQPS := cfg.MaxQueuedPerSender
	if maxQPS <= 0 {
		maxQPS = DefaultMaxQueuedPerSender
	}
	maxQT := cfg.MaxQueuedTotal
	if maxQT <= 0 {
		maxQT = DefaultMaxQueuedTotal
	}

	return &PriorityMempool{
		cfg:                cfg,
		priorityIndex:      skiplist.New(skiplist.GreaterThanFunc(compareEntries)),
		entries:            make(map[txKey]*txEntry),
		senders:            make(map[string]*senderState),
		txEncoder:          txEncoder,
		tiers:              tiers,
		tierDistribution:   dist,
		maxQueuedPerSender: maxQPS,
		maxQueuedTotal:     maxQT,
	}
}

// SetAccountKeeper sets the account keeper used for querying on-chain sequences.
func (p *PriorityMempool) SetAccountKeeper(ak AccountKeeper) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.ak = ak
}

// SetEventCh stores the cometbft event channel for event dispatch.
func (p *PriorityMempool) SetEventCh(ch chan<- cmtmempool.AppMempoolEvent) {
	p.eventCh.Store(&ch)
}

// SetMaxQueuedPerSender overrides the default per-sender queued tx limit.
func (p *PriorityMempool) SetMaxQueuedPerSender(n int) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.maxQueuedPerSender = n
}

// SetMaxQueuedTotal overrides the default total queued tx limit.
func (p *PriorityMempool) SetMaxQueuedTotal(n int) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.maxQueuedTotal = n
}

// getOrCreateSenderLocked returns the senderState for the given sender, creating one if needed.
// the caller must hold p.mtx.
func (p *PriorityMempool) getOrCreateSenderLocked(sender string) *senderState {
	s, ok := p.senders[sender]
	if !ok {
		s = &senderState{
			active: make(map[uint64]*txEntry),
			queued: make(map[uint64]*txEntry),
		}
		p.senders[sender] = s
	}

	return s
}

// DefaultMempoolCleaningInterval is the default interval for the mempool cleaning worker.
const DefaultMempoolCleaningInterval = time.Second * 5

// StartCleaningWorker starts a background worker that periodically cleans up
// stale transactions.
func (p *PriorityMempool) StartCleaningWorker(baseApp BaseApp, ak AccountKeeper, interval time.Duration) {
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
				p.cleanUpEntries(baseApp, ak)
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

// safeGetContext tries to get a non-panicking context from the BaseApp.
func safeGetContext(bApp BaseApp) (ctx sdk.Context, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()

	// use simulate context to avoid state mutation during cleanup
	ctx = bApp.GetContextForSimulate(nil)
	ok = true

	return
}

// cleanUpEntries removes transactions from users whose on-chain sequence
// has advanced beyond the sequences of their pending transactions.
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

		// collect stale entries below on-chain sequence
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

		// collect invalid entries by running ante handler in nonce order
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
	var finalRemoved []*txEntry
	for _, entry := range removed {
		if existing, ok := p.entries[entry.key]; ok {
			p.removeEntryLocked(existing)
			finalRemoved = append(finalRemoved, existing)
		}
	}
	p.mtx.Unlock()

	for _, entry := range finalRemoved {
		p.pushEvent(cmtmempool.EventTxRemoved, entry.bytes)
	}
}

// Contains returns true if the transaction is in the active or queued pool.
func (p *PriorityMempool) Contains(tx sdk.Tx) bool {
	key, err := txKeyFromTx(tx)
	if err != nil {
		return false
	}

	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if _, ok := p.entries[key]; ok {
		return true
	}
	if s := p.senders[key.sender]; s != nil {
		_, exists := s.queued[key.nonce]
		return exists
	}

	return false
}

// CountTx returns the total number of active and queued transactions.
func (p *PriorityMempool) CountTx() int {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	return len(p.entries) + int(p.queuedCount.Load())
}

// GetTxDistribution returns the number of transactions per configured tier.
func (p *PriorityMempool) GetTxDistribution() map[string]uint64 {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	out := make(map[string]uint64, len(p.tierDistribution)+1)
	maps.Copy(out, p.tierDistribution)
	if n := p.queuedCount.Load(); n > 0 {
		out["queued"] = uint64(n)
	}

	return out
}

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
	size := int64(len(bz))

	var gas uint64
	if feeTx, ok := tx.(sdk.FeeTx); ok {
		gas = feeTx.GetGas()
	} else {
		return fmt.Errorf("tx does not implement FeeTx")
	}

	p.mtx.Lock()
	ss := p.getOrCreateSenderLocked(key.sender)
	if !ss.hasActiveNext && p.ak != nil {
		p.mtx.Unlock()
		seq, seqOk := p.fetchSequence(sdkCtx, key.sender)
		p.mtx.Lock()

		// refetch sender state in case it was removed while we were unlocked
		ss = p.getOrCreateSenderLocked(key.sender)

		if !ss.hasActiveNext && seqOk {
			ss.activeNext = seq
			ss.hasActiveNext = true
		}
	}

	switch {
	case ss.hasActiveNext && key.nonce < ss.activeNext:
		p.mtx.Unlock()
		return fmt.Errorf("tx nonce %d is stale for sender %s (expected >= %d)", key.nonce, key.sender, ss.activeNext)

	case ss.hasActiveNext && key.nonce > ss.activeNext:
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
		p.mtx.Unlock()

		if evicted != nil {
			p.pushEvent(cmtmempool.EventTxRemoved, evicted.bytes)
		}
		return nil

	default:
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

		// check if entry already exists in active pool
		existing, hasExisting := p.entries[entry.key]
		if hasExisting {
			if entry.priority < existing.priority {
				p.mtx.Unlock()
				return nil
			}
		}

		if ok, ev := p.canAcceptLocked(sdkCtx, entry.tier, entry.priority, entry.size, entry.gas, existing); ok {
			if hasExisting {
				p.removeEntryLocked(existing)
				removed = append(removed, existing)
			}
			p.removeEntriesLocked(ev...)
			removed = append(removed, ev...)
		} else {
			p.mtx.Unlock()
			p.pushRemovedEvents(removed)
			return sdkmempool.ErrMempoolTxMaxCapacity
		}

		p.addEntryLocked(entry)

		// advance activeNext and promote continuous queued entries
		var promoted []*txEntry
		if ss.hasActiveNext {
			ss.activeNext = max(ss.activeNext, key.nonce+1)
			toPromote := p.collectPromotableLocked(ss)
			for _, pe := range toPromote {
				pe.order = p.nextOrder()
				peCtx := sdkCtx.WithTxBytes(pe.bytes)
				pe.tier = p.selectTier(peCtx, pe.tx)
				if accepted, ev := p.canAcceptLocked(peCtx, pe.tier, pe.priority, pe.size, pe.gas, nil); accepted {
					p.removeEntriesLocked(ev...)
					removed = append(removed, ev...)
					p.addEntryLocked(pe)
					promoted = append(promoted, pe)
				}
			}
		}

		p.mtx.Unlock()

		p.pushRemovedEvents(removed)
		p.pushEvent(cmtmempool.EventTxInserted, entry.bytes)
		for _, pe := range promoted {
			p.pushEvent(cmtmempool.EventTxInserted, pe.bytes)
		}

		return nil
	}
}

// Remove removes the tx from the active pool or queued pool.
func (p *PriorityMempool) Remove(tx sdk.Tx) error {
	key, err := txKeyFromTx(tx)
	if err != nil {
		return err
	}

	p.mtx.Lock()
	// try active pool first
	if entry, ok := p.entries[key]; ok {
		p.removeEntryLocked(entry)
		p.mtx.Unlock()
		p.pushEvent(cmtmempool.EventTxRemoved, entry.bytes)
		return nil
	}

	// try queued pool
	if s := p.senders[key.sender]; s != nil {
		if entry, exists := s.queued[key.nonce]; exists {
			delete(s.queued, key.nonce)
			p.queuedCount.Add(-1)
			p.cleanupSenderLocked(key.sender)
			removedBytes := entry.bytes
			p.mtx.Unlock()
			p.pushEvent(cmtmempool.EventTxRemoved, removedBytes)
			return nil
		}
	}
	p.mtx.Unlock()

	return sdkmempool.ErrTxNotFound
}

// Select returns an iterator over the active priority-ordered entries.
func (p *PriorityMempool) Select(_ context.Context, _ [][]byte) sdkmempool.Iterator {
	p.mtx.RLock()
	if p.priorityIndex.Len() == 0 {
		p.mtx.RUnlock()
		return nil
	}

	entries := make([]TxInfoEntry, 0, p.priorityIndex.Len())
	for node := p.priorityIndex.Front(); node != nil; node = node.Next() {
		e := node.Value.(*txEntry)
		entries = append(entries, TxInfoEntry{
			Tx: e.tx,
			Info: TxInfo{
				Sender:   e.key.sender,
				Sequence: e.sequence,
				Size:     e.size,
				GasLimit: e.gas,
				TxBytes:  e.bytes,
				Tier:     p.tierName(e.tier),
			},
		})
	}
	p.mtx.RUnlock()

	return &priorityIterator{
		entries: entries,
		idx:     0,
	}
}

// Lookup returns the transaction hash for the given sender and nonce.
func (p *PriorityMempool) Lookup(sender string, nonce uint64) (string, bool) {
	key := txKey{sender: sender, nonce: nonce}
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if entry, ok := p.entries[key]; ok {
		return TxHash(entry.bytes), true
	}
	if s := p.senders[sender]; s != nil {
		if entry, exists := s.queued[nonce]; exists {
			return TxHash(entry.bytes), true
		}
	}

	return "", false
}

// GetTxInfo returns information about a transaction.
func (p *PriorityMempool) GetTxInfo(ctx sdk.Context, tx sdk.Tx) (TxInfo, error) {
	key, err := txKeyFromTx(tx)
	if err != nil {
		return TxInfo{}, err
	}

	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if entry, ok := p.entries[key]; ok {
		return TxInfo{
			Size:     entry.size,
			GasLimit: entry.gas,
			Sender:   entry.key.sender,
			Sequence: entry.sequence,
			TxBytes:  entry.bytes,
			Tier:     p.tierName(entry.tier),
		}, nil
	}

	if s := p.senders[key.sender]; s != nil {
		if entry, exists := s.queued[key.nonce]; exists {
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

// NextExpectedSequence returns the activeNext for a sender.
func (p *PriorityMempool) NextExpectedSequence(sender string) (uint64, bool, error) {
	p.mtx.RLock()
	s := p.senders[sender]
	if s == nil || !s.hasActiveNext {
		p.mtx.RUnlock()
		return 0, false, nil
	}
	next := s.activeNext
	p.mtx.RUnlock()

	return next, true, nil
}

// IteratePendingTxs iterates over sorted active pool entries, calling fn for each. Stops early if fn returns false.
func (p *PriorityMempool) IteratePendingTxs(fn func(sender string, nonce uint64, tx sdk.Tx) bool) {
	type item struct {
		sender string
		nonce  uint64
		tx     sdk.Tx
	}

	p.mtx.RLock()
	var items []item
	for sender, ss := range p.senders {
		for nonce, entry := range ss.active {
			items = append(items, item{sender, nonce, entry.tx})
		}
	}
	p.mtx.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		if items[i].sender != items[j].sender {
			return items[i].sender < items[j].sender
		}
		return items[i].nonce < items[j].nonce
	})

	for _, it := range items {
		if !fn(it.sender, it.nonce, it.tx) {
			return
		}
	}
}

// IterateQueuedTxs iterates over sorted queued pool entries, calling fn for each. Stops early if fn returns false.
func (p *PriorityMempool) IterateQueuedTxs(fn func(sender string, nonce uint64, tx sdk.Tx) bool) {
	type item struct {
		sender string
		nonce  uint64
		tx     sdk.Tx
	}

	p.mtx.RLock()
	var items []item
	for sender, ss := range p.senders {
		for nonce, entry := range ss.queued {
			items = append(items, item{sender, nonce, entry.tx})
		}
	}
	p.mtx.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		if items[i].sender != items[j].sender {
			return items[i].sender < items[j].sender
		}
		return items[i].nonce < items[j].nonce
	})

	for _, it := range items {
		if !fn(it.sender, it.nonce, it.tx) {
			return
		}
	}
}

// PromoteQueued evicts stale queued entries, promotes sequential queued entries,
// and refreshes activeNext for all tracked senders.
func (p *PriorityMempool) PromoteQueued(ctx context.Context) {
	if p.ak == nil {
		return
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// snapshot senders, partitioned by whether they have queued entries
	p.mtx.Lock()
	var queuedSenders, activeOnlySenders []string
	for sender, ss := range p.senders {
		if !ss.hasActiveNext {
			continue
		}
		if len(ss.queued) > 0 {
			queuedSenders = append(queuedSenders, sender)
		} else {
			activeOnlySenders = append(activeOnlySenders, sender)
		}
	}
	p.mtx.Unlock()

	if len(queuedSenders) == 0 && len(activeOnlySenders) == 0 {
		return
	}

	// fetch on-chain sequences outside the lock
	type seqResult struct {
		onChainSeq uint64
	}
	seqs := make(map[string]*seqResult, len(queuedSenders))
	for _, sender := range queuedSenders {
		seq, ok := p.fetchSequence(sdkCtx, sender)
		if !ok {
			continue
		}
		seqs[sender] = &seqResult{onChainSeq: seq}
	}

	// process under lock
	p.mtx.Lock()
	var staleEntries, promoted []*txEntry
	var removed []*txEntry

	for sender, sr := range seqs {
		ss := p.senders[sender]
		if ss == nil {
			continue
		}
		newActive := max(sr.onChainSeq, ss.activeNext)

		// evict stale queued entries
		for nonce, entry := range ss.queued {
			if nonce < sr.onChainSeq {
				staleEntries = append(staleEntries, entry)
				delete(ss.queued, nonce)
				p.queuedCount.Add(-1)
			}
		}

		// advance activeNext, collect and promote
		ss.activeNext = newActive
		toPromote := p.collectPromotableLocked(ss)
		for _, pe := range toPromote {
			pe.order = p.nextOrder()
			peCtx := sdkCtx.WithTxBytes(pe.bytes)
			pe.tier = p.selectTier(peCtx, pe.tx)
			if accepted, ev := p.canAcceptLocked(peCtx, pe.tier, pe.priority, pe.size, pe.gas, nil); accepted {
				p.removeEntriesLocked(ev...)
				removed = append(removed, ev...)
				p.addEntryLocked(pe)
				promoted = append(promoted, pe)
			}
		}

		// cleanup sender if fully drained
		if len(ss.queued) == 0 && (sr.onChainSeq >= ss.activeNext || len(ss.active) == 0) {
			p.cleanupSenderLocked(sender)
		}
	}

	// active only senders, clean up if no pool entries remain
	for _, sender := range activeOnlySenders {
		if ss := p.senders[sender]; ss != nil && len(ss.active) == 0 {
			p.cleanupSenderLocked(sender)
		}
	}

	p.mtx.Unlock()

	for _, entry := range staleEntries {
		p.pushEvent(cmtmempool.EventTxRemoved, entry.bytes)
	}
	p.pushRemovedEvents(removed)
	for _, entry := range promoted {
		p.pushEvent(cmtmempool.EventTxInserted, entry.bytes)
	}
}

// fetchSequence queries the on-chain sequence for a sender.
func (p *PriorityMempool) fetchSequence(ctx sdk.Context, sender string) (uint64, bool) {
	addr, err := sdk.AccAddressFromBech32(sender)
	if err != nil {
		return 0, false
	}

	seq, err := p.ak.GetSequence(ctx, addr)
	if err != nil {
		// AccountKeeper.GetSequence returns an error only when the account does not
		// exist yet. Treat that as sequence 0 and mark the lookup as usable.
		return 0, true
	}

	return seq, true
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
	} else if p.maxQueuedTotal > 0 && int(p.queuedCount.Load()) >= p.maxQueuedTotal {
		return false, nil
	}

	ss.queued[key.nonce] = entry
	p.queuedCount.Add(1)

	return true, evicted
}

// collectPromotableLocked removes queued txs with continuous nonces starting from
// activeNext and returns them for promotion. the caller must hold p.mtx.
func (p *PriorityMempool) collectPromotableLocked(ss *senderState) []*txEntry {
	if len(ss.queued) == 0 {
		return nil
	}

	var entries []*txEntry
	next := ss.activeNext
	for {
		entry, exists := ss.queued[next]
		if !exists {
			break
		}
		entries = append(entries, entry)
		delete(ss.queued, next)
		p.queuedCount.Add(-1)
		next++
	}

	if len(entries) > 0 {
		ss.activeNext = next
	}

	return entries
}

// cleanupSenderLocked removes the sender state if fully empty. the caller must hold p.mtx.
func (p *PriorityMempool) cleanupSenderLocked(sender string) {
	if ss := p.senders[sender]; ss != nil && ss.isEmpty() {
		delete(p.senders, sender)
	}
}

// pushEvent sends an event to the cometbft event channel.
func (p *PriorityMempool) pushEvent(eventType cmtmempool.AppMempoolEventType, txBytes []byte) {
	chPtr := p.eventCh.Load()
	if chPtr == nil {
		return
	}

	cmtTx := cmttypes.Tx(txBytes)
	select {
	case *chPtr <- cmtmempool.AppMempoolEvent{
		Type:  eventType,
		TxKey: cmtTx.Key(),
		Tx:    cmtTx,
	}:
	default:
	}
}

// pushRemovedEvents sends EventTxRemoved for each entry.
func (p *PriorityMempool) pushRemovedEvents(entries []*txEntry) {
	for _, entry := range entries {
		p.pushEvent(cmtmempool.EventTxRemoved, entry.bytes)
	}
}

type txEntry struct {
	tx       sdk.Tx
	priority int64
	size     int64
	key      txKey
	sequence uint64
	order    int64
	tier     int
	gas      uint64
	bytes    []byte
}

// priorityIterator walks entries in the order determined by the priority index.
type priorityIterator struct {
	entries []TxInfoEntry
	idx     int
}

// Next advances the iterator to the next entry and returns nil when exhausted.
func (it *priorityIterator) Next() sdkmempool.Iterator {
	it.idx++
	if it.idx >= len(it.entries) {
		return nil
	}
	return it
}

// Tx returns the tx currently pointed to by the iterator.
func (it *priorityIterator) Tx() sdk.Tx {
	if it.idx >= len(it.entries) || it.idx < 0 {
		return nil
	}
	return it.entries[it.idx].Tx
}

// TxInfo returns the metadata for the entry currently pointed to by the iterator.
func (it *priorityIterator) TxInfo() TxInfo {
	if it.idx >= len(it.entries) || it.idx < 0 {
		return TxInfo{}
	}
	return it.entries[it.idx].Info
}

// compareEntries orders txEntries by tier, priority, order, sender, and nonce.
func compareEntries(a, b any) int {
	left := a.(*txEntry)
	right := b.(*txEntry)

	// lower tier wins
	if left.tier != right.tier {
		if left.tier < right.tier {
			return -1
		}
		return 1
	}

	// higher priority value wins
	if left.priority != right.priority {
		if left.priority > right.priority {
			return -1
		}
		return 1
	}

	// maintain FIFO order for same priority
	if left.order != right.order {
		if left.order < right.order {
			return -1
		}
		return 1
	}

	if left.key.sender != right.key.sender {
		return strings.Compare(left.key.sender, right.key.sender)
	}

	switch {
	case left.key.nonce < right.key.nonce:
		return -1
	case left.key.nonce > right.key.nonce:
		return 1
	default:
		return 0
	}
}

// nextOrder returns a unique sequence number used to preserve insertion order.
func (p *PriorityMempool) nextOrder() int64 {
	return atomic.AddInt64(&p.orderSeq, 1)
}

// selectTier returns the index of the first matching tier matcher for the tx.
func (p *PriorityMempool) selectTier(ctx sdk.Context, tx sdk.Tx) int {
	for idx, tier := range p.tiers {
		if tier.Matcher == nil || tier.Matcher(ctx, tx) {
			return idx
		}
	}
	return len(p.tiers) - 1
}

type tierMatcher struct {
	Name    string
	Matcher TierMatcher
}

// tierName returns the configured name for a tier index, or empty if invalid.
func (p *PriorityMempool) tierName(idx int) string {
	if idx < 0 || idx >= len(p.tiers) {
		return ""
	}
	return p.tiers[idx].Name
}

// buildTierMatchers canonicalizes the configured tiers into matcher helpers and ensures a default tier.
func buildTierMatchers(cfg PriorityMempoolConfig) []tierMatcher {
	matchers := make([]tierMatcher, 0, len(cfg.Tiers)+1)
	for idx, tier := range cfg.Tiers {
		if tier.Matcher == nil {
			continue
		}

		name := strings.TrimSpace(tier.Name)
		if name == "" {
			name = fmt.Sprintf("tier-%d", idx)
		}

		matchers = append(matchers, tierMatcher{
			Name:    name,
			Matcher: tier.Matcher,
		})
	}

	matchers = append(matchers, tierMatcher{
		Name:    "default",
		Matcher: func(ctx sdk.Context, tx sdk.Tx) bool { return true },
	})

	return matchers
}

// initTierDistribution creates a zeroed counter map for each named tier.
func initTierDistribution(tiers []tierMatcher) map[string]uint64 {
	dist := make(map[string]uint64, len(tiers))
	for _, tier := range tiers {
		if tier.Name == "" {
			continue
		}
		dist[tier.Name] = 0
	}
	return dist
}

// canAcceptLocked checks whether a tx can be accepted and returns the list of
// entries that should be evicted to make room. It does not mutate pool state.
// If exclude is non-nil, capacity planning treats it as already absent.
func (p *PriorityMempool) canAcceptLocked(ctx sdk.Context, tier int, priority int64, size int64, gas uint64, exclude *txEntry) (bool, []*txEntry) {
	var evictList []*txEntry

	// First enforce per-tx hard limits. If the candidate can never fit into a
	// block, reject it without evicting existing mempool entries.
	blockParams := ctx.ConsensusParams().Block
	blockMaxBytes := blockParams.MaxBytes
	if blockMaxBytes > 0 && size > blockMaxBytes {
		return false, evictList
	}

	blockMaxGas := blockParams.MaxGas
	if blockMaxGas > 0 && gas > uint64(blockMaxGas) {
		return false, evictList
	}

	// Capacity eviction comes after hard checks so we only evict when the new tx
	// is otherwise admissible.
	if p.cfg.MaxTx > 0 {
		targetLen := p.cfg.MaxTx - 1 // one slot needed for candidate tx
		curLen := len(p.entries)
		if exclude != nil {
			if _, ok := p.entries[exclude.key]; ok {
				curLen--
			}
		}
		for node := p.priorityIndex.Back(); curLen > targetLen; node = node.Prev() {
			if node == nil {
				return false, evictList
			}

			entry := node.Value.(*txEntry)
			if exclude != nil && entry.key == exclude.key {
				continue
			}
			if !p.isBetterThan(entry, tier, priority) {
				return false, evictList
			}

			evictList = append(evictList, entry)
			curLen--
		}
	}

	return true, evictList
}

// isBetterThan determines whether a new entry should outrank an existing one.
func (p *PriorityMempool) isBetterThan(entry *txEntry, tier int, priority int64) bool {
	if tier != entry.tier {
		return tier < entry.tier
	}
	return priority > entry.priority
}

// addEntryLocked inserts the tx entry into the priority index and updates sender/tier bookkeeping.
func (p *PriorityMempool) addEntryLocked(entry *txEntry) {
	p.priorityIndex.Set(entry, entry)
	p.entries[entry.key] = entry
	p.getOrCreateSenderLocked(entry.key.sender).active[entry.key.nonce] = entry

	if name := p.tierName(entry.tier); name != "" {
		p.tierDistribution[name]++
	}
}

// removeEntriesLocked removes multiple entries and returns the removed entries for event dispatch.
func (p *PriorityMempool) removeEntriesLocked(entries ...*txEntry) {
	for _, entry := range entries {
		p.removeEntryLocked(entry)
	}
}

// removeEntryLocked evicts an entry from the priority index and adjusts sender/tier counts.
func (p *PriorityMempool) removeEntryLocked(entry *txEntry) {
	if entry == nil {
		return
	}

	p.priorityIndex.Remove(entry)
	delete(p.entries, entry.key)
	if s := p.senders[entry.key.sender]; s != nil {
		delete(s.active, entry.key.nonce)
	}

	if name := p.tierName(entry.tier); name != "" {
		if count, ok := p.tierDistribution[name]; ok && count > 0 {
			p.tierDistribution[name] = count - 1
		}
	}
}

// txKeyFromTx extracts the sender address and nonce that uniquely identifies the tx.
func txKeyFromTx(tx sdk.Tx) (txKey, error) {
	sender, sequence, err := FirstSignature(tx)
	if err != nil {
		return txKey{}, err
	}
	return txKey{
		sender: sender.String(),
		nonce:  sequence,
	}, nil
}
