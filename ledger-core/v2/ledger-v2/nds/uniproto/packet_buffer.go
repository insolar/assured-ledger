// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"errors"
	"io"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewReceiveBuffer(regularLimit, priorityLimit, largeLimit int, defReceiver Receiver) ReceiveBuffer {
	switch {
	case defReceiver == nil:
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
		priorityBuf: priorityBuf,
		regularBuf:  make(chan smallPacket, regularLimit),
		largeSema:   synckit.NewSemaphore(largeLimit),
		defReceiver: defReceiver,
	}
}

var _ Receiver = ReceiveBuffer{}

type ReceiveBuffer struct {
	priorityBuf  chan smallPacket
	regularBuf   chan smallPacket
	oob          ProtocolSet
	priority     [ProtocolTypeCount]PacketSet
	receivers    Receivers
	defReceiver  Receiver
	discardedFn  func(apinetwork.Address, ProtocolType)
	largeSema    synckit.Semaphore
	largeTimeout time.Duration
}

// For initialization only
func (p *ReceiveBuffer) SetDiscardHandler(fn func(apinetwork.Address, ProtocolType)) {
	p.discardedFn = fn
}

// For initialization only
func (p *ReceiveBuffer) SetLargePacketQueueTimeout(d time.Duration) {
	p.largeTimeout = d
}

// For initialization only
func (p *ReceiveBuffer) SetOutOfBandProtocols(oob ProtocolSet) {
	p.oob = oob
}

// For initialization only
func (p *ReceiveBuffer) SetOutOfBand(pt ProtocolType, val bool) {
	p.oob = p.oob.Set(pt, val)
}

// For initialization only
func (p *ReceiveBuffer) SetPriorityPackets(pp [ProtocolTypeCount]PacketSet) {
	if p.priorityBuf == nil {
		panic(throw.IllegalState())
	}
	p.priority = pp
}

// For initialization only
func (p *ReceiveBuffer) SetPriorityByProtocol(pt ProtocolType, val bool) {
	if p.priorityBuf == nil {
		panic(throw.IllegalState())
	}
	if val {
		p.priority[pt] = AllPackets
	} else {
		p.priority[pt] = 0
	}
}

// For initialization only
func (p *ReceiveBuffer) SetPriority(pt ProtocolType, pk uint8, val bool) {
	if p.priorityBuf == nil {
		panic(throw.IllegalState())
	}
	p.priority[pt] = p.priority[pt].Set(pk, val)
}

// For initialization only
func (p *ReceiveBuffer) SetProtocolReceiver(pt ProtocolType, val Receiver) {
	p.receivers[pt] = val
}

func (p ReceiveBuffer) ReceiveSmallPacket(rp *ReceivedPacket, b []byte) {
	b = append([]byte(nil), b...) // make a copy

	pt := rp.Header.GetProtocolType()
	if p.oob.Has(pt) {
		rec := p.receivers[pt]
		if rec == nil {
			rec = p.defReceiver
		}
		go rec.ReceiveSmallPacket(rp, b)
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

func (p ReceiveBuffer) ReceiveLargePacket(rp *ReceivedPacket, preRead []byte, r io.LimitedReader) error {
	pt := rp.Header.GetProtocolType()
	if !p.oob.Has(pt) {
		if !p.largeSema.LockTimeout(p.largeTimeout) {
			if p.discardedFn != nil {
				p.discardedFn(rp.From, pt)
			}
			return errors.New("timeout")
		}
		defer p.largeSema.Unlock()
	}

	rec := p.receivers[pt]
	if rec == nil {
		rec = p.defReceiver
	}
	return rec.ReceiveLargePacket(rp, preRead, r)
}

func (p ReceiveBuffer) RunWorkers(count int, priorityOnly bool) {
	receivers := p.receivers
	for i := range receivers {
		if receivers[i] == nil {
			receivers[i] = p.defReceiver
		}
	}

	switch {
	case priorityOnly:
		if p.priorityBuf == nil {
			panic(throw.IllegalState())
		}
		for ; count > 0; count-- {
			go runSoloWorker(receivers, p.priorityBuf)
		}
	case p.priorityBuf == nil:
		for ; count > 0; count-- {
			go runSoloWorker(receivers, p.regularBuf)
		}
	default:
		for ; count > 0; count-- {
			go runDuoWorker(receivers, p.priorityBuf, p.regularBuf)
		}
	}
}

func (p ReceiveBuffer) Close() {
	close(p.regularBuf)
	close(p.priorityBuf)
	p.largeSema.Close()
}

func runDuoWorker(receivers Receivers, priority, regular <-chan smallPacket) {
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
		receiver := receivers[call.packet.Header.GetProtocolType()]
		receiver.ReceiveSmallPacket(call.packet, call.b)
	}
}

func runSoloWorker(receivers Receivers, priority <-chan smallPacket) {
	for {
		call, ok := <-priority
		if !ok {
			return
		}
		receiver := receivers[call.packet.Header.GetProtocolType()]
		receiver.ReceiveSmallPacket(call.packet, call.b)
	}
}

type smallPacket struct {
	packet *ReceivedPacket
	b      []byte
}
