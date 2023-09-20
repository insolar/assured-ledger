package synckit

import (
	"sync"
)

func NewSyncQueue(locker sync.Locker) SyncQueue {
	if locker == nil {
		panic("illegal value")
	}
	return SyncQueue{locker: locker}
}

func NewSignalCondQueue(signal *sync.Cond) SyncQueue {
	if signal == nil {
		panic("illegal value")
	}
	return SyncQueue{locker: signal.L, signalFn: signal.Broadcast}
}

func NewSignalFuncQueue(locker sync.Locker, signalFn func()) SyncQueue {
	if locker == nil {
		panic("illegal value")
	}
	return SyncQueue{locker: locker, signalFn: signalFn}
}

func NewNoSyncQueue() SyncQueue {
	return SyncQueue{locker: DummyLocker()}
}

type SyncFunc func(interface{})
type SyncFuncList []SyncFunc

type SyncQueue struct {
	locker   sync.Locker
	signalFn func()
	queue    SyncFuncList
}

func (p *SyncQueue) Locker() sync.Locker {
	return p.locker
}

func (p *SyncQueue) IsZero() bool {
	return p.locker == nil
}

func (p *SyncQueue) Add(fn SyncFunc) {
	if fn == nil {
		panic("illegal value")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	p.queue = append(p.queue, fn)
	if p.signalFn != nil {
		p.signalFn()
	}
}

func (p *SyncQueue) Flush() SyncFuncList {
	p.locker.Lock()
	defer p.locker.Unlock()

	if len(p.queue) == 0 {
		return nil
	}

	nextCap := cap(p.queue)
	if nextCap > 128 && len(p.queue)<<1 < nextCap {
		nextCap >>= 1
	}
	queue := p.queue
	p.queue = make(SyncFuncList, 0, nextCap)

	return queue
}

func (p *SyncQueue) AddAll(list SyncFuncList) {
	if len(list) == 0 {
		return
	}

	p.locker.Lock()
	defer p.locker.Unlock()

	p.queue = append(p.queue, list...)
}
