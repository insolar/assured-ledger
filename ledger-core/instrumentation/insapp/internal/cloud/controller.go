// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cloud

import (
	"context"
	"crypto/ecdsa"
	"sort"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat/memstor"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/censusimpl"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/watermill"
	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
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

type controlledNode struct {
	BeatAppender               beat.Appender
	dispatcher                 beat.Dispatcher
	router                     watermill.Router
	PlatformCryptographyScheme cryptography.PlatformCryptographyScheme
	KeyProcessor               cryptography.KeyProcessor
	cfg                        configuration.Configuration

	svf cryptkit.SignatureVerifierFactory

	cert nodeinfo.Certificate

	profile *adapters.StaticProfile
}

func NewController() Controller {
	return Controller{
		lock:  &sync.RWMutex{},
		nodes: make(map[reference.Global]*controlledNode),
		start: time.Now(),
	}
}

type Controller struct {
	lock  *sync.RWMutex
	start time.Time
	nodes map[reference.Global]*controlledNode
}

func (n Controller) nodeCount() int {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return len(n.nodes)
}

func (n Controller) addNode(nodeRef reference.Global, netNode controlledNode) {
	n.lock.Lock()
	defer n.lock.Unlock()

	if _, exists := n.nodes[nodeRef]; exists {
		panic(throw.IllegalState())
	}
	netNode.svf = adapters.NewTransportCryptographyFactory(netNode.PlatformCryptographyScheme)

	netNode.profile = adapters.NewStaticProfile(node.GenerateShortID(nodeRef),
		netNode.cert.GetRole(), member.SpecialRoleNone, adapters.DefaultStartPower,
		adapters.NewStaticProfileExtension(node.GenerateShortID(nodeRef), nodeRef, cryptkit.NewSignature(longbits.Bits512{}, "")),
		adapters.NewOutbound(netNode.cfg.Host.Transport.Address),
		legacyadapter.NewECDSAPublicKeyStoreFromPK(netNode.cert.GetPublicKey()),
		legacyadapter.NewECDSASignatureKeyHolder(netNode.cert.GetPublicKey().(*ecdsa.PublicKey), netNode.KeyProcessor),
		cryptkit.NewSignedDigest(
			cryptkit.NewDigest(longbits.Bits512{}, legacyadapter.SHA3Digest512),
			cryptkit.NewSignature(longbits.Bits512{}, legacyadapter.SHA3Digest512.SignedBy(legacyadapter.SECP256r1Sign)),
		),
	)

	n.nodes[nodeRef] = &netNode
}

func (n Controller) getNode(nodeID reference.Global) (*controlledNode, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return n.unsafeGetNode(nodeID)
}

func (n Controller) unsafeGetNode(nodeID reference.Global) (*controlledNode, error) {
	node, ok := n.nodes[nodeID]
	if !ok {
		return nil, throw.E("no node found for ref", struct{ reference reference.Global }{reference: nodeID})
	}
	return node, nil
}

func (n Controller) sendMessageHandler(msg *message.Message) error {
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

	node, err := n.getNode(nodeRef)
	if err != nil {
		panic(throw.IllegalState())
	}

	err = node.router.Pub.Publish(defaults.TopicIncoming, msg)
	if err != nil {
		return throw.W(err, "error while publish msg to TopicIncoming")
	}

	return nil
}

func (n Controller) Distribute(_ context.Context, packet pulsar.PulsePacket) {
	n.lock.Lock()
	defer n.lock.Unlock()

	populationNodes := make([]profiles.StaticProfile, 0, len(n.nodes))
	for _, netNode := range n.nodes {
		populationNodes = append(populationNodes, netNode.profile)
	}

	newBeatData := beat.Beat{
		Data:      adapters.NewPulseData(packet),
		StartedAt: time.Now(),
	}

	for _, netNode := range n.nodes {
		onlinePopulation := censusimpl.NewManyNodePopulation(populationNodes, netNode.profile.GetStaticNodeID(), netNode.svf)

		newBeat := beat.Beat{
			Data:      newBeatData.Data,
			StartedAt: newBeatData.StartedAt,
			Online:    prepareManyNodePopulation(netNode.profile.GetStaticNodeID(), onlinePopulation),
		}

		err := netNode.BeatAppender.AddCommittedBeat(newBeat)
		if err != nil {
			panic(err)
		}

		sink, ackFn := beat.NewAck(func(data beat.AckData) {})

		netNode.dispatcher.PrepareBeat(sink)

		ackFn(true)

		netNode.dispatcher.CommitBeat(newBeat)
	}
}

func prepareManyNodePopulation(id node.ShortNodeID, op censusimpl.ManyNodePopulation) *censusimpl.ManyNodePopulation {
	dp := censusimpl.NewDynamicPopulationCopySelf(&op)
	for _, np := range op.GetProfiles() {
		if np.GetNodeID() != id {
			dp.AddProfile(np.GetStatic())
		}
	}

	pfs := dp.GetUnorderedProfiles()
	for _, np := range pfs {
		np.SetOpMode(member.ModeNormal)
		pw := np.GetStatic().GetStartPower()
		np.SetPower(pw)
	}

	sort.SliceStable(pfs, func(i, j int) bool {
		// Power sorting is REVERSED
		return pfs[j].GetDeclaredPower() < pfs[i].GetDeclaredPower()
	})

	idx := member.AsIndex(0)
	for _, np := range pfs {
		np.SetIndex(idx)
		idx++
	}

	ap, _ := dp.CopyAndSeparate(false, nil)
	return ap
}

func (n Controller) NetworkInitFunc(cfg configuration.Configuration, cm *component.Manager) (NetworkSupport, network.Status, error) {
	statusNetwork := &cloudStatus{
		net: &n,
		cfg: cfg,
	}

	return statusNetwork, statusNetwork, nil
}

type cloudStatus struct {
	CertificateManager         nodeinfo.CertificateManager             `inject:""`
	PlatformCryptographyScheme cryptography.PlatformCryptographyScheme `inject:""`
	KeyProcessor               cryptography.KeyProcessor               `inject:""`

	BeatAppender beat.Appender

	Certificate nodeinfo.Certificate
	router      watermill.Router
	dispatcher  beat.Dispatcher

	cfg configuration.Configuration

	net *Controller
}

func (s *cloudStatus) Start(_ context.Context) error {
	s.Certificate = s.CertificateManager.GetCertificate()

	s.net.addNode(s.Certificate.GetNodeRef(), controlledNode{
		cert:                       s.Certificate,
		dispatcher:                 s.dispatcher,
		BeatAppender:               s.BeatAppender,
		router:                     s.router,
		PlatformCryptographyScheme: s.PlatformCryptographyScheme,
		cfg:                        s.cfg,
		KeyProcessor:               s.KeyProcessor,
	})

	return nil
}

func (s *cloudStatus) AddDispatcher(dispatcher beat.Dispatcher) {
	s.dispatcher = dispatcher
}

func (s *cloudStatus) GetBeatHistory() beat.History {
	s.BeatAppender = memstor.NewStorageMem()
	return s.BeatAppender
}

func (s *cloudStatus) GetLocalNodeRole() member.PrimaryRole {
	return s.Certificate.GetRole()
}

func (s *cloudStatus) GetNodeSnapshot(number pulse.Number) beat.NodeSnapshot {
	panic("implement me")
}

func (s *cloudStatus) FindAnyLatestNodeSnapshot() beat.NodeSnapshot {
	panic("implement me")
}

func (s *cloudStatus) GetCert(ctx context.Context, global reference.Global) (nodeinfo.Certificate, error) {
	node, err := s.net.getNode(global)
	if err != nil {
		return nil, throw.E("node not found")
	}
	return node.cert, nil
}

func (s *cloudStatus) CreateMessagesRouter(ctx context.Context) messagesender.MessageRouter {
	s.net.lock.Lock()
	defer s.net.lock.Unlock()

	s.router = watermill.NewRouter(ctx, s.net.sendMessageHandler)

	return s.router
}

func (s *cloudStatus) GetLocalNodeReference() reference.Holder {
	return s.Certificate.GetNodeRef()
}

func (s *cloudStatus) GetNetworkStatus() network.StatusReply {
	node, err := s.net.getNode(s.Certificate.GetNodeRef())
	if err != nil {
		panic(throw.IllegalState())
	}
	state := network.CompleteNetworkState
	if _, err := node.BeatAppender.LatestTimeBeat(); err != nil {
		state = network.WaitPulsar
	}

	nodeLen := s.net.nodeCount()
	return network.StatusReply{
		NetworkState:    state,
		LocalRef:        s.Certificate.GetNodeRef(),
		LocalRole:       s.Certificate.GetRole(),
		ActiveListSize:  nodeLen,
		WorkingListSize: nodeLen,

		Version:   version.Version,
		Timestamp: time.Now(),
		StartTime: s.net.start,
	}
}
