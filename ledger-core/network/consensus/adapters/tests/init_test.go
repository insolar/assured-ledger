// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package tests

import (
	"context"
	"crypto"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat/memstor"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/gateway"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network/mutable"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

var (
	keyProcessor = platformpolicy.NewKeyProcessor()
	scheme       = platformpolicy.NewPlatformCryptographyScheme()
)

var (
	shortNodeIdOffset = 1000
	portOffset        = 10000
)

type candidate struct {
	profiles.StaticProfile
	profiles.StaticProfileExtension
}

func testCase(stopAfter, startCaseAfter time.Duration, test func()) {
	startedAt := time.Now()

	ticker := time.NewTicker(time.Second)
	stopTest := time.After(stopAfter)
	startCase := time.After(startCaseAfter)
	for {
		select {
		case <-ticker.C:
			fmt.Println("===", time.Since(startedAt), "=================================================")
		case <-stopTest:
			return
		case <-startCase:
			test()
		}
	}
}

type InitializedNodes struct {
	addresses      []string
	controllers    []consensus.Controller
	nodeKeepers    []beat.NodeKeeper
	transports     []transport.DatagramTransport
	contexts       []context.Context
	staticProfiles []profiles.StaticProfile
}

type GeneratedNodes struct {
	nodes          []nodeinfo.NetworkNode
	meta           []*nodeMeta
	discoveryNodes []nodeinfo.NetworkNode
}

func generateNodes(countNeutral, countHeavy, countLight, countVirtual int, discoveryNodes []nodeinfo.NetworkNode) (GeneratedNodes, error) {
	nodeIdentities := generateNodeIdentities(countNeutral, countHeavy, countLight, countVirtual)
	nodeInfos := generateNodeInfos(nodeIdentities)
	nodes, dn, err := nodesFromInfo(nodeInfos)

	if len(discoveryNodes) > 0 {
		dn = discoveryNodes
	}

	if err != nil {
		return GeneratedNodes{}, err
	}

	return GeneratedNodes{
		nodes:          nodes,
		meta:           nodeInfos,
		discoveryNodes: dn,
	}, nil
}

func newNodes(size int) InitializedNodes {
	return InitializedNodes{
		addresses:      make([]string, size),
		controllers:    make([]consensus.Controller, size),
		transports:     make([]transport.DatagramTransport, size),
		contexts:       make([]context.Context, size),
		staticProfiles: make([]profiles.StaticProfile, size),
		nodeKeepers:    make([]beat.NodeKeeper, size),
	}
}

func initNodes(ctx context.Context, mode consensus.Mode, nodes GeneratedNodes, strategy NetStrategy) (InitializedNodes, error) {
	ns := newNodes(len(nodes.nodes))

	for i, n := range nodes.nodes {
		nodeKeeper := memstor.NewNodeKeeper(nodeinfo.NodeRef(n), nodeinfo.NodeRole(n))
		// nodeKeeper.SetInitialSnapshot(nodes.nodes)
		ns.nodeKeepers[i] = nodeKeeper

		certificateManager := initCrypto(n, nodes.discoveryNodes)
		datagramHandler := adapters.NewDatagramHandler()

		conf := configuration.NewHostNetwork().Transport
		conf.Address = nodeinfo.NodeAddr(n)
		ns.addresses[i] = conf.Address

		transportFactory := transport.NewFactory(conf)
		datagramTransport, err := transportFactory.CreateDatagramTransport(datagramHandler)
		if err != nil {
			return InitializedNodes{}, err
		}

		delayTransport := strategy.GetLink(datagramTransport)
		ns.transports[i] = delayTransport

		controller := consensus.New(ctx, consensus.Dep{
			KeyProcessor:       keyProcessor,
			CertificateManager: certificateManager,
			KeyStore:           keystore.NewInplaceKeyStore(nodes.meta[i].privateKey),
			TransportCryptography: adapters.NewTransportCryptographyFactory(scheme),

			NodeKeeper:        nodeKeeper,
			DatagramTransport: delayTransport,

			LocalNodeProfile: nil, // TODO

			StateGetter: &nshGen{nshDelay: defaultNshGenerationDelay},
			PulseChanger: &pulseChanger{
				nodeKeeper: nodeKeeper,
			},
			StateUpdater: &stateUpdater{
				nodeKeeper: nodeKeeper,
			},
			EphemeralController: &ephemeralController{
				allowed: true,
			},
		}).ControllerFor(mode, datagramHandler)

		ns.controllers[i] = controller
		ctx, _ = inslogger.WithFields(ctx, map[string]interface{}{
			"node_id":      n.GetNodeID(),
			"node_address": nodeinfo.NodeAddr(n),
		})
		ns.contexts[i] = ctx
		err = delayTransport.Start(ctx)
		if err != nil {
			return InitializedNodes{}, err
		}

		ns.staticProfiles[i] = n.GetStatic()
	}

	return ns, nil
}

func initPulsar(ctx context.Context, delta uint16, ns InitializedNodes) {
	pulsar := NewPulsar(delta, ns.addresses, ns.transports)
	go func() {
		for {
			pulsar.Pulse(ctx, 4+len(ns.staticProfiles)/10)
		}
	}()
}

func initLogger(t *testing.T) context.Context {
	cfg := configuration.NewLog()
	cfg.LLBufferSize = 0

	instestlogger.SetTestOutputWithCfg(t, cfg)
	ctx, _ := inslogger.InitNodeLoggerByGlobal("", "")
	return ctx
}

func generateNodeIdentities(countNeutral, countHeavy, countLight, countVirtual int) []nodeIdentity {
	r := make([]nodeIdentity, 0, countNeutral+countHeavy+countLight+countVirtual)

	r = _generateNodeIdentity(r, countNeutral, member.PrimaryRoleUnknown)
	r = _generateNodeIdentity(r, countHeavy, member.PrimaryRoleHeavyMaterial)
	r = _generateNodeIdentity(r, countLight, member.PrimaryRoleLightMaterial)
	r = _generateNodeIdentity(r, countVirtual, member.PrimaryRoleVirtual)

	return r
}

func _generateNodeIdentity(r []nodeIdentity, count int, role member.PrimaryRole) []nodeIdentity {
	for i := 0; i < count; i++ {
		port := portOffset
		r = append(r, nodeIdentity{
			role: role,
			addr: fmt.Sprintf("127.0.0.1:%d", port),
		})
		portOffset += 1
	}
	return r
}

func generateNodeInfos(nodeIdentities []nodeIdentity) []*nodeMeta {
	nodeInfos := make([]*nodeMeta, 0, len(nodeIdentities))
	for _, ni := range nodeIdentities {
		privateKey, _ := keyProcessor.GeneratePrivateKey()
		publicKey := keyProcessor.ExtractPublicKey(privateKey)

		nodeInfos = append(nodeInfos, &nodeMeta{
			nodeIdentity: ni,
			publicKey:    publicKey,
			privateKey:   privateKey,
		})
	}
	return nodeInfos
}

type nodeIdentity struct {
	role member.PrimaryRole
	addr string
}

type nodeMeta struct {
	nodeIdentity
	privateKey crypto.PrivateKey
	publicKey  crypto.PublicKey
}

func getAnnounceSignature(
	node nodeinfo.NetworkNode,
	isDiscovery bool,
	kp cryptography.KeyProcessor,
	key crypto.PrivateKey,
	scheme cryptography.PlatformCryptographyScheme,
) ([]byte, *cryptography.Signature, error) {

	addr, err := endpoints.NewIPAddress(nodeinfo.NodeAddr(node))
	if err != nil {
		return nil, nil, err
	}

	pk, err := kp.ExportPublicKeyBinary(adapters.ECDSAPublicKeyOfNode(node))
	if err != nil {
		return nil, nil, err
	}

	return gateway.CalcAnnounceSignature(node.GetNodeID(), nodeinfo.NodeRole(node), addr,
		node.GetStatic().GetStartPower(), isDiscovery, pk, keystore.NewInplaceKeyStore(key), scheme)
}

func nodesFromInfo(nodeInfos []*nodeMeta) ([]nodeinfo.NetworkNode, []nodeinfo.NetworkNode, error) {
	nodes := make([]nodeinfo.NetworkNode, len(nodeInfos))
	discoveryNodes := make([]nodeinfo.NetworkNode, 0)

	for i, info := range nodeInfos {
		var isDiscovery bool
		if info.role == member.PrimaryRoleHeavyMaterial || info.role == member.PrimaryRoleUnknown {
			isDiscovery = true
		}

		nn := newNetworkNode(info.addr, info.role, info.publicKey)
		nodes[i] = nn
		if isDiscovery {
			discoveryNodes = append(discoveryNodes, nn)
		}

		d, s, err := getAnnounceSignature(nn, isDiscovery,
			keyProcessor, info.privateKey, scheme,
		)
		if err != nil {
			return nil, nil, err
		}

		dsg := cryptkit.NewSignedDigest(
			cryptkit.NewDigest(longbits.NewBits512FromBytes(d), legacyadapter.SHA3Digest512),
			cryptkit.NewSignature(longbits.NewBits512FromBytes(s.Bytes()), legacyadapter.SHA3Digest512.SignedBy(legacyadapter.SECP256r1Sign)),
		)

		nn.SetSignature(dsg)
	}

	return nodes, discoveryNodes, nil
}

func newNetworkNode(addr string, role member.PrimaryRole, pk crypto.PublicKey) *mutable.Node {
	n := mutable.NewTestNode(gen.UniqueGlobalRef(), role, addr)
	n.SetShortID(node.ShortNodeID(shortNodeIdOffset))
	n.SetPublicKeyStore(adapters.ECDSAPublicKeyAsPublicKeyStore(pk))
	n.SetNodePublicKey(adapters.ECDSAPublicKeyAsSignatureKeyHolder(pk, keyProcessor))

	shortNodeIdOffset += 1
	return n
}

func initCrypto(node nodeinfo.NetworkNode, discoveryNodes []nodeinfo.NetworkNode) *mandates.CertificateManager {
	pubKey := adapters.ECDSAPublicKeyOfNode(node)

	publicKey, _ := keyProcessor.ExportPublicKeyPEM(pubKey)

	bootstrapNodes := make([]mandates.BootstrapNode, 0, len(discoveryNodes))
	for _, dn := range discoveryNodes {
		pubKey := adapters.ECDSAPublicKeyOfNode(dn)
		pubKeyBuf, _ := keyProcessor.ExportPublicKeyPEM(pubKey)

		bootstrapNode := mandates.NewBootstrapNode(
			pubKey,
			string(pubKeyBuf[:]),
			nodeinfo.NodeAddr(dn),
			nodeinfo.NodeRef(dn).String(),
			nodeinfo.NodeRole(dn).String(),
		)
		bootstrapNodes = append(bootstrapNodes, *bootstrapNode)
	}

	cert := &mandates.Certificate{
		AuthorizationCertificate: mandates.AuthorizationCertificate{
			PublicKey: string(publicKey[:]),
			Reference: nodeinfo.NodeRef(node).String(),
			Role:      nodeinfo.NodeRole(node).String(),
		},
		BootstrapNodes: bootstrapNodes,
	}

	// dump cert and read it again from json for correct private files initialization
	jsonCert, _ := cert.Dump()
	cert, _ = mandates.ReadCertificateFromReader(pubKey, keyProcessor, strings.NewReader(jsonCert))
	return mandates.NewCertificateManager(cert)
}

const defaultNshGenerationDelay = time.Millisecond * 0

type nshGen struct {
	nshDelay time.Duration
}

func (ng *nshGen) RequestNodeState(fn chorus.NodeStateFunc) {
	delay := ng.nshDelay
	if delay != 0 {
		time.Sleep(delay)
	}

	nshBytes := longbits.Bits512{}
	rand.Read(nshBytes[:])

	fn(api.UpstreamState{NodeState: cryptkit.NewDigest(nshBytes, "random")})
}

func (ng *nshGen) CancelNodeState() {}

type pulseChanger struct {
	nodeKeeper beat.NodeKeeper
}

func (pc *pulseChanger) ChangeBeat(ctx context.Context, report api.UpstreamReport, pu beat.Beat) {
	inslogger.FromContext(ctx).Info(">>>>>> Change pulse called")
	if err := pc.nodeKeeper.AddCommittedBeat(pu); err != nil {
		panic(err)
	}
}

type stateUpdater struct {
	nodeKeeper beat.NodeKeeper
}

func (su *stateUpdater) UpdateState(ctx context.Context, beat beat.Beat) {
	inslogger.FromContext(ctx).Info(">>>>>> Update state called")

	if err := su.nodeKeeper.AddExpectedBeat(beat); err != nil {
		panic(err)
	}
	if beat.IsFromPulsar() {
		return
	}
	if err := su.nodeKeeper.AddCommittedBeat(beat); err != nil {
		panic(err)
	}
}

type ephemeralController struct {
	allowed bool
}

func (e *ephemeralController) EphemeralMode(census.OnlinePopulation) bool {
	return e.allowed
}
