package abcipp

// senderState tracks all sender mempool state, active entries in the priority
// index, queued future nonce entries, and the next expected insertion nonce.
type senderState struct {
	onChainSeq uint64

	activeMin uint64
	activeMax uint64
	active    map[uint64]*txEntry

	queuedMin uint64
	queuedMax uint64
	queued    map[uint64]*txEntry
}

func (s *senderState) isEmpty() bool {
	return len(s.active) == 0 && len(s.queued) == 0
}

// nextExpectedNonce returns the sender cursor used for stale checks/promotion.
// When active entries exist, derive from active tail (max+1).
// When active is empty, use cached on-chain sequence.
func (s *senderState) nextExpectedNonce() uint64 {
	if len(s.active) == 0 {
		return s.onChainSeq
	}
	return max(s.onChainSeq, s.activeMax+1)
}

// setOnChainSeqLocked updates cached on-chain sequence for a sender.
func (s *senderState) setOnChainSeqLocked(seq uint64) {
	s.onChainSeq = seq
}

// resetActiveRangeLocked clears the cached active nonce range.
func (s *senderState) resetActiveRangeLocked() {
	s.activeMin = 0
	s.activeMax = 0
}

// setActiveRangeOnInsertLocked updates the active nonce range after insertion.
func (s *senderState) setActiveRangeOnInsertLocked(nonce uint64) {
	if len(s.active) == 1 {
		s.activeMin = nonce
		s.activeMax = nonce
		return
	}
	s.activeMin = min(s.activeMin, nonce)
	s.activeMax = max(s.activeMax, nonce)
}

// setActiveRangeOnRemoveLocked updates the cached range after removing one active nonce.
//
// Design note: we intentionally update bounds only when the removed nonce is
// on a boundary (activeMin/activeMax). The mempool active set is maintained as
// a contiguous prefix from on-chain sequence, so removing a middle nonce is not
// a valid steady-state transition; callers that violate this are detected by
// invariant checks instead of being silently "healed" here.
func (s *senderState) setActiveRangeOnRemoveLocked(removedNonce uint64) {
	if len(s.active) == 0 {
		s.resetActiveRangeLocked()
		return
	}
	if removedNonce == s.activeMin {
		for n := s.activeMin + 1; ; n++ {
			if _, ok := s.active[n]; ok {
				s.activeMin = n
				break
			}
			if n >= s.activeMax {
				s.activeMin = s.activeMax
				break
			}
		}
		return
	}
	if removedNonce == s.activeMax {
		for n := s.activeMax - 1; ; n-- {
			if _, ok := s.active[n]; ok {
				s.activeMax = n
				break
			}
			if n <= s.activeMin {
				s.activeMax = s.activeMin
				break
			}
		}
	}
}

// resetQueuedRangeLocked clears cached queued nonce bounds.
func (s *senderState) resetQueuedRangeLocked() {
	s.queuedMin = 0
	s.queuedMax = 0
}

// setQueuedRangeOnInsertLocked updates queued bounds after insertion.
func (s *senderState) setQueuedRangeOnInsertLocked(nonce uint64) {
	if len(s.queued) == 1 {
		s.queuedMin = nonce
		s.queuedMax = nonce
		return
	}
	s.queuedMin = min(s.queuedMin, nonce)
	s.queuedMax = max(s.queuedMax, nonce)
}

// setQueuedRangeOnRemoveLocked updates queued bounds after nonce removal.
func (s *senderState) setQueuedRangeOnRemoveLocked(removedNonce uint64) {
	if len(s.queued) == 0 {
		s.resetQueuedRangeLocked()
		return
	}
	// Intentional tradeoff: boundary recovery scans between cached min/max and
	// the next existing nonce. Sender-local queued size is capped, so this stays
	// bounded while keeping range maintenance simple and map-only.
	if removedNonce == s.queuedMin {
		for n := s.queuedMin + 1; ; n++ {
			if _, ok := s.queued[n]; ok {
				s.queuedMin = n
				break
			}
			if n >= s.queuedMax {
				s.queuedMin = s.queuedMax
				break
			}
		}
		return
	}
	if removedNonce == s.queuedMax {
		for n := s.queuedMax - 1; ; n-- {
			if _, ok := s.queued[n]; ok {
				s.queuedMax = n
				break
			}
			if n <= s.queuedMin {
				s.queuedMax = s.queuedMin
				break
			}
		}
	}
}
