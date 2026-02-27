package abcipp

import (
	"context"
	"sync"
	"sync/atomic"

	cmtmempool "github.com/cometbft/cometbft/mempool"
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

type RemovalReason uint8

const (
	// RemovalReasonCapacityEvicted is used when low-priority active txs are removed
	// to satisfy pool capacity checks.
	RemovalReasonCapacityEvicted RemovalReason = iota
	// RemovalReasonCommittedInBlock is used when a tx is removed because it was
	// included in a block.
	RemovalReasonCommittedInBlock
	// RemovalReasonAnteRejectedInPrepare is used when a tx fails ante during
	// proposal construction or recheck-time validation paths.
	RemovalReasonAnteRejectedInPrepare
)

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

	eventCh     atomic.Pointer[chan<- cmtmempool.AppMempoolEvent]
	eventMu     sync.Mutex
	eventQueue  []cmtmempool.AppMempoolEvent
	eventNotify chan struct{}
	eventStop   chan struct{}
	eventDone   chan struct{}
}

// NewPriorityMempool creates a new PriorityMempool with the provided limits.
// AccountKeeper is required for sender sequence routing.
func NewPriorityMempool(cfg PriorityMempoolConfig, txEncoder sdk.TxEncoder, ak AccountKeeper) *PriorityMempool {
	if txEncoder == nil {
		panic("tx encoder is required")
	}
	if ak == nil {
		panic("account keeper is required")
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

	p := &PriorityMempool{
		cfg:                cfg,
		priorityIndex:      skiplist.New(skiplist.GreaterThanFunc(compareEntries)),
		entries:            make(map[txKey]*txEntry),
		senders:            make(map[string]*senderState),
		txEncoder:          txEncoder,
		tiers:              tiers,
		tierDistribution:   dist,
		maxQueuedPerSender: maxQPS,
		maxQueuedTotal:     maxQT,
		ak:                 ak,
		eventNotify:        make(chan struct{}, 1),
		eventStop:          make(chan struct{}),
		eventDone:          make(chan struct{}),
	}
	go p.eventDispatchLoop()
	return p
}

// Stop signals all background workers to exit and waits for them to finish.
func (p *PriorityMempool) Stop() {
	p.StopCleaningWorker()
	p.StopEventDispatch()
}

// SetMaxQueuedPerSender overrides the default per-sender queued tx limit.
func (p *PriorityMempool) SetMaxQueuedPerSender(n int) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.maxQueuedPerSender = n
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

// PromoteQueued evicts stale queued entries, promotes sequential queued entries,
// and refreshes sender on-chain sequence for all tracked senders.
func (p *PriorityMempool) PromoteQueued(ctx context.Context) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// snapshot senders, partitioned by whether they have queued entries
	p.mtx.Lock()
	if p.ak == nil {
		p.mtx.Unlock()
		return
	}
	ak := p.ak

	var queuedSenders, activeOnlySenders []string
	for sender, ss := range p.senders {
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
		seq, ok := fetchSequence(sdkCtx, ak, sender)
		if !ok {
			continue
		}
		seqs[sender] = &seqResult{onChainSeq: seq}
	}

	// process under lock
	p.mtx.Lock()
	var promoted []*txEntry
	var removed []*txEntry

	for sender, sr := range seqs {
		ss := p.senders[sender]
		if ss == nil {
			continue
		}
		ss.setOnChainSeqLocked(sr.onChainSeq)

		//  Reconcile sender pools by dropping entries below the latest on-chain sequence.
		removed = append(removed, p.removeStaleLocked(ss, sr.onChainSeq)...)

		// collect and promote from current sender cursor.
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

		// cleanup sender if fully drained
		if len(ss.queued) == 0 && (sr.onChainSeq >= ss.nextExpectedNonce() || len(ss.active) == 0) {
			p.cleanupSenderLocked(sender)
		}
	}

	// active only senders, clean up if no pool entries remain
	for _, sender := range activeOnlySenders {
		if ss := p.senders[sender]; ss != nil && len(ss.active) == 0 {
			p.cleanupSenderLocked(sender)
		}
	}

	p.enqueueRemovedEvents(removed)
	for _, entry := range promoted {
		p.enqueueEvent(cmtmempool.EventTxInserted, entry.bytes)
	}
	p.mtx.Unlock()
}

// cleanupSenderLocked removes the sender state if fully empty. the caller must hold p.mtx.
func (p *PriorityMempool) cleanupSenderLocked(sender string) {
	if ss := p.senders[sender]; ss != nil && ss.isEmpty() {
		delete(p.senders, sender)
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

// nextOrder returns a unique sequence number used to preserve insertion order.
func (p *PriorityMempool) nextOrder() int64 {
	return atomic.AddInt64(&p.orderSeq, 1)
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

// addEntryLocked inserts the tx entry into the priority index and updates sender/tier bookkeeping.
func (p *PriorityMempool) addEntryLocked(entry *txEntry) {
	p.priorityIndex.Set(entry, entry)
	p.entries[entry.key] = entry
	ss := p.getOrCreateSenderLocked(entry.key.sender)
	ss.active[entry.key.nonce] = entry
	ss.setActiveRangeOnInsertLocked(entry.key.nonce)

	if name := p.tierName(entry.tier); name != "" {
		p.tierDistribution[name]++
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
		if _, exists := s.active[entry.key.nonce]; exists {
			delete(s.active, entry.key.nonce)
			s.setActiveRangeOnRemoveLocked(entry.key.nonce)
		}
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
