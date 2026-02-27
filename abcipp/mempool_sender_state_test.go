package abcipp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSenderStateNextExpectedNonce(t *testing.T) {
	ss := &senderState{
		onChainSeq: 3,
		active:     make(map[uint64]*txEntry),
		queued:     make(map[uint64]*txEntry),
	}

	// Empty active uses on-chain sequence.
	require.Equal(t, uint64(3), ss.nextExpectedNonce())

	// Active tail takes precedence when it is ahead of on-chain.
	ss.active[3] = &txEntry{sequence: 3}
	ss.active[4] = &txEntry{sequence: 4}
	ss.activeMin, ss.activeMax = 3, 4
	require.Equal(t, uint64(5), ss.nextExpectedNonce())

	// On-chain can still be ahead of active tail.
	ss.setOnChainSeqLocked(8)
	require.Equal(t, uint64(8), ss.nextExpectedNonce())
}

func TestSenderStateActiveRangeUpdates(t *testing.T) {
	ss := &senderState{
		active: make(map[uint64]*txEntry),
		queued: make(map[uint64]*txEntry),
	}

	ss.active[5] = &txEntry{sequence: 5}
	ss.setActiveRangeOnInsertLocked(5)
	require.Equal(t, uint64(5), ss.activeMin)
	require.Equal(t, uint64(5), ss.activeMax)

	ss.active[3] = &txEntry{sequence: 3}
	ss.setActiveRangeOnInsertLocked(3)
	require.Equal(t, uint64(3), ss.activeMin)
	require.Equal(t, uint64(5), ss.activeMax)

	ss.active[7] = &txEntry{sequence: 7}
	ss.setActiveRangeOnInsertLocked(7)
	require.Equal(t, uint64(3), ss.activeMin)
	require.Equal(t, uint64(7), ss.activeMax)

	// Removing min should move to the next existing nonce.
	delete(ss.active, 3)
	ss.setActiveRangeOnRemoveLocked(3)
	require.Equal(t, uint64(5), ss.activeMin)
	require.Equal(t, uint64(7), ss.activeMax)

	// Removing max should move down to previous existing nonce.
	delete(ss.active, 7)
	ss.setActiveRangeOnRemoveLocked(7)
	require.Equal(t, uint64(5), ss.activeMin)
	require.Equal(t, uint64(5), ss.activeMax)

	// Removing last should reset both bounds.
	delete(ss.active, 5)
	ss.setActiveRangeOnRemoveLocked(5)
	require.Equal(t, uint64(0), ss.activeMin)
	require.Equal(t, uint64(0), ss.activeMax)
}

func TestSenderStateQueuedRangeUpdates(t *testing.T) {
	ss := &senderState{
		active: make(map[uint64]*txEntry),
		queued: make(map[uint64]*txEntry),
	}

	ss.queued[10] = &txEntry{sequence: 10}
	ss.setQueuedRangeOnInsertLocked(10)
	require.Equal(t, uint64(10), ss.queuedMin)
	require.Equal(t, uint64(10), ss.queuedMax)

	ss.queued[8] = &txEntry{sequence: 8}
	ss.setQueuedRangeOnInsertLocked(8)
	ss.queued[12] = &txEntry{sequence: 12}
	ss.setQueuedRangeOnInsertLocked(12)
	require.Equal(t, uint64(8), ss.queuedMin)
	require.Equal(t, uint64(12), ss.queuedMax)

	// Removing min should skip holes and find the next existing value.
	delete(ss.queued, 8)
	ss.setQueuedRangeOnRemoveLocked(8)
	require.Equal(t, uint64(10), ss.queuedMin)
	require.Equal(t, uint64(12), ss.queuedMax)

	// Removing max should move down to previous existing value.
	delete(ss.queued, 12)
	ss.setQueuedRangeOnRemoveLocked(12)
	require.Equal(t, uint64(10), ss.queuedMin)
	require.Equal(t, uint64(10), ss.queuedMax)

	// Removing last should reset bounds.
	delete(ss.queued, 10)
	ss.setQueuedRangeOnRemoveLocked(10)
	require.Equal(t, uint64(0), ss.queuedMin)
	require.Equal(t, uint64(0), ss.queuedMax)
}
