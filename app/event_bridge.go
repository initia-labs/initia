package app

import (
	cmtmempool "github.com/cometbft/cometbft/mempool"
	"github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/abcipp"
)

var _ abcipp.TxEventListener = (*cometEventBridge)(nil)

// cometEventBridge bridges PriorityMempool's TxEventListener callbacks into
// CometBFT's AppMempoolEvent channel so the reactor can track insertions and
// removals for gossip and cache management.
type cometEventBridge struct {
	txEncoder sdk.TxEncoder
	eventCh   chan<- cmtmempool.AppMempoolEvent
}

// OnTxInserted is called when a transaction is accepted into the PriorityMempool.
func (b *cometEventBridge) OnTxInserted(tx sdk.Tx) {
	txBytes, err := b.txEncoder(tx)
	if err != nil {
		return
	}
	cmtTx := types.Tx(txBytes)
	select {
	case b.eventCh <- cmtmempool.AppMempoolEvent{
		Type:  cmtmempool.EventTxInserted,
		TxKey: cmtTx.Key(),
		Tx:    cmtTx,
	}:
	default:
	}
}

// OnTxRemoved is called when a transaction is evicted from the PriorityMempool.
func (b *cometEventBridge) OnTxRemoved(tx sdk.Tx) {
	txBytes, err := b.txEncoder(tx)
	if err != nil {
		return
	}
	cmtTx := types.Tx(txBytes)
	select {
	case b.eventCh <- cmtmempool.AppMempoolEvent{
		Type:  cmtmempool.EventTxRemoved,
		TxKey: cmtTx.Key(),
	}:
	default:
	}
}
