package abcipp

import (
	"context"
	"fmt"
	"maps"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/huandu/skiplist"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var _ Mempool = (*PriorityMempool)(nil)

// PriorityMempoolConfig configures the limits enforced by the priority mempool.
// A MaxTx value less than or equal to zero is treated as "unlimited."
type PriorityMempoolConfig struct {
	MaxTx int // total transaction limit
	Tiers []Tier

	// AnteHandler to filter out invalid transactions from the mempool
	AnteHandler sdk.AnteHandler
}

type TierMatcher func(ctx sdk.Context, tx sdk.Tx) bool

type Tier struct {
	Name    string
	Matcher TierMatcher
}

// TxEventListener can be registered to observe when transactions enter or leave
// the mempool.
type TxEventListener interface {
	OnTxInserted(tx sdk.Tx)
	OnTxRemoved(tx sdk.Tx)
}

type txKey struct {
	sender string
	nonce  uint64
}

// PriorityMempool is a transaction pool that keeps high-priority submissions
// flowing with low latency while still making progress on lower-priority ones.
// Transactions are tracked per user and ordered by priority so we can keep
// concurrency high, favor priority levels, and efficiently drop all pending
// messages from a user when a block is committed.
//
// The mempool caps total transactions and bytes, ordering the retained entries
// by priority across all users so higher-quality traffic stays ahead. The
// limits guard against memory exhaustion attacks while still letting us batch
// transactions efficiently.
type PriorityMempool struct {
	mtx              sync.Mutex
	cfg              PriorityMempoolConfig
	txEncoder        sdk.TxEncoder
	priorityIndex    *skiplist.SkipList
	entries          map[txKey]*txEntry
	userBuckets      map[string]*userBucket
	orderSeq         int64
	listeners        []TxEventListener
	tiers            []tierMatcher
	tierDistribution map[string]uint64
	cleaningStopCh   chan struct{}
	cleaningDoneCh   chan struct{}
}

// NewPriorityMempool creates a new PriorityMempool with the provided limits.
func NewPriorityMempool(cfg PriorityMempoolConfig, txEncoder sdk.TxEncoder) *PriorityMempool {
	if txEncoder == nil {
		panic("tx encoder is required")
	}
	tiers := buildTierMatchers(cfg)
	dist := initTierDistribution(tiers)
	return &PriorityMempool{
		cfg:              cfg,
		priorityIndex:    skiplist.New(skiplist.GreaterThanFunc(compareEntries)),
		entries:          make(map[txKey]*txEntry),
		userBuckets:      make(map[string]*userBucket),
		txEncoder:        txEncoder,
		tiers:            tiers,
		tierDistribution: dist,
	}
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

	buckets := p.snapshotBuckets()
	var removed []*txEntry
	for sender, bucket := range buckets {
		startSeq, ok := bucket.snapshotStart()
		if !ok {
			continue
		}
		accountAddr, err := sdk.AccAddressFromBech32(sender)
		if err != nil {
			continue
		}
		accountSeq, err := ak.GetSequence(sdkCtx, accountAddr)
		if err != nil {
			continue
		}

		// remove stale entries below on-chain sequence
		if accountSeq > startSeq {
			removed = append(removed, bucket.collectStale(accountSeq)...)
			startSeq = accountSeq
		}

		// remove invalid entries starting from current on-chain sequence
		removed = append(removed, bucket.collectInvalid(sdkCtx, p.cfg.AnteHandler, startSeq)...)
	}

	if len(removed) == 0 {
		return
	}

	p.mtx.Lock()
	listeners := copyListeners(p.listeners)
	var finalRemoved []*txEntry
	for _, entry := range removed {
		if existing, ok := p.entries[entry.key]; ok {
			finalRemoved = append(finalRemoved, p.removeEntry(existing))
		}
	}
	p.mtx.Unlock()

	p.dispatchRemoved(listeners, finalRemoved)
}

// RegisterEventListener registers an observer that will be notified whenever
// transactions are inserted or removed from the pool.
func (p *PriorityMempool) RegisterEventListener(listener TxEventListener) {
	if listener == nil {
		return
	}

	p.mtx.Lock()
	defer p.mtx.Unlock()
	p.listeners = append(p.listeners, listener)
}

// copyListeners produces a shallow snapshot of listeners so we can iterate without holding the lock.
func copyListeners(list []TxEventListener) []TxEventListener {
	if len(list) == 0 {
		return nil
	}
	copied := make([]TxEventListener, len(list))
	copy(copied, list)
	return copied
}

// Contains implements Mempool.
func (p *PriorityMempool) Contains(tx sdk.Tx) bool {
	key, err := txKeyFromTx(tx)
	if err != nil {
		return false
	}

	p.mtx.Lock()
	defer p.mtx.Unlock()
	_, ok := p.entries[key]
	return ok
}

// CountTx implements Mempool.
func (p *PriorityMempool) CountTx() int {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	return len(p.entries)
}

// GetTxDistribution returns the number of transactions per configured tier.
func (p *PriorityMempool) GetTxDistribution() map[string]uint64 {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	out := make(map[string]uint64, len(p.tierDistribution))
	maps.Copy(out, p.tierDistribution)
	return out
}

// Insert implements Mempool.
func (p *PriorityMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	priority := sdkCtx.Priority()

	key, err := txKeyFromTx(tx)
	if err != nil {
		return err
	}

	bz, size, err := p.txBytesAndSize(tx)
	if err != nil {
		return err
	}

	var gas uint64
	if feeTx, ok := tx.(sdk.FeeTx); ok {
		gas = feeTx.GetGas()
	} else {
		return fmt.Errorf("tx does not implement FeeTx")
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

	p.mtx.Lock()
	listeners := copyListeners(p.listeners)
	var removed []*txEntry

	if existing, ok := p.entries[entry.key]; ok {
		if entry.priority < existing.priority {
			p.mtx.Unlock()
			return nil
		}
		removed = append(removed, p.removeEntry(existing))
	}

	ok, evicted := p.canAccept(sdkCtx, entry.tier, entry.priority, entry.size, entry.gas)
	removed = append(removed, evicted...)
	if !ok {
		p.mtx.Unlock()
		p.dispatchRemoved(listeners, removed)
		return sdkmempool.ErrMempoolTxMaxCapacity
	}

	p.addEntry(entry)
	p.mtx.Unlock()

	p.dispatchRemoved(listeners, removed)
	p.dispatchInserted(listeners, entry.tx)
	return nil
}

// Remove implements Mempool.
func (p *PriorityMempool) Remove(tx sdk.Tx) error {
	key, err := txKeyFromTx(tx)
	if err != nil {
		return err
	}

	p.mtx.Lock()
	listeners := copyListeners(p.listeners)
	entry, ok := p.entries[key]
	if !ok {
		p.mtx.Unlock()
		return sdkmempool.ErrTxNotFound
	}

	removed := p.removeEntry(entry)
	p.mtx.Unlock()

	p.dispatchRemoved(listeners, []*txEntry{removed})
	return nil
}

// Select implements Mempool.
func (p *PriorityMempool) Select(ctx context.Context, _ [][]byte) sdkmempool.Iterator {
	p.mtx.Lock()
	if p.priorityIndex.Len() == 0 {
		p.mtx.Unlock()
		return nil
	}

	entries := make([]*txEntry, 0, p.priorityIndex.Len())
	for node := p.priorityIndex.Front(); node != nil; node = node.Next() {
		entries = append(entries, node.Value.(*txEntry))
	}
	p.mtx.Unlock()

	return &priorityIterator{
		entries: entries,
		idx:     0,
	}
}

// Lookup returns the transaction hash for the given sender and nonce.
func (p *PriorityMempool) Lookup(sender string, nonce uint64) (string, bool) {
	key := txKey{sender: sender, nonce: nonce}
	p.mtx.Lock()
	defer p.mtx.Unlock()
	entry, ok := p.entries[key]
	if !ok {
		return "", false
	}
	return TxHash(entry.bytes), true
}

// GetTxInfo implements Mempool.
func (p *PriorityMempool) GetTxInfo(ctx sdk.Context, tx sdk.Tx) (TxInfo, error) {
	key, err := txKeyFromTx(tx)
	if err != nil {
		return TxInfo{}, err
	}

	p.mtx.Lock()
	defer p.mtx.Unlock()
	entry, ok := p.entries[key]
	if !ok {
		return TxInfo{}, sdkmempool.ErrTxNotFound
	}
	tierName := p.tierName(entry.tier)
	return TxInfo{
		Size:     entry.size,
		GasLimit: entry.gas,
		Sender:   entry.key.sender,
		Sequence: entry.sequence,
		TxBytes:  entry.bytes,
		Tier:     tierName,
	}, nil
}

// HasSenderEntries returns true if the pool contains any entries for the given sender.
func (p *PriorityMempool) HasSenderEntries(sender string) bool {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	_, ok := p.userBuckets[sender]
	return ok
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

const emptySeq = math.MaxUint64

type userBucket struct {
	mu      sync.Mutex
	entries map[uint64]*txEntry
	start   uint64
	next    uint64
}

// newUserBucket creates an empty bucket for a single sender.
func newUserBucket() *userBucket {
	return &userBucket{
		entries: make(map[uint64]*txEntry),
		start:   emptySeq,
	}
}

// add records a tx entry in the bucket and updates bounds.
func (b *userBucket) add(entry *txEntry) {
	if entry.sequence < b.start {
		b.start = entry.sequence
	}
	if entry.sequence+1 > b.next {
		b.next = entry.sequence + 1
	}
	b.entries[entry.sequence] = entry
}

// remove deletes an entry from the bucket and keeps start/next consistent.
func (b *userBucket) remove(sequence uint64) {
	delete(b.entries, sequence)
	if len(b.entries) == 0 {
		b.start = emptySeq
		b.next = 0
		return
	}
	b.recomputeBounds()
}

// recomputeBounds recalculates the min/max sequence in the bucket.
func (b *userBucket) recomputeBounds() {
	minSeq := uint64(math.MaxUint64)
	maxSeq := uint64(0)
	for seq := range b.entries {
		if seq < minSeq {
			minSeq = seq
		}
		if seq > maxSeq {
			maxSeq = seq
		}
	}
	b.start = minSeq
	b.next = maxSeq + 1
}

// snapshotStart returns the current start sequence or false if empty.
func (b *userBucket) snapshotStart() (uint64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.start == emptySeq {
		return 0, false
	}
	return b.start, true
}

// collectStale gathers entries whose sequence is below the provided threshold.
func (b *userBucket) collectStale(upto uint64) []*txEntry {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.start == emptySeq || b.start >= upto {
		return nil
	}

	var entries []*txEntry
	for seq := b.start; seq < upto; seq++ {
		if entry, ok := b.entries[seq]; ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

// collectInvalid gathers entries that fail anteHandler checks starting from start.
func (b *userBucket) collectInvalid(ctx sdk.Context, anteHandler sdk.AnteHandler, start uint64) []*txEntry {
	if anteHandler == nil {
		return nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.next == 0 {
		return nil
	}

	// if we find first invalid entry, trash all next entries to the end
	failed := false
	var invalidEntries []*txEntry
	for seq := start; seq < b.next; seq++ {
		if entry, ok := b.entries[seq]; ok {
			if failed {
				invalidEntries = append(invalidEntries, entry)
				continue
			}

			sdkCtx, write := ctx.WithTxBytes(entry.bytes).WithIsReCheckTx(true).CacheContext()
			if _, err := anteHandler(sdkCtx, entry.tx, false); err != nil {
				failed = true
				invalidEntries = append(invalidEntries, entry)
				continue
			}

			write()
		}
	}

	return invalidEntries
}

func (p *PriorityMempool) snapshotBuckets() map[string]*userBucket {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	targets := make(map[string]*userBucket, len(p.userBuckets))
	maps.Copy(targets, p.userBuckets)
	return targets
}

// priorityIterator walks entries in the order determined by the priority index.
type priorityIterator struct {
	entries []*txEntry
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
	return it.entries[it.idx].tx
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

// canAccept checks whether a tx can remain in the pool given the configured limits and evicts as needed.
func (p *PriorityMempool) canAccept(ctx sdk.Context, tier int, priority int64, size int64, gas uint64) (bool, []*txEntry) {
	var removed []*txEntry

	if p.cfg.MaxTx > 0 {
		for len(p.entries) >= p.cfg.MaxTx {
			evicted := p.evictLower(tier, priority)
			if evicted == nil {
				return false, removed
			}
			removed = append(removed, evicted)
		}
	}

	blockParams := ctx.ConsensusParams().Block
	blockMaxBytes := blockParams.MaxBytes
	if blockMaxBytes > 0 && size > blockMaxBytes {
		return false, removed
	}

	blockMaxGas := blockParams.MaxGas
	if blockMaxGas > 0 && gas > uint64(blockMaxGas) {
		return false, removed
	}

	return true, removed
}

// evictLower removes the lowest-priority entry that is worse than the provided tier/priority.
func (p *PriorityMempool) evictLower(tier int, priority int64) *txEntry {
	back := p.priorityIndex.Back()
	if back == nil {
		return nil
	}

	entry := back.Value.(*txEntry)
	if !p.isBetterThan(entry, tier, priority) {
		return nil
	}

	return p.removeEntry(entry)
}

// isBetterThan determines whether a new entry should outrank an existing one.
func (p *PriorityMempool) isBetterThan(entry *txEntry, tier int, priority int64) bool {
	if tier != entry.tier {
		return tier < entry.tier
	}
	return priority > entry.priority
}

// addEntry inserts the tx entry into all indexes and updates per-sender/tier bookkeeping.
func (p *PriorityMempool) addEntry(entry *txEntry) {
	p.priorityIndex.Set(entry, entry)
	p.entries[entry.key] = entry

	bucket, ok := p.userBuckets[entry.key.sender]
	if !ok {
		bucket = newUserBucket()
		p.userBuckets[entry.key.sender] = bucket
	}

	bucket.mu.Lock()
	bucket.add(entry)
	bucket.mu.Unlock()

	if name := p.tierName(entry.tier); name != "" {
		p.tierDistribution[name]++
	}
}

// removeEntry evicts an entry from every structure and adjusts tier counts.
func (p *PriorityMempool) removeEntry(entry *txEntry) *txEntry {
	if entry == nil {
		return nil
	}

	if bucket, ok := p.userBuckets[entry.key.sender]; ok {
		bucket.mu.Lock()
		bucket.remove(entry.sequence)
		empty := len(bucket.entries) == 0
		bucket.mu.Unlock()
		if empty {
			delete(p.userBuckets, entry.key.sender)
		}
	}

	p.priorityIndex.Remove(entry)
	delete(p.entries, entry.key)

	if name := p.tierName(entry.tier); name != "" {
		if count, ok := p.tierDistribution[name]; ok && count > 0 {
			p.tierDistribution[name] = count - 1
		}
	}

	return entry
}

// dispatchInserted notifies listeners about a newly accepted transaction.
func (p *PriorityMempool) dispatchInserted(listeners []TxEventListener, tx sdk.Tx) {
	if len(listeners) == 0 {
		return
	}

	for _, l := range listeners {
		l.OnTxInserted(tx)
	}
}

// dispatchRemoved notifies listeners about transactions that left the pool.
func (p *PriorityMempool) dispatchRemoved(listeners []TxEventListener, entries []*txEntry) {
	if len(listeners) == 0 || len(entries) == 0 {
		return
	}

	for _, entry := range entries {
		for _, l := range listeners {
			l.OnTxRemoved(entry.tx)
		}
	}
}

// txBytesAndSize encodes the tx and returns its bytes with length.
func (p *PriorityMempool) txBytesAndSize(tx sdk.Tx) ([]byte, int64, error) {
	bz, err := p.txEncoder(tx)
	if err != nil {
		return nil, 0, err
	}
	return bz, int64(len(bz)), nil
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
