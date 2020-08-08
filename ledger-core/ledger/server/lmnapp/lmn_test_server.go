// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmnapp

import (
	"context"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/secrets"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/testpop"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewTestServer(t logcommon.TestingLogger) *TestServer {
	instestlogger.SetTestOutput(t)
	s := &TestServer{ t: t.(minimock.Tester) }

	pop := testpop.CreateManyNodePopulationMock(s.t, 1, member.PrimaryRoleLightMaterial)
	s.pg = testutils.NewPulseGenerator(10, pop)
	s.pg.Generate()

	return s
}

type TestServer struct {
	t     minimock.Tester
	cfg   configuration.Ledger
	app   *insconveyor.AppCompartment
	pg    *testutils.PulseGenerator
	fn    insconveyor.ImposerFunc
	state atomickit.StartStopFlag
}

func (p *TestServer) T() minimock.Tester {
	return p.t
}

func (p *TestServer) SetConfig(cfg configuration.Ledger) {
	if p.state.WasStarted() {
		panic(throw.IllegalState())
	}
	p.cfg = cfg
}

func (p *TestServer) SetImposer(fn insconveyor.ImposerFunc) {
	switch {
	case p.state.WasStarted():
		panic(throw.IllegalState())
	case fn == nil:
		panic(throw.IllegalValue())
	case p.fn != nil:
		panic(throw.IllegalState())
	}
	p.fn = fn
}

func (p *TestServer) Start() {
	if p.state.WasStarted() {
		panic(throw.IllegalState())
	}

	pcs := platformpolicy.NewPlatformCryptographyScheme()
	kp := platformpolicy.NewKeyProcessor()
	sk, err := secrets.GenerateKeyPair()
	require.NoError(p.t, err)

	// Reuse default compartment construction logic
	p.app = NewAppCompartment(p.cfg, insapp.AppComponents{
		// Provide mocks for application-level / cross-compartment dependencies here
		MessageSender: messagesender.NewServiceMock(p.t),
		CryptoScheme: legacyadapter.New(pcs, kp, keystore.NewInplaceKeyStore(sk.Private)),
	})
	require.NotNil(p.t, p.app)

	p.app.SetImposer(func(params *insconveyor.ImposedParams) {
		// put here default behavior overrides for tests

		// ......

		// and of default behavior overrides for tests

		if p.fn != nil {
			p.fn(params)
		}

		if params.ComponentInterceptFn != nil {
			// Individual tests should not use params.ComponentInterceptFn
			panic(throw.IllegalState())
		}

		// set params.ComponentInterceptFn here to do post-checks on components
	})

	ctx := context.Background()

	err = p.app.Init(ctx)
	require.NoError(p.t, err)

	if !p.state.Start() {
		panic(throw.IllegalState())
	}

	err = p.app.Start(ctx)
	require.NoError(p.t, err)
}

func (p *TestServer) NextPulse() {
	p.pg.Generate()

	if !p.state.IsActive() {
		return
	}

	bd := p.app.GetBeatDispatcher()
	ack, _ := beat.NewAck(func(beat.AckData) {})
	bd.PrepareBeat(ack)
	bd.CommitBeat(p.pg.GetLastBeat())
}

func (p *TestServer) Stop() {
	p.state.DoDiscard(func() {
		// not started
	}, func() {
		err := p.app.Stop(context.Background())
		require.NoError(p.t, err)
	})
}
