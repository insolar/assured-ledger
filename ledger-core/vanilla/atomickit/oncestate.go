/*
 * Copyright 2020 Insolar Network Ltd.
 * All rights reserved.
 * This material is licensed under the Insolar License version 1.0,
 * available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package atomickit

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Once struct {
	done uint32
	m    sync.Mutex
}

func (p *Once) IsDone() bool {
	return atomic.LoadUint32(&p.done) != 0
}

func (p *Once) GetDone() uint32 {
	return atomic.LoadUint32(&p.done)
}

func (p *Once) Do(f func()) {
	if atomic.LoadUint32(&p.done) == 0 {
		p.doSlow(func() uint32 {
			f()
			return 1
		})
	}
}

func (p *Once) DoWithValue(f func() uint32) {
	if atomic.LoadUint32(&p.done) == 0 {
		p.doSlow(f)
	}
}

func (p *Once) doSlow(f func() uint32) {
	p.m.Lock()
	defer p.m.Unlock()
	if atomic.LoadUint32(&p.done) == 0 {
		defer atomic.CompareAndSwapUint32(&p.done, 0, 1) // set 1 when f() has returned 0
		atomic.StoreUint32(&p.done, f())
	}
}

/***********************************************************/

type OnceFlag struct {
	done int32
}

func (p *OnceFlag) IsSet() bool {
	return atomic.LoadInt32(&p.done) == 1
}

func (p *OnceFlag) Set() bool {
	return atomic.CompareAndSwapInt32(&p.done, 0, 1)
}

func (p *OnceFlag) DoSet(f func()) bool {
	if !atomic.CompareAndSwapInt32(&p.done, 0, -1) {
		return false
	}
	p.doSlow(f)
	return true
}

func (p *OnceFlag) DoSpin(f func()) bool {
	if !atomic.CompareAndSwapInt32(&p.done, 0, -1) {
		for !p.IsSet() {
			runtime.Gosched()
		}
		return false
	}
	p.doSlow(f)
	return true
}

func (p *OnceFlag) doSlow(f func()) {
	defer atomic.StoreInt32(&p.done, 1)
	f()
}

/***********************************************************/

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
			switch {
			case i < 10:
				runtime.Gosched()
			case i < 100:
				time.Sleep(time.Microsecond)
			default:
				time.Sleep(time.Millisecond)
			}
		default:
			return false
		}
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
