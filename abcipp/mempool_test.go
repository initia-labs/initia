package abcipp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/core/address"
	txsigning "cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/tx/signing/direct"
	cmtmempool "github.com/cometbft/cometbft/mempool"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

func TestPriorityMempoolAllowsOutOfOrder(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()

	// Insert seq 5, accepted
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")); err != nil {
		t.Fatalf("failed to insert initial tx: %v", err)
	}

	// Insert seq 7, also accepted (no AccountKeeper means no nonce routing)
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 7, 1000, "default")); err != nil {
		t.Fatalf("failed to insert out-of-order tx: %v", err)
	}

	if mp.CountTx() != 2 {
		t.Fatalf("expected 2 entries, got %d", mp.CountTx())
	}

	// Both should appear in Select
	count := 0
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		count++
	}
	if count != 2 {
		t.Fatalf("expected 2 entries in Select, got %d", count)
	}
}

func TestPriorityMempoolLifecycle(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	tx := newTestTxWithPriv(priv, 5, 1000, "default")

	if err := mp.Insert(ctx, tx); err != nil {
		t.Fatalf("insert tx: %v", err)
	}

	if !mp.Contains(tx) {
		t.Fatalf("expected tx to be tracked")
	}

	info, err := mp.GetTxInfo(sdkCtx, tx)
	if err != nil {
		t.Fatalf("get tx info: %v", err)
	}
	if info.Sequence != 5 || info.Tier != "default" {
		t.Fatalf("unexpected tx info %+v", info)
	}

	if hash, ok := mp.Lookup(tx.sender.String(), tx.sequence); !ok || hash == "" {
		t.Fatalf("lookup failed")
	}

	if err := mp.Remove(tx); err != nil {
		t.Fatalf("remove tx: %v", err)
	}

	if _, err := mp.GetTxInfo(sdkCtx, tx); err != sdkmempool.ErrTxNotFound {
		t.Fatalf("expected ErrTxNotFound after removal, got %v", err)
	}
}

func TestPriorityMempoolSelectOrdersByTierAndTracksDistribution(t *testing.T) {
	tiers := []Tier{
		testTierMatcher("high"),
		testTierMatcher("low"),
	}
	mp := newTestPriorityMempool(t, tiers)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	lowTx := newTestTx(testAddress(1), 1, 1000, "low")
	highTx := newTestTx(testAddress(2), 1, 1000, "high")

	if err := mp.Insert(ctx, lowTx); err != nil {
		t.Fatalf("insert low priority: %v", err)
	}
	if err := mp.Insert(ctx, highTx); err != nil {
		t.Fatalf("insert high priority: %v", err)
	}

	order := make([]string, 0, 2)
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		tt, ok := it.Tx().(*testTx)
		if !ok {
			t.Fatalf("unexpected tx type %T", it.Tx())
		}
		order = append(order, tt.tier)
	}

	if len(order) != 2 {
		t.Fatalf("expected two entries, got %d", len(order))
	}
	if order[0] != "high" || order[1] != "low" {
		t.Fatalf("expected tier order [high low], got %v", order)
	}

	dist := mp.GetTxDistribution()
	if dist["high"] != 1 || dist["low"] != 1 {
		t.Fatalf("unexpected distribution %v", dist)
	}
}

func TestPriorityMempoolCleanUpEntries(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)
	baseApp := testBaseApp{ctx: sdkCtx}

	priv := secp256k1.GenPrivKey()
	tx1 := newTestTxWithPriv(priv, 1, 1000, "default")
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")

	if err := mp.Insert(ctx, tx1); err != nil {
		t.Fatalf("insert first tx: %v", err)
	}
	if err := mp.Insert(ctx, tx2); err != nil {
		t.Fatalf("insert second tx: %v", err)
	}

	keeper := newMockAccountKeeper()
	keeper.SetSequence(tx1.sender, 3)

	mp.cleanUpEntries(baseApp, keeper)

	if mp.CountTx() != 0 {
		t.Fatalf("expected stale entries removed, still have %d", mp.CountTx())
	}

	if mp.Contains(tx1) || mp.Contains(tx2) {
		t.Fatalf("stale entries should be gone")
	}
}

func TestPriorityMempoolCleanUpAnteErrors(t *testing.T) {
	accountKeeper := newTestAccountKeeper(authcodec.NewBech32Codec("initia"))
	bankKeeper := newTestBankKeeper()
	signModeHandler := txsigning.NewHandlerMap(direct.SignModeHandler{})

	anteHandler, err := authante.NewAnteHandler(authante.HandlerOptions{
		AccountKeeper:   accountKeeper,
		BankKeeper:      bankKeeper,
		SignModeHandler: signModeHandler,
	})
	if err != nil {
		t.Fatalf("failed to build ante handler: %v", err)
	}

	mp := NewPriorityMempool(PriorityMempoolConfig{
		MaxTx:       10,
		AnteHandler: anteHandler,
	}, testTxEncoder)

	sdkCtx := testSDKContext()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	accountKeeper.SetAccount(sdkCtx, authtypes.NewBaseAccountWithAddress(sender))
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx := newTestTxWithPriv(priv, 0, 1000, "default")
	txBytes, err := testTxEncoder(tx)
	if err != nil {
		t.Fatalf("encode tx: %v", err)
	}
	recheckCtx := sdkCtx.WithTxBytes(txBytes).WithIsReCheckTx(true)
	if _, err := anteHandler(recheckCtx, tx, false); err == nil {
		t.Fatalf("expected ante to reject tx without funds")
	}
	if err := mp.Insert(ctx, tx); err != nil {
		t.Fatalf("insert tx: %v", err)
	}

	if mp.CountTx() != 1 {
		t.Fatalf("expected tx to be in mempool before cleanup")
	}

	baseApp := testBaseApp{ctx: sdkCtx}
	mp.cleanUpEntries(baseApp, accountKeeper)

	if mp.CountTx() != 0 {
		t.Fatalf("expected ante failure to remove tx, still have %d", mp.CountTx())
	}

	if mp.Contains(tx) {
		t.Fatalf("expected tx removed after ante failure")
	}
}

func TestPriorityMempoolPreservesInsertionOrder(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx1 := newTestTx(testAddress(1), 1, 1000, "default")
	tx2 := newTestTx(testAddress(2), 1, 1000, "default")

	if err := mp.Insert(ctx, tx1); err != nil {
		t.Fatalf("insert first tx: %v", err)
	}
	if err := mp.Insert(ctx, tx2); err != nil {
		t.Fatalf("insert second tx: %v", err)
	}

	var order []sdk.AccAddress
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		tt, ok := it.Tx().(*testTx)
		if !ok {
			t.Fatalf("unexpected tx type %T", it.Tx())
		}
		order = append(order, tt.sender)
	}

	if len(order) != 2 {
		t.Fatalf("expected two entries, got %d", len(order))
	}
	if !order[0].Equals(tx1.sender) || !order[1].Equals(tx2.sender) {
		t.Fatalf("expected FIFO insertion order, got %v", order)
	}
}

func TestPriorityMempoolOrdersByTierPriorityAndOrder(t *testing.T) {
	tiers := []Tier{
		testTierMatcher("high"),
		testTierMatcher("low"),
	}
	mp := newTestPriorityMempool(t, tiers)
	sdkCtx := testSDKContext()

	ctxPriority5 := sdk.WrapSDKContext(sdkCtx.WithPriority(5))
	ctxPriority10 := sdk.WrapSDKContext(sdkCtx.WithPriority(10))
	ctxPriority100 := sdk.WrapSDKContext(sdkCtx.WithPriority(100))

	highLowPriority1 := newTestTx(testAddress(1), 1, 1000, "high")
	highHighPriority := newTestTx(testAddress(2), 1, 1000, "high")
	highLowPriority2 := newTestTx(testAddress(3), 1, 1000, "high")
	lowHighPriority := newTestTx(testAddress(4), 1, 1000, "low")

	if err := mp.Insert(ctxPriority100, lowHighPriority); err != nil {
		t.Fatalf("insert low tier high priority: %v", err)
	}
	if err := mp.Insert(ctxPriority5, highLowPriority1); err != nil {
		t.Fatalf("insert high tier low priority #1: %v", err)
	}
	if err := mp.Insert(ctxPriority10, highHighPriority); err != nil {
		t.Fatalf("insert high tier high priority: %v", err)
	}
	if err := mp.Insert(ctxPriority5, highLowPriority2); err != nil {
		t.Fatalf("insert high tier low priority #2: %v", err)
	}

	var order []sdk.AccAddress
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		tt, ok := it.Tx().(*testTx)
		if !ok {
			t.Fatalf("unexpected tx type %T", it.Tx())
		}
		order = append(order, tt.sender)
	}

	expected := []sdk.AccAddress{
		highHighPriority.sender,
		highLowPriority1.sender,
		highLowPriority2.sender,
		lowHighPriority.sender,
	}
	if len(order) != len(expected) {
		t.Fatalf("expected %d entries, got %d", len(expected), len(order))
	}
	for idx, sender := range expected {
		if !order[idx].Equals(sender) {
			t.Fatalf("unexpected order at %d: got %v expected %v", idx, order[idx], sender)
		}
	}
}

// drainEvents reads all buffered events from the channel and returns counts.
func drainEvents(ch <-chan cmtmempool.AppMempoolEvent) (inserted, removed int) {
	const idleWindow = 2 * time.Millisecond
	const maxWait = 500 * time.Millisecond

	idle := time.NewTimer(idleWindow)
	defer idle.Stop()
	deadline := time.NewTimer(maxWait)
	defer deadline.Stop()

	for {
		select {
		case ev := <-ch:
			switch ev.Type {
			case cmtmempool.EventTxInserted:
				inserted++
			case cmtmempool.EventTxRemoved:
				removed++
			}
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(idleWindow)
		case <-idle.C:
			return
		case <-deadline.C:
			return
		}
	}
}

func newTestMempoolWithKeeper(t *testing.T, keeper *mockAccountKeeper) (*PriorityMempool, *mockAccountKeeper) {
	t.Helper()
	if keeper == nil {
		keeper = newMockAccountKeeper()
	}
	mp := NewPriorityMempool(PriorityMempoolConfig{
		MaxTx: 100,
	}, testTxEncoder)
	mp.SetAccountKeeper(keeper)
	return mp, keeper
}

func TestQueuedMempoolQueuesOutOfOrderSequences(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 5)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 5, active
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")); err != nil {
		t.Fatalf("insert seq 5: %v", err)
	}

	// Insert seq 7, future nonce, should be queued
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 7, 1000, "default")); err != nil {
		t.Fatalf("expected future nonce to be queued, got: %v", err)
	}

	if mp.CountTx() != 2 {
		t.Fatalf("expected 2 entries (1 active + 1 queued), got %d", mp.CountTx())
	}

	// Seq 7 should NOT appear in Select (only active entries)
	count := 0
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
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

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 5 (active), seq 7 (queued)
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")); err != nil {
		t.Fatalf("insert seq 5: %v", err)
	}
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 7, 1000, "default")); err != nil {
		t.Fatalf("insert seq 7: %v", err)
	}

	// Insert seq 6, fills the gap, should auto-promote seq 7
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 6, 1000, "default")); err != nil {
		t.Fatalf("insert seq 6: %v", err)
	}

	// All 3 should now be active
	count := 0
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
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

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Stale nonce 4 should be rejected
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 4, 1000, "default")); err == nil {
		t.Fatalf("expected stale nonce to be rejected")
	}

	// Stale nonce 3 should be rejected
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")); err == nil {
		t.Fatalf("expected stale nonce to be rejected")
	}

	// Matching nonce 5 should work
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")); err != nil {
		t.Fatalf("failed to insert matching nonce: %v", err)
	}
}

func TestQueuedMempoolPromoteQueued(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active) and seq 2, 3 (queued, gap at 1)
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")); err != nil {
		t.Fatalf("insert seq 0: %v", err)
	}
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")); err != nil {
		t.Fatalf("insert seq 2: %v", err)
	}
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")); err != nil {
		t.Fatalf("insert seq 3: %v", err)
	}

	// Only seq 0 should be active
	count := 0
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		count++
	}
	if count != 1 {
		t.Fatalf("expected 1 active entry, got %d", count)
	}

	// Simulate block commit, on-chain sequence advances to 2
	keeper.SetSequence(sender, 2)
	mp.PromoteQueued(sdkCtx)

	// After promotion, seq 0 still in pool + seq 2 and 3 promoted
	count = 0
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		count++
	}
	if count != 3 {
		t.Fatalf("expected 3 active entries after PromoteQueued (seq 0 still in pool + promoted 2,3), got %d", count)
	}
}

// TestPromoteQueuedResetsActiveNextAfterDrain verifies that PromoteQueued
// correctly promotes queued txs when the active pool has been fully drained
// by block inclusion, but activeNext was advanced past the on-chain sequence.
//
// This reproduces a bug where burst submitted txs arrive partially in-order
// (advancing activeNext) and partially not (going to the queue). When
// the correct order active txs get included in blocks and removed, activeNext
// stays at its peak.
func TestPromoteQueuedResetsActiveNextAfterDrain(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// simulating burst arrival, seq 0, 1, 2 arrive in-order (go to active,
	// activeNext advances to 3), then seq 5, 6, 7 arrive before seq 3, 4
	for _, seq := range []uint64{0, 1, 2} {
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, seq, 1000, "default")))
	}
	for _, seq := range []uint64{5, 6, 7} {
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, seq, 1000, "default")))
	}

	// 3 active (seq 0,1,2) + 3 queued (seq 5,6,7)
	activeCount := 0
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		activeCount++
	}
	require.Equal(t, 3, activeCount, "seq 0,1,2 should be active")
	require.Equal(t, 6, mp.CountTx(), "3 active + 3 queued = 6 total")

	// Now seq 3 and 4 arrive, they match activeNext (3, then 4), get promoted
	// to active, and also promote seq 5, 6, 7 from queued via the
	// continuous nonce chain. activeNext advances to 8.
	for _, seq := range []uint64{3, 4} {
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, seq, 1000, "default")))
	}

	// all 8 should now be active
	activeCount = 0
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		activeCount++
	}
	require.Equal(t, 8, activeCount, "all 8 txs should be active after chain promotion")

	// simulating blocks committing seq 0..4, then seq 5..7 arrive on another
	// node via gossip later. on that node, the active pool drains first.
	// we simulate this by removing seq 0..7 (block inclusion) and noting
	// that activeNext is still 8.
	for i := uint64(0); i < 8; i++ {
		// ignore error since Remove returns ErrTxNotFound for already-removed txs
		mp.Remove(newTestTxWithPriv(priv, i, 1000, "default"))
	}

	// active pool is empty, activeNext=8
	activeCount = 0
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		activeCount++
	}
	require.Equal(t, 0, activeCount, "active pool should be empty")

	// advancing keeper sequence to 5, only seq 0..4 committed
	keeper.SetSequence(sender, 5)

	// now new txs with seq 5, 6, 7 arrive via gossip from another node.
	// and get rejected by Insert as stale (nonce 5 < activeNext 8).
	// this is the scenario, the node already saw these txs,
	// promoted them, they got included/removed, but a slow gossip peer
	// re-sends them. they simply can't re-enter.
	//
	// but the issue is when seq 5,6,7 were never removed from queued
	// on some other node. let's simulate that by directly adding to the queue.
	mp.mtx.Lock()
	ss := mp.getOrCreateSenderLocked(sender.String())
	for _, seq := range []uint64{5, 6, 7} {
		tx := newTestTxWithPriv(priv, seq, 1000, "default")
		bz, _ := mp.txEncoder(tx)
		entry := &txEntry{
			tx:       tx,
			priority: 1000,
			size:     int64(len(bz)),
			key:      txKey{sender: sender.String(), nonce: seq},
			sequence: seq,
			tier:     queuedTier,
			bytes:    bz,
		}
		ss.queued[seq] = entry
		mp.queuedCount.Add(1)
	}
	mp.mtx.Unlock()

	// state: active=0, queued=3 (seq 5,6,7), activeNext=8, onChainSeq=5
	require.Equal(t, 3, mp.CountTx(), "should have 3 queued txs")

	// PromoteQueued should detect active is empty, reset activeNext to
	// onChainSeq (5), then promote seq 5,6,7 from queued to active.
	mp.PromoteQueued(sdkCtx)

	activeCount = 0
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		activeCount++
	}
	require.Equal(t, 3, activeCount,
		"PromoteQueued should promote all 3 queued txs (seq 5..7) after active pool drain")
}

func TestQueuedMempoolSelectOnlyReturnsActive(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active), seq 2 (queued)
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")); err != nil {
		t.Fatalf("insert seq 0: %v", err)
	}
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")); err != nil {
		t.Fatalf("insert seq 2: %v", err)
	}

	// Select should only return active entries
	count := 0
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
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

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active), seq 2 and 3 (queued)
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")); err != nil {
		t.Fatalf("insert seq 0: %v", err)
	}
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")); err != nil {
		t.Fatalf("insert seq 2: %v", err)
	}
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")); err != nil {
		t.Fatalf("insert seq 3: %v", err)
	}

	if mp.CountTx() != 3 {
		t.Fatalf("expected CountTx=3 (1 active + 2 queued), got %d", mp.CountTx())
	}
}

func TestQueuedMempoolGetTxDistributionIncludesQueued(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Empty pool, no queued tier
	dist := mp.GetTxDistribution()
	require.Zero(t, dist["queued"])

	// Insert seq 0 (active, "default" tier), seq 2 and 3 (queued)
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")))

	dist = mp.GetTxDistribution()
	require.Equal(t, uint64(1), dist["default"])
	require.Equal(t, uint64(2), dist["queued"])

	// Fill gap, seq 1 promotes seq 2 and 3 into the active pool
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 1, 1000, "default")))

	dist = mp.GetTxDistribution()
	require.Equal(t, uint64(4), dist["default"])
	require.Zero(t, dist["queued"])
}

func TestQueuedMempoolContainsChecksQueued(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")

	if err := mp.Insert(ctx, tx0); err != nil {
		t.Fatalf("insert seq 0: %v", err)
	}
	if err := mp.Insert(ctx, tx2); err != nil {
		t.Fatalf("insert seq 2: %v", err)
	}

	if !mp.Contains(tx0) {
		t.Fatalf("expected Contains=true for active tx")
	}
	if !mp.Contains(tx2) {
		t.Fatalf("expected Contains=true for queued tx")
	}
}

func TestQueuedMempoolRemoveFromQueued(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")

	if err := mp.Insert(ctx, tx0); err != nil {
		t.Fatalf("insert seq 0: %v", err)
	}
	if err := mp.Insert(ctx, tx2); err != nil {
		t.Fatalf("insert seq 2: %v", err)
	}

	// Remove active tx
	if err := mp.Remove(tx0); err != nil {
		t.Fatalf("remove active tx: %v", err)
	}
	if mp.Contains(tx0) {
		t.Fatalf("expected active tx removed")
	}

	// Remove queued tx
	if err := mp.Remove(tx2); err != nil {
		t.Fatalf("remove queued tx: %v", err)
	}
	if mp.Contains(tx2) {
		t.Fatalf("expected queued tx removed")
	}

	if mp.CountTx() != 0 {
		t.Fatalf("expected 0 entries, got %d", mp.CountTx())
	}

	// Remove non-existent tx
	tx3 := newTestTxWithPriv(priv, 3, 1000, "default")
	if err := mp.Remove(tx3); err != sdkmempool.ErrTxNotFound {
		t.Fatalf("expected ErrTxNotFound, got %v", err)
	}
}

func TestQueuedMempoolPerSenderLimit(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	mp.SetMaxQueuedPerSender(3)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active)
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")); err != nil {
		t.Fatalf("insert seq 0: %v", err)
	}

	// Queue seq 5, 6, 7 fills per sender limit of 3
	for _, seq := range []uint64{5, 6, 7} {
		if err := mp.Insert(ctx, newTestTxWithPriv(priv, seq, 1000, "default")); err != nil {
			t.Fatalf("insert seq %d: %v", seq, err)
		}
	}
	if mp.CountTx() != 4 {
		t.Fatalf("expected 4 (1 active + 3 queued), got %d", mp.CountTx())
	}

	// Seq 8 has the highest nonce, should be silently rejected
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 8, 1000, "default")); err != nil {
		t.Fatalf("insert seq 8 should silently skip, got error: %v", err)
	}
	if mp.CountTx() != 4 {
		t.Fatalf("expected still 4 after rejected insert, got %d", mp.CountTx())
	}

	// Seq 3 has lower nonce than the highest (7), should evict seq 7 and insert
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")); err != nil {
		t.Fatalf("insert seq 3: %v", err)
	}
	if mp.CountTx() != 4 {
		t.Fatalf("expected 4 after eviction+insert, got %d", mp.CountTx())
	}

	// Verify seq 7 evicted, seq 3 present
	if _, ok := mp.Lookup(sender.String(), 7); ok {
		t.Fatalf("expected seq 7 to be evicted")
	}
	if _, ok := mp.Lookup(sender.String(), 3); !ok {
		t.Fatalf("expected seq 3 to be present")
	}
}

func TestQueuedMempoolGlobalLimit(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	mp.SetMaxQueuedTotal(3)

	// Use different senders to avoid a per sender limit
	var privs [4]*secp256k1.PrivKey
	for i := range privs {
		privs[i] = secp256k1.GenPrivKey()
		sender := sdk.AccAddress(privs[i].PubKey().Address())
		keeper.SetSequence(sender, 0)

		// Insert seq 0 (active) for each sender
		if err := mp.Insert(ctx, newTestTxWithPriv(privs[i], 0, 1000, "default")); err != nil {
			t.Fatalf("insert active for sender %d: %v", i, err)
		}
	}

	// Queue future-nonce txs from 3 senders, fills global limit
	for i := 0; i < 3; i++ {
		if err := mp.Insert(ctx, newTestTxWithPriv(privs[i], 2, 1000, "default")); err != nil {
			t.Fatalf("queue for sender %d: %v", i, err)
		}
	}

	// 4th sender queued tx should be silently rejected (global limit)
	if err := mp.Insert(ctx, newTestTxWithPriv(privs[3], 2, 1000, "default")); err != nil {
		t.Fatalf("insert should silently skip, got error: %v", err)
	}

	// 4 active + 3 queued = 7
	if mp.CountTx() != 7 {
		t.Fatalf("expected 7 total (4 active + 3 queued), got %d", mp.CountTx())
	}
}

func TestQueuedMempoolStalenessRecoveryAfterCleanup(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 5)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 5 active, activeNext becomes 6
	tx5 := newTestTxWithPriv(priv, 5, 1000, "default")
	if err := mp.Insert(ctx, tx5); err != nil {
		t.Fatalf("insert seq 5: %v", err)
	}

	// Simulate a cleaning worker removing the active tx (ante failure)
	if err := mp.Remove(tx5); err != nil {
		t.Fatalf("remove tx: %v", err)
	}
	mp.PromoteQueued(sdkCtx)

	// Now the sender should be able to resubmit nonce 5
	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")); err != nil {
		t.Fatalf("expected re-submission of nonce 5 to succeed after PromoteQueued, got: %v", err)
	}
}

func TestQueuedMempoolGetTxInfo(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx2 := newTestTxWithPriv(priv, 2, 2000, "default")

	require.NoError(t, mp.Insert(ctx, tx0))
	require.NoError(t, mp.Insert(ctx, tx2))

	// Active tx info
	info, err := mp.GetTxInfo(sdkCtx, tx0)
	require.NoError(t, err)
	require.Equal(t, sender.String(), info.Sender)
	require.Equal(t, uint64(0), info.Sequence)
	require.True(t, info.Size > 0)
	require.Equal(t, uint64(1000), info.GasLimit)
	require.NotEmpty(t, info.Tier)
	require.NotEqual(t, "queued", info.Tier)
	require.NotEmpty(t, info.TxBytes)

	// Queued tx info
	info, err = mp.GetTxInfo(sdkCtx, tx2)
	require.NoError(t, err)
	require.Equal(t, sender.String(), info.Sender)
	require.Equal(t, uint64(2), info.Sequence)
	require.True(t, info.Size > 0)
	require.Equal(t, uint64(2000), info.GasLimit)
	require.Equal(t, "queued", info.Tier)
	require.NotEmpty(t, info.TxBytes)

	// Non-existent tx
	tx9 := newTestTxWithPriv(priv, 9, 1000, "default")
	_, err = mp.GetTxInfo(sdkCtx, tx9)
	require.ErrorIs(t, err, sdkmempool.ErrTxNotFound)
}

func TestQueuedMempoolNextExpectedSequence(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Unknown sender returns false
	_, ok, err := mp.NextExpectedSequence(sender.String())
	require.NoError(t, err)
	require.False(t, ok)

	// Insert seq 0 and activeNext becomes 1
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))

	next, ok, err := mp.NextExpectedSequence(sender.String())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, uint64(1), next)

	// Insert seq 1 and activeNext becomes 2
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 1, 1000, "default")))

	next, ok, err = mp.NextExpectedSequence(sender.String())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, uint64(2), next)
}

func TestQueuedMempoolSameNonceReplacement(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()

	// Insert seq 0 (active)
	ctx := sdk.WrapSDKContext(sdkCtx.WithPriority(100))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))

	// Queue seq 5 with priority 100
	ctx = sdk.WrapSDKContext(sdkCtx.WithPriority(100))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")))
	require.Equal(t, 2, mp.CountTx())

	// Replace seq 5 with higher priority 200 should succeed
	ctx = sdk.WrapSDKContext(sdkCtx.WithPriority(200))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 5, 2000, "default")))
	require.Equal(t, 2, mp.CountTx()) // count unchanged (replacement)

	// Try replacing seq 5 with lower priority 50 should be silently rejected
	ctx = sdk.WrapSDKContext(sdkCtx.WithPriority(50))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 5, 500, "default")))
	require.Equal(t, 2, mp.CountTx())
}

func TestQueuedMempoolSameNonceReplacementAtCapacityDoesNotEvictOther(t *testing.T) {
	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	senderA := sdk.AccAddress(privA.PubKey().Address())
	senderB := sdk.AccAddress(privB.PubKey().Address())

	mp := NewPriorityMempool(PriorityMempoolConfig{
		MaxTx: 2,
	}, testTxEncoder)

	// Fill pool with two active txs.
	ctxA := sdk.WrapSDKContext(testSDKContext().WithPriority(100))
	txA := newTestTxWithPriv(privA, 0, 1000, "default")
	require.NoError(t, mp.Insert(ctxA, txA))

	ctxB := sdk.WrapSDKContext(testSDKContext().WithPriority(10))
	txB := newTestTxWithPriv(privB, 0, 1000, "default")
	require.NoError(t, mp.Insert(ctxB, txB))
	require.Equal(t, 2, mp.CountTx())

	// Replace A with higher priority while pool is at capacity.
	ctxA2 := sdk.WrapSDKContext(testSDKContext().WithPriority(200))
	txA2 := newTestTxWithPriv(privA, 0, 2000, "default")
	require.NoError(t, mp.Insert(ctxA2, txA2))

	// Replacement should not evict unrelated txB.
	require.Equal(t, 2, mp.CountTx())

	hashA, ok := mp.Lookup(senderA.String(), 0)
	require.True(t, ok)
	bzA2, err := testTxEncoder(txA2)
	require.NoError(t, err)
	require.Equal(t, TxHash(bzA2), hashA)

	hashB, ok := mp.Lookup(senderB.String(), 0)
	require.True(t, ok)
	bzB, err := testTxEncoder(txB)
	require.NoError(t, err)
	require.Equal(t, TxHash(bzB), hashB)
}

func TestQueuedMempoolEventDispatch(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active), EventTxInserted
	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx0))
	inserted, removed := drainEvents(eventCh)
	require.Equal(t, 1, inserted, "active insert should fire EventTxInserted")
	require.Equal(t, 0, removed)

	// Insert seq 2 (queued), no event
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx2))
	inserted, removed = drainEvents(eventCh)
	require.Equal(t, 0, inserted, "queued insert should not fire EventTxInserted")
	require.Equal(t, 0, removed)

	// Remove queued tx, EventTxRemoved
	require.NoError(t, mp.Remove(tx2))
	inserted, removed = drainEvents(eventCh)
	require.Equal(t, 0, inserted)
	require.Equal(t, 1, removed, "queued remove should fire EventTxRemoved")

	// Reinsert seq 2 (queued) and then fill the gap with seq 1
	tx2b := newTestTxWithPriv(priv, 2, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx2b))
	drainEvents(eventCh)

	tx1 := newTestTxWithPriv(priv, 1, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx1))
	inserted, removed = drainEvents(eventCh)
	// seq 1 inserted (active) + seq 2 promoted
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

	mp, _ := newTestMempoolWithKeeper(t, keeper)

	// Sender A active only (seq 0), no queued entries
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privA, 0, 1000, "default")))

	// Sender B active (seq 0) + queued (seq 2)
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privB, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privB, 2, 1000, "default")))

	// PromoteQueued. B on-chain seq advances, promoting seq 2
	keeper.SetSequence(senderB, 2)
	mp.PromoteQueued(sdkCtx)

	// B seq 2 should now be promoted
	activeCount := 0
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		activeCount++
	}
	require.Equal(t, 3, activeCount, "A(seq0) + B(seq0) + B(seq2 promoted)")

	// activeNext should be refreshed from the pool state
	next, ok, _ := mp.NextExpectedSequence(senderA.String())
	require.True(t, ok)
	require.Equal(t, uint64(1), next)

	// remove A active tx (simulate cleaning worker)
	txA0 := newTestTxWithPriv(privA, 0, 1000, "default")
	require.NoError(t, mp.Remove(txA0))

	// PromoteQueued. A has no pool entries and no queued, should be cleaned up
	mp.PromoteQueued(sdkCtx)

	_, ok, _ = mp.NextExpectedSequence(senderA.String())
	require.False(t, ok, "A should be cleaned from activeNext after pool cleanup")

	// A can now reinsert (fresh lookup from store)
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privA, 0, 1000, "default")))
}

func TestQueuedMempoolPromoteQueuedEvictsStale(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active), seq 2, 3, 5 (queued)
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")))
	require.Equal(t, 4, mp.CountTx())

	// simulate block commit, on-chain advances past seq 2 and 3
	keeper.SetSequence(sender, 4)
	drainEvents(eventCh) // clear events from inserts
	mp.PromoteQueued(sdkCtx)

	// Seq 2, 3 should be evicted (stale), seq 5 remains queued (gap at 4)
	_, removed := drainEvents(eventCh)
	require.Equal(t, 2, removed, "stale seq 2 and 3 should fire EventTxRemoved")

	// seq 0 still in active pool + seq 5 still queued = 2
	require.Equal(t, 2, mp.CountTx())

	// Verify seq 5 is still queued
	_, ok := mp.Lookup(sender.String(), 5)
	require.True(t, ok, "seq 5 should still be queued")
}

// collectEvents returns typed slices of tx bytes.
func collectEvents(ch <-chan cmtmempool.AppMempoolEvent) (inserted, removed [][]byte) {
	const idleWindow = 2 * time.Millisecond
	const maxWait = 500 * time.Millisecond

	idle := time.NewTimer(idleWindow)
	defer idle.Stop()
	deadline := time.NewTimer(maxWait)
	defer deadline.Stop()

	for {
		select {
		case ev := <-ch:
			switch ev.Type {
			case cmtmempool.EventTxInserted:
				inserted = append(inserted, ev.Tx)
			case cmtmempool.EventTxRemoved:
				removed = append(removed, ev.Tx)
			default:
				panic("unhandled default case")
			}
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(idleWindow)
		case <-idle.C:
			return
		case <-deadline.C:
			return
		}
	}
}

func collectNEvents(t *testing.T, ch <-chan cmtmempool.AppMempoolEvent, n int, timeout time.Duration) []cmtmempool.AppMempoolEvent {
	t.Helper()

	events := make([]cmtmempool.AppMempoolEvent, 0, n)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for len(events) < n {
		select {
		case ev := <-ch:
			events = append(events, ev)
		case <-timer.C:
			t.Fatalf("timed out waiting for %d events, got %d", n, len(events))
		}
	}

	return events
}

func encodeTx(t *testing.T, tx sdk.Tx) []byte {
	t.Helper()
	bz, err := testTxEncoder(tx)
	require.NoError(t, err)
	return bz
}

func TestEventOrderSameNonceReplacement(t *testing.T) {
	priv := secp256k1.GenPrivKey()

	mp := newTestPriorityMempool(t, nil)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 1)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), tx0))

	tx1 := newTestTxWithPriv(priv, 0, 2000, "default")
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(20)), tx1))

	events := collectNEvents(t, eventCh, 3, 2*time.Second)
	require.Equal(t, cmtmempool.EventTxInserted, events[0].Type, "first event should be initial insert")
	require.Equal(t, cmtmempool.EventTxRemoved, events[1].Type, "replacement must remove old tx first")
	require.Equal(t, cmtmempool.EventTxInserted, events[2].Type, "replacement must insert new tx after removal")
}

func TestEventNoDropUnderBackPressure(t *testing.T) {
	priv := secp256k1.GenPrivKey()

	mp := newTestPriorityMempool(t, nil)
	// tiny channel to force backpressure on comet side
	eventCh := make(chan cmtmempool.AppMempoolEvent, 1)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()

	const replacements = 100

	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(1)), newTestTxWithPriv(priv, 0, 1000, "default")))
	for i := range replacements {
		tx := newTestTxWithPriv(priv, 0, uint64(2000+i), "default")
		require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(int64(i+2))), tx))
	}

	events := collectNEvents(t, eventCh, 1+replacements*2, 5*time.Second)
	var inserted, removed int
	for _, ev := range events {
		switch ev.Type {
		case cmtmempool.EventTxInserted:
			inserted++
		case cmtmempool.EventTxRemoved:
			removed++
		}
	}

	require.Equal(t, replacements+1, inserted, "initial insert + one insert per successful replacement")
	require.Equal(t, replacements, removed, "one removed per successful replacement")
}

func TestStopEventDispatch_IdempotentWithoutStart(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	require.NotPanics(t, func() {
		mp.StopEventDispatch()
		mp.StopEventDispatch()
	})
}

func TestStopEventDispatch_IdempotentAfterStart(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 1)
	mp.SetEventCh(eventCh)

	done := make(chan struct{})
	go func() {
		mp.StopEventDispatch()
		mp.StopEventDispatch()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for idempotent StopEventDispatch")
	}
}

func TestEventActiveInsertAndRemove(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx0))

	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, 1, "active insert => 1 EventTxInserted")
	require.Len(t, rem, 0)
	require.Equal(t, encodeTx(t, tx0), ins[0])

	// remove active tx
	require.NoError(t, mp.Remove(tx0))
	ins, rem = collectEvents(eventCh)
	require.Len(t, ins, 0)
	require.Len(t, rem, 1, "active remove => 1 EventTxRemoved")
	require.Equal(t, encodeTx(t, tx0), rem[0])
}

func TestEventQueuedInsertNoEvent(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// insert active seq 0
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	drainEvents(eventCh) // clear

	// insert seq 5. should be queued
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")))
	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, 0, "queued insert should NOT fire EventTxInserted")
	require.Len(t, rem, 0, "queued insert should NOT fire EventTxRemoved")
}

func TestEventQueuedRemove(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	tx5 := newTestTxWithPriv(priv, 5, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx5))
	drainEvents(eventCh)

	require.NoError(t, mp.Remove(tx5))
	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, 0)
	require.Len(t, rem, 1, "queued remove => EventTxRemoved")
}

func TestEventSameNonceQueuedReplacement(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()

	// seq 0 active
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx), newTestTxWithPriv(priv, 0, 1000, "default")))
	// seq 5 queued with priority 10
	tx5a := newTestTxWithPriv(priv, 5, 1000, "default")
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), tx5a))
	drainEvents(eventCh)

	// replace queued seq 5 with higher priority 100
	tx5b := newTestTxWithPriv(priv, 5, 2000, "default")
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(100)), tx5b))

	ins, rem := collectEvents(eventCh)
	// queued replacement: evicted old fires EventTxRemoved, no insert event (still queued)
	require.Len(t, rem, 1, "queued replacement should fire EventTxRemoved for old entry")
	require.Equal(t, encodeTx(t, tx5a), rem[0])
	require.Len(t, ins, 0, "queued replacement should not fire EventTxInserted")
}

func TestEventQueuedPerSenderEviction(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	mp.SetMaxQueuedPerSender(2)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// seq 0 active
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	// queue seq 5, 6 (filling sender limit of 2)
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")))
	tx6 := newTestTxWithPriv(priv, 6, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx6))
	drainEvents(eventCh)

	// insert seq 3, lower than highest queued (6), should evict 6
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")))

	ins, rem := collectEvents(eventCh)
	require.Len(t, rem, 1, "per-sender eviction should fire EventTxRemoved for highest-nonce evicted tx")
	require.Equal(t, encodeTx(t, tx6), rem[0])
	require.Len(t, ins, 0, "per-sender eviction inserts to queue, no EventTxInserted")
}

func TestEventQueuedPerSenderRejectHighestNonce(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	mp.SetMaxQueuedPerSender(2)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 6, 1000, "default")))
	drainEvents(eventCh)

	// seq 10 >= highest queued (6), silently rejected. no events
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 10, 1000, "default")))

	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, 0)
	require.Len(t, rem, 0, "rejected queued insert should not fire any events")
}

func TestEventCapacityEvictionOnInsert(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	// create mempool with MaxTx=2
	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 2}, testTxEncoder)
	mp.SetAccountKeeper(keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	privC := secp256k1.GenPrivKey()
	keeper.SetSequence(sdk.AccAddress(privA.PubKey().Address()), 0)
	keeper.SetSequence(sdk.AccAddress(privB.PubKey().Address()), 0)
	keeper.SetSequence(sdk.AccAddress(privC.PubKey().Address()), 0)

	// fill pool, A(pri=10), B(pri=5)
	txA := newTestTxWithPriv(privA, 0, 1000, "default")
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), txA))
	txB := newTestTxWithPriv(privB, 0, 1000, "default")
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(5)), txB))
	drainEvents(eventCh)

	// insert C with priority 20, should evict B, the lowest priority
	txC := newTestTxWithPriv(privC, 0, 1000, "default")
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(20)), txC))

	ins, rem := collectEvents(eventCh)
	require.Len(t, rem, 1, "capacity eviction should fire EventTxRemoved for evicted tx")
	require.Equal(t, encodeTx(t, txB), rem[0])
	require.Len(t, ins, 1, "new tx should fire EventTxInserted")
	require.Equal(t, encodeTx(t, txC), ins[0])
}

func TestEventCapacityEvictionReject(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 2}, testTxEncoder)
	mp.SetAccountKeeper(keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	privC := secp256k1.GenPrivKey()
	keeper.SetSequence(sdk.AccAddress(privA.PubKey().Address()), 0)
	keeper.SetSequence(sdk.AccAddress(privB.PubKey().Address()), 0)
	keeper.SetSequence(sdk.AccAddress(privC.PubKey().Address()), 0)

	// filling pool. A(pri=10), B(pri=5)
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 0, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(5)), newTestTxWithPriv(privB, 0, 1000, "default")))
	drainEvents(eventCh)

	// inserting C with priority 1, lower than all. should be rejected
	err := mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(1)), newTestTxWithPriv(privC, 0, 1000, "default"))
	require.Error(t, err)

	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, 0, "rejected insert should not fire EventTxInserted")
	require.Len(t, rem, 0, "rejected insert should not fire EventTxRemoved")
}

func TestEventGapFillPromotion(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// seq 0 active, seq 2, 3 queued (gap at 1)
	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx0))
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx2))
	tx3 := newTestTxWithPriv(priv, 3, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx3))
	drainEvents(eventCh)

	// Fill gap with seq 1. should promote seq 2 and 3
	tx1 := newTestTxWithPriv(priv, 1, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx1))

	ins, rem := collectEvents(eventCh)
	require.Len(t, rem, 0)

	// seq 1 inserted + seq 2 promoted + seq 3 promoted = 3 EventTxInserted
	require.Len(t, ins, 3, "gap-fill should fire EventTxInserted for seq 1 + promoted seq 2 + promoted seq 3")
}

func TestEventPromoteQueuedStaleEviction(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// seq 0 active, seq 2, 3, 5 queued
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx2))
	tx3 := newTestTxWithPriv(priv, 3, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx3))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")))
	drainEvents(eventCh)

	// on-chain advances past 2 and 3 (to 4)
	keeper.SetSequence(sender, 4)
	mp.PromoteQueued(sdkCtx)

	ins, rem := collectEvents(eventCh)
	// seq 2 and 3 stale => 2 EventTxRemoved
	require.Len(t, rem, 2, "PromoteQueued should fire EventTxRemoved for stale queued txs")
	// seq 5 still has gap (need 4) => no promotion
	require.Len(t, ins, 0, "no promotion because gap remains at seq 4")
}

func TestEventPromoteQueuedPromotionAndCapacityEviction(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	// MaxTx=3
	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 3}, testTxEncoder)
	mp.SetAccountKeeper(keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	senderA := sdk.AccAddress(privA.PubKey().Address())
	senderB := sdk.AccAddress(privB.PubKey().Address())
	keeper.SetSequence(senderA, 0)
	keeper.SetSequence(senderB, 0)

	// A seq 0 (active, pri=100), queued seq 2 (pri=50)
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(100)), newTestTxWithPriv(privA, 0, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(50)), newTestTxWithPriv(privA, 2, 1000, "default")))

	// B seq 0 (active, pri=1), seq 1 (active, pri=1)
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(1)), newTestTxWithPriv(privB, 0, 1000, "default")))
	// pool is now full at 3 active entries (A:0, B:0 + need to fit B:1)
	// Actually, A:0 and B:0 = 2 active + A:2 is queued. Let's add another active.
	drainEvents(eventCh)

	// on-chain for A advances to 2 so should promote
	keeper.SetSequence(senderA, 2)
	mp.PromoteQueued(sdkCtx)

	ins, rem := collectEvents(eventCh)

	// seq 2 gets promoted into the active pool. the pool was at 2 active (A=0, B=0) with MaxTx=3, so no eviction needed.
	require.Len(t, ins, 1, "PromoteQueued should fire EventTxInserted for promoted tx")
	require.Len(t, rem, 0, "no eviction needed when pool has capacity")
}

func TestEventPromoteQueuedCapacityEvictsLowPriority(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	// MaxTx=2 so promotion must evict
	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 2}, testTxEncoder)
	mp.SetAccountKeeper(keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	senderA := sdk.AccAddress(privA.PubKey().Address())
	senderB := sdk.AccAddress(privB.PubKey().Address())
	keeper.SetSequence(senderA, 0)
	keeper.SetSequence(senderB, 0)

	// A seq 0 active (pri=100), seq 2 queued (pri=50)
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(100)), newTestTxWithPriv(privA, 0, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(50)), newTestTxWithPriv(privA, 2, 1000, "default")))

	// B seq 0 active (pri=1), the lowest priority
	txB := newTestTxWithPriv(privB, 0, 1000, "default")
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(1)), txB))
	require.Equal(t, 2, mp.CountTx()-1) // 2 active + 1 queued => CountTx=3
	drainEvents(eventCh)

	// on-chain for A advances to 2, so A seq 2 should promote, evicting B seq 0
	keeper.SetSequence(senderA, 2)
	mp.PromoteQueued(sdkCtx)

	ins, rem := collectEvents(eventCh)
	require.GreaterOrEqual(t, len(ins), 1, "promoted tx should fire EventTxInserted")
	require.GreaterOrEqual(t, len(rem), 1, "capacity eviction during promotion should fire EventTxRemoved")
}

func TestEventCleanUpEntriesRemoval(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx1 := newTestTxWithPriv(priv, 1, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx0))
	require.NoError(t, mp.Insert(ctx, tx1))
	drainEvents(eventCh)

	// advancing on-chain sequence past both txs
	keeper.SetSequence(sender, 5)
	baseApp := testBaseApp{ctx: sdkCtx}
	mp.cleanUpEntries(baseApp, keeper)

	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, 0)
	require.Len(t, rem, 2, "cleanUpEntries should fire EventTxRemoved for each stale entry")
}

func TestEventMultipleSendersInsertRemove(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)

	const numSenders = 5
	privs := make([]*secp256k1.PrivKey, numSenders)
	for i := range privs {
		privs[i] = secp256k1.GenPrivKey()
		keeper.SetSequence(sdk.AccAddress(privs[i].PubKey().Address()), 0)
	}

	// each sender inserts seq 0 (active) and seq 2 (queued)
	txs := make([][]*testTx, numSenders)
	for i := 0; i < numSenders; i++ {
		ctx := sdk.WrapSDKContext(sdkCtx)
		tx0 := newTestTxWithPriv(privs[i], 0, 1000, "default")
		tx2 := newTestTxWithPriv(privs[i], 2, 1000, "default")
		require.NoError(t, mp.Insert(ctx, tx0))
		require.NoError(t, mp.Insert(ctx, tx2))
		txs[i] = []*testTx{tx0, tx2}
	}

	ins, rem := collectEvents(eventCh)
	require.Len(t, ins, numSenders, "each sender's active insert should fire EventTxInserted")
	require.Len(t, rem, 0, "queued inserts should not fire events")

	// filling gaps for all senders (insert seq 1 => promotes seq 2)
	for i := 0; i < numSenders; i++ {
		ctx := sdk.WrapSDKContext(sdkCtx)
		tx1 := newTestTxWithPriv(privs[i], 1, 1000, "default")
		require.NoError(t, mp.Insert(ctx, tx1))
	}

	ins, rem = collectEvents(eventCh)
	// each sender has 1 insert (seq 1) + 1 promoted (seq 2) = 2 per sender
	require.Len(t, ins, numSenders*2, "gap-fill for %d senders should fire %d EventTxInserted", numSenders, numSenders*2)
	require.Len(t, rem, 0)
}

func TestEventConcurrentInsertsSameNonce(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 1000)
	mp.SetEventCh(eventCh)
	sdkCtx := testSDKContext()

	const goroutines = 10
	done := make(chan struct{})

	// all goroutines here are racing to insert seq 0 with increasing priority
	for i := 0; i < goroutines; i++ {
		go func(priority int64) {
			ctx := sdk.WrapSDKContext(sdkCtx.WithPriority(priority))
			tx := newTestTxWithPriv(priv, 0, 1000, "default")
			_ = mp.Insert(ctx, tx)
			done <- struct{}{}
		}(int64(i + 1))
	}
	for i := 0; i < goroutines; i++ {
		<-done
	}

	ins, rem := collectEvents(eventCh)
	// exactly 1 tx should be in the pool
	require.Equal(t, 1, mp.CountTx(), "only one tx should survive concurrent same-nonce inserts")

	// the first insertion fires EventTxInserted. the following successful replacements fire
	// EventTxRemoved + EventTxInserted. failed replacements fire nothing.
	// inserted >= 1 (at least the first one)
	require.GreaterOrEqual(t, len(ins), 1, "at least one EventTxInserted for first insert")

	// now every replacement adds 1 removed + 1 inserted, so inserted == removed + 1
	require.Equal(t, len(ins), len(rem)+1,
		"inserted events should equal removed events + 1 (initial insert)")
}

func TestEventConcurrentInsertsVariousNoncesMultipleSenders(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 10000)
	mp.SetEventCh(eventCh)

	const numSenders = 5
	const txsPerSender = 5

	privs := make([]*secp256k1.PrivKey, numSenders)
	for i := range privs {
		privs[i] = secp256k1.GenPrivKey()
		keeper.SetSequence(sdk.AccAddress(privs[i].PubKey().Address()), 0)
	}

	done := make(chan struct{})
	totalGoroutines := numSenders * txsPerSender

	// each sender inserts seq 0..4 concurrently from different goroutines
	for i := 0; i < numSenders; i++ {
		for seq := uint64(0); seq < txsPerSender; seq++ {
			go func(priv *secp256k1.PrivKey, s uint64) {
				ctx := sdk.WrapSDKContext(sdkCtx.WithPriority(int64(s + 1)))
				tx := newTestTxWithPriv(priv, s, 1000, "default")
				_ = mp.Insert(ctx, tx)
				done <- struct{}{}
			}(privs[i], seq)
		}
	}

	for i := 0; i < totalGoroutines; i++ {
		<-done
	}

	ins, rem := collectEvents(eventCh)

	// all txs should be in the pool, since each sender has unique nonces
	require.Equal(t, numSenders*txsPerSender, mp.CountTx(),
		"all txs should be in pool (no duplicates across senders)")

	// also no replacements since each sender has unique nonces, so no removes
	require.Len(t, rem, 0, "no replacements expected across different senders with unique nonces")

	// counting inserted events. active inserts fire events, queued do not.
	// due to concurrency, the exact split between active and queued depends on timing.
	require.GreaterOrEqual(t, len(ins), numSenders,
		"at least one EventTxInserted per sender (for seq 0)")
	require.LessOrEqual(t, len(ins), numSenders*txsPerSender,
		"at most all txs fire EventTxInserted")
}

func TestEventPromoteQueuedMultipleSenders(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)

	const numSenders = 3
	privs := make([]*secp256k1.PrivKey, numSenders)
	senders := make([]sdk.AccAddress, numSenders)
	for i := range privs {
		privs[i] = secp256k1.GenPrivKey()
		senders[i] = sdk.AccAddress(privs[i].PubKey().Address())
		keeper.SetSequence(senders[i], 0)
	}

	ctx := sdk.WrapSDKContext(sdkCtx)

	// each sender has seq 0 (active), seq 2, 3 (queued, gap at 1)
	for i := 0; i < numSenders; i++ {
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privs[i], 0, 1000, "default")))
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privs[i], 2, 1000, "default")))
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privs[i], 3, 1000, "default")))
	}
	drainEvents(eventCh)

	// advancing all senders on-chain to 2 so seq 2 and 3 get promoted
	for i := 0; i < numSenders; i++ {
		keeper.SetSequence(senders[i], 2)
	}
	mp.PromoteQueued(sdkCtx)

	ins, rem := collectEvents(eventCh)

	// each sender has seq 2 promoted + seq 3 promoted = 2 EventTxInserted
	require.Equal(t, numSenders*2, len(ins), "each sender should get 2 promoted txs")
	require.Len(t, rem, 0, "no stale txs to remove")
}

type txKeyType = [sha256.Size]byte

// reactorSim simulates the cometbft reactor's insertedTxs tracking.
// it should be cumulative, calling drain() repeatedly to process new events.
type reactorSim struct {
	ch          <-chan cmtmempool.AppMempoolEvent
	insertedTxs map[txKeyType]bool
	hasValidTxs bool
}

func newReactorSim(ch <-chan cmtmempool.AppMempoolEvent) *reactorSim {
	return &reactorSim{
		ch:          ch,
		insertedTxs: make(map[txKeyType]bool),
	}
}

// drain processes all buffered events and updates the cumulative state.
func (r *reactorSim) drain() {
	const idleWindow = 2 * time.Millisecond
	const maxWait = 500 * time.Millisecond

	idle := time.NewTimer(idleWindow)
	defer idle.Stop()
	deadline := time.NewTimer(maxWait)
	defer deadline.Stop()

	for {
		select {
		case ev := <-r.ch:
			switch ev.Type {
			case cmtmempool.EventTxInserted:
				r.insertedTxs[ev.TxKey] = true
				r.hasValidTxs = true
			case cmtmempool.EventTxRemoved:
				delete(r.insertedTxs, ev.TxKey)
				if len(r.insertedTxs) == 0 {
					r.hasValidTxs = false
				}
			}
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(idleWindow)
		case <-idle.C:
			return
		case <-deadline.C:
			return
		}
	}
}

// activeCount returns the number of active pool entries via Select iterator.
func activeCount(mp *PriorityMempool) int {
	n := 0
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		n++
	}
	return n
}

func activeKeySet(mp *PriorityMempool) map[txKeyType]bool {
	out := make(map[txKeyType]bool)
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		tx := it.Tx()
		if tx == nil {
			continue
		}
		bz, err := testTxEncoder(tx)
		if err != nil {
			continue
		}
		key := cmttypes.Tx(bz).Key()
		out[key] = true
	}
	return out
}

func reactorMatchesActiveSet(mp *PriorityMempool, reactor *reactorSim) bool {
	reactor.drain()
	active := activeKeySet(mp)
	if len(active) != len(reactor.insertedTxs) {
		return false
	}

	for k := range active {
		if !reactor.insertedTxs[k] {
			return false
		}
	}

	for k := range reactor.insertedTxs {
		if !active[k] {
			return false
		}
	}

	return reactor.hasValidTxs == (len(active) > 0)
}

func TestReactorInvariant_InsertRemoveLifecycle(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	reactor := newReactorSim(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// inserting seq 0, 1, 2 (all active)
	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx1 := newTestTxWithPriv(priv, 1, 1000, "default")
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx0))
	require.NoError(t, mp.Insert(ctx, tx1))
	require.NoError(t, mp.Insert(ctx, tx2))

	reactor.drain()
	require.Equal(t, 3, len(reactor.insertedTxs), "reactor should track 3 active txs")
	require.Equal(t, activeCount(mp), len(reactor.insertedTxs), "reactor map size should match active pool")
	require.True(t, reactor.hasValidTxs, "hasValidTxs should be true")

	// remove tx1
	require.NoError(t, mp.Remove(tx1))
	reactor.drain()
	require.Equal(t, 2, len(reactor.insertedTxs))
	require.Equal(t, activeCount(mp), len(reactor.insertedTxs))
	require.True(t, reactor.hasValidTxs)

	// now remove the remaining
	require.NoError(t, mp.Remove(tx0))
	require.NoError(t, mp.Remove(tx2))
	reactor.drain()
	require.Equal(t, 0, len(reactor.insertedTxs))
	require.Equal(t, activeCount(mp), len(reactor.insertedTxs))
	require.False(t, reactor.hasValidTxs, "hasValidTxs should be false when all txs removed")
}

func TestReactorInvariant_QueuedPromotionAndStaleEviction(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	const numSenders = 3
	privs := make([]*secp256k1.PrivKey, numSenders)
	senders := make([]sdk.AccAddress, numSenders)
	for i := range privs {
		privs[i] = secp256k1.GenPrivKey()
		senders[i] = sdk.AccAddress(privs[i].PubKey().Address())
		keeper.SetSequence(senders[i], 0)
	}

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 1000)
	mp.SetEventCh(eventCh)
	reactor := newReactorSim(eventCh)
	ctx := sdk.WrapSDKContext(sdkCtx)

	// each sender has seq 0 (active), seq 2, 3, 5 (queued)
	for i := 0; i < numSenders; i++ {
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privs[i], 0, 1000, "default")))
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privs[i], 2, 1000, "default")))
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privs[i], 3, 1000, "default")))
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privs[i], 5, 1000, "default")))
	}

	reactor.drain()
	require.Equal(t, numSenders, len(reactor.insertedTxs), "only seq 0 per sender should be active")
	require.Equal(t, activeCount(mp), len(reactor.insertedTxs))
	require.True(t, reactor.hasValidTxs)

	// advancing on-chain to 4. so seq 2, 3 become stale, seq 5 still has gap
	for i := 0; i < numSenders; i++ {
		keeper.SetSequence(senders[i], 4)
	}
	mp.PromoteQueued(sdkCtx)

	reactor.drain()
	require.Equal(t, activeCount(mp), len(reactor.insertedTxs),
		"reactor map should match active pool after stale eviction")
	require.True(t, reactor.hasValidTxs, "seq 0 still active, hasValidTxs should be true")

	// advancing on-chain to 5. now seq 5 becomes promotable
	for i := 0; i < numSenders; i++ {
		keeper.SetSequence(senders[i], 5)
	}
	mp.PromoteQueued(sdkCtx)

	reactor.drain()
	require.Equal(t, activeCount(mp), len(reactor.insertedTxs),
		"reactor map should match active pool after promotion")
	require.True(t, reactor.hasValidTxs)
}

func TestReactorInvariant_GapFillChainPromotion(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	reactor := newReactorSim(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// seq 0 active, seq 2, 3, 4 queued (gap at 1)
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 4, 1000, "default")))

	reactor.drain()
	require.Equal(t, 1, len(reactor.insertedTxs), "only seq 0 active before gap fill")

	// filling the gap with seq 1, promotes 2, 3, 4
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 1, 1000, "default")))

	reactor.drain()
	require.Equal(t, 5, activeCount(mp))
	require.Equal(t, activeCount(mp), len(reactor.insertedTxs),
		"reactor should track all 5 active txs after chain promotion")
	require.True(t, reactor.hasValidTxs)
}

func TestReactorInvariant_CapacityEviction(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 3}, testTxEncoder)
	mp.SetAccountKeeper(keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	reactor := newReactorSim(eventCh)

	privs := make([]*secp256k1.PrivKey, 4)
	for i := range privs {
		privs[i] = secp256k1.GenPrivKey()
		keeper.SetSequence(sdk.AccAddress(privs[i].PubKey().Address()), 0)
	}

	// filling the pool with 3 txs at priorities 10, 20, 30
	for i := 0; i < 3; i++ {
		ctx := sdk.WrapSDKContext(sdkCtx.WithPriority(int64((i + 1) * 10)))
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privs[i], 0, 1000, "default")))
	}
	reactor.drain()
	require.Equal(t, 3, len(reactor.insertedTxs))

	// inserting the 4th with priority 50, should evict the lowest (priority 10)
	ctx := sdk.WrapSDKContext(sdkCtx.WithPriority(50))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privs[3], 0, 1000, "default")))

	reactor.drain()
	require.Equal(t, 3, len(reactor.insertedTxs), "pool should still have 3 after eviction+insert")
	require.Equal(t, activeCount(mp), len(reactor.insertedTxs))
	require.True(t, reactor.hasValidTxs)
}

func TestReactorInvariant_CleanUpEntries(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	reactor := newReactorSim(eventCh)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 1, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")))

	reactor.drain()
	require.Equal(t, 3, len(reactor.insertedTxs))

	// advancing on-chain seq past all, cleanup should remove all
	keeper.SetSequence(sender, 10)
	baseApp := testBaseApp{ctx: sdkCtx}
	mp.cleanUpEntries(baseApp, keeper)

	reactor.drain()
	require.Equal(t, 0, len(reactor.insertedTxs), "all txs cleaned up")
	require.Equal(t, activeCount(mp), len(reactor.insertedTxs))
	require.False(t, reactor.hasValidTxs, "hasValidTxs should be false when all txs removed via cleanup")
}

func TestReactorInvariant_ConcurrentMultiSenderWorkload(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 10000)
	mp.SetEventCh(eventCh)
	reactor := newReactorSim(eventCh)

	const numSenders = 10
	const seqsPerSender = 5

	privs := make([]*secp256k1.PrivKey, numSenders)
	for i := range privs {
		privs[i] = secp256k1.GenPrivKey()
		keeper.SetSequence(sdk.AccAddress(privs[i].PubKey().Address()), 0)
	}

	// concurrent inserts. with each sender inserts seq 0..4
	var wg sync.WaitGroup
	wg.Add(numSenders * seqsPerSender)
	for i := 0; i < numSenders; i++ {
		for seq := uint64(0); seq < seqsPerSender; seq++ {
			go func(priv *secp256k1.PrivKey, s uint64) {
				defer wg.Done()
				ctx := sdk.WrapSDKContext(sdkCtx.WithPriority(int64(s + 1)))
				_ = mp.Insert(ctx, newTestTxWithPriv(priv, s, 1000, "default"))
			}(privs[i], seq)
		}
	}
	wg.Wait()

	reactor.drain()
	active := activeCount(mp)
	require.Equal(t, active, len(reactor.insertedTxs),
		"reactor insertedTxs count must match active pool after concurrent inserts")
	if active > 0 {
		require.True(t, reactor.hasValidTxs)
	}

	// PromoteQueued should fix any remaining queued txs
	for i := 0; i < numSenders; i++ {
		keeper.SetSequence(sdk.AccAddress(privs[i].PubKey().Address()), seqsPerSender)
	}
	mp.PromoteQueued(sdkCtx)

	reactor.drain()
	active = activeCount(mp)
	require.Equal(t, active, len(reactor.insertedTxs),
		"reactor insertedTxs count must match active pool after PromoteQueued")
	require.Equal(t, numSenders*seqsPerSender, active,
		"all txs should be active after PromoteQueued")
	require.True(t, reactor.hasValidTxs)

	// removing all txs one by one
	for i := 0; i < numSenders; i++ {
		for seq := uint64(0); seq < seqsPerSender; seq++ {
			tx := newTestTxWithPriv(privs[i], seq, 1000, "default")
			_ = mp.Remove(tx)
		}
	}

	reactor.drain()
	require.Equal(t, 0, len(reactor.insertedTxs), "all txs removed")
	require.Equal(t, activeCount(mp), len(reactor.insertedTxs))
	require.False(t, reactor.hasValidTxs, "hasValidTxs must be false when pool is empty")
}

func TestReactorInvariant_ConcurrentRandomizedStress(t *testing.T) {
	sdkCtx := testSDKContext()
	ctxWithPriority := func(priority int64) context.Context {
		return sdk.WrapSDKContext(sdkCtx.WithPriority(priority))
	}

	const (
		rounds         = 5
		numSenders     = 20
		workers        = 32
		opsPerWorker   = 300
		maxNonce       = 6
		insertRatioPct = 70
	)

	for round := range rounds {
		mp := newTestPriorityMempool(t, nil)
		eventCh := make(chan cmtmempool.AppMempoolEvent, 1) // force heavy backpressure
		mp.SetEventCh(eventCh)
		reactor := newReactorSim(eventCh)

		privs := make([]*secp256k1.PrivKey, numSenders)
		for i := range privs {
			privs[i] = secp256k1.GenPrivKey()
		}

		var wg sync.WaitGroup
		wg.Add(workers)
		for w := range workers {
			seed := int64(round*10_000 + w + 1)
			go func(seed int64) {
				defer wg.Done()
				rng := rand.New(rand.NewSource(seed))
				for range opsPerWorker {
					senderIdx := rng.Intn(numSenders)
					nonce := uint64(rng.Intn(maxNonce))
					priority := int64(rng.Intn(10_000) + 1)
					tx := newTestTxWithPriv(privs[senderIdx], nonce, uint64(1000+nonce), "default")

					if rng.Intn(100) < insertRatioPct {
						_ = mp.Insert(ctxWithPriority(priority), tx)
					} else {
						_ = mp.Remove(tx)
					}
				}
			}(seed)
		}
		wg.Wait()

		require.Eventually(t, func() bool { return reactorMatchesActiveSet(mp, reactor) },
			5*time.Second, 10*time.Millisecond,
			"round %d: reactor state must converge to active pool state", round)
	}
}

// Nonce gap prevention tests
func TestPromoteQueuedRequeuesOnCapacityExhaustion(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	// MaxTx=3
	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 3}, testTxEncoder)
	mp.SetAccountKeeper(keeper)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	privC := secp256k1.GenPrivKey()
	senderA := sdk.AccAddress(privA.PubKey().Address())
	senderB := sdk.AccAddress(privB.PubKey().Address())
	senderC := sdk.AccAddress(privC.PubKey().Address())
	keeper.SetSequence(senderA, 0)
	keeper.SetSequence(senderB, 0)
	keeper.SetSequence(senderC, 0)

	// A seq 0 active (pri=50), queued seq 2, 3, 4 (pri=10)
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(50)), newTestTxWithPriv(privA, 0, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 2, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 3, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 4, 1000, "default")))

	// B seq 0 active (pri=200)
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(200)), newTestTxWithPriv(privB, 0, 1000, "default")))
	// C seq 0 active (pri=200)
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(200)), newTestTxWithPriv(privC, 0, 1000, "default")))

	// pool has active 3 (A=0, B=0, C=0) at capacity. A has queued 2, 3, 4
	require.Equal(t, 6, mp.CountTx())

	// advancing A on-chain to 2 so seq 2, 3, 4 become promotable
	keeper.SetSequence(senderA, 2)
	mp.PromoteQueued(sdkCtx)

	// A=2 (pri=10) since we can't evict B=0 or C=0 (both pri=200), so promotion fails. all three are requeued not lost.
	_, ok2 := mp.Lookup(senderA.String(), 2)
	_, ok3 := mp.Lookup(senderA.String(), 3)
	_, ok4 := mp.Lookup(senderA.String(), 4)
	require.True(t, ok2, "seq 2 should be re-queued after failed promotion, not lost")
	require.True(t, ok3, "seq 3 should be re-queued after failed promotion, not lost")
	require.True(t, ok4, "seq 4 should be re-queued after failed promotion, not lost")

	// activeNext should be rolled back to 2
	next, ok, _ := mp.NextExpectedSequence(senderA.String())
	require.True(t, ok)
	require.Equal(t, uint64(2), next, "activeNext should roll back to first failed promotion nonce")

	// nothing was lost
	require.Equal(t, 6, mp.CountTx(), "no entries should be lost")
}

func TestInsertGapFillRequeuesOnCapacityExhaustion(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	// MaxTx=3. here we have A=0 and B=0 filling 2 slots. A=1 gap fill uses the 3rd slot, ultimately A=2 can't fit
	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 3}, testTxEncoder)
	mp.SetAccountKeeper(keeper)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	senderA := sdk.AccAddress(privA.PubKey().Address())
	senderB := sdk.AccAddress(privB.PubKey().Address())
	keeper.SetSequence(senderA, 0)
	keeper.SetSequence(senderB, 0)

	// B seq 0 active (pri=200)
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(200)), newTestTxWithPriv(privB, 0, 1000, "default")))

	// A= seq 0 active (pri=100), seq 2, 3 queued (pri=10)
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(100)), newTestTxWithPriv(privA, 0, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 2, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 3, 1000, "default")))

	// the pool now has active 2 (A:0, B:0). A has queued 2, 3. only 1 slot free remaining.

	// inserting A=1 (gap fill, pri=100). A=1 gets the last slot.
	// then A=2 tries to promote here, but since the pool is full, and we can't evict B=0 (pri=200).
	// expecting a requeued, no nonce gap.
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(100)), newTestTxWithPriv(privA, 1, 1000, "default")))

	// verify no entries were lost
	_, ok2 := mp.Lookup(senderA.String(), 2)
	_, ok3 := mp.Lookup(senderA.String(), 3)
	require.True(t, ok2, "seq 2 should still exist (re-queued after capacity exhaustion)")
	require.True(t, ok3, "seq 3 should still exist (re-queued after capacity exhaustion)")

	// total should be A=0, A=1, B=0 active + A=2, A=3. final queued = 5
	require.Equal(t, 5, mp.CountTx(), "no entries should be lost due to capacity")

	// activeNext should be at 2 (first requeued nonce)
	next, ok, _ := mp.NextExpectedSequence(senderA.String())
	require.True(t, ok)
	require.Equal(t, uint64(2), next, "activeNext should roll back to first failed promotion nonce")
}

func TestInsertActiveClearsRequeuedSameNonceFromQueued(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	// MaxTx=3 to force promotion failure and requeue.
	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 3}, testTxEncoder)
	mp.SetAccountKeeper(keeper)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	privC := secp256k1.GenPrivKey()
	senderA := sdk.AccAddress(privA.PubKey().Address())
	senderB := sdk.AccAddress(privB.PubKey().Address())
	senderC := sdk.AccAddress(privC.PubKey().Address())
	keeper.SetSequence(senderA, 0)
	keeper.SetSequence(senderB, 0)
	keeper.SetSequence(senderC, 0)

	// A: active 0 + queued 2,3,4.
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(50)), newTestTxWithPriv(privA, 0, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 2, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 3, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 4, 1000, "default")))
	// Fill active capacity with stronger txs from other senders.
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(200)), newTestTxWithPriv(privB, 0, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(200)), newTestTxWithPriv(privC, 0, 1000, "default")))
	require.Equal(t, 6, mp.CountTx())

	// Force promotion attempt to fail and requeue A:2,3,4 with activeNext=2.
	keeper.SetSequence(senderA, 2)
	mp.PromoteQueued(sdkCtx)

	// Insert A:2 as active; it should not remain in queued.
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(300)), newTestTxWithPriv(privA, 2, 1000, "default")))

	queuedNonces := map[uint64]struct{}{}
	mp.IterateQueuedTxs(func(sender string, nonce uint64, _ sdk.Tx) bool {
		if sender == senderA.String() {
			queuedNonces[nonce] = struct{}{}
		}
		return true
	})
	_, stillQueued := queuedNonces[2]
	require.False(t, stillQueued, "nonce 2 should not remain queued after active insert")
	require.Equal(t, 5, mp.CountTx(), "A:2 should move from queued to active without duplicate accounting")
}

func TestInsertActiveRejectPreservesQueuedEntry(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	// MaxTx=2
	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 2}, testTxEncoder)
	mp.SetAccountKeeper(keeper)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	senderA := sdk.AccAddress(privA.PubKey().Address())
	senderB := sdk.AccAddress(privB.PubKey().Address())
	keeper.SetSequence(senderA, 0)
	keeper.SetSequence(senderB, 0)

	// first A inserts nonce0 (active, priority 50) + A inserts nonce2 (queued, priority 10)
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(50)), newTestTxWithPriv(privA, 0, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(10)), newTestTxWithPriv(privA, 2, 1000, "default")))
	// then B inserts nonce0 (active, priority 200) fills the pool.
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(200)), newTestTxWithPriv(privB, 0, 1000, "default")))

	require.Equal(t, 3, mp.CountTx()) // 2 active + 1 queued

	// advancing on-chain seq so PromoteQueued attempts to promote A nonce2.
	// but promotion fails because the active pool is full and A nonce2 (priority 10)
	// can't evict anyone. requeueEntriesLocked puts it back, activeNext rolls to 2.
	keeper.SetSequence(senderA, 2)
	mp.PromoteQueued(sdkCtx)

	// sanity check, A nonce2 should still be queued after failed promotion
	_, ok := mp.Lookup(senderA.String(), 2)
	require.True(t, ok, "nonce 2 must still exist after failed promotion")
	countBefore := mp.CountTx()

	// now try inserting A nonce2 with priority 20, which is higher than the queued entry (10) but
	// lower than all active entries (50, 200). canAcceptLocked should reject.
	err := mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(20)), newTestTxWithPriv(privA, 2, 2000, "default"))
	require.ErrorIs(t, err, sdkmempool.ErrMempoolTxMaxCapacity)

	// the original queued entry must survive the failed insert.
	_, ok = mp.Lookup(senderA.String(), 2)
	require.True(t, ok, "queued entry at nonce 2 must not be lost when active insert is rejected")
	require.Equal(t, countBefore, mp.CountTx(), "CountTx must not change on rejected insert")
}

func TestReactorInvariant_PromotionCapacityRequeue(t *testing.T) {
	keeper := newMockAccountKeeper()
	sdkCtx := testSDKContext()

	// MaxTx=2
	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 2}, testTxEncoder)
	mp.SetAccountKeeper(keeper)
	eventCh := make(chan cmtmempool.AppMempoolEvent, 100)
	mp.SetEventCh(eventCh)
	reactor := newReactorSim(eventCh)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	senderA := sdk.AccAddress(privA.PubKey().Address())
	senderB := sdk.AccAddress(privB.PubKey().Address())
	keeper.SetSequence(senderA, 0)
	keeper.SetSequence(senderB, 0)

	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(100)), newTestTxWithPriv(privA, 0, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(50)), newTestTxWithPriv(privA, 2, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(50)), newTestTxWithPriv(privA, 3, 1000, "default")))
	require.NoError(t, mp.Insert(sdk.WrapSDKContext(sdkCtx.WithPriority(200)), newTestTxWithPriv(privB, 0, 1000, "default")))

	reactor.drain()
	require.Equal(t, activeCount(mp), len(reactor.insertedTxs),
		"reactor should match active pool before promotion attempt")

	// attempt promotion that will fail due to capacity
	keeper.SetSequence(senderA, 2)
	mp.PromoteQueued(sdkCtx)

	reactor.drain()
	require.Equal(t, activeCount(mp), len(reactor.insertedTxs),
		"reactor should still match active pool after failed promotion + requeue")
	require.True(t, reactor.hasValidTxs, "active txs still exist")

	// now free capacity and promote
	require.NoError(t, mp.Remove(newTestTxWithPriv(privB, 0, 1000, "default")))
	mp.PromoteQueued(sdkCtx)

	reactor.drain()
	require.Equal(t, activeCount(mp), len(reactor.insertedTxs),
		"reactor should match after successful promotion")
}

func TestQueuedMempoolLookupQueued(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	mp, _ := newTestMempoolWithKeeper(t, keeper)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Insert seq 0 (active) and seq 3 (queued)
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")))

	// Lookup active tx
	hash, ok := mp.Lookup(sender.String(), 0)
	require.True(t, ok)
	require.NotEmpty(t, hash)

	// Lookup queued tx
	hash, ok = mp.Lookup(sender.String(), 3)
	require.True(t, ok)
	require.NotEmpty(t, hash)

	// Lookup non-existent
	_, ok = mp.Lookup(sender.String(), 99)
	require.False(t, ok)
}

// BenchmarkPromoteQueued benchmarks PromoteQueued with 100 users each having 10 txs (1000 total).
// nonces 1-9 are inserted first (queued), then nonce 0 fills the gap (active and auto promote chain)
func BenchmarkPromoteQueued(b *testing.B) {
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	const numUsers = 100
	const txsPerUser = 10

	privs := make([]cryptotypes.PrivKey, numUsers)
	for i := 0; i < numUsers; i++ {
		privs[i] = secp256k1.GenPrivKey()
	}

	txs := make([][]sdk.Tx, numUsers)
	for i := 0; i < numUsers; i++ {
		txs[i] = make([]sdk.Tx, txsPerUser)
		for seq := uint64(0); seq < txsPerUser; seq++ {
			txs[i][seq] = newTestTxWithPriv(privs[i], seq, 1000, "default")
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		keeper := newMockAccountKeeper()
		mp := NewPriorityMempool(PriorityMempoolConfig{
			MaxTx:              numUsers * txsPerUser,
			MaxQueuedPerSender: txsPerUser,
			MaxQueuedTotal:     numUsers * txsPerUser,
		}, testTxEncoder)
		mp.SetAccountKeeper(keeper)

		for j := 0; j < numUsers; j++ {
			sender := sdk.AccAddress(privs[j].PubKey().Address())
			keeper.SetSequence(sender, 0)
		}

		for j := 0; j < numUsers; j++ {
			for seq := uint64(1); seq < txsPerUser; seq++ {
				if err := mp.Insert(ctx, txs[j][seq]); err != nil {
					b.Fatalf("insert failed user=%d seq=%d: %v", j, seq, err)
				}
			}
		}

		for j := 0; j < numUsers; j++ {
			sender := sdk.AccAddress(privs[j].PubKey().Address())
			keeper.SetSequence(sender, txsPerUser)
		}

		b.StartTimer()
		mp.PromoteQueued(ctx)
	}
}

type mockAccountKeeper struct {
	sequences map[string]uint64
}

func newMockAccountKeeper() *mockAccountKeeper {
	return &mockAccountKeeper{
		sequences: make(map[string]uint64),
	}
}

func (m *mockAccountKeeper) SetSequence(addr sdk.AccAddress, seq uint64) {
	m.sequences[string(addr.Bytes())] = seq
}

func (m *mockAccountKeeper) GetSequence(_ context.Context, addr sdk.AccAddress) (uint64, error) {
	key := string(addr.Bytes())
	seq, ok := m.sequences[key]
	if !ok {
		return 0, fmt.Errorf("sequence not found for %s", addr)
	}
	return seq, nil
}

type testBaseApp struct {
	ctx sdk.Context
}

func (b testBaseApp) GetContextForSimulate(_ []byte) sdk.Context {
	return b.ctx
}

type testAccountKeeper struct {
	accounts     map[string]sdk.AccountI
	params       authtypes.Params
	moduleAddrs  map[string]sdk.AccAddress
	addressCodec address.Codec
}

func newTestAccountKeeper(codec address.Codec) *testAccountKeeper {
	moduleAddr := sdk.AccAddress(bytes.Repeat([]byte{0xA}, 20))
	return &testAccountKeeper{
		accounts: make(map[string]sdk.AccountI),
		params:   authtypes.DefaultParams(),
		moduleAddrs: map[string]sdk.AccAddress{
			authtypes.FeeCollectorName: moduleAddr,
		},
		addressCodec: codec,
	}
}

func (k *testAccountKeeper) GetParams(ctx context.Context) authtypes.Params {
	return k.params
}

func (k *testAccountKeeper) GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	return k.accounts[addr.String()]
}

func (k *testAccountKeeper) SetAccount(ctx context.Context, acc sdk.AccountI) {
	k.accounts[acc.GetAddress().String()] = acc
}

func (k *testAccountKeeper) GetModuleAddress(name string) sdk.AccAddress {
	return k.moduleAddrs[name]
}

func (k *testAccountKeeper) AddressCodec() address.Codec {
	return k.addressCodec
}

func (k *testAccountKeeper) GetSequence(ctx context.Context, addr sdk.AccAddress) (uint64, error) {
	account := k.GetAccount(ctx, addr)
	if account == nil {
		return 0, fmt.Errorf("account not found for %s", addr)
	}
	return account.GetSequence(), nil
}

type testBankKeeper struct {
	balances    map[string]sdk.Coins
	moduleCoins map[string]sdk.Coins
}

func newTestBankKeeper() *testBankKeeper {
	return &testBankKeeper{
		balances:    make(map[string]sdk.Coins),
		moduleCoins: make(map[string]sdk.Coins),
	}
}

func (b *testBankKeeper) IsSendEnabledCoins(ctx context.Context, coins ...sdk.Coin) error {
	return nil
}

func (b *testBankKeeper) SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	if !b.hasBalance(from, amt) {
		return sdkerrors.ErrInsufficientFunds
	}
	b.debit(from, amt)
	b.balances[to.String()] = b.balance(to).Add(amt...)
	return nil
}

func (b *testBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	if !b.hasBalance(senderAddr, amt) {
		return sdkerrors.ErrInsufficientFunds
	}
	b.debit(senderAddr, amt)
	b.moduleCoins[recipientModule] = b.moduleCoins[recipientModule].Add(amt...)
	return nil
}

func (b *testBankKeeper) balance(addr sdk.AccAddress) sdk.Coins {
	coins := b.balances[addr.String()]
	if coins == nil {
		return sdk.NewCoins()
	}
	return coins
}

func (b *testBankKeeper) hasBalance(addr sdk.AccAddress, amt sdk.Coins) bool {
	return b.balance(addr).IsAllGTE(amt)
}

func (b *testBankKeeper) debit(addr sdk.AccAddress, amt sdk.Coins) {
	remaining := b.balance(addr).Sub(amt...)
	if remaining.IsZero() {
		delete(b.balances, addr.String())
		return
	}
	b.balances[addr.String()] = remaining
}
