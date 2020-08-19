// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/insolar/component-manager"
	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	transport2 "github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/transport"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/version"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/gateway/bootstrap"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/host"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/rules"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

const (
	bootstrapTimeoutMessage = "Bootstrap timeout exceeded"
)

// Base is abstract class for gateways
type Base struct {
	component.Initer

	Self                network.Gateway
	Gatewayer           network.Gatewayer                       `inject:""`
	NodeKeeper          beat.NodeKeeper                         `inject:""`
	CryptographyService cryptography.Service                    `inject:""`
	CryptographyScheme  cryptography.PlatformCryptographyScheme `inject:""`
	CertificateManager  nodeinfo.CertificateManager             `inject:""`
	HostNetwork         network.HostNetwork                     `inject:""`
	PulseAppender       beat.Appender                           `inject:""`
	PulseManager        chorus.Conductor                        `inject:""`
	BootstrapRequester  bootstrap.Requester                     `inject:""`
	KeyProcessor        cryptography.KeyProcessor               `inject:""`
	Aborter             network.Aborter                         `inject:""`
	TransportFactory    transport.Factory                       `inject:""`

	transportCrypt    transport2.CryptographyAssistant
	datagramHandler   *adapters.DatagramHandler
	datagramTransport transport.DatagramTransport

	ConsensusMode       consensus.Mode
	consensusInstaller  consensus.Installer
	ConsensusController consensus.Controller
	consensusStarted    uint32

	Options        *network.Options
	bootstrapTimer *time.Timer // nolint
	bootstrapETA   time.Duration
	localCandidate *adapters.Candidate
	localStatic    profiles.StaticProfile

	// Next request backoff.
	backoff time.Duration // nolint

	pulseWatchdog *pulseWatchdog

	isDiscovery     bool                   // nolint
	isJoinAssistant bool                   // nolint
	joinAssistant   nodeinfo.DiscoveryNode // joinAssistant
}

// NewGateway creates new gateway on top of existing
func (g *Base) NewGateway(ctx context.Context, state network.State) network.Gateway {
	inslogger.FromContext(ctx).Infof("NewGateway %s", state.String())
	switch state {
	case network.NoNetworkState:
		g.Self = newNoNetwork(g)
	case network.CompleteNetworkState:
		g.Self = newComplete(g)
	case network.JoinerBootstrap:
		g.Self = newJoinerBootstrap(g)
	case network.DiscoveryBootstrap:
		g.Self = newDiscoveryBootstrap(g)
	case network.WaitConsensus:
		err := g.StartConsensus(ctx)
		if err != nil {
			g.FailState(ctx, fmt.Sprintf("Failed to start consensus: %s", err))
		}
		g.Self = newWaitConsensus(g)
	case network.WaitMajority:
		g.Self = newWaitMajority(g)
	case network.WaitMinRoles:
		g.Self = newWaitMinRoles(g)
	case network.WaitPulsar:
		g.Self = newWaitPulsar(g)
	default:
		inslogger.FromContext(ctx).Panic("Try to switch network to unknown state. Memory of process is inconsistent.")
	}
	return g.Self
}

func (g *Base) Init(ctx context.Context) error {
	g.pulseWatchdog = newPulseWatchdog(ctx, g.Gatewayer.Gateway(), g.Options.PulseWatchdogTimeout)

	g.HostNetwork.RegisterRequestHandler(
		types.Authorize, g.discoveryMiddleware(g.announceMiddleware(g.HandleNodeAuthorizeRequest)), // validate cert
	)
	g.HostNetwork.RegisterRequestHandler(
		types.Bootstrap, g.announceMiddleware(g.HandleNodeBootstrapRequest), // provide joiner claim
	)
	g.HostNetwork.RegisterRequestHandler(types.UpdateSchedule, g.HandleUpdateSchedule)
	g.HostNetwork.RegisterRequestHandler(types.Reconnect, g.HandleReconnect)

	g.bootstrapETA = g.Options.BootstrapTimeout

	return g.initConsensus(ctx)
}

func (g *Base) Stop(ctx context.Context) error {
	err := g.datagramTransport.Stop(ctx)
	if err != nil {
		return throw.W(err, "failed to stop datagram transport")
	}

	g.pulseWatchdog.Stop()
	return nil
}

func (g *Base) initConsensus(ctx context.Context) error {
	g.ConsensusMode = consensus.Joiner
	g.datagramHandler = adapters.NewDatagramHandler()
	datagramTransport, err := g.TransportFactory.CreateDatagramTransport(g.datagramHandler)
	if err != nil {
		return throw.W(err, "failed to create datagramTransport")
	}
	g.datagramTransport = datagramTransport
	g.transportCrypt = adapters.NewTransportCryptographyFactory(g.CryptographyScheme)


	// transport start should be here because of TestComponents tests, couldn't localNodeAsCandidate with 0 port
	err = g.datagramTransport.Start(ctx)
	if err != nil {
		return throw.W(err, "failed to start datagram transport")
	}

	err = g.localNodeAsCandidate()
	if err != nil {
		return throw.W(err, "failed to localNodeAsCandidate")
	}

	proxy := consensusProxy{g.Gatewayer}
	g.consensusInstaller = consensus.New(ctx, consensus.Dep{
		KeyProcessor:        g.KeyProcessor,
		CertificateManager:  g.CertificateManager,
		KeyStore:            getKeyStore(g.CryptographyService),
		NodeKeeper:          g.NodeKeeper,
		LocalNodeProfile:    g.localStatic, // initialized by localNodeAsCandidate()
		StateGetter:         proxy,
		PulseChanger:        proxy,
		StateUpdater:        proxy,
		DatagramTransport:   g.datagramTransport,
		EphemeralController: g,
		TransportCryptography: g.transportCrypt,
	})

	return nil
}

func (g *Base) localNodeAsCandidate() error {
	cert := g.CertificateManager.GetCertificate()

	staticProfile, err := CreateLocalNodeProfile(g.NodeKeeper, cert,
		g.datagramTransport.Address(),
		g.KeyProcessor, g.CryptographyService, g.CryptographyScheme)

	if err != nil {
		return err
	}

	g.localStatic = staticProfile
	g.localCandidate = adapters.NewCandidate(staticProfile, g.KeyProcessor)
	return nil
}

func (g *Base) GetLocalNodeStaticProfile() profiles.StaticProfile {
	if g.localStatic == nil {
		panic(throw.IllegalState())
	}
	return g.localStatic
}


func (g *Base) StartConsensus(ctx context.Context) error {

	cert := g.CertificateManager.GetCertificate()
	if network.OriginIsJoinAssistant(cert) {
		// one of the nodes has to be in consensus.ReadyNetwork state,
		// all other nodes has to be in consensus.Joiner
		// TODO: fix Assistant node can't join existing network
		g.ConsensusMode = consensus.ReadyNetwork
	}

	g.ConsensusController = g.consensusInstaller.ControllerFor(g.ConsensusMode, g.datagramHandler)
	g.ConsensusController.RegisterFinishedNotifier(func(ctx context.Context, report network.Report) {
		g.Gatewayer.Gateway().OnConsensusFinished(ctx, report)
	})

	atomic.StoreUint32(&g.consensusStarted, 1)
	return nil
}

func (g *Base) RequestNodeState(fn chorus.NodeStateFunc) {
	// This is wrong, but current structure of gateway requires too much hassle to make it right
	fn(api.UpstreamState{
		NodeState: cryptkit.NewDigest(longbits.Bits512{}, "empty"),
	})
}

func (g *Base) CancelNodeState() {}

func (g *Base) OnPulseFromConsensus(ctx context.Context, pu beat.Beat) {
	g.pulseWatchdog.Reset()

	if err := g.NodeKeeper.AddCommittedBeat(pu); err != nil {
		inslogger.FromContext(ctx).Panic(err)
	}

	nodeCount := int64(pu.Online.GetIndexedCount())
	inslogger.FromContext(ctx).Debugf("[ AddCommittedBeat ] Population size: %d", nodeCount)
	stats.Record(ctx, network.ActiveNodes.M(nodeCount))

	// nodes := g.NodeKeeper.GetNodeSnapshot(pu.PulseNumber).GetOnlineNodes()
	// inslogger.FromContext(ctx).Debugf("OnPulseFromConsensus: %d : epoch %d : nodes %d", pu.PulseNumber, pu.PulseEpoch, len(nodes))
}

// UpdateState called then Consensus is done
func (g *Base) UpdateState(ctx context.Context, pu beat.Beat) {
	err := g.NodeKeeper.AddExpectedBeat(pu)
	if err == nil && !pu.IsFromPulsar() {
		err = g.NodeKeeper.AddCommittedBeat(pu)
	}

	if err != nil {
		inslogger.FromContext(ctx).Panic(err)
	}

	nodeCount := int64(pu.Online.GetIndexedCount())
	inslogger.FromContext(ctx).Debugf("[ AddCommittedBeat ] Population size: %d", nodeCount)
}

func (g *Base) BeforeRun(ctx context.Context, pulse pulse.Data) {}

// Auther casts us to Auther or obtain it in another way
func (g *Base) Auther() network.Auther {
	if ret, ok := g.Self.(network.Auther); ok {
		return ret
	}
	panic("Our network gateway suddenly is not an Auther")
}

// Bootstrapper casts us to Bootstrapper or obtain it in another way
func (g *Base) Bootstrapper() network.Bootstrapper {
	if ret, ok := g.Self.(network.Bootstrapper); ok {
		return ret
	}
	panic("Our network gateway suddenly is not an Bootstrapper")
}

// GetCert method returns node certificate by requesting sign from discovery nodes
func (g *Base) GetCert(ctx context.Context, ref reference.Global) (nodeinfo.Certificate, error) {
	return nil, throw.New("GetCert() in non active mode")
}

// ValidateCert validates node certificate
func (g *Base) ValidateCert(ctx context.Context, authCert nodeinfo.AuthorizationCertificate) (bool, error) {
	return mandates.VerifyAuthorizationCertificate(g.CryptographyService, g.CertificateManager.GetCertificate().GetDiscoveryNodes(), authCert)
}

// ============= Bootstrap =======

func (g *Base) checkCanAnnounceCandidate(context.Context) error {
	// 1. Current node is JoinAssistant:
	// 		could announce candidate when network is initialized
	// 		NB: announcing in WaitConsensus state is allowed
	// 2. Otherwise:
	// 		could announce candidate when JoinAssistant node found in *active* list and initial consensus passed
	// 		NB: announcing in WaitConsensus state is *NOT* allowed

	state := g.Gatewayer.Gateway().GetState()
	if state > network.WaitConsensus {
		return nil
	}

	if state == network.WaitConsensus && g.isJoinAssistant {
		return nil
	}

	pn := pulse.Unknown
	if na := g.NodeKeeper.FindAnyLatestNodeSnapshot(); na != nil {
		pn = na.GetPulseNumber()
	}

	return throw.Errorf(
		"can't announce candidate: pulse=%d state=%s",
		pn, state,
	)
}

func (g *Base) announceMiddleware(handler network.RequestHandler) network.RequestHandler {
	return func(ctx context.Context, request network.ReceivedPacket) (network.Packet, error) {
		if err := g.checkCanAnnounceCandidate(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, request)
	}
}

func (g *Base) discoveryMiddleware(handler network.RequestHandler) network.RequestHandler {
	return func(ctx context.Context, request network.ReceivedPacket) (network.Packet, error) {
		if !network.OriginIsDiscovery(g.CertificateManager.GetCertificate()) {
			return nil, throw.New("only discovery nodes could authorize other nodes, this is not a discovery node")
		}
		return handler(ctx, request)
	}
}

func (g *Base) HandleNodeBootstrapRequest(ctx context.Context, request network.ReceivedPacket) (network.Packet, error) {
	if request.GetRequest() == nil || request.GetRequest().GetBootstrap() == nil {
		return nil, throw.Errorf("process bootstrap: got invalid protobuf request message: %s", request)
	}

	data := request.GetRequest().GetBootstrap()

	var nodes []nodeinfo.NetworkNode
	if na := g.NodeKeeper.FindAnyLatestNodeSnapshot(); na != nil {
		nodes = na.GetPopulation().GetProfiles()
	}

	hasCollision := false
	if len(nodes) > 1 {
		hasCollision = network.CheckShortIDCollision(nodes, data.CandidateProfile.ShortID)
	} else {
		hasCollision = data.CandidateProfile.ShortID == g.localStatic.GetStaticNodeID()
	}

	if hasCollision {
		return g.HostNetwork.BuildResponse(ctx, request, &packet.BootstrapResponse{Code: packet.UpdateShortID}), nil
	}

	err := bootstrap.ValidatePermit(data.Permit, g.CertificateManager.GetCertificate(), g.CryptographyService)
	if err != nil {
		inslogger.FromContext(ctx).Warnf("Rejected bootstrap request from node %s: %s", request.GetSender(), err.Error())
		return g.HostNetwork.BuildResponse(ctx, request, &packet.BootstrapResponse{Code: packet.Reject}), nil
	}

	type candidate struct {
		profiles.StaticProfile
		profiles.StaticProfileExtension
	}

	profile := adapters.Candidate(data.CandidateProfile).StaticProfile(g.KeyProcessor)

	err = g.ConsensusController.AddJoinCandidate(candidate{profile, profile.GetExtension()})
	if err != nil {
		inslogger.FromContext(ctx).Warnf("Retry Failed to AddJoinCandidate  %s: %s", request.GetSender(), err.Error())
		return g.HostNetwork.BuildResponse(ctx, request, &packet.BootstrapResponse{Code: packet.Retry}), nil
	}

	inslogger.FromContext(ctx).Infof("=== AddJoinCandidate id = %d, address = %s ", data.CandidateProfile.ShortID, data.CandidateProfile.Address)

	return g.HostNetwork.BuildResponse(ctx, request,
		&packet.BootstrapResponse{
			Code:       packet.Accepted,
			ETASeconds: uint32(g.bootstrapETA.Seconds()),
		}), nil
}

// validateTimestamp returns true if difference between timestamp ant current UTC < delta
func validateTimestamp(timestamp int64, delta time.Duration) bool {
	return time.Now().UTC().Sub(time.Unix(timestamp, 0)) < delta
}

func (g *Base) HandleNodeAuthorizeRequest(ctx context.Context, request network.ReceivedPacket) (network.Packet, error) {
	if request.GetRequest() == nil || request.GetRequest().GetAuthorize() == nil {
		return nil, throw.Errorf("process authorize: got invalid protobuf request message: %s", request)
	}
	data := request.GetRequest().GetAuthorize().AuthorizeData

	if data.Version != version.Version {
		return nil, throw.Errorf("version mismatch in AuthorizeRequest: local=%s received=%s", version.Version, data.Version)
	}

	// TODO: move time.Minute to config
	if !validateTimestamp(data.Timestamp, time.Minute) {
		return g.HostNetwork.BuildResponse(ctx, request, &packet.AuthorizeResponse{
			Code:      packet.WrongTimestamp,
			Timestamp: time.Now().UTC().Unix(),
		}), nil
	}

	cert, err := mandates.Deserialize(data.Certificate, platformpolicy.NewKeyProcessor())
	if err != nil {
		return g.HostNetwork.BuildResponse(ctx, request, &packet.AuthorizeResponse{Code: packet.WrongMandate, Error: err.Error()}), nil
	}

	valid, err := g.Gatewayer.Gateway().Auther().ValidateCert(ctx, cert)
	if err != nil || !valid {
		if err == nil {
			err = throw.New("certificate validation failed")
		}

		inslogger.FromContext(ctx).Warn("AuthorizeRequest: ", err)
		return g.HostNetwork.BuildResponse(ctx, request, &packet.AuthorizeResponse{Code: packet.WrongMandate, Error: err.Error()}), nil
	}

	var nodes []nodeinfo.NetworkNode
	var discoveryCount int
	if na := g.NodeKeeper.FindAnyLatestNodeSnapshot(); na != nil {
		nodes = na.GetPopulation().GetProfiles()
	}

	var reconnectHost *host.Host
	if g.isJoinAssistant && len(nodes) > 1 && nodes != nil /* != is to fix annoying GoLand hint */ {
		randNode := nodes[rand.Intn(len(nodes))]
		reconnectHost, err = host.NewHostNS(nodeinfo.NodeAddr(randNode), nodeinfo.NodeRef(randNode), randNode.GetNodeID())

		discoveryCount := len(network.FindDiscoveriesInNodeList(nodes, g.CertificateManager.GetCertificate()))
		if discoveryCount == 0 {
			err = throw.New("missing discoveries")
			inslogger.FromContext(ctx).Warn("AuthorizeRequest: ", err)
			return nil, err
		}
	} else {
		// workaround bootstrap to the local node
		reconnectHost, err = host.NewHostNS(g.localStatic.GetDefaultEndpoint().GetIPAddress().String(),
			g.localStatic.GetExtension().GetReference(), g.localStatic.GetStaticNodeID())
		discoveryCount = 1
	}

	if err != nil {
		err = throw.W(err, "failed to get reconnectHost")
		inslogger.FromContext(ctx).Warn("AuthorizeRequest: ", err)
		return nil, err
	}

	pubKey, err := g.KeyProcessor.ExportPublicKeyPEM(adapters.ECDSAPublicKeyOfProfile(g.localStatic))
	if err != nil {
		err = throw.W(err, "failed to export PK")
		inslogger.FromContext(ctx).Warn("AuthorizeRequest: ", err)
		return nil, err
	}

	permit, err := bootstrap.CreatePermit(g.NodeKeeper.GetLocalNodeReference(),
		reconnectHost, pubKey,
		g.CryptographyService,
	)
	if err != nil {
		return nil, err
	}

	return g.HostNetwork.BuildResponse(ctx, request, &packet.AuthorizeResponse{
		Code:           packet.Success,
		Timestamp:      time.Now().UTC().Unix(),
		Permit:         permit,
		DiscoveryCount: uint32(discoveryCount),
	}), nil
}

func (g *Base) HandleUpdateSchedule(ctx context.Context, request network.ReceivedPacket) (network.Packet, error) {
	// TODO:
	return g.HostNetwork.BuildResponse(ctx, request, &packet.UpdateScheduleResponse{}), nil
}

func (g *Base) HandleReconnect(ctx context.Context, request network.ReceivedPacket) (network.Packet, error) {
	if request.GetRequest() == nil || request.GetRequest().GetReconnect() == nil {
		return nil, throw.Errorf("process reconnect: got invalid protobuf request message: %s", request)
	}

	// check permit, if permit from Discovery node
	// request.GetRequest().GetReconnect().Permit

	// TODO:
	return g.HostNetwork.BuildResponse(ctx, request, &packet.ReconnectResponse{}), nil
}

func (g *Base) OnConsensusFinished(ctx context.Context, report network.Report) {
	inslogger.FromContext(ctx).Infof("OnConsensusFinished for pulse %d", report.PulseNumber)
}

func (g *Base) EphemeralMode(nodes census.OnlinePopulation) bool {
	if nodes == nil {
		return true
	}
	_, majorityErr := rules.CheckMajorityRule(g.CertificateManager.GetCertificate(), nodes)
	minRoleErr := rules.CheckMinRole(g.CertificateManager.GetCertificate(), nodes)

	return majorityErr != nil || minRoleErr != nil
}

func (g *Base) FailState(ctx context.Context, reason string) {
	na := g.NodeKeeper.FindAnyLatestNodeSnapshot()

	addr := ""
	if na != nil {
		addr = nodeinfo.NodeAddr(na.GetPopulation().GetLocalProfile())
	}
	wrapReason := fmt.Sprintf("Abort node: ref=%s address=%s role=%v state=%s, reason=%s",
		g.NodeKeeper.GetLocalNodeReference(),
		addr,
		g.NodeKeeper.GetLocalNodeRole(),
		g.Gatewayer.Gateway().GetState().String(),
		reason,
	)
	g.Aborter.Abort(ctx, wrapReason)
}
