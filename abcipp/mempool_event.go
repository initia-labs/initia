package abcipp

import (
	cmtmempool "github.com/cometbft/cometbft/mempool"
	cmttypes "github.com/cometbft/cometbft/types"
)

// SetEventCh stores the cometbft event channel for event dispatch.
func (p *PriorityMempool) SetEventCh(ch chan<- cmtmempool.AppMempoolEvent) {
	p.eventCh.Store(&ch)
	// Wake the dispatch loop in case events were queued before channel wiring.
	select {
	case p.eventNotify <- struct{}{}:
	default:
	}
}

// StopEventDispatch signals the event dispatcher goroutine and waits for exit.
func (p *PriorityMempool) StopEventDispatch() {
	select {
	case <-p.eventStop:
		return // already stopped
	default:
		close(p.eventStop)
	}
	<-p.eventDone
}

// enqueueEvent appends one event to the internal FIFO dispatch queue.
func (p *PriorityMempool) enqueueEvent(eventType cmtmempool.AppMempoolEventType, txBytes []byte) {
	if p.eventCh.Load() == nil {
		return
	}

	cmtTx := cmttypes.Tx(txBytes)
	ev := cmtmempool.AppMempoolEvent{
		Type:  eventType,
		TxKey: cmtTx.Key(),
		Tx:    cmtTx,
	}

	p.eventMu.Lock()
	select {
	case <-p.eventStop:
		p.eventMu.Unlock()
		return
	default:
	}
	p.eventQueue = append(p.eventQueue, ev)
	p.eventMu.Unlock()

	select {
	case p.eventNotify <- struct{}{}:
	default:
	}
}

// enqueueRemovedEvents appends EventTxRemoved for each removed tx entry.
func (p *PriorityMempool) enqueueRemovedEvents(entries []*txEntry) {
	for _, entry := range entries {
		p.enqueueEvent(cmtmempool.EventTxRemoved, entry.bytes)
	}
}

// eventDispatchLoop forwards queued app-mempool events to cometbft.
func (p *PriorityMempool) eventDispatchLoop() {
	defer close(p.eventDone)

	for {
		select {
		case <-p.eventStop:
			return
		case <-p.eventNotify:
		}

		chPtr := p.eventCh.Load()
		if chPtr == nil {
			continue
		}

		for {
			p.eventMu.Lock()
			if len(p.eventQueue) == 0 {
				p.eventMu.Unlock()
				break
			}
			ev := p.eventQueue[0]
			p.eventQueue[0] = cmtmempool.AppMempoolEvent{}
			p.eventQueue = p.eventQueue[1:]
			p.eventMu.Unlock()

			select {
			case *chPtr <- ev:
			case <-p.eventStop:
				return
			}
		}
	}
}
