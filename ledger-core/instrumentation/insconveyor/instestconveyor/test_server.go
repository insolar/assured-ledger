// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package instestconveyor

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
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/testpop"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewTestServerTemplate(t logcommon.TestingLogger, filterFn logcommon.ErrorFilterFunc) *ServerTemplate {
	instestlogger.SetTestOutputWithErrorFilter(t, filterFn)
	s := &ServerTemplate{t: t.(minimock.Tester)}

	pop := testpop.CreateManyNodePopulationMock(s.t, 1, member.PrimaryRoleLightMaterial)
	s.pg = testutils.NewPulseGenerator(10, pop, nil)
	s.pg.Generate()

	return s
}

type ServerTemplate struct {
	// set by constructor
	t           minimock.Tester
	appFn       AppCompartmentFunc
	appImposeFn insconveyor.ImposerFunc
	pg          *testutils.PulseGenerator

	// set by setters
	cfg configuration.Configuration
	ac  *insapp.AppComponents
	fn  insconveyor.ImposerFunc

	// set by init
	app         *insconveyor.AppCompartment
	ctxCancelFn context.CancelFunc
	state       atomickit.StartStopFlag
}

func (p *ServerTemplate) T() minimock.Tester {
	return p.t
}

type AppCompartmentFunc = func(configuration.Configuration, insapp.AppComponents) *insconveyor.AppCompartment

// InitTemplate is used to set up a default behavior for a test server. Can only be called once.
// Handler (appFn) must be provided to create an app compartment.
// Handler (appImposeFn) is optional and will be invoked before a handler set by SetImposer().
// Handler (appImposeFn) can set insconveyor.ImposerFunc.ComponentInterceptFn.
func (p *ServerTemplate) InitTemplate(appFn AppCompartmentFunc, appImposeFn insconveyor.ImposerFunc) {
	switch {
	case p.state.WasStarted():
		panic(throw.IllegalState())
	case p.appFn != nil:
		panic(throw.IllegalState())
	}
	p.appFn = appFn
	p.appImposeFn = appImposeFn
}

// SetConfig updates config. Can only be called before the server is initialized / started.
func (p *ServerTemplate) SetConfig(cfg configuration.Configuration) {
	if p.state.WasStarted() {
		panic(throw.IllegalState())
	}
	p.cfg = cfg
}

// SetImposer sets per-test override logic. Can only be called once and before the server is initialized / started.
// Handler is NOT allowed to set insconveyor.ImposerFunc.ComponentInterceptFn.
func (p *ServerTemplate) SetImposer(fn insconveyor.ImposerFunc) {
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

// SetAppComponents sets per-test overrides for app components. Nil values will be replaced by default.
// Can only be called once and before the server is initialized / started.
func (p *ServerTemplate) SetAppComponents(ac insapp.AppComponents) {
	switch {
	case p.state.WasStarted():
		panic(throw.IllegalState())
	case p.ac != nil:
		panic(throw.IllegalState())
	}
	p.ac = &ac
}

// Start initializes and starts the test server.
func (p *ServerTemplate) Start() {
	switch {
	case p.state.WasStarted():
		panic(throw.IllegalState())
	case p.appFn == nil:
		panic(throw.IllegalState())
	}

	var ac insapp.AppComponents
	if p.ac != nil {
		ac = *p.ac
	}

	if ac.MessageSender == nil {
		ac.MessageSender = messagesender.NewServiceMock(p.t)
	}
	if ac.CryptoScheme == nil {
		pcs := platformpolicy.NewPlatformCryptographyScheme()
		kp := platformpolicy.NewKeyProcessor()
		sk, err := secrets.GenerateKeyPair()
		require.NoError(p.t, err)

		ac.CryptoScheme = legacyadapter.New(pcs, kp, keystore.NewInplaceKeyStore(sk.Private))
	}
	if reference.IsEmpty(ac.LocalNodeRef) {
		ac.LocalNodeRef = gen.UniqueGlobalRefWithPulse(pulse.MinTimePulse)
	}

	ctx := context.Background()
	ctx, p.ctxCancelFn = context.WithCancel(ctx)

	p.app = p.appFn(p.cfg, ac)
	require.NotNil(p.t, p.app)

	p.app.SetImposer(func(params *insconveyor.ImposedParams) {
		params.CompartmentSetup.ConveyorConfig.ConveyorMachineConfig.SlotMachineLogger = nil
		sml := &params.CompartmentSetup.ConveyorConfig.SlotMachineConfig.SlotMachineLogger
		*sml = nil

		if p.appImposeFn != nil {
			// set params.ComponentInterceptFn here to do post-checks on components
			p.appImposeFn(params)
			p.appImposeFn = nil
		}

		if p.fn != nil {
			cif := params.ComponentInterceptFn
			params.ComponentInterceptFn = nil

			p.fn(params)
			p.fn = nil

			if params.ComponentInterceptFn != nil {
				// Individual tests should not use params.ComponentInterceptFn
				panic(throw.IllegalState())
			}
			params.ComponentInterceptFn = cif
		}

		switch {
		case *sml != nil:
		case convlog.UseTextConvLog():
			*sml = convlog.MachineLogger{}
		default:
			*sml = insconveyor.ConveyorLoggerFactory{}
		}

		if params.EventJournal != nil {
			*sml = params.EventJournal.InterceptSlotMachineLog(*sml, params.InitContext.Done())
		}
	})

	err := p.app.Init(ctx)
	require.NoError(p.t, err)

	if !p.state.Start() {
		panic(throw.IllegalState())
	}

	err = p.app.Start(ctx)
	require.NoError(p.t, err)
}

// IncrementPulse generates and applies next pulse.
func (p *ServerTemplate) IncrementPulse() {
	p.pg.Generate()

	if !p.state.IsActive() {
		return
	}

	bd := p.app.GetBeatDispatcher()
	ack, _ := beat.NewAck(func(beat.AckData) {})
	bd.PrepareBeat(ack)
	bd.CommitBeat(p.pg.GetLastBeat())
}

// Stop cancels context and initiates stop of the app compartment.
func (p *ServerTemplate) Stop() {
	p.state.DoDiscard(func() {
		// not started
	}, func() {
		ctx := context.Background()

		p.ctxCancelFn()
		err := p.app.Stop(ctx)
		require.NoError(p.t, err)
	})
}

// App returns app compartment. Panics when wasn't started.
func (p *ServerTemplate) App() *insconveyor.AppCompartment {
	if p.app == nil {
		panic(throw.IllegalState())
	}
	return p.app
}

// Injector returns a dependency injector with dependencies available for the app compartment.
// There is NO access to dependencies added by pulse slots. Panics when wasn't started.
func (p *ServerTemplate) Injector() injector.DependencyInjector {
	return injector.NewDependencyInjector(struct{}{}, p.App().Conveyor(), nil)
}

// Pulsar returns a pulse generator. Panics when zero.
func (p *ServerTemplate) Pulsar() *testutils.PulseGenerator {
	if p.pg == nil {
		panic(throw.IllegalState())
	}
	return p.pg
}

func (p *ServerTemplate) LastPulseNumber() pulse.Number {
	return p.Pulsar().GetLastPulseData().PulseNumber
}
