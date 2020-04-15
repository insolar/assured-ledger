// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type ReceiverFunc func(apinetwork.PayloadCompleteness, apinetwork.Serializable) error

type ProtocolController struct {
	pType     uniproto.ProtocolType
	factory   apinetwork.DeserializationFactory
	receiveFn ReceiverFunc
	receiver  packetReceiver
	dedup     receiveDeduplicator
	smSender  retrySender
	lgSender  largeSender
}

func (p *ProtocolController) reportError(err error) {

}

func (p *ProtocolController) receiveState(packet *uniproto.ReceivedPacket, payload *StatePacket) error {
	peerID := packet.Header.SourceID

	if payload.BodyRq != 0 {
		dPeer, ok := packet.Peer.GetProtoInfo(p.pType).(*DeliveryPeer)
		if !ok {
			err := throw.RemoteBreach("peer is not a node")
			p.reportError(err)
			return err
		}

		shid := AsShipmentID(peerID, payload.BodyRq)
		if msg := p.smSender.get(shid); msg != nil && msg.markBodyRq() {
			// TODO send body
		} else {
			dPeer.sendReject(payload.BodyRq)
		}
	}

	for _, id := range payload.AckList {
		shid := AsShipmentID(peerID, id)
		if msg := p.smSender.get(shid); msg != nil {
			msg.markAck()
		}
	}

	for _, id := range payload.BodyAckList {
		shid := AsShipmentID(peerID, id)
		if msg := p.smSender.get(shid); msg != nil {
			msg.markBodyAck()
		}
	}

	for _, id := range payload.RejectList {
		shid := AsShipmentID(peerID, id)
		if msg := p.smSender.get(shid); msg != nil {
			msg.markReject()
		}
	}

	return nil
}

func (p *ProtocolController) receiveParcel(packet *uniproto.ReceivedPacket, payload *ParcelPacket) error {
	dPeer, ok := packet.Peer.GetOrCreateProtoInfo(p.pType, p.createProtoPeer).(*DeliveryPeer)
	if !ok {
		err := throw.RemoteBreach("peer is not a node")
		p.reportError(err)
		return err
	}

	//if payload.RepeatedSend {
	//	// TODO collect stats
	//} else {
	//
	//}

	peerID := packet.Header.SourceID

	if payload.ReturnId != 0 {
		retID := AsShipmentID(peerID, payload.ReturnId)
		if msg := p.smSender.get(retID); msg != nil {
			msg.markBodyAck()
		}
	}

	if payload.ParcelType == apinetwork.BodyPayload {
		dPeer.sendBodyAck(payload.ParcelId)
	} else {
		dPeer.sendAck(payload.ParcelId)
	}

	if !p.dedup.Add(DedupId(payload.ParcelId)) {
		return nil
	}

	// TODO in-proc unique node id
	err := p.receiveFn(payload.ParcelType, payload.Data)
	if err != nil {
		p.reportError(err)
	}
	return err
}

func (p *ProtocolController) createProtoPeer(peer uniproto.Peer) io.Closer {
	id := peer.GetNodeID()
	if id.IsAbsent() {
		return nil
	}

	dp := &DeliveryPeer{peerID: id}
	dp.outbound = &peerProxy{peer: dp}
	return dp
}

func (p *ProtocolController) send(to apinetwork.Address, payload *ParcelPacket) error {
	// check valid
	// payload.
}
