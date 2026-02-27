package abcipp

import (
	"testing"
	"time"

	cmtmempool "github.com/cometbft/cometbft/mempool"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// newTestMempoolWithEvents builds a mempool with event channel wiring so each
// test can focus on state transitions.
func newTestMempoolWithEvents(t *testing.T, maxTx int) (*PriorityMempool, *mockAccountKeeper, sdk.Context, chan cmtmempool.AppMempoolEvent) {
	t.Helper()

	keeper := newMockAccountKeeper()
	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: maxTx}, testTxEncoder, keeper)
	sdkCtx := testSDKContext()
	eventCh := make(chan cmtmempool.AppMempoolEvent, 256)
	mp.SetEventCh(eventCh)
	t.Cleanup(func() {
		assertInvariant(t, mp)
	})

	return mp, keeper, sdkCtx, eventCh
}

// TestGapFillPromotesQueuedChain verifies the main happy-path flow:
// 0 active + 2/3 queued -> insert 1 -> 2/3 promoted.
func TestGapFillPromotesQueuedChain(t *testing.T) {
	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, 100)
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")))
	drainEvents(eventCh)

	// Filling the gap should insert seq1 and promote seq2/3 in one step.
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 1, 1000, "default")))

	ins, rem := collectEvents(eventCh)
	require.Len(t, rem, 0)
	require.Len(t, ins, 3, "seq1 insert + seq2/3 promotions should each emit EventTxInserted")
	require.Equal(t, 4, activeCount(mp))
	require.Equal(t, 4, mp.CountTx())

	next, ok, err := mp.NextExpectedSequence(sender.String())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, uint64(4), next)
}

// TestRemoveCommittedAdvancesCursor checks the block-commit remove path:
// Remove() should advance sender on-chain cursor and allow direct continuation.
func TestRemoveCommittedAdvancesCursor(t *testing.T) {
	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, 100)
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx0))
	require.NoError(t, mp.Insert(ctx, tx2))
	drainEvents(eventCh)

	// Commit-style removal advances onChainSeq to 1.
	require.NoError(t, mp.Remove(tx0))
	_, removedCount := drainEvents(eventCh)
	require.Equal(t, 1, removedCount, "committed tx removal should emit EventTxRemoved")

	next, ok, err := mp.NextExpectedSequence(sender.String())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, uint64(1), next)

	// Inserting seq1 should also promote queued seq2.
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 1, 1000, "default")))
	ins, rem := collectEvents(eventCh)
	require.Len(t, rem, 0)
	require.Len(t, ins, 2, "seq1 insert and seq2 promotion should both emit inserted events")
}

// TestRemoveWithReasonDoesNotAdvanceCursor verifies non-commit remove
// is treated as local rejection and does not imply chain progression.
func TestRemoveWithReasonDoesNotAdvanceCursor(t *testing.T) {
	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, 100)
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 5)

	tx5 := newTestTxWithPriv(priv, 5, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx5))
	drainEvents(eventCh)

	require.NoError(t, mp.RemoveWithReason(tx5, RemovalReasonAnteRejectedInPrepare))
	_, rem := drainEvents(eventCh)
	require.Equal(t, 1, rem, "ante-rejected removal should emit EventTxRemoved")

	// After non-commit cleanup, sender can submit the same on-chain nonce again.
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")))
}

// TestCapacityDemotionHasNoRemovedEvent enforces current policy:
// capacity pressure can demote active txs to queued, but demotion is not removal.
func TestCapacityDemotionHasNoRemovedEvent(t *testing.T) {
	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, 2)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	privC := secp256k1.GenPrivKey()
	keeper.SetSequence(sdk.AccAddress(privA.PubKey().Address()), 0)
	keeper.SetSequence(sdk.AccAddress(privB.PubKey().Address()), 0)
	keeper.SetSequence(sdk.AccAddress(privC.PubKey().Address()), 0)

	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 0, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(5)), newTestTxWithPriv(privB, 0, 1000, "default")))
	drainEvents(eventCh)

	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(20)), newTestTxWithPriv(privC, 0, 1000, "default")))

	ins, rem := collectEvents(eventCh)
	require.Len(t, rem, 0, "demotion keeps tx in mempool, so EventTxRemoved must not fire")
	require.Len(t, ins, 1, "only newly accepted tx should emit EventTxInserted")
	require.Equal(t, 3, mp.CountTx(), "2 active + 1 queued after demotion")
}

// TestPromoteQueuedRemovesStale validates stale cleanup policy in PromoteQueued:
// stale active+queued entries are removed, and gapped future queued entries stay queued.
func TestPromoteQueuedRemovesStale(t *testing.T) {
	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, 100)
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")))
	drainEvents(eventCh)

	// Chain moved to 4: active 0 and queued 2/3 become stale, queued 5 still has gap at 4.
	keeper.SetSequence(sender, 4)
	mp.PromoteQueued(sdkCtx)

	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, 0)
	require.Len(t, rem, 3, "stale active 0 and stale queued 2/3 should be removed")

	_, staleActiveExists := mp.Lookup(sender.String(), 0)
	require.False(t, staleActiveExists, "stale active entry should be removed during PromoteQueued")

	_, exists := mp.Lookup(sender.String(), 5)
	require.True(t, exists, "future queued nonce with a gap should remain queued")
}

// TestQueuedSameNonceReplacement verifies queued same-nonce replacement
// keeps only the highest-priority tx and emits exactly one remove event.
func TestQueuedSameNonceReplacement(t *testing.T) {
	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, 32)
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	txLow := newTestTxWithPriv(priv, 5, 1000, "default")
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), txLow))
	drainEvents(eventCh)

	// Higher-priority same nonce should replace queued tx and emit one remove event.
	txHigh := newTestTxWithPriv(priv, 5, 1000, "default")
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(100)), txHigh))

	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, 0, "queued replacement should not emit inserted event")
	require.Len(t, rem, 1, "queued replacement should emit one removed event for replaced tx")
	require.Equal(t, encodeTx(t, txLow), rem[0])
	require.Equal(t, 2, mp.CountTx(), "one active + one queued should remain")
}

// TestQueuedPerSenderCapEvictsHighestNonce verifies per-sender queued
// cap policy: when full, inserting a lower nonce evicts the current highest nonce.
func TestQueuedPerSenderCapEvictsHighestNonce(t *testing.T) {
	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, 64)
	mp.SetMaxQueuedPerSender(2)
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")))
	tx4 := newTestTxWithPriv(priv, 4, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx4))
	drainEvents(eventCh)

	// Insert nonce=3 while cap full; nonce=4 should be evicted.
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")))
	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, 0)
	require.Len(t, rem, 1, "highest queued nonce should be evicted")
	require.Equal(t, encodeTx(t, tx4), rem[0])

	_, ok2 := mp.Lookup(sender.String(), 2)
	_, ok3 := mp.Lookup(sender.String(), 3)
	_, ok4 := mp.Lookup(sender.String(), 4)
	require.True(t, ok2)
	require.True(t, ok3)
	require.False(t, ok4)
}

// TestQueuedTotalCapSilentlyRejectsFutureNonce verifies global queued
// limit behavior: inserts beyond global queued capacity are skipped without error.
func TestQueuedTotalCapSilentlyRejectsFutureNonce(t *testing.T) {
	keeper := newMockAccountKeeper()
	mp := NewPriorityMempool(PriorityMempoolConfig{
		MaxTx:          64,
		MaxQueuedTotal: 2,
	}, testTxEncoder, keeper)
	sdkCtx := testSDKContext()
	eventCh := make(chan cmtmempool.AppMempoolEvent, 128)
	mp.SetEventCh(eventCh)
	t.Cleanup(func() {
		assertInvariant(t, mp)
	})
	ctx := sdk.WrapSDKContext(sdkCtx)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	privC := secp256k1.GenPrivKey()
	senderA := sdk.AccAddress(privA.PubKey().Address())
	senderB := sdk.AccAddress(privB.PubKey().Address())
	senderC := sdk.AccAddress(privC.PubKey().Address())
	keeper.SetSequence(senderA, 0)
	keeper.SetSequence(senderB, 0)
	keeper.SetSequence(senderC, 0)

	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privA, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privB, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privC, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privA, 2, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privB, 2, 1000, "default")))
	drainEvents(eventCh)

	// This queued insert should be silently skipped because total queued is full.
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privC, 2, 1000, "default")))
	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, 0)
	require.Len(t, rem, 0)
	require.Equal(t, 5, mp.CountTx(), "3 active + 2 queued expected")
}

// TestPromotionCapacityFailureRequeuesChain verifies promotion under capacity
// pressure promotes what fits and requeues the remaining suffix without loss.
func TestPromotionCapacityFailureRequeuesChain(t *testing.T) {
	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, 3)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	privC := secp256k1.GenPrivKey()
	senderA := sdk.AccAddress(privA.PubKey().Address())
	keeper.SetSequence(senderA, 0)
	keeper.SetSequence(sdk.AccAddress(privB.PubKey().Address()), 0)
	keeper.SetSequence(sdk.AccAddress(privC.PubKey().Address()), 0)

	// Active filled by A:0, B:0, C:0. A:2/3/4 are queued.
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(50)), newTestTxWithPriv(privA, 0, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 2, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 3, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 4, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(200)), newTestTxWithPriv(privB, 0, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(200)), newTestTxWithPriv(privC, 0, 1000, "default")))
	drainEvents(eventCh)

	keeper.SetSequence(senderA, 2)
	mp.PromoteQueued(sdkCtx)

	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, 1, "first promotable nonce should be inserted")
	require.Len(t, rem, 1, "stale active nonce should be removed during reconcile")
	require.Equal(t, 5, mp.CountTx(), "stale active removed, one queued promoted, suffix requeued")

	_, ok2 := mp.Lookup(senderA.String(), 2)
	_, ok3 := mp.Lookup(senderA.String(), 3)
	_, ok4 := mp.Lookup(senderA.String(), 4)
	require.True(t, ok2)
	require.True(t, ok3)
	require.True(t, ok4)

	next, ok, err := mp.NextExpectedSequence(senderA.String())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, uint64(3), next, "cursor should advance to first requeued nonce after one promotion")
}

// TestNonCommitRemovalCleansSenderState verifies non-commit removal of
// sender head cleans sender state when no active/queued entries remain.
func TestNonCommitRemovalCleansSenderState(t *testing.T) {
	mp, keeper, sdkCtx, _ := newTestMempoolWithEvents(t, 32)
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx0))
	require.NoError(t, mp.RemoveWithReason(tx0, RemovalReasonAnteRejectedInPrepare))

	_, ok, err := mp.NextExpectedSequence(sender.String())
	require.NoError(t, err)
	require.False(t, ok, "sender state should be cleaned when no entries remain")
}

// TestActiveReplacementEventOrder verifies active same-nonce replacement
// emits events in order: removed old tx first, then inserted new tx.
func TestActiveReplacementEventOrder(t *testing.T) {
	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, 32)

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	txLow := newTestTxWithPriv(priv, 0, 1000, "default")
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), txLow))
	drainEvents(eventCh)

	txHigh := newTestTxWithPriv(priv, 0, 1000, "default")
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(100)), txHigh))

	events := collectNEvents(t, eventCh, 2, 2*time.Second)
	require.Equal(t, cmtmempool.EventTxRemoved, events[0].Type)
	require.Equal(t, cmtmempool.EventTxInserted, events[1].Type)
	require.Equal(t, encodeTx(t, txLow), []byte(events[0].Tx))
	require.Equal(t, encodeTx(t, txHigh), []byte(events[1].Tx))
}

func TestValidateInvariantsDetectsCorruption(t *testing.T) {
	t.Run("queuedCountMismatch", func(t *testing.T) {
		keeper := newMockAccountKeeper()
		mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 32}, testTxEncoder, keeper)
		ctx := sdk.WrapSDKContext(testSDKContext())

		priv := secp256k1.GenPrivKey()
		sender := sdk.AccAddress(priv.PubKey().Address())
		keeper.SetSequence(sender, 0)

		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")))

		mp.mtx.Lock()
		mp.queuedCount.Add(1)
		mp.mtx.Unlock()

		err := mp.ValidateInvariants()
		require.Error(t, err)
		require.ErrorContains(t, err, "queued count mismatch")
	})

	t.Run("activeRangeBoundaryMissing", func(t *testing.T) {
		keeper := newMockAccountKeeper()
		mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 32}, testTxEncoder, keeper)
		ctx := sdk.WrapSDKContext(testSDKContext())

		priv := secp256k1.GenPrivKey()
		sender := sdk.AccAddress(priv.PubKey().Address())
		keeper.SetSequence(sender, 0)

		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 1, 1000, "default")))

		mp.mtx.Lock()
		ss := mp.senders[sender.String()]
		delete(ss.active, 1)
		mp.mtx.Unlock()

		err := mp.ValidateInvariants()
		require.Error(t, err)
		require.ErrorContains(t, err, "activeMax")
	})

	t.Run("globalEntryWithoutSenderState", func(t *testing.T) {
		keeper := newMockAccountKeeper()
		mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 32}, testTxEncoder, keeper)
		ctx := sdk.WrapSDKContext(testSDKContext())

		priv := secp256k1.GenPrivKey()
		sender := sdk.AccAddress(priv.PubKey().Address())
		keeper.SetSequence(sender, 0)
		tx := newTestTxWithPriv(priv, 0, 1000, "default")
		require.NoError(t, mp.Insert(ctx, tx))

		mp.mtx.Lock()
		delete(mp.senders, sender.String())
		mp.mtx.Unlock()

		err := mp.ValidateInvariants()
		require.Error(t, err)
		require.ErrorContains(t, err, "active/global count mismatch")
	})
}
