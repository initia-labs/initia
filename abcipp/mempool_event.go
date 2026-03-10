package abcipp

import (
	"sync"

	cmtmempool "github.com/cometbft/cometbft/mempool"
	cmttypes "github.com/cometbft/cometbft/types"
)

// SetEventCh stores the cometbft event channel for event dispatch.
func (p *PriorityMempool) SetEventCh(ch chan<- cmtmempool.AppMempoolEvent) {
	p.eventCh.Store(&ch)
	// Wake the comet dispatch goroutine in case events were queued before channel wiring.
	select {
	case p.cometNotify <- struct{}{}:
	default:
	}
}

// SetAppEventCh stores an app-side event channel that receives all events.
// Unlike the cometbft channel (SetEventCh), this channel is not filtered.
// Apps use it for internal tracking.
func (p *PriorityMempool) SetAppEventCh(ch chan<- cmtmempool.AppMempoolEvent) {
	p.appEventCh.Store(&ch)
	// Wake the app dispatch goroutine in case events were queued before channel wiring.
	select {
	case p.appNotify <- struct{}{}:
	default:
	}
}

// StopEventDispatch signals the event dispatcher goroutines and waits for exit.
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

// enqueueEvent appends one event to the relevant internal FIFO queues.
func (p *PriorityMempool) enqueueEvent(eventType cmtmempool.AppMempoolEventType, txBytes []byte) {
	hasCometCh := p.eventCh.Load() != nil
	hasAppCh := p.appEventCh.Load() != nil
	if !hasCometCh && !hasAppCh {
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
	// EventTxQueued is filtered from the comet channel to avoid double gossip.
	if hasCometCh && eventType != cmtmempool.EventTxQueued {
		p.cometQueue = append(p.cometQueue, ev)
	}
	if hasAppCh {
		p.appQueue = append(p.appQueue, ev)
	}
	p.eventMu.Unlock()

	if hasCometCh && eventType != cmtmempool.EventTxQueued {
		select {
		case p.cometNotify <- struct{}{}:
		default:
		}
	}
	if hasAppCh {
		select {
		case p.appNotify <- struct{}{}:
		default:
		}
	}
}

// enqueueRemovedEvents appends EventTxRemoved for each removed tx entry.
func (p *PriorityMempool) enqueueRemovedEvents(entries []*txEntry) {
	for _, entry := range entries {
		p.enqueueEvent(cmtmempool.EventTxRemoved, entry.bytes)
	}
}

// eventDispatchLoop launches two independent goroutines:
//   - cometbft dispatcher: receives EventTxInserted and EventTxRemoved only.
//     EventTxQueued is filtered because ProxyMempool already emits its own
//     EventTxQueued to the reactor for gossip.
//   - app dispatcher: receives all events including EventTxQueued.
//     Apps use this for internal tracking (e.g. txpool cache).
//
// The two goroutines drain separate queues, so a slow consumer on one side
// never blocks the other.
func (p *PriorityMempool) eventDispatchLoop() {
	defer close(p.eventDone)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); p.cometDispatchLoop() }()
	go func() { defer wg.Done(); p.appDispatchLoop() }()
	wg.Wait()
}

func (p *PriorityMempool) cometDispatchLoop() {
	for {
		select {
		case <-p.eventStop:
			return
		case <-p.cometNotify:
		}

		chPtr := p.eventCh.Load()
		if chPtr == nil {
			continue
		}

		for {
			p.eventMu.Lock()
			if len(p.cometQueue) == 0 {
				p.eventMu.Unlock()
				break
			}
			ev := p.cometQueue[0]
			p.cometQueue[0] = cmtmempool.AppMempoolEvent{}
			p.cometQueue = p.cometQueue[1:]
			p.eventMu.Unlock()

			select {
			case *chPtr <- ev:
			case <-p.eventStop:
				return
			}
		}
	}
}

func (p *PriorityMempool) appDispatchLoop() {
	for {
		select {
		case <-p.eventStop:
			return
		case <-p.appNotify:
		}

		chPtr := p.appEventCh.Load()
		if chPtr == nil {
			continue
		}

		for {
			p.eventMu.Lock()
			if len(p.appQueue) == 0 {
				p.eventMu.Unlock()
				break
			}
			ev := p.appQueue[0]
			p.appQueue[0] = cmtmempool.AppMempoolEvent{}
			p.appQueue = p.appQueue[1:]
			p.eventMu.Unlock()

			select {
			case *chPtr <- ev:
			case <-p.eventStop:
				return
			}
		}
	}
}
