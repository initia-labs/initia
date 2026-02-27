package abcipp

import (
	"testing"
	"time"

	cmtmempool "github.com/cometbft/cometbft/mempool"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
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

func TestCleanUpEntriesAnteFailureRemovesFailedSuffix(t *testing.T) {
	keeper := newMockAccountKeeper()
	ante := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		tt := tx.(*testTx)
		if tt.sequence >= 1 {
			return ctx, sdkmempool.ErrTxNotFound
		}
		return ctx, nil
	}
	mp := NewPriorityMempool(PriorityMempoolConfig{
		MaxTx:       32,
		AnteHandler: ante,
	}, testTxEncoder, keeper)
	sdkCtx := testSDKContext()
	eventCh := make(chan cmtmempool.AppMempoolEvent, 128)
	mp.SetEventCh(eventCh)
	t.Cleanup(func() {
		assertInvariant(t, mp)
	})
	ctx := sdk.WrapSDKContext(sdkCtx)
	baseApp := testBaseApp{ctx: sdkCtx}

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx1 := newTestTxWithPriv(priv, 1, 1000, "default")
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx0))
	require.NoError(t, mp.Insert(ctx, tx1))
	require.NoError(t, mp.Insert(ctx, tx2))
	drainEvents(eventCh)

	// Ante fails at nonce 1, cleanup must remove nonce 1 and all following entries.
	mp.cleanUpEntries(baseApp, keeper)

	_, rem := drainEvents(eventCh)
	require.Equal(t, 2, rem)
	require.True(t, mp.Contains(tx0))
	require.False(t, mp.Contains(tx1))
	require.False(t, mp.Contains(tx2))
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
	mp.StartCleaningWorker(baseApp, keeper, 5*time.Millisecond)

	require.Eventually(t, func() bool {
		return mp.CountTx() == 0
	}, time.Second, 10*time.Millisecond)

	mp.StopCleaningWorker()
	require.NotPanics(t, func() {
		mp.StopCleaningWorker() // idempotent stop
	})
}

func TestStartCleaningWorkerIsNoopWhenAlreadyRunning(t *testing.T) {
	mp, keeper, sdkCtx, _ := newTestMempoolWithEvents(t, 32)
	baseApp := testBaseApp{ctx: sdkCtx}

	// First start with default interval path.
	mp.StartCleaningWorker(baseApp, keeper, 0)

	mp.mtx.RLock()
	firstStopCh := mp.cleaningStopCh
	firstDoneCh := mp.cleaningDoneCh
	mp.mtx.RUnlock()
	require.NotNil(t, firstStopCh)
	require.NotNil(t, firstDoneCh)

	// Second start must be a no-op while worker is already running.
	mp.StartCleaningWorker(baseApp, keeper, time.Millisecond)

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
