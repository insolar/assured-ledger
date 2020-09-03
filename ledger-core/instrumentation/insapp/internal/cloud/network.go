// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cloud

import (
	"context"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/watermill"
	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/version"
)

// NetworkSupport provides network-related functions to an app compartment
type NetworkSupport interface {
	beat.NodeNetwork
	nodeinfo.CertificateGetter

	CreateMessagesRouter(context.Context) messagesender.MessageRouter

	AddDispatcher(beat.Dispatcher)
	GetBeatHistory() beat.History
}

type Node struct {
	pulseManager chorus.Conductor
	router       watermill.Router

	cert nodeinfo.Certificate
}

func NewNetwork() Network {
	return Network{
		lock:  &sync.RWMutex{},
		nodes: make(map[reference.Global]Node),
		start: time.Now(),
	}
}

type Network struct {
	lock  *sync.RWMutex
	start time.Time
	nodes map[reference.Global]Node
}

func (n Network) nodeCount() int {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return len(n.nodes)
}

func (n Network) getFirstBeat() beat.Beat {
	return beat.Beat{
		Data: pulse.Data{
			PulseNumber: pulse.MinTimePulse,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.MinTimePulse,
				NextPulseDelta: 100,
				PrevPulseDelta: 0,
				Timestamp:      pulse.MinTimePulse,
				PulseEntropy:   longbits.Bits256{},
			},
		},
	}
}

func (n Network) UpdateNode(cert nodeinfo.Certificate, pulseManager chorus.Conductor) {
	n.lock.Lock()
	defer n.lock.Unlock()

	node, err := n.getNode(cert.GetNodeRef())
	if err != nil {
		panic(throw.IllegalState())
	}

	node.pulseManager = pulseManager

	err = pulseManager.CommitPulseChange(n.getFirstBeat())
	if err != nil {
		panic(err)
	}
}

func (n Network) GetNode(nodeID reference.Global) (*Node, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return n.getNode(nodeID)
}

func (n Network) getNode(nodeID reference.Global) (*Node, error) {
	node, ok := n.nodes[nodeID]
	if !ok {
		return nil, throw.E("no node found for ref", struct{ reference reference.Global }{reference: nodeID})
	}
	return &node, nil
}

func (n Network) sendMessageHandler(msg *message.Message) error {
	receiver := msg.Metadata.Get(defaults.Receiver)
	if receiver == "" {
		return throw.E("failed to send message: Receiver in message metadata is not set")
	}
	nodeRef, err := reference.GlobalFromString(receiver)
	if err != nil {
		return throw.W(err, "failed to send message: Receiver in message metadata is invalid")
	}
	if nodeRef.IsEmpty() {
		return throw.E("failed to send message: Receiver in message metadata is empty")
	}

	node, err := n.GetNode(nodeRef)
	if err != nil {
		panic(throw.IllegalState())
	}

	err = node.router.Pub.Publish(defaults.TopicIncoming, msg)
	if err != nil {
		return throw.W(err, "error while publish msg to TopicIncoming")
	}

	return nil
}

func (n Network) Distribute(_ context.Context, packet pulsar.PulsePacket) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	for _, node := range n.nodes {
		node.pulseManager.CommitPulseChange(beat.Beat{
			Data: adapters.NewPulseData(packet),
		})
	}
}

func (n Network) NetworkInitFunc(cert nodeinfo.Certificate) (NetworkSupport, network.Status, error) {
	cloudNetwork := &cloudStatus{
		nodeRef: cert.GetNodeRef(),
		role:    cert.GetRole(),
		cert:    cert,
		net:     &n,
	}
	return cloudNetwork, cloudNetwork, nil
}

type cloudStatus struct {
	nodeRef reference.Global
	role    member.PrimaryRole
	cert    nodeinfo.Certificate
	net     *Network
}

func (s *cloudStatus) AddDispatcher(dispatcher beat.Dispatcher) {
	panic("implement me")
}

func (s *cloudStatus) GetBeatHistory() beat.History {
	panic("implement me")
}

func (s *cloudStatus) GetLocalNodeRole() member.PrimaryRole {
	return s.role
}

func (s *cloudStatus) GetNodeSnapshot(number pulse.Number) beat.NodeSnapshot {
	panic("implement me")
}

func (s *cloudStatus) FindAnyLatestNodeSnapshot() beat.NodeSnapshot {
	panic("implement me")
}

func (s *cloudStatus) GetCert(ctx context.Context, global reference.Global) (nodeinfo.Certificate, error) {
	node, err := s.net.GetNode(global)
	if err != nil {
		return nil, throw.E("node not found")
	}
	return node.cert, nil
}

func (s *cloudStatus) CreateMessagesRouter(ctx context.Context) messagesender.MessageRouter {
	s.net.lock.Lock()
	defer s.net.lock.Unlock()

	router := watermill.NewRouter(ctx, s.net.sendMessageHandler)

	s.net.nodes[s.cert.GetNodeRef()] = Node{
		cert:   s.cert,
		router: router,
	}

	return router
}

func (s *cloudStatus) GetLocalNodeReference() reference.Holder {
	return s.nodeRef
}

func (s *cloudStatus) GetNetworkStatus() network.StatusReply {
	nodeLen := s.net.nodeCount()
	return network.StatusReply{
		NetworkState:    network.CompleteNetworkState,
		LocalRef:        s.nodeRef,
		LocalRole:       s.role,
		ActiveListSize:  nodeLen,
		WorkingListSize: nodeLen,

		Version:   version.Version,
		Timestamp: time.Now(),
		StartTime: s.net.start,
	}
}
