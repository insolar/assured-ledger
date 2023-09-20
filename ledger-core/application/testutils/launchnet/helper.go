package launchnet

import (
	"context"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/cloud"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Helper struct {
	pulseLock *sync.Mutex

	pulseGenerator    *testutils.PulseGenerator
	NodeController    insapp.MultiController
	NetworkController *cloud.NetworkController

	ConfigurationProvider *cloud.ConfigurationProvider
}

func (h *Helper) IncrementPulse(ctx context.Context) {
	h.pulseLock.Lock()
	defer h.pulseLock.Unlock()

	h.incrementPulse(ctx)
}

func (h *Helper) incrementPulse(ctx context.Context) {
	h.pulseGenerator.Generate()

	h.NetworkController.Distribute(ctx, h.pulseGenerator.GetLastPulsePacket())
}

func (h *Helper) SyncPulse(ctx context.Context) {
	h.NetworkController.Distribute(ctx, h.pulseGenerator.GetLastPulsePacket())
}

func (h *Helper) PartialPulse(ctx context.Context, packet pulsar.PulsePacket, whiteList map[reference.Global]struct{}) {
	h.pulseLock.Lock()
	defer h.pulseLock.Unlock()

	h.pulseGenerator.Generate()
	h.NetworkController.PartialDistribute(ctx, packet, whiteList)
}

func (h *Helper) AddNode(role member.PrimaryRole) reference.Global {
	nodeRef := h.ConfigurationProvider.RunNode(role)
	started, err := h.NodeController.AppStart(nodeRef.String())
	if err != nil {
		panic(throw.W(err, "failed to start new node"))
	}
	if !started {
		panic(throw.E("failed to start node"))
	}
	return nodeRef
}

func (h *Helper) StopNode(role member.PrimaryRole, nodeRef reference.Global) {
	h.ConfigurationProvider.FreeNode(role, nodeRef)
	stopped, err := h.NodeController.AppStop(nodeRef.String())
	if err != nil {
		panic(throw.W(err, "failed to stop node"))
	}
	if !stopped {
		panic(throw.E("failed to stop node"))
	}
}

func (h *Helper) GetAPIAddresses() []string {
	appConfigs := h.ConfigurationProvider.GetAppConfigs()
	apiAddresses := make([]string, 0, len(appConfigs))
	for _, el := range appConfigs {
		apiAddresses = append(apiAddresses, el.TestWalletAPI.Address)
	}
	return apiAddresses
}

func (h *Helper) GetAdminAddresses() []string {
	appConfigs := h.ConfigurationProvider.GetAppConfigs()
	adminAddresses := make([]string, 0, len(appConfigs))
	for _, el := range appConfigs {
		adminAddresses = append(adminAddresses, el.AdminAPIRunner.Address)
	}
	return adminAddresses
}
