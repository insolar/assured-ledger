// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/gateway/bootstrap"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	mock "github.com/insolar/assured-ledger/ledger-core/testutils/network"
)

type fixture struct {
	mc              *minimock.Controller
	joinerBootstrap *JoinerBootstrap
	gatewayer       *mock.GatewayerMock
	requester       *bootstrap.RequesterMock
}

func createFixture(t *testing.T) fixture {
	mc := minimock.NewController(t)
	cert := &mandates.Certificate{}
	gatewayer := mock.NewGatewayerMock(mc)
	requester := bootstrap.NewRequesterMock(mc)

	joinerBootstrap := newJoinerBootstrap(&Base{
		CertificateManager: mandates.NewCertificateManager(cert),
		BootstrapRequester: requester,
		Gatewayer:          gatewayer,
		originCandidate:    &adapters.Candidate{},
	})

	return fixture{
		mc:              mc,
		joinerBootstrap: joinerBootstrap,
		gatewayer:       gatewayer,
		requester:       requester,
	}
}

var ErrUnknown = errors.New("unknown error")

func TestJoinerBootstrap_Run_AuthorizeRequestFailed(t *testing.T) {
	instestlogger.SetTestOutput(t)

	f := createFixture(t)
	defer f.mc.Finish()
	defer f.mc.Wait(time.Minute)

	f.gatewayer.SwitchStateMock.Set(func(ctx context.Context, state nodeinfo.NetworkState, pulse network.NetworkedPulse) {
		assert.Equal(t, nodeinfo.NoNetworkState, state)
	})

	f.requester.AuthorizeMock.Set(func(ctx context.Context, c2 nodeinfo.Certificate) (pp1 *packet.Permit, err error) {
		return nil, ErrUnknown
	})

	assert.Equal(t, nodeinfo.JoinerBootstrap, f.joinerBootstrap.GetState())
	f.joinerBootstrap.Run(context.Background(), EphemeralPulse)
}

func TestJoinerBootstrap_Run_BootstrapRequestFailed(t *testing.T) {
	instestlogger.SetTestOutput(t)

	f := createFixture(t)
	defer f.mc.Finish()
	defer f.mc.Wait(time.Minute)

	f.gatewayer.SwitchStateMock.Set(func(ctx context.Context, state nodeinfo.NetworkState, pulse network.NetworkedPulse) {
		assert.Equal(t, nodeinfo.NoNetworkState, state)
	})

	f.requester.AuthorizeMock.Set(func(ctx context.Context, c2 nodeinfo.Certificate) (pp1 *packet.Permit, err error) {
		return &packet.Permit{}, nil
	})

	f.requester.BootstrapMock.Set(func(context.Context, *packet.Permit, adapters.Candidate, network.NetworkedPulse) (bp1 *packet.BootstrapResponse, err error) {
		return nil, ErrUnknown
	})

	f.joinerBootstrap.Run(context.Background(), EphemeralPulse)
}

func TestJoinerBootstrap_Run_BootstrapSucceeded(t *testing.T) {
	instestlogger.SetTestOutput(t)

	f := createFixture(t)
	defer f.mc.Finish()
	defer f.mc.Wait(time.Minute)

	f.gatewayer.SwitchStateMock.Set(func(ctx context.Context, state nodeinfo.NetworkState, puls network.NetworkedPulse) {
		assert.Equal(t, pulse.Number(123), puls.PulseNumber)
		assert.Equal(t, nodeinfo.WaitConsensus, state)
	})

	f.requester.AuthorizeMock.Set(func(ctx context.Context, c2 nodeinfo.Certificate) (pp1 *packet.Permit, err error) {
		return &packet.Permit{}, nil
	})

	f.requester.BootstrapMock.Set(func(ctx context.Context, pp1 *packet.Permit, c2 adapters.Candidate, pp2 network.NetworkedPulse) (bp1 *packet.BootstrapResponse, err error) {
		p := rms.PulseProto{PulseNumber: 123}
		return &packet.BootstrapResponse{
			ETASeconds: 90,
			Pulse:      p,
		}, nil
	})

	f.joinerBootstrap.Run(context.Background(), EphemeralPulse)

	assert.Equal(t, true, f.joinerBootstrap.bootstrapTimer.Stop())
	assert.Equal(t, time.Duration(0), f.joinerBootstrap.backoff)
	assert.Equal(t, time.Duration(time.Second*90), f.joinerBootstrap.bootstrapETA)
}
