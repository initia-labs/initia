package abcipp

import (
	"context"
	"fmt"
	"testing"
	"time"

	cmtmempool "github.com/cometbft/cometbft/mempool"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// drainEvents reads buffered app-mempool events and returns inserted/removed counts.
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

// collectEvents returns buffered inserted/removed event tx bytes.
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

// collectNEvents waits for exactly n events or fails on timeout.
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

// encodeTx serializes a test tx and fails the test if encoding fails.
func encodeTx(t *testing.T, tx sdk.Tx) []byte {
	t.Helper()
	bz, err := testTxEncoder(tx)
	require.NoError(t, err)
	return bz
}

// activeCount counts only active (selectable) txs in the mempool.
func activeCount(mp *PriorityMempool) int {
	count := 0
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		count++
	}
	return count
}

// assertInvariant fails the test immediately if mempool invariants are broken.
func assertInvariant(t *testing.T, mp *PriorityMempool) {
	t.Helper()
	require.NotPanics(t, func() {
		mp.AssertInvariant()
	})
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
