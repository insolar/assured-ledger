package cloud

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat/memstor"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/watermill"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/version"
)

type cloudStatus struct {
	CertificateManager         nodeinfo.CertificateManager             `inject:""`
	PlatformCryptographyScheme cryptography.PlatformCryptographyScheme `inject:""`
	KeyProcessor               cryptography.KeyProcessor               `inject:""`

	BeatAppender beat.Appender

	Certificate nodeinfo.Certificate
	router      watermill.Router
	dispatcher  beat.Dispatcher

	cfg configuration.Configuration

	net *NetworkController
}

func (s *cloudStatus) Start(_ context.Context) error {
	s.Certificate = s.CertificateManager.GetCertificate()

	s.net.addNode(s.Certificate.GetNodeRef(), controlledNode{
		cert:                       s.Certificate,
		dispatcher:                 s.dispatcher,
		beatAppender:               s.BeatAppender,
		router:                     s.router,
		platformCryptographyScheme: s.PlatformCryptographyScheme,
		cfg:                        s.cfg,
		keyProcessor:               s.KeyProcessor,
	})

	return nil
}

func (s *cloudStatus) Stop(_ context.Context) error {
	s.net.nodeLeave(s.Certificate.GetNodeRef())

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

func (s *cloudStatus) GetCert(_ context.Context, global reference.Global) (nodeinfo.Certificate, error) {
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
	pulse, err := node.beatAppender.LatestTimeBeat()
	switch {
	// if there was not a single pulse
	case err != nil:
		state = network.WaitPulsar
	default:
		// lets ensure that latest beat has ancestor
		_, err = node.beatAppender.TimeBeat(pulse.PrevPulseNumber())
		if err != nil {
			state = network.WaitPulsar
		}
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
