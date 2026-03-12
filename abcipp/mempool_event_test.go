package abcipp

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	cmtmempool "github.com/cometbft/cometbft/mempool"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestStopEventDispatchIsIdempotent(t *testing.T) {
	mp, _, _, _ := newTestMempoolWithEvents(t, 8)

	require.NotPanics(t, func() {
		mp.StopEventDispatch()
	})
	require.NotPanics(t, func() {
		mp.StopEventDispatch() // already stopped
	})
}

// TestAppEventChReceivesAllEvents verifies that the app channel receives all
// event types (including EventTxQueued) while the comet channel filters it out.
func TestAppEventChReceivesAllEvents(t *testing.T) {
	keeper := newMockAccountKeeper()
	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 32}, log.NewNopLogger(), testTxEncoder, keeper)
	t.Cleanup(func() { mp.StopEventDispatch() })

	cometCh := make(chan cmtmempool.AppMempoolEvent, 16)
	appCh := make(chan cmtmempool.AppMempoolEvent, 16)
	mp.SetEventCh(cometCh)
	mp.SetAppEventCh(appCh)

	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	// Insert nonce 0 (active) — emits EventTxInserted on both channels.
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))

	// Insert nonce 2 (queued, future) — emits EventTxQueued only on app channel.
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 2, 1000, "default")))

	collectN := func(ch chan cmtmempool.AppMempoolEvent, n int) []cmtmempool.AppMempoolEventType {
		types := make([]cmtmempool.AppMempoolEventType, 0, n)
		deadline := time.After(500 * time.Millisecond)
		for len(types) < n {
			select {
			case ev := <-ch:
				types = append(types, ev.Type)
			case <-deadline:
				return types
			}
		}
		return types
	}

	// Comet channel: only EventTxInserted (EventTxQueued is filtered).
	cometTypes := collectN(cometCh, 1)
	require.Equal(t, []cmtmempool.AppMempoolEventType{cmtmempool.EventTxInserted}, cometTypes)
	require.Empty(t, cometCh, "comet channel must not receive EventTxQueued")

	// App channel: EventTxInserted then EventTxQueued, no filtering.
	appTypes := collectN(appCh, 2)
	require.ElementsMatch(t,
		[]cmtmempool.AppMempoolEventType{cmtmempool.EventTxInserted, cmtmempool.EventTxQueued},
		appTypes,
	)
}

// TestAppEventChIndependentFromCometCh verifies that a blocked app channel
// does not stall delivery to the comet channel, and vice-versa.
func TestAppEventChIndependentFromCometCh(t *testing.T) {
	keeper := newMockAccountKeeper()
	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 64}, log.NewNopLogger(), testTxEncoder, keeper)
	t.Cleanup(func() { mp.StopEventDispatch() })

	// cometCh is buffered; appCh is unbuffered (always full / blocking).
	cometCh := make(chan cmtmempool.AppMempoolEvent, 32)
	appCh := make(chan cmtmempool.AppMempoolEvent) // unbuffered — blocks until read
	mp.SetEventCh(cometCh)
	mp.SetAppEventCh(appCh)

	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	const n = 10
	privs := make([]*secp256k1.PrivKey, n)
	for i := range privs {
		p := secp256k1.GenPrivKey()
		privs[i] = p
		keeper.SetSequence(sdk.AccAddress(p.PubKey().Address()), 0)
	}

	// Insert n txs (one per sender so no queuing).
	for _, p := range privs {
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(p, 0, 1000, "default")))
	}

	// Comet channel should receive all n events promptly even though nobody
	// is reading from appCh.
	deadline := time.After(500 * time.Millisecond)
	received := 0
	for received < n {
		select {
		case <-cometCh:
			received++
		case <-deadline:
			t.Fatalf("comet channel stalled by blocked app channel: got %d/%d events", received, n)
		}
	}
}

func TestEnqueueEventAfterStopIsIgnored(t *testing.T) {
	mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 8}, log.NewNopLogger(), testTxEncoder, newMockAccountKeeper())
	eventCh := make(chan cmtmempool.AppMempoolEvent, 8)
	mp.SetEventCh(eventCh)

	mp.StopEventDispatch()

	// After dispatcher stop, enqueue must be ignored and channel stays empty.
	mp.enqueueEvent(cmtmempool.EventTxInserted, []byte("stopped"))
	select {
	case <-eventCh:
		t.Fatal("unexpected event emitted after dispatcher stop")
	case <-time.After(20 * time.Millisecond):
	}
}
