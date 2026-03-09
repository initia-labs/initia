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

// SetAppEventCh stores an app-side event channel that receives all events.
// Unlike the cometbft channel (SetEventCh), this channel is not filtered.
// Apps use it for internal tracking.
func (p *PriorityMempool) SetAppEventCh(ch chan<- cmtmempool.AppMempoolEvent) {
	p.appEventCh.Store(&ch)
	// Wake the dispatch loop in case events were queued before channel wiring.
	select {
	case p.eventNotify <- struct{}{}:
	default:
	}
}

// StopEventDispatch signals the event dispatcher goroutine and waits for exit.
func (p *PriorityMempool) StopEventDispatch() {
	p.eventMu.Lock()
	select {
	case <-p.eventStop:
		p.eventMu.Unlock()
		return // already stopped
	default:
		close(p.eventStop)
	}
	p.eventMu.Unlock()
	<-p.eventDone
}

// enqueueEvent appends one event to the internal FIFO dispatch queue.
func (p *PriorityMempool) enqueueEvent(eventType cmtmempool.AppMempoolEventType, txBytes []byte) {
	if p.eventCh.Load() == nil && p.appEventCh.Load() == nil {
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

// eventDispatchLoop forwards queued app-mempool events to consumers.
// Events are dispatched to two channels:
//   - eventCh (cometbft): receives EventTxInserted and EventTxRemoved only.
//     EventTxQueued is filtered because ProxyMempool already emits its own
//     EventTxQueued to the reactor for gossip.
//   - appEventCh (app): receives all events including EventTxQueued.
//     Apps use this for internal tracking (e.g. txpool cache).
func (p *PriorityMempool) eventDispatchLoop() {
	defer close(p.eventDone)

	for {
		select {
		case <-p.eventStop:
			return
		case <-p.eventNotify:
		}

		cometChPtr := p.eventCh.Load()
		appChPtr := p.appEventCh.Load()
		if cometChPtr == nil && appChPtr == nil {
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

			// dispatch to cometbft (filter EventTxQueued to avoid double gossip)
			if cometChPtr != nil && ev.Type != cmtmempool.EventTxQueued {
				select {
				case *cometChPtr <- ev:
				case <-p.eventStop:
					return
				}
			}

			// dispatch to app (all events, non-blocking to avoid stalling cometbft)
			if appChPtr != nil {
				select {
				case *appChPtr <- ev:
				default:
				}
			}
		}
	}
}
