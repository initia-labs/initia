package abcipp

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	cmtmempool "github.com/cometbft/cometbft/mempool"
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
