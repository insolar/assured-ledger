package uniserver

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
)

var _ uniproto.Dispatcher = &Dispatcher{}

// Dispatcher provides registration and management of protocols.
// Set of protocols can't be changed after Seal.
// But individual protocols can be enabled or disabled at any time with GetMode / SetMode functions.
type Dispatcher struct {
	mutex     sync.Mutex
	manager   uniproto.PeerManager
	mode      atomickit.Uint32
	state     atomickit.Uint32
	lastPulse pulse.Range

	controllers [uniproto.ProtocolTypeCount]uniproto.Controller
	receivers   [uniproto.ProtocolTypeCount]uniproto.Receiver
	descripts   uniproto.Descriptors
	registered  uniproto.ProtocolSet
	started     uniproto.ProtocolSet
}

func (p *Dispatcher) RegisterProtocolByFunc(fn uniproto.RegisterProtocolFunc) {
	fn(p.RegisterProtocol)
}

func (p *Dispatcher) RegisterProtocol(pt uniproto.ProtocolType, desc uniproto.Descriptor, ctl uniproto.Controller, rcv uniproto.Receiver) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch {
	case ctl == nil:
		panic(throw.IllegalValue())
	case rcv == nil:
		panic(throw.IllegalValue())
	case !desc.IsSupported():
		panic(throw.IllegalValue())
	case p.IsSealed():
		panic(throw.IllegalState())
	case p.controllers[pt] != nil:
		panic(throw.IllegalState())
	}
	p.receivers[pt] = rcv
	p.controllers[pt] = ctl
	p.descripts[pt] = desc
	p.registered = p.registered.Set(pt, true)
}

const (
	_ = iota
	sealed
	started
	stopped
)

// IsSealed returns true when set of protocols was sealed and can't be changed.
func (p *Dispatcher) IsSealed() bool {
	return p.state.Load() > 0
}

func (p *Dispatcher) IsStarted() bool {
	return p.state.Load() == started
}

func (p *Dispatcher) Seal() uniproto.Descriptors {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.registered == 0 {
		panic(throw.IllegalState())
	}
	p.state.CompareAndSwap(0, sealed)
	return p.descripts
}

func (p *Dispatcher) Start(manager uniproto.PeerManager) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch {
	case manager == nil:
		panic(throw.IllegalValue())
	case p.registered == 0:
		panic(throw.IllegalState())
	case p.manager != nil:
		panic(throw.IllegalState())
	case p.state.Load() > sealed:
		panic(throw.IllegalState())
	}

	p.manager = manager

	mode := p.GetMode()
	mode = mode.SetAllowedSet(mode.AllowedSet() & p.registered)
	mode = mode.SetAllowedSet(p._startProtocols(mode.AllowedSet()))
	p.mode.Store(uint32(mode))
	p.state.Store(started)
}

func (p *Dispatcher) NextPulse(pr pulse.Range) {
	p.mutex.Lock()
	p.lastPulse = pr
	started := p.started
	p.mutex.Unlock()

	started.ForEach(func(pt uniproto.ProtocolType) bool {
		p.controllers[pt].NextPulse(pr)
		return false
	})
}

func (p *Dispatcher) Stop() {
	started := p.stop()
	started.ForEach(func(pt uniproto.ProtocolType) bool {
		p.controllers[pt].Stop()
		return false
	})
}

func (p *Dispatcher) stop() uniproto.ProtocolSet {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	switch p.state.Load() {
	case stopped:
		return 0
	case started:
		p.state.Store(stopped)
	default:
		panic(throw.IllegalState())
	}
	p.mode.Store(0)
	started := p.started
	p.started = 0
	return started
}

func (p *Dispatcher) GetReceiver(pt uniproto.ProtocolType) uniproto.Receiver {
	if !p.IsSealed() {
		panic(throw.IllegalState())
	}
	return p.receivers[pt]
}

func (p *Dispatcher) GetDescriptors() uniproto.Descriptors {
	if !p.IsSealed() {
		panic(throw.IllegalState())
	}
	return p.descripts
}

func (p *Dispatcher) GetMode() uniproto.ConnectionMode {
	return uniproto.ConnectionMode(p.mode.Load())
}

// SetMode changes a set of available protocols.
// When Dispatcher was started, changing a set of allowed protocols will result in starting/stopping of relevant protocols.
func (p *Dispatcher) SetMode(mode uniproto.ConnectionMode) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.IsStarted() {
		allowed := p._startProtocols(mode.AllowedSet())
		mode = mode.SetAllowedSet(mode.AllowedSet() & allowed)
	}
	p.mode.Store(uint32(mode))
}

func (p *Dispatcher) _startProtocols(set uniproto.ProtocolSet) uniproto.ProtocolSet {
	set &= p.registered
	set &^= p.started

	set.ForEach(func(pt uniproto.ProtocolType) bool {
		p._startProtocol(pt)
		return false
	})

	return p.started
}

func (p *Dispatcher) _startProtocol(pt uniproto.ProtocolType) {
	p.started = p.started.Set(pt, true)
	ctl := p.controllers[pt]
	ctl.Start(p.manager)
	if p.lastPulse != nil {
		ctl.NextPulse(p.lastPulse)
	}
}
