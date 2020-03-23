// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l4

import (
	"io"
	"math"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/network/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
)

var _ MessageDelivery = &messageDelivery{}

func NewMessageDelivery() MessageDelivery {
	return &messageDelivery{}
}

type messageDelivery struct {
	nodeMapper NodeDeliveryMapper
	nodes      map[apinetwork.ShortNodeID]*nodeDelivery
}

func (p *messageDelivery) ShipTo(to DeliveryAddress, parcel DeliveryParcel, needsTag bool) (DeliveryTag, error) {
	panic("implement me")
}

func (p *messageDelivery) ShipReturn(to ReturnAddress, parcel DeliveryParcel, needsTag bool) (DeliveryTag, error) {
	panic("implement me")
}

func (p *messageDelivery) readError(id apinetwork.ShortNodeID, err error) {

}

type NodeDeliveryMapper interface {
	MapAddress(DeliveryAddress) apinetwork.ShortNodeID
	IsOnlineNode(apinetwork.ShortNodeID) bool
	ConnectNodeTransport(apinetwork.ShortNodeID, NodeTransportReceiver) (NodeTransportSender, error)
	SetNodeStatusCallback(func(apinetwork.ShortNodeID, NodeStatusUpdate))
}

type NodeStatusUpdate uint8

const (
	NodeAbsent NodeStatusUpdate = iota
	NodeActive
	NodeSuspended
	NodePowerZero
)

type NodeTransportReceiver interface {
	//ReceiveOffline()
	Receive(h apinetwork.Header, r *iokit.LimitedReader)
	ReceiveConnectionless(apinetwork.Header, []byte)
}

type NodeTransportSender interface {
	ShortNodeID() apinetwork.ShortNodeID

	MaxConnectionlessSize() int
	SendConnectionless([]byte) error

	Send(io.WriterTo) error
	SendBytes(io.WriterTo) error
}

type msgParcel struct {
	id     ParcelId
	parcel DeliveryParcel

	//to       DeliveryAddress
	returnId ParcelId

	// state headSent, headAck, bodySent

	report *DeliveryTag
}

var _ NodeTransportReceiver = &nodeDelivery{}

type nodeDelivery struct {
	parent *messageDelivery
	sender NodeTransportSender

	suspended map[ParcelId]*msgParcel

	receiveMutex sync.Mutex
	received     receiveDeduplicator
	ackBundle    []ParcelId
}

func (p *nodeDelivery) Receive(h apinetwork.Header, r *iokit.LimitedReader) {
	//b := make([]byte, fullSize)
	//if _, err := io.ReadFull(r, b); err != nil {
	//	p.parent.readError(p.sender.ShortNodeID(), err)
	//}
	//p.receive(h, b, false)
	// check signature
}

func (p *nodeDelivery) ReceiveConnectionless(h apinetwork.Header, b []byte) {
	p.receive(h, b, true)
}

func (p *nodeDelivery) receive(h apinetwork.Header, b []byte, connectionless bool) {
	p.receiveMutex.Lock()
	defer p.receiveMutex.Unlock()

}

type msgOut struct {
	out    io.WriterTo
	size   int
	parcel *msgParcel
}

type msgBundle struct {
	node   *nodeDelivery
	bundle []msgOut
}

type receiveDeduplicator struct {
	min, max  ParcelId
	received  map[ParcelId]struct{}
	prevCount int
}

func (p *receiveDeduplicator) Has(id ParcelId) bool {
	if id >= p.min && id <= p.max {
		return id != 0
	}
	return p.has(id)
}

func (p *receiveDeduplicator) has(id ParcelId) bool {
	_, ok := p.received[id]
	return ok
}

func (p *receiveDeduplicator) Add(id ParcelId) bool {
	switch {
	case id == 0:
		return false
	case id < p.min:
		if p.has(id) {
			return false
		}
		if id == p.min-1 {
			for p.min--; p.min > 1 && p.has(p.min-1); p.min-- {
				delete(p.received, p.min-1)
			}
			return true
		}
	case p.max == 0:
		p.min = id
		p.max = id
		return true
	case id > p.max:
		if p.has(id) {
			return false
		}
		if id == p.max+1 {
			for p.max++; p.max < math.MaxUint64 && p.has(p.max+1); p.max++ {
				delete(p.received, p.max+1)
			}
			return true
		}
	default:
		return false
	}
	p.received[id] = struct{}{}
	return true
}

func (p *receiveDeduplicator) Flush() {
	const minCount = 32
	n := len(p.received)
	switch h := p.prevCount >> 1; {
	case n > h:
		//
	case h < minCount>>1:
		p.received = nil
		p.prevCount = minCount
		return
	default:
		n = h
	}
	p.received = make(map[ParcelId]struct{}, n)
	p.prevCount = n
}
