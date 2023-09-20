package synckit

import (
	"context"
	"sync/atomic"
)

func NewChainedCancel() *ChainedCancel {
	return &ChainedCancel{}
}

type ChainedCancel struct {
	state uint32 // atomic
	chain atomic.Value
}

const (
	stateCancelled = 1 << iota
	stateChainHandlerBeingSet
	stateChainHandlerSet
)

func (p *ChainedCancel) Cancel() {
	if p == nil {
		return
	}
	for {
		lastState := atomic.LoadUint32(&p.state)
		switch {
		case lastState&stateCancelled != 0:
			return
		case !atomic.CompareAndSwapUint32(&p.state, lastState, lastState|stateCancelled):
			continue
		case lastState == stateChainHandlerSet:
			p.runChain()
		}
		return
	}
}

func (p *ChainedCancel) runChain() {
	// here is a potential problem, because Go spec doesn't provide ANY ordering on atomic operations
	// but Go compiler does provide some guarantees, so lets hope for the best

	fn := (p.chain.Load()).(context.CancelFunc)
	if fn == nil {
		// this can only happen when atomic ordering is broken
		panic("unexpected atomic ordering")
	}

	// prevent repeated calls as well as retention of references & possible memory leaks
	// type cast is mandatory due to specifics of atomic.Value
	p.chain.Store(context.CancelFunc(func() {}))
	fn()
}

func (p *ChainedCancel) IsCancelled() bool {
	return p != nil && atomic.LoadUint32(&p.state)&stateCancelled != 0
}

/*
	SetChain sets a chained function once.
	The chained function can only be set once to a non-null value, further calls will panic.
    But if the chained function was not set, the SetChain(nil) can be called multiple times.

	The chained function is guaranteed to be called only once, it will also be called on set when IsCancelled is already true.
*/
func (p *ChainedCancel) SetChain(chain context.CancelFunc) {
	if chain == nil {
		if p.chain.Load() == nil {
			return
		}
		panic("illegal state")
	}
	for {
		lastState := atomic.LoadUint32(&p.state)
		switch {
		case lastState&^stateCancelled != 0: // chain is set or being set
			panic("illegal state")
		case !atomic.CompareAndSwapUint32(&p.state, lastState, lastState|stateChainHandlerBeingSet): //
			continue
		}
		break
	}

	p.chain.Store(chain)

	for {
		lastState := atomic.LoadUint32(&p.state)
		switch {
		case lastState&^stateCancelled != stateChainHandlerBeingSet:
			// this can only happen when atomic ordering is broken
			panic("unexpected atomic ordering")
		case !atomic.CompareAndSwapUint32(&p.state, lastState, (lastState&stateCancelled)|stateChainHandlerSet):
			continue
		case lastState&stateCancelled != 0:
			// if cancel was set then call the chained cancel fn here
			// otherwise, the cancelling process will be responsible to call the chained cancel
			p.runChain()
		}
		return
	}
}
