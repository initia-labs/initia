package abcipp

import (
	"context"
	"maps"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

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

// NextExpectedSequence returns the next expected nonce cursor for a sender.
func (p *PriorityMempool) NextExpectedSequence(sender string) (uint64, bool, error) {
	p.mtx.RLock()
	s := p.senders[sender]
	if s == nil {
		p.mtx.RUnlock()
		return 0, false, nil
	}
	next := s.nextExpectedNonce()
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
