// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package atomickit

import (
	"runtime"
	"sync/atomic"
	"time"
)

const (
	starting = -1
	active   = 1
	stopping = 2
	stopped  = 3
)

type StartStopFlag struct {
	done int32
}

func (p *StartStopFlag) IsActive() bool {
	return atomic.LoadInt32(&p.done) == active
}

func (p *StartStopFlag) IsStarting() bool {
	return atomic.LoadInt32(&p.done) < 0
}

func (p *StartStopFlag) WasStarted() bool {
	return atomic.LoadInt32(&p.done) != 0
}

func (p *StartStopFlag) WasStopped() bool {
	return atomic.LoadInt32(&p.done) >= stopping
}

func (p *StartStopFlag) IsStopping() bool {
	return atomic.LoadInt32(&p.done) == stopping
}

func (p *StartStopFlag) IsStopped() bool {
	return atomic.LoadInt32(&p.done) == stopped
}

func (p *StartStopFlag) Status() (isActive, wasStarted bool) {
	n := atomic.LoadInt32(&p.done)
	return n == active, n != 0
}

func (p *StartStopFlag) DoStart(f func()) bool {
	if !atomic.CompareAndSwapInt32(&p.done, 0, starting) {
		return false
	}
	p.doSlow(f, active)
	return true
}

func (p *StartStopFlag) DoStop(f func()) bool {
	if !atomic.CompareAndSwapInt32(&p.done, active, stopping) {
		return false
	}
	p.doSlow(f, stopped)
	return true
}

func (p *StartStopFlag) DoDiscard(discardFn, stopFn func()) bool {
	for i := uint(0); ; i++ {
		switch atomic.LoadInt32(&p.done) {
		case 0:
			if atomic.CompareAndSwapInt32(&p.done, 0, stopping) {
				p.doSlow(discardFn, stopped)
				return true
			}
		case active:
			return p.DoStop(stopFn)
		case starting:
			spinWait(int(i))
		default:
			return false
		}
	}
}

func spinWait(spinCount int) {
	switch {
	case spinCount < 10:
		runtime.Gosched()
	case spinCount < 100:
		time.Sleep(time.Microsecond)
	default:
		time.Sleep(time.Millisecond)
	}
}

func (p *StartStopFlag) DoDiscardByOne(fn func(wasStarted bool)) bool {
	if fn == nil {
		return p.DoDiscard(nil, nil)
	}
	return p.DoDiscard(func() {
		fn(false)
	}, func() {
		fn(true)
	})
}

func (p *StartStopFlag) Start() bool {
	return atomic.CompareAndSwapInt32(&p.done, 0, active)
}

func (p *StartStopFlag) Stop() bool {
	return atomic.CompareAndSwapInt32(&p.done, active, stopped)
}

func (p *StartStopFlag) doSlow(f func(), status int32) {
	upd := int32(stopped)
	defer func() {
		atomic.StoreInt32(&p.done, upd)
	}()

	if f != nil {
		f()
	}
	upd = status
}
