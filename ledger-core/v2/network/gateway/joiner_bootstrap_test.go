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

	"github.com/insolar/assured-ledger/ledger-core/v2/certificate"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/gateway/bootstrap"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	mock "github.com/insolar/assured-ledger/ledger-core/v2/testutils/network"
)

type fixture struct {
	mc              *minimock.Controller
	joinerBootstrap *JoinerBootstrap
	gatewayer       *mock.GatewayerMock
	requester       *bootstrap.RequesterMock
}

func createFixture(t *testing.T) fixture {
	mc := minimock.NewController(t)
	cert := &certificate.Certificate{}
	gatewayer := mock.NewGatewayerMock(mc)
	requester := bootstrap.NewRequesterMock(mc)

	joinerBootstrap := newJoinerBootstrap(&Base{
		CertificateManager: certificate.NewCertificateManager(cert),
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
	f := createFixture(t)
	defer f.mc.Finish()
	defer f.mc.Wait(time.Minute)

	f.gatewayer.SwitchStateMock.Set(func(ctx context.Context, state node.NetworkState, pulse pulsestor.Pulse) {
		assert.Equal(t, node.NoNetworkState, state)
	})

	f.requester.AuthorizeMock.Set(func(ctx context.Context, c2 node.Certificate) (pp1 *packet.Permit, err error) {
		return nil, ErrUnknown
	})

	assert.Equal(t, node.JoinerBootstrap, f.joinerBootstrap.GetState())
	f.joinerBootstrap.Run(context.Background(), *pulsestor.EphemeralPulse)
}

func TestJoinerBootstrap_Run_BootstrapRequestFailed(t *testing.T) {
	f := createFixture(t)
	defer f.mc.Finish()
	defer f.mc.Wait(time.Minute)

	f.gatewayer.SwitchStateMock.Set(func(ctx context.Context, state node.NetworkState, pulse pulsestor.Pulse) {
		assert.Equal(t, node.NoNetworkState, state)
	})

	f.requester.AuthorizeMock.Set(func(ctx context.Context, c2 node.Certificate) (pp1 *packet.Permit, err error) {
		return &packet.Permit{}, nil
	})

	f.requester.BootstrapMock.Set(func(ctx context.Context, pp1 *packet.Permit, c2 adapters.Candidate, pp2 *pulsestor.Pulse) (bp1 *packet.BootstrapResponse, err error) {
		return nil, ErrUnknown
	})

	f.joinerBootstrap.Run(context.Background(), *pulsestor.EphemeralPulse)
}

func TestJoinerBootstrap_Run_BootstrapSucceeded(t *testing.T) {
	f := createFixture(t)
	defer f.mc.Finish()
	defer f.mc.Wait(time.Minute)

	f.gatewayer.SwitchStateMock.Set(func(ctx context.Context, state node.NetworkState, puls pulsestor.Pulse) {
		assert.Equal(t, pulse.Number(123), puls.PulseNumber)
		assert.Equal(t, node.WaitConsensus, state)
	})

	f.requester.AuthorizeMock.Set(func(ctx context.Context, c2 node.Certificate) (pp1 *packet.Permit, err error) {
		return &packet.Permit{}, nil
	})

	f.requester.BootstrapMock.Set(func(ctx context.Context, pp1 *packet.Permit, c2 adapters.Candidate, pp2 *pulsestor.Pulse) (bp1 *packet.BootstrapResponse, err error) {
		p := pulsestor.PulseProto{PulseNumber: 123}
		return &packet.BootstrapResponse{
			ETASeconds: 90,
			Pulse:      p,
		}, nil
	})

	f.joinerBootstrap.Run(context.Background(), *pulsestor.EphemeralPulse)

	assert.Equal(t, true, f.joinerBootstrap.bootstrapTimer.Stop())
	assert.Equal(t, time.Duration(0), f.joinerBootstrap.backoff)
	assert.Equal(t, time.Duration(time.Second*90), f.joinerBootstrap.bootstrapETA)
}
