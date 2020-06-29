// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package tests

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/serialization"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/node"
	"github.com/insolar/assured-ledger/ledger-core/network/nodenetwork"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
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
	nodeKeepers    []network.NodeKeeper
	transports     []transport.DatagramTransport
	contexts       []context.Context
	staticProfiles []profiles.StaticProfile
}

type GeneratedNodes struct {
	nodes          []node2.NetworkNode
	meta           []*nodeMeta
	discoveryNodes []node2.NetworkNode
}

func generateNodes(countNeutral, countHeavy, countLight, countVirtual int, discoveryNodes []node2.NetworkNode) (*GeneratedNodes, error) {
	nodeIdentities := generateNodeIdentities(countNeutral, countHeavy, countLight, countVirtual)
	nodeInfos := generateNodeInfos(nodeIdentities)
	nodes, dn, err := nodesFromInfo(nodeInfos)

	if len(discoveryNodes) > 0 {
		dn = discoveryNodes
	}

	if err != nil {
		return nil, err
	}

	return &GeneratedNodes{
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
		nodeKeepers:    make([]network.NodeKeeper, size),
	}
}

func initNodes(ctx context.Context, mode consensus.Mode, nodes GeneratedNodes, strategy NetStrategy) (*InitializedNodes, error) {
	ns := newNodes(len(nodes.nodes))

	for i, n := range nodes.nodes {
		nodeKeeper := nodenetwork.NewNodeKeeper(n)
		nodeKeeper.SetInitialSnapshot(nodes.nodes)
		ns.nodeKeepers[i] = nodeKeeper

		certificateManager := initCrypto(n, nodes.discoveryNodes)
		datagramHandler := adapters.NewDatagramHandler()

		conf := configuration.NewHostNetwork().Transport
		conf.Address = n.Address()
		ns.addresses[i] = n.Address()

		transportFactory := transport.NewFactory(conf)
		datagramTransport, err := transportFactory.CreateDatagramTransport(datagramHandler)
		if err != nil {
			return nil, err
		}

		delayTransport := strategy.GetLink(datagramTransport)
		ns.transports[i] = delayTransport

		controller := consensus.New(ctx, consensus.Dep{
			KeyProcessor:       keyProcessor,
			Scheme:             scheme,
			CertificateManager: certificateManager,
			KeyStore:           keystore.NewInplaceKeyStore(nodes.meta[i].privateKey),

			NodeKeeper:        nodeKeeper,
			DatagramTransport: delayTransport,

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
			"node_id":      n.ShortID(),
			"node_address": n.Address(),
		})
		ns.contexts[i] = ctx
		err = delayTransport.Start(ctx)
		if err != nil {
			return nil, err
		}

		ns.staticProfiles[i] = adapters.NewStaticProfile(n, certificateManager.GetCertificate(), keyProcessor)
	}

	return &ns, nil
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

	r = _generateNodeIdentity(r, countNeutral, node2.StaticRoleUnknown)
	r = _generateNodeIdentity(r, countHeavy, node2.StaticRoleHeavyMaterial)
	r = _generateNodeIdentity(r, countLight, node2.StaticRoleLightMaterial)
	r = _generateNodeIdentity(r, countVirtual, node2.StaticRoleVirtual)

	return r
}

func _generateNodeIdentity(r []nodeIdentity, count int, role node2.StaticRole) []nodeIdentity {
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
	role node2.StaticRole
	addr string
}

type nodeMeta struct {
	nodeIdentity
	privateKey crypto.PrivateKey
	publicKey  crypto.PublicKey
}

func getAnnounceSignature(
	node node2.NetworkNode,
	isDiscovery bool,
	kp cryptography.KeyProcessor,
	key *ecdsa.PrivateKey,
	scheme cryptography.PlatformCryptographyScheme,
) ([]byte, *cryptography.Signature, error) {

	brief := serialization.NodeBriefIntro{}
	brief.ShortID = node.ShortID()
	brief.SetPrimaryRole(adapters.StaticRoleToPrimaryRole(node.Role()))
	if isDiscovery {
		brief.SpecialRoles = member.SpecialRoleDiscovery
	}
	brief.StartPower = 10

	addr, err := endpoints.NewIPAddress(node.Address())
	if err != nil {
		return nil, nil, err
	}
	copy(brief.Endpoint[:], addr[:])

	pk, err := kp.ExportPublicKeyBinary(node.PublicKey())
	if err != nil {
		return nil, nil, err
	}

	copy(brief.NodePK[:], pk)

	buf := &bytes.Buffer{}
	err = brief.SerializeTo(nil, buf)
	if err != nil {
		return nil, nil, err
	}

	data := buf.Bytes()
	data = data[:len(data)-64]

	digest := scheme.IntegrityHasher().Hash(data)
	sign, err := scheme.DigestSigner(key).Sign(digest)
	if err != nil {
		return nil, nil, err
	}

	return digest, sign, nil
}

func nodesFromInfo(nodeInfos []*nodeMeta) ([]node2.NetworkNode, []node2.NetworkNode, error) {
	nodes := make([]node2.NetworkNode, len(nodeInfos))
	discoveryNodes := make([]node2.NetworkNode, 0)

	for i, info := range nodeInfos {
		var isDiscovery bool
		if info.role == node2.StaticRoleHeavyMaterial || info.role == node2.StaticRoleUnknown {
			isDiscovery = true
		}

		nn := newNetworkNode(info.addr, info.role, info.publicKey)
		nodes[i] = nn
		if isDiscovery {
			discoveryNodes = append(discoveryNodes, nn)
		}

		d, s, err := getAnnounceSignature(
			nn,
			isDiscovery,
			keyProcessor,
			info.privateKey.(*ecdsa.PrivateKey),
			scheme,
		)
		if err != nil {
			return nil, nil, err
		}
		nn.(node.MutableNode).SetSignature(d, *s)
	}

	return nodes, discoveryNodes, nil
}

func newNetworkNode(addr string, role node2.StaticRole, pk crypto.PublicKey) node.MutableNode {
	n := node.NewNode(
		gen.UniqueGlobalRef(),
		role,
		pk,
		addr,
		"",
	)
	mn := n.(node.MutableNode)
	mn.SetShortID(node2.ShortNodeID(shortNodeIdOffset))

	shortNodeIdOffset += 1
	return mn
}

func initCrypto(node node2.NetworkNode, discoveryNodes []node2.NetworkNode) *mandates.CertificateManager {
	pubKey := node.PublicKey()

	publicKey, _ := keyProcessor.ExportPublicKeyPEM(pubKey)

	bootstrapNodes := make([]mandates.BootstrapNode, 0, len(discoveryNodes))
	for _, dn := range discoveryNodes {
		pubKey := dn.PublicKey()
		pubKeyBuf, _ := keyProcessor.ExportPublicKeyPEM(pubKey)

		bootstrapNode := mandates.NewBootstrapNode(
			pubKey,
			string(pubKeyBuf[:]),
			dn.Address(),
			dn.ID().String(),
			dn.Role().String(),
		)
		bootstrapNodes = append(bootstrapNodes, *bootstrapNode)
	}

	cert := &mandates.Certificate{
		AuthorizationCertificate: mandates.AuthorizationCertificate{
			PublicKey: string(publicKey[:]),
			Reference: node.ID().String(),
			Role:      node.Role().String(),
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

func (ng *nshGen) State() []byte {
	delay := ng.nshDelay
	if delay != 0 {
		time.Sleep(delay)
	}

	nshBytes := make([]byte, 64)
	rand.Read(nshBytes)

	return nshBytes
}

type pulseChanger struct {
	nodeKeeper network.NodeKeeper
}

func (pc *pulseChanger) ChangePulse(ctx context.Context, pulse pulsestor.Pulse) {
	inslogger.FromContext(ctx).Info(">>>>>> Change pulse called")
	pc.nodeKeeper.MoveSyncToActive(ctx, pulse.PulseNumber)
}

type stateUpdater struct {
	nodeKeeper network.NodeKeeper
}

func (su *stateUpdater) UpdateState(ctx context.Context, pulseNumber pulse.Number, nodes []node2.NetworkNode, cloudStateHash []byte) {
	inslogger.FromContext(ctx).Info(">>>>>> Update state called")

	su.nodeKeeper.Sync(ctx, pulseNumber, nodes)
}

type ephemeralController struct {
	allowed bool
}

func (e *ephemeralController) EphemeralMode(nodes []node2.NetworkNode) bool {
	return e.allowed
}
