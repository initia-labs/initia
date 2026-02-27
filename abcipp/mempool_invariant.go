package abcipp

import "fmt"

// ValidateInvariants verifies internal mempool consistency and returns an error
// when any sender/global bookkeeping invariant is broken.
//
// This method is intended for diagnostics and tests.
func (p *PriorityMempool) ValidateInvariants() error {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	totalQueued := 0
	totalActive := 0

	for sender, ss := range p.senders {
		if ss == nil {
			return fmt.Errorf("sender %s has nil sender state", sender)
		}
		if ss.active == nil || ss.queued == nil {
			return fmt.Errorf("sender %s has nil active/queued map", sender)
		}

		activeLen := len(ss.active)
		queuedLen := len(ss.queued)
		totalActive += activeLen
		totalQueued += queuedLen

		if activeLen == 0 {
			if ss.activeMin != 0 || ss.activeMax != 0 {
				return fmt.Errorf("sender %s has empty active but non-zero range [%d,%d]", sender, ss.activeMin, ss.activeMax)
			}
		} else {
			if ss.activeMin > ss.activeMax {
				return fmt.Errorf("sender %s has invalid active range [%d,%d]", sender, ss.activeMin, ss.activeMax)
			}
			if _, ok := ss.active[ss.activeMin]; !ok {
				return fmt.Errorf("sender %s activeMin %d missing from active set", sender, ss.activeMin)
			}
			if _, ok := ss.active[ss.activeMax]; !ok {
				return fmt.Errorf("sender %s activeMax %d missing from active set", sender, ss.activeMax)
			}
			// Active nonces must remain contiguous by design.
			for nonce := ss.activeMin; nonce <= ss.activeMax; nonce++ {
				if _, ok := ss.active[nonce]; !ok {
					return fmt.Errorf("sender %s active range broken: missing nonce %d in [%d,%d]", sender, nonce, ss.activeMin, ss.activeMax)
				}
				if nonce == ss.activeMax {
					break
				}
			}
		}

		if queuedLen == 0 {
			if ss.queuedMin != 0 || ss.queuedMax != 0 {
				return fmt.Errorf("sender %s has empty queued but non-zero range [%d,%d]", sender, ss.queuedMin, ss.queuedMax)
			}
		} else {
			if ss.queuedMin > ss.queuedMax {
				return fmt.Errorf("sender %s has invalid queued range [%d,%d]", sender, ss.queuedMin, ss.queuedMax)
			}
			if _, ok := ss.queued[ss.queuedMin]; !ok {
				return fmt.Errorf("sender %s queuedMin %d missing from queued set", sender, ss.queuedMin)
			}
			if _, ok := ss.queued[ss.queuedMax]; !ok {
				return fmt.Errorf("sender %s queuedMax %d missing from queued set", sender, ss.queuedMax)
			}
		}

		expectedNext := ss.onChainSeq
		if activeLen > 0 {
			expectedNext = max(ss.onChainSeq, ss.activeMax+1)
		}
		if next := ss.nextExpectedNonce(); next != expectedNext {
			return fmt.Errorf("sender %s nextExpected mismatch: got %d, expected %d", sender, next, expectedNext)
		}

		for nonce, entry := range ss.active {
			if entry == nil {
				return fmt.Errorf("sender %s has nil active entry at nonce %d", sender, nonce)
			}
			if entry.key.sender != sender || entry.key.nonce != nonce {
				return fmt.Errorf("sender %s active entry key mismatch at nonce %d", sender, nonce)
			}
			if entry.tier == queuedTier {
				return fmt.Errorf("sender %s active entry at nonce %d marked as queued tier", sender, nonce)
			}
			global, ok := p.entries[entry.key]
			if !ok {
				return fmt.Errorf("sender %s active entry at nonce %d missing from global entries", sender, nonce)
			}
			if global != entry {
				return fmt.Errorf("sender %s active entry pointer mismatch at nonce %d", sender, nonce)
			}
		}

		for nonce, entry := range ss.queued {
			if entry == nil {
				return fmt.Errorf("sender %s has nil queued entry at nonce %d", sender, nonce)
			}
			if entry.key.sender != sender || entry.key.nonce != nonce {
				return fmt.Errorf("sender %s queued entry key mismatch at nonce %d", sender, nonce)
			}
			// queued entries can temporarily carry a non-queued tier when they were
			// previously active and requeued after a failed promotion attempt.
			if _, ok := p.entries[entry.key]; ok {
				return fmt.Errorf("sender %s queued entry at nonce %d exists in global active entries", sender, nonce)
			}
			if nonce < ss.queuedMin || nonce > ss.queuedMax {
				return fmt.Errorf("sender %s queued nonce %d outside queued range [%d,%d]", sender, nonce, ss.queuedMin, ss.queuedMax)
			}
		}
	}

	if totalActive != len(p.entries) {
		return fmt.Errorf("active/global count mismatch: sender active=%d, global entries=%d", totalActive, len(p.entries))
	}

	for key, entry := range p.entries {
		ss := p.senders[key.sender]
		if ss == nil {
			return fmt.Errorf("global entry %s/%d has no sender state", key.sender, key.nonce)
		}
		if ss.active[key.nonce] != entry {
			return fmt.Errorf("global entry %s/%d not found in sender active map", key.sender, key.nonce)
		}
	}

	if int64(totalQueued) != p.queuedCount.Load() {
		return fmt.Errorf("queued count mismatch: sender queued=%d, queuedCount=%d", totalQueued, p.queuedCount.Load())
	}

	return nil
}

// AssertInvariant panics when internal mempool invariants are broken.
// Use this in debug/test paths when immediate fail-fast behavior is preferred.
func (p *PriorityMempool) AssertInvariant() {
	if err := p.ValidateInvariants(); err != nil {
		panic(fmt.Sprintf("mempool invariant violation: %v", err))
	}
}
