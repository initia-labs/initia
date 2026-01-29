package abcipp

import (
	"context"
	"fmt"
	"maps"
	"math"
	"strings"
	"sync"
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
	ak               AccountKeeper
	txEncoder        sdk.TxEncoder
	priorityIndex    *skiplist.SkipList
	entries          map[txKey]*txEntry
	userBuckets      map[string]*userBucket
	orderSeq         int64
	listeners        []TxEventListener
	tiers            []tierMatcher
	tierDistribution map[string]uint64
	recheckMu        sync.Mutex
	recheckActive    bool
	recheckCtx       sdk.Context
	recheckPending   bool
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
		ak:               nil,
		priorityIndex:    skiplist.New(skiplist.LessThanFunc(compareEntries)),
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
func (p *PriorityMempool) StartCleaningWorker(baseApp BaseApp, ak AccountKeeper, interval time.Duration) {
	p.ak = ak
	go func() {
		timer := time.NewTicker(interval)
		for range timer.C {
			p.cleanUpEntries(baseApp, ak)
		}
	}()
}

// cleanUpEntries removes transactions from users whose on-chain sequence
// has advanced beyond the sequences of their pending transactions.
func (p *PriorityMempool) cleanUpEntries(baseApp BaseApp, ak AccountKeeper) {
	sdkCtx := baseApp.GetContextForCheckTx(nil)

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
		if accountSeq <= startSeq {
			continue
		}
		removed = append(removed, bucket.collectStale(accountSeq)...)
	}

	if len(removed) == 0 {
		return
	}

	p.mtx.Lock()
	listeners := append([]TxEventListener(nil), p.listeners...)
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

// Contains implements Mempool.
func (p *PriorityMempool) Contains(tx sdk.Tx) bool {
	sender, sequence, err := FirstSignature(tx)
	if err != nil {
		return false
	}

	key := txKey{sender: sender.String(), nonce: sequence}
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
	for name, count := range p.tierDistribution {
		out[name] = count
	}
	return out
}

// Insert implements Mempool.
func (p *PriorityMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	priority := sdkCtx.Priority()

	sender, sequence, err := FirstSignature(tx)
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
		key: txKey{
			sender: sender.String(),
			nonce:  sequence,
		},
		sequence: sequence,
		order:    p.nextOrder(),
		tier:     p.selectTier(sdkCtx, tx),
		gas:      gas,
		bytes:    bz,
	}

	p.mtx.Lock()
	listeners := append([]TxEventListener(nil), p.listeners...)
	var removed []*txEntry
	isNewEntry := true

	if existing, ok := p.entries[entry.key]; ok {
		if entry.priority < existing.priority {
			p.mtx.Unlock()
			return nil
		}
		removed = append(removed, p.removeEntry(existing))
		isNewEntry = false
	}

	if isNewEntry {
		if expectedSeq, enforce, err := p.expectedNextSequenceLocked(sdkCtx, entry.key.sender); err != nil {
			p.mtx.Unlock()
			return err
		} else if enforce && entry.sequence != expectedSeq {
			p.mtx.Unlock()
			return fmt.Errorf("tx sequence %d is out of order for sender %s (expected %d)", entry.sequence, entry.key.sender, expectedSeq)
		}
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

// GetTxInfo implements Mempool.
func (p *PriorityMempool) GetTxInfo(ctx sdk.Context, tx sdk.Tx) (TxInfo, error) {
	sender, sequence, err := FirstSignature(tx)
	if err != nil {
		return TxInfo{}, err
	}

	key := txKey{sender: sender.String(), nonce: sequence}
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

// Remove implements Mempool.
func (p *PriorityMempool) Remove(tx sdk.Tx) error {
	sender, sequence, err := FirstSignature(tx)
	if err != nil {
		return err
	}

	key := txKey{sender: sender.String(), nonce: sequence}
	p.mtx.Lock()
	listeners := append([]TxEventListener(nil), p.listeners...)
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

func newUserBucket() *userBucket {
	return &userBucket{
		entries: make(map[uint64]*txEntry),
		start:   emptySeq,
	}
}

func (b *userBucket) add(entry *txEntry) {
	if entry.sequence < b.start {
		b.start = entry.sequence
	}
	if entry.sequence+1 > b.next {
		b.next = entry.sequence + 1
	}
	b.entries[entry.sequence] = entry
}

func (b *userBucket) remove(sequence uint64) {
	delete(b.entries, sequence)
	if len(b.entries) == 0 {
		b.start = emptySeq
		b.next = 0
		return
	}
	b.recomputeBounds()
}

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

func (b *userBucket) snapshotStart() (uint64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.start == emptySeq {
		return 0, false
	}
	return b.start, true
}

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

func (b *userBucket) nextSequence() (uint64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.next == 0 {
		return 0, false
	}
	return b.next, true
}

func (p *PriorityMempool) Recheck(ctx sdk.Context) error {
	if p.ak == nil {
		return nil
	}

	p.recheckMu.Lock()
	if p.recheckActive {
		p.recheckCtx = ctx
		p.recheckPending = true
		p.recheckMu.Unlock()
		return nil
	}

	p.recheckCtx = ctx
	p.recheckPending = false
	p.recheckActive = true
	p.recheckMu.Unlock()

	go p.recheckWorker(ctx)
	return nil
}

func (p *PriorityMempool) recheckWorker(ctx sdk.Context) {
	for {
		p.runRecheckOnce(ctx)

		p.recheckMu.Lock()
		if !p.recheckPending {
			p.recheckActive = false
			p.recheckMu.Unlock()
			return
		}
		ctx = p.recheckCtx
		p.recheckPending = false
		p.recheckMu.Unlock()
	}
}

func (p *PriorityMempool) runRecheckOnce(ctx sdk.Context) {
	targets := p.snapshotBuckets()
	toRemove := p.collectStaleEntries(targets, ctx)
	if len(toRemove) == 0 {
		return
	}

	p.mtx.Lock()
	listeners := append([]TxEventListener(nil), p.listeners...)
	var removed []*txEntry
	for _, entry := range toRemove {
		if existing, ok := p.entries[entry.key]; ok {
			removed = append(removed, p.removeEntry(existing))
		}
	}
	p.mtx.Unlock()

	p.dispatchRemoved(listeners, removed)
}

func (p *PriorityMempool) snapshotBuckets() map[string]*userBucket {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	targets := make(map[string]*userBucket, len(p.userBuckets))
	maps.Copy(targets, p.userBuckets)
	return targets
}

func (p *PriorityMempool) collectStaleEntries(targets map[string]*userBucket, ctx sdk.Context) []*txEntry {
	if p.ak == nil {
		return nil
	}

	ctxCtx := ctx.Context()
	var toRemove []*txEntry
	for sender, bucket := range targets {
		startSeq, ok := bucket.snapshotStart()
		if !ok {
			continue
		}
		addr, err := sdk.AccAddressFromBech32(sender)
		if err != nil {
			continue
		}
		seq, err := p.ak.GetSequence(ctxCtx, addr)
		if err != nil {
			continue
		}
		if seq <= startSeq {
			continue
		}
		toRemove = append(toRemove, bucket.collectStale(seq)...)
	}
	return toRemove
}

type priorityIterator struct {
	entries []*txEntry
	idx     int
}

func (it *priorityIterator) Next() sdkmempool.Iterator {
	it.idx++
	if it.idx >= len(it.entries) {
		return nil
	}
	return it
}

func (it *priorityIterator) Tx() sdk.Tx {
	if it.idx >= len(it.entries) || it.idx < 0 {
		return nil
	}
	return it.entries[it.idx].tx
}

func compareEntries(a, b any) int {
	left := a.(*txEntry)
	right := b.(*txEntry)

	if left.tier != right.tier {
		if left.tier < right.tier {
			return -1
		}
		return 1
	}

	if left.priority != right.priority {
		if left.priority > right.priority {
			return -1
		}
		return 1
	}

	if left.order != right.order {
		if left.order > right.order {
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

func (p *PriorityMempool) nextOrder() int64 {
	p.orderSeq++
	return p.orderSeq
}

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

func (p *PriorityMempool) tierName(idx int) string {
	if idx < 0 || idx >= len(p.tiers) {
		return ""
	}
	return p.tiers[idx].Name
}

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

func (p *PriorityMempool) expectedNextSequenceLocked(ctx sdk.Context, sender string) (uint64, bool, error) {
	bucket, ok := p.userBuckets[sender]
	if !ok {
		return 0, false, nil
	}
	next, ok := bucket.nextSequence()
	return next, ok, nil
}

// NextExpectedSequence returns the next expected sequence for a sender.
func (p *PriorityMempool) NextExpectedSequence(ctx sdk.Context, sender string) (uint64, bool, error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	return p.expectedNextSequenceLocked(ctx, sender)
}

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

func (p *PriorityMempool) isBetterThan(entry *txEntry, tier int, priority int64) bool {
	if tier != entry.tier {
		return tier < entry.tier
	}
	return priority > entry.priority
}

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

func (p *PriorityMempool) dispatchInserted(listeners []TxEventListener, tx sdk.Tx) {
	if len(listeners) == 0 {
		return
	}

	for _, l := range listeners {
		l.OnTxInserted(tx)
	}
}

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

func (p *PriorityMempool) txBytesAndSize(tx sdk.Tx) ([]byte, int64, error) {
	bz, err := p.txEncoder(tx)
	if err != nil {
		return nil, 0, err
	}
	return bz, int64(len(bz)), nil
}
