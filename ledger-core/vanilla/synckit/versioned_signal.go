// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package synckit

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

func NewVersionedSignal() VersionedSignal {
	return VersionedSignal{}
}

type VersionedSignal struct {
	signalVersion *SignalVersion // atomic
}

func (p *VersionedSignal) _signalVersion() *unsafe.Pointer {
	return (*unsafe.Pointer)((unsafe.Pointer)(&p.signalVersion))
}

func (p *VersionedSignal) NextBroadcast() {
	sv := (*SignalVersion)(atomic.SwapPointer(p._signalVersion(), nil))
	sv.signal()
}

func (p *VersionedSignal) BroadcastAndMark() *SignalVersion {
	nsv := newSignalVersion()
	sv := (*SignalVersion)(atomic.SwapPointer(p._signalVersion(), (unsafe.Pointer)(nsv)))
	sv.signal()
	return nsv
}

func (p *VersionedSignal) Mark() *SignalVersion {
	var nsv *SignalVersion
	for {
		sv := (*SignalVersion)(atomic.LoadPointer(p._signalVersion()))
		switch {
		case sv != nil:
			return sv
		case nsv == nil: // avoid repetitive new
			nsv = newSignalVersion()
		}
		if atomic.CompareAndSwapPointer(p._signalVersion(), nil, (unsafe.Pointer)(nsv)) {
			return nsv
		}
	}
}

func NewNeverSignal() *SignalVersion {
	return newSignalVersion()
}

func newSignalVersion() *SignalVersion {
	sv := SignalVersion{}
	sv.wg.Add(1)
	return &sv
}

type signalChannel = chan struct{}

type SignalVersion struct {
	next *SignalVersion
	wg   sync.WaitGroup // is cheaper than channel and doesn't need additional heap allocation
	c    *signalChannel // atomic
}

func (p *SignalVersion) _signalChannel() *unsafe.Pointer {
	return (*unsafe.Pointer)((unsafe.Pointer)(&p.c))
}

func (p *SignalVersion) getSignalChannel() *signalChannel {
	return (*signalChannel)(atomic.LoadPointer(p._signalChannel()))
}

func (p *SignalVersion) signal() {
	if p == nil {
		return
	}
	p.next.signal() // older signals must fire first

	var closedSignal *SignalChannel // explicit type decl to avoid passing of something wrong into unsafe.Pointer conversion
	closedSignal = &closedChan

	atomic.CompareAndSwapPointer(p._signalChannel(), nil, (unsafe.Pointer)(closedSignal))
	p.wg.Done()
}

func (p *SignalVersion) Wait() {
	if p == nil {
		return
	}

	p.wg.Wait()
}

func (p *SignalVersion) ChannelIf(choice bool, def <-chan struct{}) <-chan struct{} {
	if choice {
		return p.Channel()
	}
	return def
}

func (p *SignalVersion) Channel() <-chan struct{} {
	if p == nil {
		return ClosedChannel()
	}

	var wcp *signalChannel
	for {
		switch sc := p.getSignalChannel(); {
		case sc != nil:
			return *sc
		case wcp == nil:
			wcp = new(signalChannel)
			*wcp = make(signalChannel)
		}

		if atomic.CompareAndSwapPointer(p._signalChannel(), nil, (unsafe.Pointer)(wcp)) {
			go p.waitClose(wcp)
			return *wcp
		}
	}
}

func (p *SignalVersion) waitClose(wcp *signalChannel) {
	p.wg.Wait()
	close(*wcp)
}

func (p *SignalVersion) HasSignal() bool {
	if p == nil {
		return true
	}

	sc := p.getSignalChannel()
	if sc == nil {
		return false
	}
	select {
	case <-*sc:
		return true
	default:
		return false
	}
}
