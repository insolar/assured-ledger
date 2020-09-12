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

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/rules"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

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

func (g *Complete) Run(context.Context, pulse.Data) {
	if g.bootstrapTimer != nil {
		g.bootstrapTimer.Stop()
	}

	g.HostNetwork.RegisterRequestHandler(types.SignCert, g.signCertHandler)
}

func (g *Complete) GetState() network.State {
	return network.CompleteNetworkState
}

func (g *Complete) BeforeRun(ctx context.Context, pulse pulse.Data) {
	// if !pulse.PulseEpoch.IsTimeEpoch() {
	// 	panic(throw.IllegalState())
	// }
	//
	// err := g.PulseManager.CommitFirstPulseChange(pulse)
	// if err != nil {
	// 	inslogger.FromContext(ctx).Panicf("failed to set start pulse: %d, %s", pulse.PulseNumber, err.Error())
	// }
}

// GetCert method generates cert by requesting signs from discovery nodes
func (g *Complete) GetCert(ctx context.Context, registeredNodeRef reference.Global) (nodeinfo.Certificate, error) {
	pKey, role, err := g.getNodeInfo(ctx, registeredNodeRef)
	if err != nil {
		return nil, throw.W(err, "[ GetCert ] Couldn't get node info")
	}

	currentNodeCert := g.CertificateManager.GetCertificate()
	registeredNodeCert, err := mandates.NewUnsignedCertificate(currentNodeCert, pKey, role, registeredNodeRef.String())
	if err != nil {
		return nil, throw.W(err, "[ GetCert ] Couldn't create certificate")
	}

	for i, discoveryNode := range currentNodeCert.GetDiscoveryNodes() {
		sign, err := g.requestCertSign(ctx, discoveryNode, registeredNodeRef)
		if err != nil {
			return nil, throw.W(err, "[ GetCert ] Couldn't request cert sign")
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

	request := &rms.SignCertRequest{
		NodeRef: rms.NewReference(registeredNodeRef),
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
		return nil, throw.W(err, "[ SignCert ] Couldn't extract response")
	}
	return mandates.SignCert(g.CryptographyService, pKey, role, registeredNodeRef.String())
}

// signCertHandler is handler that signs certificate for some node with node own key
func (g *Complete) signCertHandler(ctx context.Context, request network.ReceivedPacket) (network.Packet, error) {
	if request.GetRequest() == nil || request.GetRequest().GetSignCert() == nil {
		inslogger.FromContext(ctx).Warnf("process SignCert: got invalid request protobuf message: %s", request)
	}
	sign, err := g.signCert(ctx, request.GetRequest().GetSignCert().NodeRef.GetValue())
	if err != nil {
		return g.HostNetwork.BuildResponse(ctx, request, &rms.ErrorResponse{Error: err.Error()}), nil
	}

	return g.HostNetwork.BuildResponse(ctx, request, &rms.SignCertResponse{Sign: sign.Bytes()}), nil
}

func (g *Complete) EphemeralMode(census.OnlinePopulation) bool {
	return false
}

func (g *Complete) UpdateState(ctx context.Context, beat beat.Beat) {

	if _, err := rules.CheckMajorityRule(g.CertificateManager.GetCertificate(), beat.Online); err != nil {
		g.FailState(ctx, err.Error())
	}

	if err := rules.CheckMinRole(g.CertificateManager.GetCertificate(), beat.Online); err != nil { // Return error
		g.FailState(ctx, err.Error())
	}

	g.Base.UpdateState(ctx, beat)
}

func (g *Complete) RequestNodeState(fn chorus.NodeStateFunc) {
	g.PulseManager.RequestNodeState(fn)
}

func (g *Complete) CancelNodeState() {
	g.PulseManager.CancelNodeState()
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

	if err := g.PulseAppender.AddCommittedBeat(pulse); err != nil {
		inslogger.FromContext(ctx).Panic("failed to append pulse: ", err.Error())
	}

	if err := g.PulseManager.CommitPulseChange(pulse); err != nil {
		logger.Fatalf("Failed to set new pulse: %s", err.Error())
	}

	logger.Infof("Set new current pulse number: %d", pulse.PulseNumber)
	stats.Record(ctx, statPulse.M(int64(pulse.PulseNumber)))
}
