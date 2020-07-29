// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
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
	"github.com/insolar/assured-ledger/ledger-core/network/node"
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
	NodeKeeper          network.NodeKeeper                      `inject:""`
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

	// nolint
	OriginProvider network.OriginProvider `inject:""`

	datagramHandler   *adapters.DatagramHandler
	datagramTransport transport.DatagramTransport

	ConsensusMode       consensus.Mode
	consensusInstaller  consensus.Installer
	ConsensusController consensus.Controller
	consensusStarted    uint32

	Options         *network.Options
	bootstrapTimer  *time.Timer // nolint
	bootstrapETA    time.Duration
	originCandidate *adapters.Candidate

	// Next request backoff.
	backoff time.Duration // nolint

	pulseWatchdog *pulseWatchdog
}

// NewGateway creates new gateway on top of existing
func (g *Base) NewGateway(ctx context.Context, state nodeinfo.NetworkState) network.Gateway {
	inslogger.FromContext(ctx).Infof("NewGateway %s", state.String())
	switch state {
	case nodeinfo.NoNetworkState:
		g.Self = newNoNetwork(g)
	case nodeinfo.CompleteNetworkState:
		g.Self = newComplete(g)
	case nodeinfo.JoinerBootstrap:
		g.Self = newJoinerBootstrap(g)
	case nodeinfo.WaitConsensus:
		err := g.StartConsensus(ctx)
		if err != nil {
			g.FailState(ctx, fmt.Sprintf("Failed to start consensus: %s", err))
		}
		g.Self = newWaitConsensus(g)
	case nodeinfo.WaitMajority:
		g.Self = newWaitMajority(g)
	case nodeinfo.WaitMinRoles:
		g.Self = newWaitMinRoles(g)
	case nodeinfo.WaitPulsar:
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

	proxy := consensusProxy{g.Gatewayer}
	g.consensusInstaller = consensus.New(ctx, consensus.Dep{
		KeyProcessor:        g.KeyProcessor,
		Scheme:              g.CryptographyScheme,
		CertificateManager:  g.CertificateManager,
		KeyStore:            getKeyStore(g.CryptographyService),
		NodeKeeper:          g.NodeKeeper,
		StateGetter:         proxy,
		PulseChanger:        proxy,
		StateUpdater:        proxy,
		DatagramTransport:   g.datagramTransport,
		EphemeralController: g,
	})

	// transport start should be here because of TestComponents tests, couldn't createOriginCandidate with 0 port
	err = g.datagramTransport.Start(ctx)
	if err != nil {
		return throw.W(err, "failed to start datagram transport")
	}

	err = g.createOriginCandidate()
	if err != nil {
		return throw.W(err, "failed to createOriginCandidate")
	}

	return nil
}

func (g *Base) createOriginCandidate() error {
	// sign origin
	origin := g.NodeKeeper.GetOrigin()
	mutableOrigin := origin.(node.MutableNode)
	mutableOrigin.SetAddress(g.datagramTransport.Address())

	digest, sign, err := getAnnounceSignature(
		origin,
		network.IsDiscoveryCert(g.CertificateManager.GetCertificate()),
		g.KeyProcessor,
		getKeyStore(g.CryptographyService),
		g.CryptographyScheme,
	)
	if err != nil {
		return err
	}
	mutableOrigin.SetSignature(digest, *sign)
	g.NodeKeeper.SetInitialSnapshot([]nodeinfo.NetworkNode{origin})

	staticProfile := adapters.NewStaticProfile(origin, g.CertificateManager.GetCertificate(), g.KeyProcessor)
	g.originCandidate = adapters.NewCandidate(staticProfile, g.KeyProcessor)
	return nil
}

func (g *Base) StartConsensus(ctx context.Context) error {

	cert := g.CertificateManager.GetCertificate()
	if network.OriginIsJoinAssistant(cert) {
		// one of the nodes has to be in consensus.ReadyNetwork state,
		// all other nodes has to be in consensus.Joiner
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

	g.NodeKeeper.MoveSyncToActive(ctx, pu.PulseNumber)

	nodes := g.NodeKeeper.GetAccessor(pu.PulseNumber).GetActiveNodes()
	inslogger.FromContext(ctx).Debugf("OnPulseFromConsensus: %d : epoch %d : nodes %d", pu.PulseNumber, pu.PulseEpoch, len(nodes))
}

// UpdateState called then Consensus done
func (g *Base) UpdateState(ctx context.Context, _ pulse.Number, nodes []nodeinfo.NetworkNode, _ []byte) {
	g.NodeKeeper.Sync(ctx, nodes)
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

func (g *Base) checkCanAnnounceCandidate(ctx context.Context) error {
	// 1. Current node is heavy:
	// 		could announce candidate when network is initialized
	// 		NB: announcing in WaitConsensus state is allowed
	// 2. Otherwise:
	// 		could announce candidate when heavy node found in *active* list and initial consensus passed
	// 		NB: announcing in WaitConsensus state is *NOT* allowed

	state := g.Gatewayer.Gateway().GetState()
	if state > nodeinfo.WaitConsensus {
		return nil
	}

	if state == nodeinfo.WaitConsensus {
		cert := g.CertificateManager.GetCertificate()
		if network.OriginIsJoinAssistant(cert) {
			return nil
		}
	}

	return throw.Errorf(
		"can't announce candidate: pulse=%d state=%s",
		g.LatestPulse(ctx).PulseNumber,
		state,
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
		if !network.IsDiscoveryCert(g.CertificateManager.GetCertificate()) {
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

	nodes := g.NodeKeeper.GetAccessor(g.LatestPulse(ctx).PulseNumber).GetActiveNodes()

	if network.CheckShortIDCollision(nodes, data.CandidateProfile.ShortID) {
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
		inslogger.FromContext(ctx).Warnf("Rejected bootstrap request from node %s: %s", request.GetSender(), err.Error())
		return g.HostNetwork.BuildResponse(ctx, request, &packet.BootstrapResponse{Code: packet.Reject}), nil
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
			err = throw.New("Certificate validation failed")
		}

		inslogger.FromContext(ctx).Warn("AuthorizeRequest with invalid cert: ", err.Error())
		return g.HostNetwork.BuildResponse(ctx, request, &packet.AuthorizeResponse{Code: packet.WrongMandate, Error: err.Error()}), nil
	}

	// TODO: get random reconnectHost
	// nodes := g.NodeKeeper.GetAccessor().GetActiveNodes()

	o := g.OriginProvider.GetOrigin()
	// workaround bootstrap to the origin node
	reconnectHost, err := host.NewHostNS(o.Address(), o.GetReference(), o.GetNodeID())
	if err != nil {
		err = throw.W(err, "Failed to get reconnectHost")
		inslogger.FromContext(ctx).Warn(err.Error())
		return nil, err
	}

	pubKey, err := g.KeyProcessor.ExportPublicKeyPEM(o.PublicKey())
	if err != nil {
		err = throw.W(err, "Failed to export public key")
		inslogger.FromContext(ctx).Warn(err.Error())
		return nil, err
	}

	permit, err := bootstrap.CreatePermit(g.OriginProvider.GetOrigin().GetReference(),
		reconnectHost,
		pubKey,
		g.CryptographyService,
	)
	if err != nil {
		return nil, err
	}

	nodes := g.NodeKeeper.GetAccessor(g.LatestPulse(ctx).PulseNumber).GetActiveNodes()

	discoveryCount := len(network.FindDiscoveriesInNodeList(nodes, g.CertificateManager.GetCertificate()))
	if discoveryCount == 0 {
		panic(throw.IllegalState())
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

func (g *Base) EphemeralMode(nodes []nodeinfo.NetworkNode) bool {
	_, majorityErr := rules.CheckMajorityRule(g.CertificateManager.GetCertificate(), nodes)
	minRoleErr := rules.CheckMinRole(g.CertificateManager.GetCertificate(), nodes)

	return majorityErr != nil || minRoleErr != nil
}

func (g *Base) FailState(ctx context.Context, reason string) {
	o := g.OriginProvider.GetOrigin()
	wrapReason := fmt.Sprintf("Abort node with address: %s role: %s state: %s, reason: %s",
		o.Address(),
		o.GetPrimaryRole().String(),
		g.Gatewayer.Gateway().GetState().String(),
		reason,
	)
	g.Aborter.Abort(ctx, wrapReason)
}

func (g *Base) LatestPulse(ctx context.Context) pulse.Data {
	if g.ConsensusController == nil {
		return pulse.NewFirstEphemeralData()
	}

	return g.ConsensusController.Chronicles().GetActiveCensus().GetPulseData()
}
