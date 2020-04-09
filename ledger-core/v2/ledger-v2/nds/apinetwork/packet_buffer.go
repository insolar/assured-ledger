// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

import (
	"errors"
	"io"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewReceiveBuffer(regularLimit, priorityLimit, largeLimit int, defReceiver ProtocolReceiver) ProtocolReceiveBuffer {
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
	return ProtocolReceiveBuffer{
		priorityBuf: priorityBuf,
		regularBuf:  make(chan smallPacket, regularLimit),
		largeSema:   synckit.NewSemaphore(largeLimit),
		defReceiver: defReceiver,
	}
}

var _ ProtocolReceiver = ProtocolReceiveBuffer{}

type ProtocolReceiveBuffer struct {
	priorityBuf  chan smallPacket
	regularBuf   chan smallPacket
	oob          ProtocolSet
	priority     [ProtocolTypeCount]PacketSet
	receivers    ProtocolReceivers
	defReceiver  ProtocolReceiver
	discardedFn  func(Address, ProtocolType)
	largeSema    synckit.Semaphore
	largeTimeout time.Duration
}

// For initialization only
func (p *ProtocolReceiveBuffer) SetDiscardHandler(fn func(Address, ProtocolType)) {
	p.discardedFn = fn
}

// For initialization only
func (p *ProtocolReceiveBuffer) SetLargePacketQueueTimeout(d time.Duration) {
	p.largeTimeout = d
}

// For initialization only
func (p *ProtocolReceiveBuffer) SetOutOfBandProtocols(oob ProtocolSet) {
	p.oob = oob
}

// For initialization only
func (p *ProtocolReceiveBuffer) SetOutOfBand(pt ProtocolType, val bool) {
	p.oob = p.oob.Set(pt, val)
}

// For initialization only
func (p *ProtocolReceiveBuffer) SetPriorityPackets(pp [ProtocolTypeCount]PacketSet) {
	if p.priorityBuf == nil {
		panic(throw.IllegalState())
	}
	p.priority = pp
}

// For initialization only
func (p *ProtocolReceiveBuffer) SetPriorityByProtocol(pt ProtocolType, val bool) {
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
func (p *ProtocolReceiveBuffer) SetPriority(pt ProtocolType, pk uint8, val bool) {
	if p.priorityBuf == nil {
		panic(throw.IllegalState())
	}
	p.priority[pt] = p.priority[pt].Set(pk, val)
}

// For initialization only
func (p *ProtocolReceiveBuffer) SetProtocolReceiver(pt ProtocolType, val ProtocolReceiver) {
	p.receivers[pt] = val
}

func (p ProtocolReceiveBuffer) ReceiveSmallPacket(from Address, packet Packet, b []byte, sigLen uint32) {
	b = append([]byte(nil), b...) // make a copy

	pt := packet.Header.GetProtocolType()
	if p.oob.Has(pt) {
		rec := p.receivers[pt]
		go rec.ReceiveSmallPacket(from, packet, b, sigLen)
	}
	buf := p.regularBuf
	if s := p.priority[pt]; s != 0 && p.priorityBuf != nil && s.Has(packet.Header.GetPacketType()) {
		buf = p.priorityBuf
	}

	if p.discardedFn != nil {
		select {
		case buf <- smallPacket{from, packet, b, sigLen}:
			return
		default:
			p.discardedFn(from, pt)
		}
	} else {
		buf <- smallPacket{from, packet, b, sigLen}
	}
}

func (p ProtocolReceiveBuffer) ReceiveLargePacket(from Address, packet Packet, preRead []byte, r io.LimitedReader, verifier PacketDataVerifier) error {
	pt := packet.Header.GetProtocolType()
	if !p.oob.Has(pt) {
		if !p.largeSema.LockTimeout(p.largeTimeout) {
			if p.discardedFn != nil {
				p.discardedFn(from, pt)
			}
			return errors.New("timeout")
		}
		defer p.largeSema.Unlock()
	}
	rec := p.receivers[pt]
	return rec.ReceiveLargePacket(from, packet, preRead, r, verifier)
}

func (p ProtocolReceiveBuffer) RunWorker(priorityOnly bool) {
	switch {
	case priorityOnly:
		if p.priorityBuf == nil {
			panic(throw.IllegalState())
		}
		go runSoloWorker(p.receivers, p.priorityBuf)
	case p.priorityBuf == nil:
		go runSoloWorker(p.receivers, p.regularBuf)
	default:
		go runDuoWorker(p.receivers, p.priorityBuf, p.regularBuf)
	}
}

func (p ProtocolReceiveBuffer) Close() {
	close(p.regularBuf)
	close(p.priorityBuf)
	p.largeSema.Close()
}

func runDuoWorker(receivers ProtocolReceivers, priority, regular <-chan smallPacket) {
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
		receiver.ReceiveSmallPacket(call.from, call.packet, call.b, call.sigLen)
	}
}

func runSoloWorker(receivers ProtocolReceivers, priority <-chan smallPacket) {
	for {
		call, ok := <-priority
		if !ok {
			return
		}
		receiver := receivers[call.packet.Header.GetProtocolType()]
		receiver.ReceiveSmallPacket(call.from, call.packet, call.b, call.sigLen)
	}
}

type smallPacket struct {
	from   Address
	packet Packet
	b      []byte
	sigLen uint32
}
