package abcipp

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/core/address"
	txsigning "cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/tx/signing/direct"
	cmtmempool "github.com/cometbft/cometbft/mempool"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
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
	_, ok, err := mp.NextExpectedSequence(sdkCtx, sender.String())
	require.NoError(t, err)
	require.False(t, ok)

	// Insert seq 0 and activeNext becomes 1
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))

	next, ok, err := mp.NextExpectedSequence(sdkCtx, sender.String())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, uint64(1), next)

	// Insert seq 1 and activeNext becomes 2
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 1, 1000, "default")))

	next, ok, err = mp.NextExpectedSequence(sdkCtx, sender.String())
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
	next, ok, _ := mp.NextExpectedSequence(sdkCtx, senderA.String())
	require.True(t, ok)
	require.Equal(t, uint64(1), next)

	// remove A active tx (simulate cleaning worker)
	txA0 := newTestTxWithPriv(privA, 0, 1000, "default")
	require.NoError(t, mp.Remove(txA0))

	// PromoteQueued. A has no pool entries and no queued, should be cleaned up
	mp.PromoteQueued(sdkCtx)

	_, ok, _ = mp.NextExpectedSequence(sdkCtx, senderA.String())
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
