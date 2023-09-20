package integration

import (
	"context"
	"crypto"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat/memstor"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/censusimpl"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"

	"github.com/insolar/assured-ledger/ledger-core/network/servicenetwork"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
)

var (
	testNetworkPort uint32 = 10000
)

const (
	UseFakeTransport = false
	UseFakeBootstrap = true

	reqTimeoutMs = 2000
	pulseDelta   = 4
	//	consensusMin		= 5 // minimum count of participants that can survive when one node leaves
	maxPulsesForJoin = 3
)

const cacheDir = "network_cache/"

func initLogger(ctx context.Context, t *testing.T, level log.Level) context.Context {
	instestlogger.SetTestOutputWithErrorFilter(t, func(s string) bool {
		switch {
		case strings.Contains(s, "fraud"):
			return true // must fail
		case strings.Contains(s, "blame"):
			return true // must fail
		case strings.Contains(s, "Failed to process packet: packet type (") && strings.Contains(s, ") limit exceeded"):
			return false // skip it
		case strings.Contains(s, "Failed to send datagram: failed to "):
			return false // skip it
		}
		return true
	})
	global.SetLevel(level)

	ctx, _ = inslogger.InitNodeLoggerByGlobal("", "")
	return ctx
}

// testSuite is base test suite
type testSuite struct {
	bootstrapCount int
	nodesCount     int
	ctx            context.Context
	bootstrapNodes []*networkNode
	pulsar         TestPulsar
	t              *testing.T
	abortFn        func(string)
}

type consensusSuite struct {
	testSuite
}

func newTestSuite(t *testing.T, bootstrapCount, nodesCount int) testSuite {
	return testSuite{
		bootstrapCount: bootstrapCount,
		nodesCount:     nodesCount,
		t:              t,
		ctx:            initLogger(instestlogger.TestContext(t), t, log.DebugLevel),
		bootstrapNodes: make([]*networkNode, 0),
	}
}

func newConsensusSuite(t *testing.T, bootstrapCount, nodesCount int) *consensusSuite {
	return &consensusSuite{
		testSuite: newTestSuite(t, bootstrapCount, nodesCount),
	}
}

// Setup creates and run network with bootstrap and common nodes once before run all tests in the suite
func (s *consensusSuite) Setup() {
	var err error
	s.pulsar, err = NewTestPulsar(reqTimeoutMs, pulseDelta)
	require.NoError(s.t, err)

	global.Info("SetupTest")

	for i := 0; i < s.bootstrapCount; i++ {
		role := member.PrimaryRoleVirtual
		if i == 0 {
			role = member.PrimaryRoleHeavyMaterial
		}
		n := s.newNetworkNodeWithRole(fmt.Sprintf("bootstrap_%d", i), role)
		s.bootstrapNodes = append(s.bootstrapNodes, n)
	}

	pulseReceivers := make([]string, 0)
	for _, n := range s.bootstrapNodes {
		pulseReceivers = append(pulseReceivers, n.host)
	}

	global.Info("Setup bootstrap nodes")
	s.SetupNodesNetwork(s.bootstrapNodes)

	if UseFakeBootstrap {
		bnodes := make([]profiles.StaticProfile, 0)
		for _, n := range s.bootstrapNodes {
			o := n.serviceNetwork.NodeKeeper.FindAnyLatestNodeSnapshot().GetPopulation().GetLocalProfile()
			sdg := nodeinfo.NodeSignedDigest(o)
			require.NotNil(s.t, sdg)
			require.NotEmpty(s.t, sdg.GetSignatureHolder().AsByteString())
			bnodes = append(bnodes, o.GetStatic())
		}

		for _, n := range s.bootstrapNodes {
			n.serviceNetwork.BaseGateway.ConsensusMode = consensus.ReadyNetwork

			pop := censusimpl.NewManyNodePopulation(bnodes, n.id, n.vf)

			pu := pulsestor.GenesisPulse
			pu.Online = &pop

			err := n.serviceNetwork.NodeKeeper.AddExpectedBeat(pu)
			require.NoError(s.t, err)
			err = n.serviceNetwork.NodeKeeper.AddCommittedBeat(pu)
			require.NoError(s.t, err)

			err = n.serviceNetwork.BaseGateway.PulseAppender.AddCommittedBeat(pu)
			require.NoError(s.t, err)
			err = n.serviceNetwork.BaseGateway.StartConsensus(s.ctx)
			require.NoError(s.t, err)
			n.serviceNetwork.Gatewayer.SwitchState(s.ctx, network.CompleteNetworkState, pu.Data)

			pulseReceivers = append(pulseReceivers, n.host)
		}
	}

	s.StartNodesNetwork(s.bootstrapNodes)

	expectedBootstrapsCount := len(s.bootstrapNodes)
	retries := 10
	for {
		activeNodes := s.bootstrapNodes[0].GetActiveNodes()
		if expectedBootstrapsCount == len(activeNodes) {
			break
		}

		retries--
		if retries == 0 {
			break
		}

		time.Sleep(2 * time.Second)
	}

	activeNodes := s.bootstrapNodes[0].GetActiveNodes()
	require.Equal(s.t, len(s.bootstrapNodes), len(activeNodes))

	global.Info("Start test pulsar")
	err = s.pulsar.Start(initLogger(s.ctx, s.t, log.ErrorLevel), pulseReceivers)
	require.NoError(s.t, err)
}

func (s *testSuite) waitResults(results chan error, expected int) {
	count := 0
	for count < expected {
		err := <-results
		require.NoError(s.t, err)
		count++
	}
}

func (s *testSuite) SetupNodesNetwork(nodes []*networkNode) {
	for _, n := range nodes {
		s.preInitNode(n)
	}

	results := make(chan error, len(nodes))
	initNode := func(node *networkNode) {
		err := node.componentManager.Init(s.ctx)
		results <- err
	}

	global.Info("Init nodes")
	for _, n := range nodes {
		go initNode(n)
	}
	s.waitResults(results, len(nodes))

	for _, n := range nodes {
		s.afterInitNode(n)
	}
}

func (s *testSuite) StartNodesNetwork(nodes []*networkNode) {
	inslogger.FromContext(s.ctx).Info("Start nodes")

	results := make(chan error, len(nodes))
	startNode := func(node *networkNode) {
		err := node.componentManager.Start(node.ctx)
		node.serviceNetwork.BaseGateway.ConsensusController.RegisterFinishedNotifier(func(ctx context.Context, report network.Report) {
			node.consensusResult <- report.PulseNumber
		})
		results <- err
	}

	for _, n := range nodes {
		go startNode(n)
	}
	s.waitResults(results, len(nodes))
}

func startNetworkSuite(t *testing.T) *consensusSuite {
	cs := newConsensusSuite(t, 5, 0)
	cs.Setup()

	return cs
}

// stopNetworkSuite shutdowns all nodes in network
func (s *consensusSuite) stopNetworkSuite() {
	global.Info("=================== stopNetworkSuite()")
	global.Info("Stop network nodes")
	//for _, n := range s.networkNodes {
	//	err := n.componentManager.Stop(n.ctx)
	//	require.NoError(s.t, err)
	//}
	global.Info("Stop bootstrap nodes")
	for _, n := range s.bootstrapNodes {
		err := n.componentManager.Stop(n.ctx)
		require.NoError(s.t, err)
	}
	global.Info("Stop test pulsar")
	err := s.pulsar.Stop(s.ctx)
	require.NoError(s.t, err)
}

// waitForNodeJoin returns true if node joined in pulsesCount
func (s *consensusSuite) waitForNodeJoin(ref reference.Global, pulsesCount int) bool {
	for i := 0; i < pulsesCount; i++ {
		pn := s.waitForConsensus(1)
		if s.isNodeInActiveLists(ref, pn) {
			return true
		}
	}
	return false
}

// waitForNodeLeave returns true if node leaved in pulsesCount
func (s *consensusSuite) waitForNodeLeave(ref reference.Global, pulsesCount int) bool {
	for i := 0; i < pulsesCount; i++ {
		pn := s.waitForConsensus(1)
		if !s.isNodeInActiveLists(ref, pn) {
			return true
		}
	}
	return false
}

func (s *consensusSuite) waitForConsensus(consensusCount int) pulse.Number {
	var p pulse.Number
	for i := 0; i < consensusCount; i++ {
		for _, n := range s.bootstrapNodes {
			select {
			case p = <-n.consensusResult:
				continue
			case <-time.After(time.Second * 12):
				panic("waitForConsensus timeout")
			}
		}

		//for _, n := range s.networkNodes {
		//	<-n.consensusResult
		//}
	}
	s.assertNetworkInConsistentState(p)
	return p
}

func (s *consensusSuite) assertNetworkInConsistentState(p pulse.Number) {
	var nodes []nodeinfo.NetworkNode

	for _, n := range s.bootstrapNodes {
		require.Equal(s.t, network.CompleteNetworkState.String(),
			n.serviceNetwork.Gatewayer.Gateway().GetState().String(),
			"Node not in CompleteNetworkState",
		)

		a := n.serviceNetwork.NodeKeeper.GetNodeSnapshot(p)
		activeNodes := a.GetPopulation().GetProfiles()
		if nodes == nil {
			nodes = activeNodes
			continue
		}

		require.Equal(s.t, len(nodes), len(activeNodes))

		for i, n  := range nodes {
			an := activeNodes[i]
			require.True(s.t, profiles.EqualStaticProfiles(n.GetStatic(), an.GetStatic(), true))
			require.Equal(s.t, n.GetNodeID(), an.GetNodeID(), i)
			require.Equal(s.t, adapters.ECDSAPublicKeyOfNode(n), adapters.ECDSAPublicKeyOfNode(an), i)
			require.Equal(s.t, n.GetDeclaredPower(), an.GetDeclaredPower(), i)
		}
	}
}

func (s *consensusSuite) waitForConsensusExcept(consensusCount int, exception reference.Global) pulse.Number {
	var p pulse.Number
	for i := 0; i < consensusCount; i++ {
		for _, n := range s.bootstrapNodes {
			if n.ref.Equal(exception) {
				continue
			}
			select {
			case p = <-n.consensusResult:
				continue
			case <-time.After(time.Second * 12):
				panic("waitForConsensus timeout")
			}
		}

	}

	s.assertNetworkInConsistentState(p)
	return p
}

// nodesCount returns count of nodes in network without testNode
func (s *testSuite) getNodesCount() int {
	return len(s.bootstrapNodes) // + len(s.networkNodes)
}

func (s *testSuite) isNodeInActiveLists(ref reference.Global, p pulse.Number) bool {
	for _, n := range s.bootstrapNodes {
		a := n.serviceNetwork.NodeKeeper.GetNodeSnapshot(p)
		if a.FindNodeByRef(ref) == nil {
			return false
		}
	}
	return true
}

func (s *testSuite) InitNode(node *networkNode) {
	if node.componentManager != nil {
		err := node.componentManager.Init(s.ctx)
		require.NoError(s.t, err)
		s.afterInitNode(node)
	}
}

func (s *testSuite) StartNode(node *networkNode) {
	if node.componentManager != nil {
		err := node.componentManager.Start(node.ctx)
		require.NoError(s.t, err)
	}
}

func (s *testSuite) StopNode(node *networkNode) {
	if node.componentManager != nil {
		err := node.componentManager.Stop(s.ctx)
		require.NoError(s.t, err)
	}
}

func (s *testSuite) GracefulStop(node *networkNode) {
	if node.componentManager != nil {
		err := node.componentManager.GracefulStop(s.ctx)
		require.NoError(s.t, err)

		err = node.componentManager.Stop(s.ctx)
		require.NoError(s.t, err)
	}
}

type networkNode struct {
	ref                 reference.Global
	id 					node.ShortNodeID
	role                member.PrimaryRole
	privateKey          crypto.PrivateKey
	cryptographyService cryptography.Service
	host                string
	ctx                 context.Context

	vf *adapters.TransportCryptographyFactory

	componentManager *component.Manager
	serviceNetwork   *servicenetwork.ServiceNetwork
	consensusResult  chan pulse.Number
}

func (s *testSuite) newNetworkNode(name string) *networkNode {
	return s.newNetworkNodeWithRole(name, member.PrimaryRoleVirtual)
}
func (s *testSuite) startNewNetworkNode(name string) *networkNode {
	testNode := s.newNetworkNode(name)
	s.preInitNode(testNode)
	s.InitNode(testNode)
	s.StartNode(testNode)
	return testNode
}

// newNetworkNode returns networkNode initialized only with ref, host address and key pair
func (s *testSuite) newNetworkNodeWithRole(name string, role member.PrimaryRole) *networkNode {
	key, err := platformpolicy.NewKeyProcessor().GeneratePrivateKey()
	require.NoError(s.t, err)
	address := "127.0.0.1:" + strconv.Itoa(incrementTestPort())

	ref := gen.UniqueGlobalRef()
	n := &networkNode{
		ref:                 ref,
		id:                  node.GenerateShortID(ref),
		role:                role,
		privateKey:          key,
		cryptographyService: platformpolicy.NewKeyBoundCryptographyService(key),
		host:                address,
		consensusResult:     make(chan pulse.Number, 1),
	}

	nodeContext, _ := inslogger.WithFields(s.ctx, map[string]interface{}{
		"node_name": name,
	})

	n.ctx = nodeContext
	return n
}

func incrementTestPort() int {
	result := atomic.AddUint32(&testNetworkPort, 1)
	return int(result)
}

func (n *networkNode) GetActiveNodes() []nodeinfo.NetworkNode {
	return n.serviceNetwork.NodeKeeper.FindAnyLatestNodeSnapshot().GetPopulation().GetProfiles()
}

func (n *networkNode) GetWorkingNodeCount() int {
	pop := n.serviceNetwork.NodeKeeper.FindAnyLatestNodeSnapshot().GetPopulation()
	return pop.GetIndexedCount() - pop.GetIdleCount()
}

func (s *testSuite) initCrypto(node *networkNode) (*mandates.CertificateManager, cryptography.Service) {
	pubKey, err := node.cryptographyService.GetPublicKey()
	require.NoError(s.t, err)

	// init certificate

	proc := platformpolicy.NewKeyProcessor()
	publicKey, err := proc.ExportPublicKeyPEM(pubKey)
	require.NoError(s.t, err)

	cert := &mandates.Certificate{}
	cert.PublicKey = string(publicKey[:])
	cert.Reference = node.ref.String()
	cert.Role = node.role.String()
	cert.BootstrapNodes = make([]mandates.BootstrapNode, 0)
	cert.MinRoles.HeavyMaterial = 1
	cert.MinRoles.Virtual = 1

	for _, b := range s.bootstrapNodes {
		pubKey, _ := b.cryptographyService.GetPublicKey()
		pubKeyBuf, err := proc.ExportPublicKeyPEM(pubKey)
		require.NoError(s.t, err)

		bootstrapNode := mandates.NewBootstrapNode(
			pubKey,
			string(pubKeyBuf[:]),
			b.host,
			b.ref.String(),
			b.role.String(),
		)

		sign, err := mandates.SignCert(b.cryptographyService, cert.PublicKey, cert.Role, cert.Reference)
		require.NoError(s.t, err)
		bootstrapNode.NodeSign = sign.Bytes()

		cert.BootstrapNodes = append(cert.BootstrapNodes, *bootstrapNode)
	}

	// dump cert and read it again from json for correct private files initialization
	jsonCert, err := cert.Dump()
	require.NoError(s.t, err)

	cert, err = mandates.ReadCertificateFromReader(pubKey, proc, strings.NewReader(jsonCert))
	require.NoError(s.t, err)
	return mandates.NewCertificateManager(cert), node.cryptographyService
}

type PublisherMock struct{}

func (p *PublisherMock) Publish(topic string, messages ...*message.Message) error {
	return nil
}

func (p *PublisherMock) Close() error {
	return nil
}

// preInitNode inits previously created node with mocks and external dependencies
func (s *testSuite) preInitNode(nd *networkNode) {
	cfg := configuration.NewConfiguration()
	cfg.Host.Transport.Address = nd.host
	cfg.Service.CacheDirectory = cacheDir + nd.host

	nd.componentManager = component.NewManager(nil)
	nd.componentManager.SetLogger(global.Logger())

	scheme := platformpolicy.NewPlatformCryptographyScheme()
	nd.componentManager.Register(scheme)
	serviceNetwork, err := servicenetwork.NewServiceNetwork(cfg, nd.componentManager)
	require.NoError(s.t, err)

	certManager, cryptographyService := s.initCrypto(nd)

	certificate := certManager.GetCertificate()
	realKeeper := memstor.NewNodeKeeper(certificate.GetNodeRef(), certificate.GetRole())

	keyProc := platformpolicy.NewKeyProcessor()

	pubMock := &PublisherMock{}
	if UseFakeTransport {
		// little hack: this Register will override transport.Factory
		// in servicenetwork internal component manager with fake factory
		nd.componentManager.Register(transport.NewFakeFactory(cfg.Host.Transport))
	} else {
		nd.componentManager.Register(transport.NewFactory(cfg.Host.Transport))
	}

	pulseManager := chorus.NewConductorMock(s.t)
	pulseManager.RequestNodeStateMock.Set(func(fn chorus.NodeStateFunc) {
		nshBytes := longbits.Bits512{}
		_, _ = rand.Read(nshBytes[:])
		fn(api.UpstreamState{NodeState: cryptkit.NewDigest(nshBytes, "random")})
	})
	pulseManager.CommitPulseChangeMock.Return(nil)
	pulseManager.CommitFirstPulseChangeMock.Return(nil)

	nd.componentManager.Inject(
		realKeeper,
		pulseManager,
		pubMock,
		certManager,
		cryptographyService,
		keystore.NewInplaceKeyStore(nd.privateKey),
		serviceNetwork,
		keyProc,
		memstor.NewStorageMem(),
	)
	nd.serviceNetwork = serviceNetwork
	nd.vf = adapters.NewTransportCryptographyFactory(scheme)

	localNodeRef := realKeeper.GetLocalNodeReference()
	nodeContext, _ := inslogger.WithFields(s.ctx, map[string]interface{}{
		"node_id":      node.GenerateShortID(localNodeRef),
		"node_address": localNodeRef,
		"node_role":    realKeeper.GetLocalNodeRole(),
	})

	nd.ctx = nodeContext
}

// afterInitNode called after component manager Init
func (s *testSuite) afterInitNode(nd *networkNode) {
	aborter := network.NewAborterMock(s.t)
	aborter.AbortMock.Set(func(ctx context.Context, reason string) {
		if s.abortFn != nil {
			s.abortFn(reason)
		} else {
			inslogger.FromContext(nd.ctx).Fatal(reason)
		}
	})
	nd.serviceNetwork.BaseGateway.Aborter = aborter

	staticProfile := nd.serviceNetwork.BaseGateway.GetLocalNodeStaticProfile()
	pop := censusimpl.NewManyNodePopulation([]profiles.StaticProfile{staticProfile}, staticProfile.GetStaticNodeID(), nd.vf)

	pu := pulsestor.GenesisPulse
	pu.Online = &pop

	nodeKeeper := nd.serviceNetwork.BaseGateway.NodeKeeper
	err := nodeKeeper.AddExpectedBeat(pu)
	require.NoError(s.t, err)
	err = nodeKeeper.AddCommittedBeat(pu)
	require.NoError(s.t, err)
}

func (s *testSuite) AssertActiveNodesCountDelta(delta int) {
	activeNodes := s.bootstrapNodes[1].GetActiveNodes()
	n := s.getNodesCount()+delta
	require.Equal(s.t, n, len(activeNodes))
}

func (s *testSuite) AssertWorkingNodesCountDelta(delta int) {
	require.Equal(s.t, s.getNodesCount()+delta, s.bootstrapNodes[0].GetWorkingNodeCount())
}
