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
	if p.done == 0 {
		defer atomic.CompareAndSwapUint32(&p.done, 0, 1) // set 1 when f() has returned 0
		p.done = f()
	}
}

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
