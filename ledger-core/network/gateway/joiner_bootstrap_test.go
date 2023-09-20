package gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/gateway/bootstrap"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
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
		localCandidate:     &adapters.Candidate{},
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
	defer f.mc.Wait(time.Second*10)

	f.gatewayer.SwitchStateMock.Set(func(ctx context.Context, state network.State, pulse pulse.Data) {
		assert.Equal(t, network.NoNetworkState, state)
	})

	f.requester.AuthorizeMock.Set(func(ctx context.Context, c2 nodeinfo.Certificate) (pp1 *rms.Permit, err error) {
		return nil, ErrUnknown
	})

	assert.Equal(t, network.JoinerBootstrap, f.joinerBootstrap.GetState())
	f.joinerBootstrap.Run(context.Background(), EphemeralPulse.Data)
}

func TestJoinerBootstrap_Run_BootstrapRequestFailed(t *testing.T) {
	instestlogger.SetTestOutput(t)

	f := createFixture(t)
	defer f.mc.Finish()
	defer f.mc.Wait(time.Second*10)

	f.gatewayer.SwitchStateMock.Set(func(ctx context.Context, state network.State, pulse pulse.Data) {
		assert.Equal(t, network.NoNetworkState, state)
	})

	f.requester.AuthorizeMock.Set(func(ctx context.Context, c2 nodeinfo.Certificate) (pp1 *rms.Permit, err error) {
		return &rms.Permit{}, nil
	})

	f.requester.BootstrapMock.Set(func(context.Context, *rms.Permit, adapters.Candidate) (bp1 *rms.BootstrapResponse, err error) {
		return nil, ErrUnknown
	})

	f.joinerBootstrap.Run(context.Background(), EphemeralPulse.Data)
}

func TestJoinerBootstrap_Run_BootstrapSucceeded(t *testing.T) {
	instestlogger.SetTestOutput(t)

	f := createFixture(t)
	defer f.mc.Finish()
	defer f.mc.Wait(time.Second*10)

	f.gatewayer.SwitchStateMock.Set(func(ctx context.Context, state network.State, puls pulse.Data) {
		assert.Equal(t, pulse.Unknown, puls.PulseNumber)
		assert.Equal(t, network.WaitConsensus, state)
	})

	f.requester.AuthorizeMock.Set(func(ctx context.Context, c2 nodeinfo.Certificate) (pp1 *rms.Permit, err error) {
		return &rms.Permit{}, nil
	})

	f.requester.BootstrapMock.Set(func(ctx context.Context, pp1 *rms.Permit, c2 adapters.Candidate) (bp1 *rms.BootstrapResponse, err error) {
		return &rms.BootstrapResponse{
			ETASeconds: 90,
		}, nil
	})

	f.joinerBootstrap.Run(context.Background(), EphemeralPulse.Data)

	assert.Equal(t, true, f.joinerBootstrap.bootstrapTimer.Stop())
	assert.Equal(t, time.Duration(0), f.joinerBootstrap.backoff)
	assert.Equal(t, time.Duration(time.Second*90), f.joinerBootstrap.bootstrapETA)
}
