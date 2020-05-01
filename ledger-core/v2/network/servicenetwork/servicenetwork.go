// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package servicenetwork

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/network"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/controller"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/gateway"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/gateway/bootstrap"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/hostnetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/nodenetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/routing"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/storage"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/termination"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/component-manager"
)

// ServiceNetwork is facade for network.
type ServiceNetwork struct {
	cfg configuration.Configuration
	cm  *component.Manager

	// dependencies
	CertificateManager insolar.CertificateManager `inject:""`

	// watermill support interfaces
	Pub message.Publisher `inject:""`

	// subcomponents
	RPC                controller.RPCController   `inject:"subcomponent"`
	PulseAccessor      storage.PulseAccessor      `inject:"subcomponent"`
	NodeKeeper         network.NodeKeeper         `inject:"subcomponent"`
	TerminationHandler network.TerminationHandler `inject:"subcomponent"`

	HostNetwork network.HostNetwork

	Gatewayer   network.Gatewayer
	BaseGateway *gateway.Base
}

// NewServiceNetwork returns a new ServiceNetwork.
func NewServiceNetwork(conf configuration.Configuration, rootCm *component.Manager) (*ServiceNetwork, error) {
	serviceNetwork := &ServiceNetwork{cm: component.NewManager(rootCm), cfg: conf}
	return serviceNetwork, nil
}

// Init implements component.Initer
func (n *ServiceNetwork) Init(ctx context.Context) error {
	hostNetwork, err := hostnetwork.NewHostNetwork(n.CertificateManager.GetCertificate().GetNodeRef().String())
	if err != nil {
		return errors.Wrap(err, "failed to create hostnetwork")
	}
	n.HostNetwork = hostNetwork

	options := network.ConfigureOptions(n.cfg)

	cert := n.CertificateManager.GetCertificate()

	nodeNetwork, err := nodenetwork.NewNodeNetwork(n.cfg.Host.Transport, cert)
	if err != nil {
		return errors.Wrap(err, "failed to create NodeNetwork")
	}

	n.BaseGateway = &gateway.Base{Options: options}
	n.Gatewayer = gateway.NewGatewayer(n.BaseGateway.NewGateway(ctx, insolar.NoNetworkState))

	table := &routing.Table{}

	n.cm.Inject(n,
		table,
		cert,
		transport.NewFactory(n.cfg.Host.Transport),
		hostNetwork,
		nodeNetwork,
		controller.NewRPCController(options),
		bootstrap.NewRequester(options),
		storage.NewMemoryStorage(),
		n.BaseGateway,
		n.Gatewayer,
		storage.NewMemoryStorage(),
		termination.NewHandler(n),
	)

	err = n.cm.Init(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to init internal components")
	}

	return nil
}

// Start implements component.Starter
func (n *ServiceNetwork) Start(ctx context.Context) error {
	err := n.cm.Start(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start component manager")
	}

	bootstrapPulse := gateway.GetBootstrapPulse(ctx, n.PulseAccessor)
	n.Gatewayer.Gateway().Run(ctx, bootstrapPulse)
	n.RPC.RemoteProcedureRegister(deliverWatermillMsg, n.processIncoming)

	return nil
}

func (n *ServiceNetwork) Leave(ctx context.Context, eta insolar.PulseNumber) {
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
	n.TerminationHandler.Leave(ctx, 0)
	logger.Info("ServiceNetwork.GracefulStop - leaving claim accepted")

	return nil
}

// Stop implements insolar.Component
func (n *ServiceNetwork) Stop(ctx context.Context) error {
	return n.cm.Stop(ctx)
}

func (n *ServiceNetwork) GetOrigin() insolar.NetworkNode {
	return n.NodeKeeper.GetOrigin()
}

func (n *ServiceNetwork) GetAccessor(p insolar.PulseNumber) network.Accessor {
	return n.NodeKeeper.GetAccessor(p)
}

func (n *ServiceNetwork) GetCert(ctx context.Context, ref *reference.Global) (insolar.Certificate, error) {
	return n.Gatewayer.Gateway().Auther().GetCert(ctx, ref)
}
