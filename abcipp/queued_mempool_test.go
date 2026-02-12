package abcipp

import (
	"context"
	"testing"

	cmtmempool "github.com/cometbft/cometbft/mempool"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

// drainEvents reads all buffered events from the channel and returns counts.
func drainEvents(ch <-chan cmtmempool.AppMempoolEvent) (inserted, removed int) {
	for {
		select {
		case ev := <-ch:
			switch ev.Type {
			case cmtmempool.EventTxInserted:
				inserted++
			case cmtmempool.EventTxRemoved:
				removed++
			}
		default:
			return
		}
	}
}

func newTestQueuedMempool(t *testing.T, keeper *mockAccountKeeper) (*QueuedMempool, *mockAccountKeeper) {
	t.Helper()
	if keeper == nil {
		keeper = newMockAccountKeeper()
	}
	txpool := NewPriorityMempool(PriorityMempoolConfig{
		MaxTx: 100,
	}, testTxEncoder)
	qm := NewQueuedMempool(txpool, testTxEncoder)
	qm.SetAccountKeeper(keeper)
	return qm, keeper
}

func TestQueuedMempoolQueuesOutOfOrderSequences(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 5)

	pm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 5, active
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")); err != nil {
		t.Fatalf("insert seq 5: %v", err)
	}

	// Insert seq 7, future nonce, should be queued
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 7, 1000, "default")); err != nil {
		t.Fatalf("expected future nonce to be queued, got: %v", err)
	}

	if pm.CountTx() != 2 {
		t.Fatalf("expected 2 entries (1 active + 1 queued), got %d", pm.CountTx())
	}

	// Seq 7 should NOT appear in Select (only active entries)
	count := 0
	for it := pm.Select(context.Background(), nil); it != nil; it = it.Next() {
		count++
	}
	if count != 1 {
		t.Fatalf("expected 1 active entry in Select, got %d", count)
	}
}

func TestQueuedMempoolAutoPromotesOnGapFill(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 5)

	pm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 5 (active), seq 7 (queued)
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")); err != nil {
		t.Fatalf("insert seq 5: %v", err)
	}
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 7, 1000, "default")); err != nil {
		t.Fatalf("insert seq 7: %v", err)
	}

	// Insert seq 6, fills the gap, should auto-promote seq 7
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 6, 1000, "default")); err != nil {
		t.Fatalf("insert seq 6: %v", err)
	}

	// All 3 should now be active
	count := 0
	for it := pm.Select(context.Background(), nil); it != nil; it = it.Next() {
		count++
	}
	if count != 3 {
		t.Fatalf("expected 3 active entries after auto-promotion, got %d", count)
	}
}

func TestQueuedMempoolRejectsStaleNonces(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 5)

	pm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Stale nonce 4 should be rejected
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 4, 1000, "default")); err == nil {
		t.Fatalf("expected stale nonce to be rejected")
	}

	// Stale nonce 3 should be rejected
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")); err == nil {
		t.Fatalf("expected stale nonce to be rejected")
	}

	// Matching nonce 5 should work
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")); err != nil {
		t.Fatalf("failed to insert matching nonce: %v", err)
	}
}

func TestQueuedMempoolPromoteQueued(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	pm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active) and seq 2, 3 (queued, gap at 1)
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")); err != nil {
		t.Fatalf("insert seq 0: %v", err)
	}
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")); err != nil {
		t.Fatalf("insert seq 2: %v", err)
	}
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")); err != nil {
		t.Fatalf("insert seq 3: %v", err)
	}

	// Only seq 0 should be active
	count := 0
	for it := pm.Select(context.Background(), nil); it != nil; it = it.Next() {
		count++
	}
	if count != 1 {
		t.Fatalf("expected 1 active entry, got %d", count)
	}

	// Simulate block commit, on-chain sequence advances to 2
	keeper.SetSequence(sender, 2)
	pm.PromoteQueued(sdkCtx)

	// After promotion, seq 0 cleaned by txpool, seq 2 and 3 promoted
	count = 0
	for it := pm.Select(context.Background(), nil); it != nil; it = it.Next() {
		count++
	}
	if count != 3 {
		t.Fatalf("expected 3 active entries after PromoteQueued (seq 0 still in txpool + promoted 2,3), got %d", count)
	}
}

func TestQueuedMempoolSelectDelegatesToTxpool(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	pm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active), seq 2 (queued)
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")); err != nil {
		t.Fatalf("insert seq 0: %v", err)
	}
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")); err != nil {
		t.Fatalf("insert seq 2: %v", err)
	}

	// Select should only return active entries
	count := 0
	for it := pm.Select(context.Background(), nil); it != nil; it = it.Next() {
		count++
	}
	if count != 1 {
		t.Fatalf("expected 1 entry from Select, got %d", count)
	}
}

func TestQueuedMempoolCountIncludesQueued(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	pm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active), seq 2 and 3 (queued)
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")); err != nil {
		t.Fatalf("insert seq 0: %v", err)
	}
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")); err != nil {
		t.Fatalf("insert seq 2: %v", err)
	}
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")); err != nil {
		t.Fatalf("insert seq 3: %v", err)
	}

	if pm.CountTx() != 3 {
		t.Fatalf("expected CountTx=3 (1 active + 2 queued), got %d", pm.CountTx())
	}
}

func TestQueuedMempoolGetTxDistributionIncludesQueued(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	qm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Empty pool, no queued tier
	dist := qm.GetTxDistribution()
	require.Zero(t, dist["queued"])

	// Insert seq 0 (active, "default" tier), seq 2 and 3 (queued)
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")))
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")))

	dist = qm.GetTxDistribution()
	require.Equal(t, uint64(1), dist["default"])
	require.Equal(t, uint64(2), dist["queued"])

	// Fill gap, seq 1 promotes seq 2 and 3 into the active pool
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 1, 1000, "default")))

	dist = qm.GetTxDistribution()
	require.Equal(t, uint64(4), dist["default"])
	require.Zero(t, dist["queued"])
}

func TestQueuedMempoolContainsChecksQueued(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	pm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")

	if err := pm.Insert(ctx, tx0); err != nil {
		t.Fatalf("insert seq 0: %v", err)
	}
	if err := pm.Insert(ctx, tx2); err != nil {
		t.Fatalf("insert seq 2: %v", err)
	}

	if !pm.Contains(tx0) {
		t.Fatalf("expected Contains=true for active tx")
	}
	if !pm.Contains(tx2) {
		t.Fatalf("expected Contains=true for queued tx")
	}
}

func TestQueuedMempoolRemoveFromQueued(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	pm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")

	if err := pm.Insert(ctx, tx0); err != nil {
		t.Fatalf("insert seq 0: %v", err)
	}
	if err := pm.Insert(ctx, tx2); err != nil {
		t.Fatalf("insert seq 2: %v", err)
	}

	// Remove active tx
	if err := pm.Remove(tx0); err != nil {
		t.Fatalf("remove active tx: %v", err)
	}
	if pm.Contains(tx0) {
		t.Fatalf("expected active tx removed")
	}

	// Remove queued tx
	if err := pm.Remove(tx2); err != nil {
		t.Fatalf("remove queued tx: %v", err)
	}
	if pm.Contains(tx2) {
		t.Fatalf("expected queued tx removed")
	}

	if pm.CountTx() != 0 {
		t.Fatalf("expected 0 entries, got %d", pm.CountTx())
	}

	// Remove non existent tx
	tx3 := newTestTxWithPriv(priv, 3, 1000, "default")
	if err := pm.Remove(tx3); err != sdkmempool.ErrTxNotFound {
		t.Fatalf("expected ErrTxNotFound, got %v", err)
	}
}

func TestQueuedMempoolPerSenderLimit(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	pm, _ := newTestQueuedMempool(t, keeper)
	pm.SetMaxQueuedPerSender(3)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active)
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")); err != nil {
		t.Fatalf("insert seq 0: %v", err)
	}

	// Queue seq 5, 6, 7 fills per sender limit of 3
	for _, seq := range []uint64{5, 6, 7} {
		if err := pm.Insert(ctx, newTestTxWithPriv(priv, seq, 1000, "default")); err != nil {
			t.Fatalf("insert seq %d: %v", seq, err)
		}
	}
	if pm.CountTx() != 4 {
		t.Fatalf("expected 4 (1 active + 3 queued), got %d", pm.CountTx())
	}

	// Seq 8 has the  highest nonce, should be silently rejected
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 8, 1000, "default")); err != nil {
		t.Fatalf("insert seq 8 should silently skip, got error: %v", err)
	}
	if pm.CountTx() != 4 {
		t.Fatalf("expected still 4 after rejected insert, got %d", pm.CountTx())
	}

	// Seq 3 has lower nonce than the highest (7) — should evict seq 7 and insert
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")); err != nil {
		t.Fatalf("insert seq 3: %v", err)
	}
	if pm.CountTx() != 4 {
		t.Fatalf("expected 4 after eviction+insert, got %d", pm.CountTx())
	}

	// Verify seq 7 evicted, seq 3 present
	if _, ok := pm.Lookup(sender.String(), 7); ok {
		t.Fatalf("expected seq 7 to be evicted")
	}
	if _, ok := pm.Lookup(sender.String(), 3); !ok {
		t.Fatalf("expected seq 3 to be present")
	}
}

func TestQueuedMempoolGlobalLimit(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	pm, _ := newTestQueuedMempool(t, keeper)
	pm.SetMaxQueuedTotal(3)

	// Use different senders to avoid a per sender limit
	var privs [4]*secp256k1.PrivKey
	for i := range privs {
		privs[i] = secp256k1.GenPrivKey()
		sender := sdk.AccAddress(privs[i].PubKey().Address())
		keeper.SetSequence(sender, 0)

		// Insert seq 0 (active) for each sender
		if err := pm.Insert(ctx, newTestTxWithPriv(privs[i], 0, 1000, "default")); err != nil {
			t.Fatalf("insert active for sender %d: %v", i, err)
		}
	}

	// Queue future-nonce txs from 3 senders, fills global limit
	for i := 0; i < 3; i++ {
		if err := pm.Insert(ctx, newTestTxWithPriv(privs[i], 2, 1000, "default")); err != nil {
			t.Fatalf("queue for sender %d: %v", i, err)
		}
	}

	// 4th sender queued tx should be silently rejected (global limit)
	if err := pm.Insert(ctx, newTestTxWithPriv(privs[3], 2, 1000, "default")); err != nil {
		t.Fatalf("insert should silently skip, got error: %v", err)
	}

	// 4 active + 3 queued = 7
	if pm.CountTx() != 7 {
		t.Fatalf("expected 7 total (4 active + 3 queued), got %d", pm.CountTx())
	}
}

func TestQueuedMempoolStalenessRecoveryAfterCleanup(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 5)

	pm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 5 active, activeNext becomes 6
	tx5 := newTestTxWithPriv(priv, 5, 1000, "default")
	if err := pm.Insert(ctx, tx5); err != nil {
		t.Fatalf("insert seq 5: %v", err)
	}

	// Simulate a cleaning worker removing the active tx (ante failure)
	if err := pm.txpool.Remove(tx5); err != nil {
		t.Fatalf("remove from txpool: %v", err)
	}
	pm.PromoteQueued(sdkCtx)

	// Now the sender should be able to resubmit nonce 5
	if err := pm.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")); err != nil {
		t.Fatalf("expected re-submission of nonce 5 to succeed after PromoteQueued, got: %v", err)
	}
}

func TestQueuedMempoolGetTxInfo(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	qm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx2 := newTestTxWithPriv(priv, 2, 2000, "default")

	require.NoError(t, qm.Insert(ctx, tx0))
	require.NoError(t, qm.Insert(ctx, tx2))

	// Active tx info, delegates to PriorityMempool
	info, err := qm.GetTxInfo(sdkCtx, tx0)
	require.NoError(t, err)
	require.Equal(t, sender.String(), info.Sender)
	require.Equal(t, uint64(0), info.Sequence)
	require.True(t, info.Size > 0)
	require.Equal(t, uint64(1000), info.GasLimit)
	require.NotEmpty(t, info.Tier)
	require.NotEqual(t, "queued", info.Tier)
	require.NotEmpty(t, info.TxBytes)

	// Queued tx info, returned by QueuedMempool directly
	info, err = qm.GetTxInfo(sdkCtx, tx2)
	require.NoError(t, err)
	require.Equal(t, sender.String(), info.Sender)
	require.Equal(t, uint64(2), info.Sequence)
	require.True(t, info.Size > 0)
	require.Equal(t, uint64(2000), info.GasLimit)
	require.Equal(t, "queued", info.Tier)
	require.NotEmpty(t, info.TxBytes)

	// Non existent tx
	tx9 := newTestTxWithPriv(priv, 9, 1000, "default")
	_, err = qm.GetTxInfo(sdkCtx, tx9)
	require.ErrorIs(t, err, sdkmempool.ErrTxNotFound)
}

func TestQueuedMempoolNextExpectedSequence(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	qm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Unknown sender falls back to txpool (which returns false)
	_, ok, err := qm.NextExpectedSequence(sdkCtx, sender.String())
	require.NoError(t, err)
	require.False(t, ok)

	// Insert seq 0 and activeNext becomes 1
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))

	next, ok, err := qm.NextExpectedSequence(sdkCtx, sender.String())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, uint64(1), next)

	// Insert seq 1 and activeNext becomes 2
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 1, 1000, "default")))

	next, ok, err = qm.NextExpectedSequence(sdkCtx, sender.String())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, uint64(2), next)
}

func TestQueuedMempoolSameNonceReplacement(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	qm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()

	// Insert seq 0 (active)
	ctx := sdk.WrapSDKContext(sdkCtx.WithPriority(100))
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))

	// Queue seq 5 with priority 100
	ctx = sdk.WrapSDKContext(sdkCtx.WithPriority(100))
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")))
	require.Equal(t, 2, qm.CountTx())

	// Replace seq 5 with higher priority 200 should succeed
	ctx = sdk.WrapSDKContext(sdkCtx.WithPriority(200))
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 5, 2000, "default")))
	require.Equal(t, 2, qm.CountTx()) // count unchanged (replacement)

	// Try replacing seq 5 with lower priority 50 should be silently rejected
	ctx = sdk.WrapSDKContext(sdkCtx.WithPriority(50))
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 5, 500, "default")))
	require.Equal(t, 2, qm.CountTx())
}

func TestQueuedMempoolEventDispatch(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	qm, _ := newTestQueuedMempool(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	qm.SetEventCh(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active), EventTxInserted
	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	require.NoError(t, qm.Insert(ctx, tx0))
	inserted, removed := drainEvents(eventCh)
	require.Equal(t, 1, inserted, "active insert should fire EventTxInserted")
	require.Equal(t, 0, removed)

	// Insert seq 2 (queued), no event (CometBFT fires EventTxQueued from CheckTx)
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")
	require.NoError(t, qm.Insert(ctx, tx2))
	inserted, removed = drainEvents(eventCh)
	require.Equal(t, 0, inserted, "queued insert should not fire EventTxInserted")
	require.Equal(t, 0, removed)

	// Remove queued tx, EventTxRemoved
	require.NoError(t, qm.Remove(tx2))
	inserted, removed = drainEvents(eventCh)
	require.Equal(t, 0, inserted)
	require.Equal(t, 1, removed, "queued remove should fire EventTxRemoved")

	// Reinsert seq 2 (queued) and then fill the gap with seq 1
	tx2b := newTestTxWithPriv(priv, 2, 1000, "default")
	require.NoError(t, qm.Insert(ctx, tx2b))
	drainEvents(eventCh)

	tx1 := newTestTxWithPriv(priv, 1, 1000, "default")
	require.NoError(t, qm.Insert(ctx, tx1))
	inserted, removed = drainEvents(eventCh)
	// seq 1 inserted (active) + seq 2 promoted (via bridge -> forwardInserted -> pushEvent)
	require.Equal(t, 2, inserted, "gap-fill should fire EventTxInserted for seq 1 + promoted seq 2")
	require.Equal(t, 0, removed)
}

func TestQueuedMempoolPromoteQueuedActiveOnlySenders(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	privA := secp256k1.GenPrivKey()
	senderA := sdk.AccAddress(privA.PubKey().Address())
	keeper.SetSequence(senderA, 0)

	privB := secp256k1.GenPrivKey()
	senderB := sdk.AccAddress(privB.PubKey().Address())
	keeper.SetSequence(senderB, 0)

	qm, _ := newTestQueuedMempool(t, keeper)

	// Sender A active only (seq 0), no queued entries
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(privA, 0, 1000, "default")))

	// Sender B active (seq 0) + queued (seq 2)
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(privB, 0, 1000, "default")))
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(privB, 2, 1000, "default")))

	// PromoteQueued. B on-chain seq advances, promoting seq 2
	keeper.SetSequence(senderB, 2)
	qm.PromoteQueued(sdkCtx)

	// B seq 2 should now be promoted
	activeCount := 0
	for it := qm.Select(context.Background(), nil); it != nil; it = it.Next() {
		activeCount++
	}
	require.Equal(t, 3, activeCount, "A(seq0) + B(seq0) + B(seq2 promoted)")

	// activeNext should be refreshed from the pool state
	next, ok, _ := qm.NextExpectedSequence(sdkCtx, senderA.String())
	require.True(t, ok)
	require.Equal(t, uint64(1), next)

	// now we remove A active tx from txpool (simulate cleaning worker)
	txA0 := newTestTxWithPriv(privA, 0, 1000, "default")
	require.NoError(t, qm.txpool.Remove(txA0))

	// PromoteQueued. A has no pool entries and no queued -> should be cleaned up
	qm.PromoteQueued(sdkCtx)

	_, ok, _ = qm.NextExpectedSequence(sdkCtx, senderA.String())
	require.False(t, ok, "A should be cleaned from activeNext after pool cleanup")

	// A can now reinsert (fresh lookup from store)
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(privA, 0, 1000, "default")))
}

func TestQueuedMempoolPromoteQueuedEvictsStale(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	qm, _ := newTestQueuedMempool(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	qm.SetEventCh(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active), seq 2, 3, 5 (queued)
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")))
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")))
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")))
	require.Equal(t, 4, qm.CountTx())

	// simulate block commit, on-chain advances past seq 2 and 3
	keeper.SetSequence(sender, 4)
	drainEvents(eventCh) // clear events from inserts
	qm.PromoteQueued(sdkCtx)

	// Seq 2, 3 should be evicted (stale), seq 5 remains queued (gap at 4)
	_, removed := drainEvents(eventCh)
	require.Equal(t, 2, removed, "stale seq 2 and 3 should fire EventTxRemoved")

	// seq 0 still in active pool + seq 5 still queued = 2
	require.Equal(t, 2, qm.CountTx())

	// Verify seq 5 is still queued
	_, ok := qm.Lookup(sender.String(), 5)
	require.True(t, ok, "seq 5 should still be queued")
}

func TestQueuedMempoolLookupQueued(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	qm, _ := newTestQueuedMempool(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active) and seq 3 (queued)
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	require.NoError(t, qm.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")))

	// Lookup active tx
	hash, ok := qm.Lookup(sender.String(), 0)
	require.True(t, ok)
	require.NotEmpty(t, hash)

	// Lookup queued tx
	hash, ok = qm.Lookup(sender.String(), 3)
	require.True(t, ok)
	require.NotEmpty(t, hash)

	// Lookup non-existent
	_, ok = qm.Lookup(sender.String(), 99)
	require.False(t, ok)
}
