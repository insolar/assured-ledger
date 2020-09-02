// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"io"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

func NewReceiveBuffer(regularLimit, priorityLimit, largeLimit int, dispatcher uniproto.Dispatcher) ReceiveBuffer {
	switch {
	case dispatcher == nil:
		panic(throw.IllegalValue())
	case regularLimit <= 0:
		panic(throw.IllegalValue())
	case priorityLimit < 0:
		panic(throw.IllegalValue())
	}
	var priorityBuf chan smallPacket
	if priorityLimit > 0 {
		priorityBuf = make(chan smallPacket, priorityLimit)
	}
	return ReceiveBuffer{
		priorityBuf:  priorityBuf,
		regularBuf:   make(chan smallPacket, regularLimit),
		largeSema:    synckit.NewSemaphore(largeLimit),
		largeTimeout: 10 * time.Second,
		dispatcher:   dispatcher,
	}
}

var _ uniproto.Receiver = ReceiveBuffer{}
var _ uniproto.Dispatcher = ReceiveBuffer{}

type ReceiveBuffer struct {
	priorityBuf  chan smallPacket
	regularBuf   chan smallPacket
	oob          uniproto.ProtocolSet
	priority     [uniproto.ProtocolTypeCount]uniproto.PacketSet
	dispatcher   uniproto.Dispatcher
	discardedFn  func(nwapi.Address, uniproto.ProtocolType)
	largeSema    synckit.Semaphore
	largeTimeout time.Duration
	hasWorkers   bool
}

// For initialization only
func (p *ReceiveBuffer) SetDiscardHandler(fn func(nwapi.Address, uniproto.ProtocolType)) {
	p.discardedFn = fn
}

// For initialization only
func (p *ReceiveBuffer) SetLargePacketQueueTimeout(d time.Duration) {
	p.largeTimeout = d
}

// For initialization only
func (p *ReceiveBuffer) SetOutOfBandProtocols(oob uniproto.ProtocolSet) {
	p.oob = oob
}

// For initialization only
func (p *ReceiveBuffer) SetOutOfBand(pt uniproto.ProtocolType, val bool) {
	p.oob = p.oob.Set(pt, val)
}

// For initialization only
func (p *ReceiveBuffer) SetPriorityPackets(pp [uniproto.ProtocolTypeCount]uniproto.PacketSet) {
	if p.priorityBuf == nil {
		panic(throw.IllegalState())
	}
	p.priority = pp
}

// For initialization only
func (p *ReceiveBuffer) SetPriorityByProtocol(pt uniproto.ProtocolType, val bool) {
	if p.priorityBuf == nil {
		panic(throw.IllegalState())
	}
	if val {
		p.priority[pt] = uniproto.AllPackets
	} else {
		p.priority[pt] = 0
	}
}

// For initialization only
func (p *ReceiveBuffer) SetPriority(pt uniproto.ProtocolType, pk uint8, val bool) {
	if p.priorityBuf == nil {
		panic(throw.IllegalState())
	}
	p.priority[pt] = p.priority[pt].Set(pk, val)
}

func (p ReceiveBuffer) Start(manager uniproto.PeerManager) {
	if !p.hasWorkers {
		panic(throw.IllegalState())
	}
	p.dispatcher.Start(manager)
}

func (p ReceiveBuffer) NextPulse(pr pulse.Range) {
	p.dispatcher.NextPulse(pr)
}

func (p ReceiveBuffer) Stop() {
	p.dispatcher.Stop()
}

func (p ReceiveBuffer) Seal() uniproto.Descriptors {
	return p.dispatcher.Seal()
}

func (p ReceiveBuffer) GetMode() uniproto.ConnectionMode {
	return p.dispatcher.GetMode()
}

func (p ReceiveBuffer) GetReceiver(uniproto.ProtocolType) uniproto.Receiver {
	return p
}

func (p ReceiveBuffer) ReceiveSmallPacket(rp *uniproto.ReceivedPacket, b []byte) {
	b = append([]byte(nil), b...) // make a copy

	pt := rp.Header.GetProtocolType()
	if p.oob.Has(pt) {
		if rec := p.dispatcher.GetReceiver(pt); rec != nil {
			go rec.ReceiveSmallPacket(rp, b)
		} else if p.discardedFn != nil {
			p.discardedFn(rp.From, pt)
		}
		return
	}

	buf := p.regularBuf
	if s := p.priority[pt]; s != 0 && p.priorityBuf != nil && s.Has(rp.Header.GetPacketType()) {
		buf = p.priorityBuf
	}

	if p.discardedFn != nil {
		select {
		case buf <- smallPacket{rp, b}:
			return
		default:
			p.discardedFn(rp.From, pt)
		}
	} else {
		buf <- smallPacket{rp, b}
	}
}

func (p ReceiveBuffer) ReceiveLargePacket(rp *uniproto.ReceivedPacket, preRead []byte, r io.LimitedReader) error {
	pt := rp.Header.GetProtocolType()
	if !p.oob.Has(pt) {
		if !p.largeSema.LockTimeout(p.largeTimeout) {
			if p.discardedFn != nil {
				p.discardedFn(rp.From, pt)
			}
			return throw.FailHere("timeout")
		}
		defer p.largeSema.Unlock()
	}

	switch rec := p.dispatcher.GetReceiver(pt); {
	case rec != nil:
		return rec.ReceiveLargePacket(rp, preRead, r)
	case p.discardedFn != nil:
		p.discardedFn(rp.From, pt)
	}
	return nil
}

// For initialization only todo: ???
func (p *ReceiveBuffer) RunWorkers(count int, priorityOnly bool) {
	switch {
	case !priorityOnly:
		for ; count > 0; count-- {
			p.hasWorkers = true
			go p.runWorker(p.priorityBuf, p.regularBuf)
		}
	case p.priorityBuf == nil:
		panic(throw.IllegalState())
	default:
		for ; count > 0; count-- {
			p.hasWorkers = true
			go p.runWorker(p.priorityBuf, nil)
		}
	}
}

func (p ReceiveBuffer) Close() {
	close(p.regularBuf)
	if p.priorityBuf != nil {
		close(p.priorityBuf)
	}
	p.largeSema.Close()
}

func (p ReceiveBuffer) runWorker(priority, regular <-chan smallPacket) {
	for {
		var call smallPacket
		ok := false
		select {
		case call, ok = <-priority:
		default:
			select {
			case call, ok = <-priority:
			case call, ok = <-regular:
			}
		}
		if !ok {
			return
		}

		pt := call.packet.Header.GetProtocolType()
		if receiver := p.dispatcher.GetReceiver(pt); receiver != nil {
			receiver.ReceiveSmallPacket(call.packet, call.b)
		} else if p.discardedFn != nil {
			p.discardedFn(call.packet.From, pt)
		}
	}
}

type smallPacket struct {
	packet *uniproto.ReceivedPacket
	b      []byte
}
