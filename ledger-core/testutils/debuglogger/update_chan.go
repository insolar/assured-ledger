package debuglogger

import (
	"runtime"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
)

type updateChan struct {
	state  atomickit.Int
	events chan UpdateEvent
}

func (p *updateChan) send(event UpdateEvent) {
	if p.state.Add(1) > 0 {
		p.events <- event
	}
	p.state.Add(-1)
}

func (p *updateChan) close() {
	const lowThreshold int = -1<<30 // NB! to avoid underflow

	n := 0
	for {
		switch n = p.state.Load(); {
		case n < 0:
			return
		case !p.state.CompareAndSwap(n, lowThreshold + n):
			continue
		}
		break
	}

	for i := 0; p.state.Load() > lowThreshold; i++ {
		if i > 10 {
			time.Sleep(time.Millisecond)
		} else {
			runtime.Gosched()
		}

		select {
		case <- p.events:
		default:
		}
	}

	close(p.events)
}

func (p *updateChan) receive() (ev UpdateEvent, ok bool) {
	ev, ok = <- p.events
	return
}
