// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package servicenetwork

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat/memstor"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeset"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/controller"
	"github.com/insolar/assured-ledger/ledger-core/network/gateway"
	"github.com/insolar/assured-ledger/ledger-core/network/gateway/bootstrap"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork"
	"github.com/insolar/assured-ledger/ledger-core/network/routing"
	"github.com/insolar/assured-ledger/ledger-core/network/termination"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

var _ beat.NodeNetwork = &ServiceNetwork{}

type ServiceNetwork struct {
	cfg configuration.Configuration
	cm  *component.Manager

	// dependencies
	CertificateManager nodeinfo.CertificateManager `inject:""`

	// subcomponents
	RPC                controller.RPCController   `inject:"subcomponent"`
	NodeKeeper         beat.NodeKeeper            `inject:"subcomponent"`
	TerminationHandler network.TerminationHandler `inject:"subcomponent"`

	HostNetwork network.HostNetwork

	Gatewayer   network.Gatewayer
	BaseGateway *gateway.Base

	router watermillRouter
}

// NewServiceNetwork returns a new ServiceNetwork.
func NewServiceNetwork(conf configuration.Configuration, rootCm *component.Manager) (*ServiceNetwork, error) {
	if rootCm != nil {
		rootCm.SetLogger(global.Logger())
	}
	serviceNetwork := &ServiceNetwork{cm: component.NewManager(rootCm), cfg: conf}
	serviceNetwork.cm.SetLogger(global.Logger())
	return serviceNetwork, nil
}

// Init implements component.Initer
func (n *ServiceNetwork) Init(ctx context.Context) error {
	hostNetwork, err := hostnetwork.NewHostNetwork(n.CertificateManager.GetCertificate().GetNodeRef().String())
	if err != nil {
		return throw.W(err, "failed to create hostnetwork")
	}
	n.HostNetwork = hostNetwork

	options := network.ConfigureOptions(n.cfg)

	cert := n.CertificateManager.GetCertificate()

	nodeNetwork, err := memstor.NewNodeNetwork(n.cfg.Host.Transport, cert)
	if err != nil {
		return throw.W(err, "failed to create NodeNetwork")
	}

	n.BaseGateway = &gateway.Base{Options: options}
	n.Gatewayer = gateway.NewGatewayer(n.BaseGateway.NewGateway(ctx, network.NoNetworkState))

	table := &routing.Table{}

	n.cm.Inject(n,
		table,
		cert,
		transport.NewFactory(n.cfg.Host.Transport),
		hostNetwork,
		nodeNetwork,
		controller.NewRPCController(options),
		bootstrap.NewRequester(options),
		memstor.NewMemoryStorage(),
		n.BaseGateway,
		n.Gatewayer,
		termination.NewHandler(n),
	)

	err = n.cm.Init(ctx)
	if err != nil {
		return throw.W(err, "failed to init internal components")
	}

	return nil
}

// Start implements component.Starter
func (n *ServiceNetwork) Start(ctx context.Context) error {
	err := n.cm.Start(ctx)
	if err != nil {
		return throw.W(err, "failed to start component manager")
	}

	p := network.NetworkedPulse{}
	p.PulseEpoch = pulse.EphemeralPulseEpoch

	n.Gatewayer.Gateway().Run(ctx, p.Data)
	n.RPC.RemoteProcedureRegister(deliverWatermillMsg, n.processIncoming)

	return nil
}

func (n *ServiceNetwork) Leave(ctx context.Context, eta pulse.Number) {
	logger := inslogger.FromContext(ctx)
	logger.Info("Gracefully stopping service network")

	// TODO: fix leave
	// n.consensusController.Leave(0)
}

func (n *ServiceNetwork) GracefulStop(ctx context.Context) error {
	logger := inslogger.FromContext(ctx)
	// node leaving from network
	// all components need to do what they want over net in gracefulStop

	logger.Info("ServiceNetwork.GracefulStop wait for accepting leaving claim")
	// TODO PLAT-594
	// For now graceful stop is broken
	// n.TerminationHandler.Leave(ctx, 0)
	logger.Info("ServiceNetwork.GracefulStop - leaving claim accepted")

	return nil
}

// Stop implements component.Stopper
func (n *ServiceNetwork) Stop(ctx context.Context) error {
	return n.cm.Stop(ctx)
}

func (n *ServiceNetwork) GetLocalNodeReference() reference.Holder {
	return n.NodeKeeper.GetLocalNodeReference()
}

func (n *ServiceNetwork) GetLocalNodeRole() member.PrimaryRole {
	return n.NodeKeeper.GetLocalNodeRole()
}

func (n *ServiceNetwork) GetNodeSnapshot(p pulse.Number) beat.NodeSnapshot {
	return n.NodeKeeper.GetNodeSnapshot(p)
}

func (n *ServiceNetwork) FindAnyLatestNodeSnapshot() beat.NodeSnapshot {
	return n.NodeKeeper.FindAnyLatestNodeSnapshot()
}

func (n *ServiceNetwork) GetCert(ctx context.Context, ref reference.Global) (nodeinfo.Certificate, error) {
	return n.Gatewayer.Gateway().Auther().GetCert(ctx, ref)
}

func (n *ServiceNetwork) CreateMessagesRouter(ctx context.Context) messagesender.MessageRouter {
	if n.router.IsZero() {
		n.router = newWatermillRouter(ctx, n.SendMessageHandler)
	}
	return n.router
}
