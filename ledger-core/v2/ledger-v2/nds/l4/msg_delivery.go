// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l4

import (
	"io"
	"math"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l2"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/apinetwork"
)

var _ MessageDelivery = &messageDelivery{}

func NewMessageDelivery() MessageDelivery {
	return &messageDelivery{}
}

type messageDelivery struct {
	nodes map[apinetwork.ShortNodeID]*nodeDelivery
}

func (p *messageDelivery) ShipTo(to DeliveryAddress, parcel DeliveryParcel, needsTag bool) (DeliveryTag, error) {
	panic("implement me")
}

func (p *messageDelivery) ShipReturn(to ReturnAddress, parcel DeliveryParcel, needsTag bool) (DeliveryTag, error) {
	panic("implement me")
}

type NodeDeliveryMapper interface {
	MapAddress(DeliveryAddress) apinetwork.ShortNodeID
	IsOnlineNode(apinetwork.ShortNodeID) bool
	ConnectNodeTransport(apinetwork.ShortNodeID, NodeTransportReceiver) (NodeTransportSender, error)
}

type NodeTransportReceiver interface {
	ReceiveOffline()
	Receive(apinetwork.Header, io.Reader)
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

	report *DeliveryTag
}

type nodeDelivery struct {
	transportTiny  l2.Transport
	transportSmall l2.Transport
	transportLarge l2.Transport

	suspended map[ParcelId]*msgParcel

	receiveMutex sync.Mutex
	received     receiveDeduplicator
	ackBundle    []ParcelId
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
