// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"
	"fmt"
	"time"

	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/node"
	"github.com/insolar/assured-ledger/ledger-core/network/rules"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
)

func newComplete(b *Base) *Complete {
	return &Complete{
		Base: b,
	}
}

type Complete struct {
	*Base
}

func (g *Complete) Run(context.Context, network.NetworkedPulse) {
	if g.bootstrapTimer != nil {
		g.bootstrapTimer.Stop()
	}

	g.HostNetwork.RegisterRequestHandler(types.SignCert, g.signCertHandler)
}

func (g *Complete) GetState() nodeinfo.NetworkState {
	return nodeinfo.CompleteNetworkState
}

func (g *Complete) BeforeRun(ctx context.Context, pulse network.NetworkedPulse) {
	if pulse.IsFromEphemeral() {
		return
	}

	err := g.PulseManager.CommitPulseChange(pulse)
	if err != nil {
		inslogger.FromContext(ctx).Panicf("failed to set start pulse: %d, %s", pulse.PulseNumber, err.Error())
	}
}

// GetCert method generates cert by requesting signs from discovery nodes
func (g *Complete) GetCert(ctx context.Context, registeredNodeRef reference.Global) (nodeinfo.Certificate, error) {
	pKey, role, err := g.getNodeInfo(ctx, registeredNodeRef)
	if err != nil {
		return nil, errors.W(err, "[ GetCert ] Couldn't get node info")
	}

	currentNodeCert := g.CertificateManager.GetCertificate()
	registeredNodeCert, err := mandates.NewUnsignedCertificate(currentNodeCert, pKey, role, registeredNodeRef.String())
	if err != nil {
		return nil, errors.W(err, "[ GetCert ] Couldn't create certificate")
	}

	for i, discoveryNode := range currentNodeCert.GetDiscoveryNodes() {
		sign, err := g.requestCertSign(ctx, discoveryNode, registeredNodeRef)
		if err != nil {
			return nil, errors.W(err, "[ GetCert ] Couldn't request cert sign")
		}
		registeredNodeCert.(*mandates.Certificate).BootstrapNodes[i].NodeSign = sign
	}
	return registeredNodeCert, nil
}

// requestCertSign method requests sign from single discovery node
func (g *Complete) requestCertSign(ctx context.Context, discoveryNode nodeinfo.DiscoveryNode, registeredNodeRef reference.Global) ([]byte, error) {
	currentNodeCert := g.CertificateManager.GetCertificate()

	if discoveryNode.GetNodeRef() == currentNodeCert.GetNodeRef() {
		sign, err := g.signCert(ctx, registeredNodeRef)
		if err != nil {
			return nil, err
		}
		return sign.Bytes(), nil
	}

	request := &packet.SignCertRequest{
		NodeRef: registeredNodeRef,
	}
	future, err := g.HostNetwork.SendRequest(ctx, types.SignCert, request, discoveryNode.GetNodeRef())
	if err != nil {
		return nil, err
	}

	p, err := future.WaitResponse(10 * time.Second)
	if err != nil {
		return nil, err
	} else if p.GetResponse().GetError() != nil {
		return nil, fmt.Errorf("[requestCertSign] Remote (%s) said %s", p.GetSender(), p.GetResponse().GetError().Error)
	}

	return p.GetResponse().GetSignCert().Sign, nil
}

func (g *Complete) getNodeInfo(ctx context.Context, nodeRef reference.Global) (string, string, error) {
	panic("deprecated")
}

func (g *Complete) signCert(ctx context.Context, registeredNodeRef reference.Global) (*cryptography.Signature, error) {
	pKey, role, err := g.getNodeInfo(ctx, registeredNodeRef)
	if err != nil {
		return nil, errors.W(err, "[ SignCert ] Couldn't extract response")
	}
	return mandates.SignCert(g.CryptographyService, pKey, role, registeredNodeRef.String())
}

// signCertHandler is handler that signs certificate for some node with node own key
func (g *Complete) signCertHandler(ctx context.Context, request network.ReceivedPacket) (network.Packet, error) {
	if request.GetRequest() == nil || request.GetRequest().GetSignCert() == nil {
		inslogger.FromContext(ctx).Warnf("process SignCert: got invalid request protobuf message: %s", request)
	}
	sign, err := g.signCert(ctx, request.GetRequest().GetSignCert().NodeRef)
	if err != nil {
		return g.HostNetwork.BuildResponse(ctx, request, &packet.ErrorResponse{Error: err.Error()}), nil
	}

	return g.HostNetwork.BuildResponse(ctx, request, &packet.SignCertResponse{Sign: sign.Bytes()}), nil
}

func (g *Complete) EphemeralMode(nodes []nodeinfo.NetworkNode) bool {
	return false
}

func (g *Complete) UpdateState(ctx context.Context, pulseNumber pulse.Number, nodes []nodeinfo.NetworkNode, cloudStateHash []byte) {
	workingNodes := node.Select(nodes, node.ListWorking)

	if _, err := rules.CheckMajorityRule(g.CertificateManager.GetCertificate(), workingNodes); err != nil {
		g.FailState(ctx, err.Error())
	}

	if err := rules.CheckMinRole(g.CertificateManager.GetCertificate(), workingNodes); err != nil { // Return error
		g.FailState(ctx, err.Error())
	}

	g.Base.UpdateState(ctx, pulseNumber, nodes, cloudStateHash)
}

func (g *Complete) OnPulseFromConsensus(ctx context.Context, pulse network.NetworkedPulse) {
	g.Base.OnPulseFromConsensus(ctx, pulse)

	done := make(chan struct{})
	defer close(done)
	pulseProcessingWatchdog(ctx, g.Base, pulse, done)

	logger := inslogger.FromContext(ctx)

	logger.Infof("Got new pulse number: %d", pulse.PulseNumber)
	ctx, span := instracer.StartSpan(ctx, "ServiceNetwork.Handlepulse")
	span.SetTag("pulse.Number", int64(pulse.PulseNumber))
	defer span.Finish()

	err := g.PulseManager.CommitPulseChange(pulse)
	if err != nil {
		logger.Fatalf("Failed to set new pulse: %s", err.Error())
	}
	logger.Infof("Set new current pulse number: %d", pulse.PulseNumber)
	stats.Record(ctx, statPulse.M(int64(pulse.PulseNumber)))
}
