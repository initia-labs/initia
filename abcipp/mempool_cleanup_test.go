package abcipp

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	cmtmempool "github.com/cometbft/cometbft/mempool"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestCleanUpEntriesRemovesStaleActive(t *testing.T) {
	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, 32)
	ctx := sdk.WrapSDKContext(sdkCtx)
	baseApp := testBaseApp{ctx: sdkCtx}

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx1 := newTestTxWithPriv(priv, 1, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx0))
	require.NoError(t, mp.Insert(ctx, tx1))
	drainEvents(eventCh)

	// Chain advanced to nonce 2, both active txs are stale.
	keeper.SetSequence(sender, 2)
	mp.cleanUpEntries(baseApp, keeper)

	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, 0)
	require.Len(t, rem, 2)
	require.Equal(t, 0, mp.CountTx())
}

func TestCleanUpEntriesRemovesStaleQueuedAndUpdatesOnChainSeq(t *testing.T) {
	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, 32)
	ctx := sdk.WrapSDKContext(sdkCtx)
	baseApp := testBaseApp{ctx: sdkCtx}

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx0))
	require.NoError(t, mp.Insert(ctx, tx2))
	drainEvents(eventCh)

	// Chain advanced to nonce 1: active nonce 0 is stale, queued nonce 2 remains.
	keeper.SetSequence(sender, 1)
	mp.cleanUpEntries(baseApp, keeper)

	_, rem := drainEvents(eventCh)
	require.Equal(t, 1, rem)
	require.False(t, mp.Contains(tx0))
	require.True(t, mp.Contains(tx2))

	mp.mtx.RLock()
	ss := mp.senders[sender.String()]
	require.NotNil(t, ss)
	require.Equal(t, uint64(1), ss.onChainSeq)
	mp.mtx.RUnlock()

	// Chain advanced to nonce 3: remaining queued tx is stale and sender should be cleaned up.
	keeper.SetSequence(sender, 3)
	mp.cleanUpEntries(baseApp, keeper)

	_, rem = drainEvents(eventCh)
	require.Equal(t, 1, rem)
	require.Equal(t, 0, mp.CountTx())

	mp.mtx.RLock()
	_, exists := mp.senders[sender.String()]
	mp.mtx.RUnlock()
	require.False(t, exists)
}

func TestCleaningWorkerCleansStaleAndStops(t *testing.T) {
	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, 32)
	ctx := sdk.WrapSDKContext(sdkCtx)
	baseApp := testBaseApp{ctx: sdkCtx}

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx1 := newTestTxWithPriv(priv, 1, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx0))
	require.NoError(t, mp.Insert(ctx, tx1))
	drainEvents(eventCh)

	// Once chain sequence advances, periodic cleanup should remove both stale txs.
	keeper.SetSequence(sender, 2)
	mp.StartCleaningWorker(baseApp, 5*time.Millisecond)

	require.Eventually(t, func() bool {
		return mp.CountTx() == 0
	}, time.Second, 10*time.Millisecond)

	mp.StopCleaningWorker()
	require.NotPanics(t, func() {
		mp.StopCleaningWorker() // idempotent stop
	})
}

func TestStartCleaningWorkerIsNoopWhenAlreadyRunning(t *testing.T) {
	mp, _, sdkCtx, _ := newTestMempoolWithEvents(t, 32)
	baseApp := testBaseApp{ctx: sdkCtx}

	// First start with default interval path.
	mp.StartCleaningWorker(baseApp, 0)

	mp.mtx.RLock()
	firstStopCh := mp.cleaningStopCh
	firstDoneCh := mp.cleaningDoneCh
	mp.mtx.RUnlock()
	require.NotNil(t, firstStopCh)
	require.NotNil(t, firstDoneCh)

	// Second start must be a no-op while worker is already running.
	mp.StartCleaningWorker(baseApp, time.Millisecond)

	mp.mtx.RLock()
	secondStopCh := mp.cleaningStopCh
	secondDoneCh := mp.cleaningDoneCh
	mp.mtx.RUnlock()
	require.Equal(t, firstStopCh, secondStopCh)
	require.Equal(t, firstDoneCh, secondDoneCh)

	mp.StopCleaningWorker()

	mp.mtx.RLock()
	require.Nil(t, mp.cleaningStopCh)
	require.Nil(t, mp.cleaningDoneCh)
	mp.mtx.RUnlock()
}

func TestCleanUpEntriesAnteCleanupRemovesRejectedHead(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()
	baseApp := testBaseApp{ctx: sdkCtx}

	rejectPriv := secp256k1.GenPrivKey()
	keepPriv := secp256k1.GenPrivKey()
	rejectSender := sdk.AccAddress(rejectPriv.PubKey().Address())
	keepSender := sdk.AccAddress(keepPriv.PubKey().Address())

	anteCalls := 0
	mp := NewPriorityMempool(
		PriorityMempoolConfig{
			MaxTx: 32,
			AnteHandler: func(ctx sdk.Context, tx sdk.Tx, _ bool) (sdk.Context, error) {
				anteCalls++
				require.True(t, ctx.IsReCheckTx())
				require.NotEmpty(t, ctx.TxBytes())

				tt, ok := tx.(*testTx)
				require.True(t, ok)
				if tt.sender.Equals(rejectSender) {
					return ctx, fmt.Errorf("ante reject")
				}

				return ctx, nil
			},
		},
		log.NewNopLogger(),
		testTxEncoder,
		keeper,
	)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 256)
	mp.SetEventCh(eventCh)
	t.Cleanup(func() {
		assertInvariant(t, mp)
	})

	keeper.SetSequence(rejectSender, 0)
	keeper.SetSequence(keepSender, 0)

	ctx := sdk.WrapSDKContext(sdkCtx)
	rejectTx := newTestTxWithPriv(rejectPriv, 0, 1000, "default")
	rejectTx1 := newTestTxWithPriv(rejectPriv, 1, 1000, "default")
	rejectTx2 := newTestTxWithPriv(rejectPriv, 2, 1000, "default")
	keepTx := newTestTxWithPriv(keepPriv, 0, 1000, "default")
	require.NoError(t, mp.Insert(ctx, rejectTx))
	require.NoError(t, mp.Insert(ctx, rejectTx1))
	require.NoError(t, mp.Insert(ctx, rejectTx2))
	require.NoError(t, mp.Insert(ctx, keepTx))
	drainEvents(eventCh)

	mp.cleanUpEntries(baseApp, keeper)

	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, 0)
	require.Len(t, rem, 1)
	require.Equal(t, encodeTx(t, rejectTx), rem[0])
	require.Equal(t, 2, anteCalls, "ante cleanup should evaluate one active head per sender")
	require.False(t, mp.Contains(rejectTx))
	require.True(t, mp.Contains(rejectTx1))
	require.True(t, mp.Contains(rejectTx2))
	require.True(t, mp.Contains(keepTx))

	// Rejected head removal should demote sender suffix (nonce 1,2) to queued.
	require.Equal(t, 1, activeCount(mp), "only keep sender head should remain active")
	queuedRejectNonces := make(map[uint64]bool)
	mp.IterateQueuedTxs(func(sender string, nonce uint64, _ sdk.Tx) bool {
		if sender == rejectSender.String() {
			queuedRejectNonces[nonce] = true
		}
		return true
	})
	require.Equal(t, map[uint64]bool{1: true, 2: true}, queuedRejectNonces, "rejected sender suffix should be queued")
}

func TestCleanUpEntriesExpiresQueuedGapAfterTTL(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()
	baseApp := testBaseApp{ctx: sdkCtx}
	mp := NewPriorityMempool(
		PriorityMempoolConfig{
			MaxTx:        32,
			QueuedGapTTL: 10 * time.Millisecond,
			AnteHandler: func(ctx sdk.Context, tx sdk.Tx, _ bool) (sdk.Context, error) {
				return ctx, nil
			},
		},
		log.NewNopLogger(),
		testTxEncoder,
		keeper,
	)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 256)
	mp.SetEventCh(eventCh)
	t.Cleanup(func() {
		assertInvariant(t, mp)
	})

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx2))
	drainEvents(eventCh)

	// First cleanup marks sender as stalled but does not expire immediately.
	mp.cleanUpEntries(baseApp, keeper)
	_, rem := drainEvents(eventCh)
	require.Equal(t, 0, rem)
	require.True(t, mp.Contains(tx2))

	time.Sleep(15 * time.Millisecond)
	mp.cleanUpEntries(baseApp, keeper)
	_, rem = drainEvents(eventCh)
	require.Equal(t, 1, rem)
	require.False(t, mp.Contains(tx2))
	require.Equal(t, 0, mp.CountTx())
}

func TestCleanUpEntriesDoesNotExpireGapWhenSenderHasActive(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()
	baseApp := testBaseApp{ctx: sdkCtx}
	mp := NewPriorityMempool(
		PriorityMempoolConfig{
			MaxTx:        32,
			QueuedGapTTL: 10 * time.Millisecond,
			AnteHandler: func(ctx sdk.Context, tx sdk.Tx, _ bool) (sdk.Context, error) {
				return ctx, nil
			},
		},
		log.NewNopLogger(),
		testTxEncoder,
		keeper,
	)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 256)
	mp.SetEventCh(eventCh)
	t.Cleanup(func() {
		assertInvariant(t, mp)
	})

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx2))
	drainEvents(eventCh)

	// Mark as stalled, then make sender active before ttl elapses.
	mp.cleanUpEntries(baseApp, keeper)
	_, rem := drainEvents(eventCh)
	require.Equal(t, 0, rem)
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	drainEvents(eventCh)

	time.Sleep(15 * time.Millisecond)
	mp.cleanUpEntries(baseApp, keeper)
	_, rem = drainEvents(eventCh)
	require.Equal(t, 0, rem)
	require.True(t, mp.Contains(tx2), "queued tx should remain while sender has active txs")
}
