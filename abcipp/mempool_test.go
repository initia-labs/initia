package abcipp

import (
	"context"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

func TestPriorityMempoolRejectsOutOfOrderSequences(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 5)

	mp := NewPriorityMempool(PriorityMempoolConfig{
		MaxTx: 10,
	}, testTxEncoder)

	sdkCtx := testSDKContext()
	if _, _, err := mp.NextExpectedSequence(sdkCtx, sender.String()); err != nil {
		t.Fatalf("expected to fetch initial account sequence: %v", err)
	}
	ctx := sdk.WrapSDKContext(sdkCtx)

	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")); err != nil {
		t.Fatalf("failed to insert initial tx: %v", err)
	}

	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 7, 1000, "default")); err == nil {
		t.Fatalf("expected sequence gap to be rejected")
	}

	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 6, 1000, "default")); err != nil {
		t.Fatalf("failed to insert sequential tx: %v", err)
	}
}

func TestPriorityMempoolNextExpectedSequenceLifecycle(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	tx := newTestTxWithPriv(priv, 5, 1000, "default")

	if _, ok, err := mp.NextExpectedSequence(sdkCtx, tx.sender.String()); err != nil {
		t.Fatalf("fetch initial sequence: %v", err)
	} else if ok {
		t.Fatalf("expected no entry for sender before insert")
	}

	if err := mp.Insert(ctx, tx); err != nil {
		t.Fatalf("insert tx: %v", err)
	}

	if seq, ok, err := mp.NextExpectedSequence(sdkCtx, tx.sender.String()); err != nil {
		t.Fatalf("fetch expected sequence: %v", err)
	} else if !ok || seq != 6 {
		t.Fatalf("expected next sequence 6, got %d (ok=%t)", seq, ok)
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

	if _, ok, err := mp.NextExpectedSequence(sdkCtx, tx.sender.String()); err != nil {
		t.Fatalf("fetch after removal: %v", err)
	} else if ok {
		t.Fatalf("expected sender to reset after removal")
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
	if order[0] == order[1] {
		t.Fatalf("expected each tier once, got %v", order)
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

	if _, ok, err := mp.NextExpectedSequence(sdkCtx, tx1.sender.String()); err != nil {
		t.Fatalf("fetch after cleanup: %v", err)
	} else if ok {
		t.Fatalf("expected sender reset after cleanup")
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

func (b testBaseApp) GetContextForCheckTx(_ []byte) sdk.Context {
	return b.ctx
}

func (b testBaseApp) IsSealed() bool {
	return true
}
