// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"
	"math"
	"sync"
	"sync/atomic"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ Service = &messageDelivery{}

func NewMessageDelivery() Service {
	return &messageDelivery{}
}

type messageDelivery struct {
	nodeMapper NodeDeliveryL3
	parcelId   uint64 // atomic
	nodes      map[apinetwork.ShortNodeID]*nodeDelivery
	thisNodeId apinetwork.ShortNodeID
}

func (p *messageDelivery) ShipReturn(to ReturnAddress, parcel DeliveryParcel, needsTag bool) (*DeliveryTag, error) {
	return p.shipTo(to.returnTo, to.returnId, parcel, needsTag)
}

func (p *messageDelivery) ShipTo(to DeliveryAddress, parcel DeliveryParcel, needsTag bool) (*DeliveryTag, error) {
	return p.shipTo(to, 0, parcel, needsTag)
}

func (p *messageDelivery) shipTo(to DeliveryAddress, returnId ParcelId, parcel DeliveryParcel, needsTag bool) (dt *DeliveryTag, err error) {
	var nd *nodeDelivery
	if nd, err = p.getNodeByAddress(to); err != nil {
		return
	}

	if needsTag {
		dt = &DeliveryTag{}
	}
	return dt, nd.send(msgParcel{
		ParcelId(atomic.AddUint64(&p.parcelId, 1)),
		parcel, returnId,
		dt,
	})
}

func (p *messageDelivery) getNodeByAddress(address DeliveryAddress) (*nodeDelivery, error) {
	//panic(throw.NotImplemented())
	nid := p.nodeMapper.MapAddress(address)
	if nid.IsAbsent() {
		return nil, throw.E("unknown node", struct{ address DeliveryAddress }{address})
	}
	return p.getNodeById(nid)
}

func (p *messageDelivery) getNodeById(nid apinetwork.ShortNodeID) (*nodeDelivery, error) {
	if nd, ok := p.nodes[nid]; ok {
		return nd, nil
	}
	sender, err := p.nodeMapper.ConnectNodeTransport(nid, p)
	if err != nil {
		return nil, err
	}
	nd := &nodeDelivery{parent: p, sender: sender}
	p.nodes[nid] = nd
	return nd, nil
}

func (p *messageDelivery) Receive(h apinetwork.Header, r *iokit.LimitedReader) {
	// use checkOnline before any actions

	switch rid, sid, tid := apinetwork.ShortNodeID(h.ReceiverID), apinetwork.ShortNodeID(h.SourceID), apinetwork.ShortNodeID(h.TargetID); {
	case sid.IsAbsent():
		// error
	case sid == p.thisNodeId:
		// error - loopback
	case rid.IsAbsent():
		// error
	case rid != p.thisNodeId:
		// wrong packet, but can't blame as ReceiverID isn't secure for MitM
	case tid == p.thisNodeId:
		if !h.IsRelayRestricted() {
			// TODO lookup SourceId by endpoint
		}

		break // all ok
	case h.IsRelayRestricted():
		// error
	default:
		// TODO relay support
		// TODO log error
		return
	}

	if h.GetProtocolType() != ProtocolMessageDelivery {
		// error
	}

	switch PacketType(h.GetPacketType()) {
	case DeliveryState:

	case DeliveryParcelHead, DeliveryParcelBody:

	}
}

func (p *messageDelivery) ReceiveConnectionless(h apinetwork.Header, b []byte) {
	panic("implement me")
}

func (p *messageDelivery) readError(id apinetwork.ShortNodeID, err error) {

}

type NodeDeliveryL3 interface {
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

func (p *nodeDelivery) send(parcel msgParcel) error {

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
