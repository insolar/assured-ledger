// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"
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

func (p *messageDelivery) ShipReturn(to ReturnAddress, parcel Shipment, needsTag bool) (*TrackingTag, error) {
	return p.shipTo(to.returnTo, to.returnId, parcel, needsTag)
}

func (p *messageDelivery) ShipTo(to DeliveryAddress, parcel Shipment, needsTag bool) (*TrackingTag, error) {
	return p.shipTo(to, 0, parcel, needsTag)
}

func (p *messageDelivery) shipTo(to DeliveryAddress, returnId ShipmentID, parcel Shipment, needsTag bool) (dt *TrackingTag, err error) {
	var nd *nodeDelivery
	if nd, err = p.getNodeByAddress(to); err != nil {
		return
	}

	if needsTag {
		dt = &TrackingTag{}
	}
	return dt, nd.send(msgShipment{
		ShipmentID(atomic.AddUint64(&p.parcelId, 1)),
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

	if h.GetProtocolType() != Protocol {
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

var _ NodeTransportReceiver = &nodeDelivery{}

type nodeDelivery struct {
	parent *messageDelivery
	sender NodeTransportSender

	suspended map[ShipmentID]*msgShipment

	receiveMutex sync.Mutex
	received     receiveDeduplicator
	ackBundle    []ShipmentID
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

func (p *nodeDelivery) send(parcel msgShipment) error {

}

type msgOut struct {
	out    io.WriterTo
	size   int
	parcel *msgShipment
}

type msgBundle struct {
	node   *nodeDelivery
	bundle []msgOut
}
