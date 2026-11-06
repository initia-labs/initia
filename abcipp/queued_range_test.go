package abcipp

import (
	"math"
	"testing"
	"time"
)

// A sender can hold queued nonces that are arbitrarily far apart (CheckTx accepts any
// future sequence with the queued extension option). Boundary recovery must not walk
// the nonce space between them.
func TestQueuedRangeRecoveryIsBoundedBySetSize(t *testing.T) {
	run := func(name string, removed uint64, remaining uint64) {
		s := &senderState{queued: map[uint64]*txEntry{}}
		s.queued[removed] = &txEntry{}
		s.queued[remaining] = &txEntry{}
		s.queuedMin = min(removed, remaining)
		s.queuedMax = max(removed, remaining)
		delete(s.queued, removed)

		done := make(chan struct{})
		go func() { s.setQueuedRangeOnRemoveLocked(removed); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatalf("%s: boundary recovery did not return within 3s (scanning nonce space)", name)
		}
		if s.queuedMin != remaining || s.queuedMax != remaining {
			t.Fatalf("%s: got range [%d,%d], want [%d,%d]", name, s.queuedMin, s.queuedMax, remaining, remaining)
		}
	}
	run("remove min, next nonce at MaxUint64", 5, math.MaxUint64)
	run("remove max, previous nonce at 5", math.MaxUint64, 5)
}
