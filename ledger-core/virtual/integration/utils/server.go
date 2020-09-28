// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"context"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat/memstor"
	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/journal"
	"github.com/insolar/assured-ledger/ledger-core/testutils/testpop"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/publisher"
	"github.com/insolar/assured-ledger/ledger-core/virtual/lmn"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache"
)

const testCryptoKey = "../../cryptography/keystore/testdata/keys.json"

type Server struct {
	pulseLock sync.Mutex

	debugLock  sync.Mutex
	debugFlags atomickit.Uint32
	suspend    synckit.ClosableSignalChannel
	cycleFn    ConveyorCycleFunc

	// real components
	virtual       *virtual.Dispatcher
	Runner        *runner.DefaultService
	messageSender *messagesender.DefaultService
	memoryCache   *memorycache.DefaultService

	// testing components and Mocks
	PublisherMock      *publisher.Mock
	JetCoordinatorMock *affinity.HelperMock
	pulseGenerator     *testutils.PulseGenerator
	pulseStorage       *memstor.StorageMem
	pulseManager       *insapp.PulseManager
	platformScheme     crypto.PlatformScheme
	Journal            *journal.Journal

	// wait and suspend operations

	// finalization
	fullStop    synckit.ClosableSignalChannel
	ctxCancelFn context.CancelFunc

	// components for testing http api
	testWalletServer *testwalletapi.TestWalletServer

	// top-level caller ID
	caller reference.Global
}

type ConveyorCycleFunc func(c *conveyor.PulseConveyor, hasActive, isIdle bool)

type Tester interface {

	// logcommon.Logger+minimock.Tester

	Helper()
	Log(...interface{})

	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Error(...interface{})
	Errorf(format string, args ...interface{})
	FailNow()
}

func NewServer(ctx context.Context, t Tester) (*Server, context.Context) {
	return newServerExt(ctx, t, NewDefaultServerOpts())
}

func NewServerWithErrorFilter(ctx context.Context, t Tester, errorFilterFn logcommon.ErrorFilterFunc) (*Server, context.Context) {
	def := NewDefaultServerOpts()
	def.ErrorFilter = errorFilterFn
	return newServerExt(ctx, t, def)
}

func NewUninitializedServer(ctx context.Context, t Tester) (*Server, context.Context) {
	def := NewDefaultServerOpts()
	def.Initialization = false
	return newServerExt(ctx, t, def)
}

func NewUninitializedServerWithErrorFilter(ctx context.Context, t Tester, errorFilterFn logcommon.ErrorFilterFunc) (*Server, context.Context) {
	def := NewDefaultServerOpts()
	def.ErrorFilter = errorFilterFn
	def.Initialization = false
	return newServerExt(ctx, t, def)
}

func generateGlobalCaller() reference.Global {
	return reference.NewSelf(reference.NewLocal(pulse.MinTimePulse, 0, gen.UniqueLocalRef().GetHash()))
}

type ServerOpts struct {
	ErrorFilter    logcommon.ErrorFilterFunc
	Initialization bool
	Delta          uint16
}

func NewDefaultServerOpts() ServerOpts {
	return ServerOpts{
		ErrorFilter:    nil,
		Initialization: true,
		Delta:          1,
	}
}

func newServerExt(ctx context.Context, t Tester, opts ServerOpts) (*Server, context.Context) {
	instestlogger.SetTestOutputWithErrorFilter(t, opts.ErrorFilter)

	if ctx == nil {
		ctx = instestlogger.TestContext(t)
	}
	ctx, cancelFn := context.WithCancel(ctx)

	s := Server{
		caller:      generateGlobalCaller(),
		fullStop:    make(synckit.ClosableSignalChannel),
		ctxCancelFn: cancelFn,
	}

	// Pulse-related components
	var (
		PulseManager *insapp.PulseManager
		Pulses       *memstor.StorageMem
	)
	{
		Pulses = memstor.NewStorageMem()
		PulseManager = insapp.NewPulseManager()
		PulseManager.PulseAppender = Pulses
	}

	s.pulseManager = PulseManager
	s.pulseStorage = Pulses
	censusMock := testpop.CreateOneNodePopulationMock(t, s.caller, member.PrimaryRoleVirtual)
	s.pulseGenerator = testutils.NewPulseGenerator(opts.Delta, censusMock, nil)
	s.incrementPulse()

	{
		keyProcessor := platformpolicy.NewKeyProcessor()
		pk, err := keyProcessor.GeneratePrivateKey()
		if err != nil {
			panic(throw.W(err, "failed to generate node PK"))
		}
		keyStore := keystore.NewInplaceKeyStore(pk)

		platformCryptographyScheme := platformpolicy.NewPlatformCryptographyScheme()
		s.platformScheme = legacyadapter.New(platformCryptographyScheme, keyProcessor, keyStore)
	}

	s.JetCoordinatorMock = affinity.NewHelperMock(t).
		MeMock.Return(s.caller).
		QueryRoleMock.Return([]reference.Global{s.caller}, nil)

	s.PublisherMock = publisher.NewMock()
	s.PublisherMock.SetResendMode(ctx, &s)

	runnerService := runner.NewService()
	if err := runnerService.Init(); err != nil {
		panic(err)
	}
	s.Runner = runnerService

	messageSender := messagesender.NewDefaultService(s.PublisherMock, s.JetCoordinatorMock, s.pulseStorage)
	s.messageSender = messageSender

	s.memoryCache = memorycache.NewDefaultService()

	var machineLogger smachine.SlotMachineLogger

	if convlog.UseTextConvLog() {
		machineLogger = convlog.MachineLogger{}
	} else {
		machineLogger = insconveyor.ConveyorLoggerFactory{}
	}
	s.Journal = journal.New()
	machineLogger = s.Journal.InterceptSlotMachineLog(machineLogger, s.fullStop)

	virtualDispatcher := virtual.NewDispatcher()
	virtualDispatcher.Runner = runnerService
	virtualDispatcher.MessageSender = messageSender
	virtualDispatcher.Affinity = s.JetCoordinatorMock
	virtualDispatcher.AuthenticationService = authentication.NewService(ctx, virtualDispatcher.Affinity)
	virtualDispatcher.MemoryCache = s.memoryCache

	virtualDispatcher.CycleFn = s.onConveyorCycle
	virtualDispatcher.EventlessSleep = -1 // disable EventlessSleep for proper WaitActiveThenIdleConveyor behavior
	virtualDispatcher.MachineLogger = machineLogger
	virtualDispatcher.MaxRunners = 4
	virtualDispatcher.ReferenceBuilder = lmn.NewRecordReferenceBuilder(s.platformScheme.RecordScheme(), s.caller)
	s.virtual = virtualDispatcher

	// re HTTP testing
	testWalletAPIConfig := configuration.TestWalletAPI{Address: "very naughty address"}
	s.testWalletServer = testwalletapi.NewTestWalletServer(inslogger.FromContext(ctx), testWalletAPIConfig, virtualDispatcher, Pulses)

	if opts.Initialization {
		s.Init(ctx)
	}

	return &s, ctx
}

func (s *Server) Init(ctx context.Context) {
	if err := s.virtual.Init(ctx); err != nil {
		panic(err)
	}
	s.virtual.MessageSender.InterceptorClear()

	s.pulseManager.AddDispatcher(s.virtual.FlowDispatcher)
	s.incrementPulse() // for sake of simplicity make sure that there is no "hanging" first pulse
	s.IncrementPulseAndWaitIdle(ctx)
}

func (s *Server) StartRecording() {
	s.StartRecordingExt(10_000, true)
}

func (s *Server) StartRecordingExt(limit int, discardOnOverflow bool) {
	s.Journal.StartRecording(limit, discardOnOverflow)
}

func (s *Server) GetPulse() beat.Beat {
	return s.pulseGenerator.GetLastBeat()
}

func (s *Server) GetPulseNumber() pulse.Number {
	return s.pulseGenerator.GetLastBeat().PulseNumber
}

func (s *Server) GetPrevPulse() beat.Beat {
	return s.pulseGenerator.GetPrevBeat()
}

func (s *Server) GetPrevPulseNumber() pulse.Number {
	return s.pulseGenerator.GetPrevBeat().PulseNumber
}

func (s *Server) incrementPulse() {
	s.pulseGenerator.Generate()

	s.pulseManager.RequestNodeState(func(api.UpstreamState) {})

	pc := s.GetPulse()
	if err := s.pulseStorage.AddCommittedBeat(pc); err != nil {
		panic(err)
	}

	if err := s.pulseManager.CommitPulseChange(pc); err != nil {
		panic(err)
	}
}

func (s *Server) IncrementPulse(context.Context) {
	s.pulseLock.Lock()
	defer s.pulseLock.Unlock()

	s.incrementPulse()
}

func (s *Server) IncrementPulseAndWaitIdle(ctx context.Context) {
	s.IncrementPulse(ctx)

	s.WaitActiveThenIdleConveyor()
}

// deprecated // use SendMsg
func (s *Server) SendMessage(_ context.Context, msg *message.Message) {
	bm := beat.NewMessageExt(msg.UUID, msg.Payload, msg)
	bm.Metadata = msg.Metadata
	s.SendMsg(bm)
}

func (s *Server) SendMsg(msg beat.Message) {
	if err := s.virtual.FlowDispatcher.Process(msg); err != nil {
		panic(err)
	}
}

func (s *Server) ReplaceRunner(svc runner.Service) {
	s.virtual.Runner = svc
}

func (s *Server) OverrideConveyorFactoryLogContext(ctx context.Context) {
	s.virtual.FactoryLogContextOverride = ctx
}

// Set limit for parallel runners. Function must be called before server.Init
// If this limit does not set it will be set by default (NumCPU() - 2)
func (s *Server) SetMaxParallelism(count int) {
	s.virtual.MaxRunners = count
}

func (s *Server) ReplaceMachinesManager(manager machine.Manager) {
	s.Runner.Manager = manager
}

func (s *Server) ReplaceCache(cache descriptor.Cache) {
	s.Runner.Cache = cache
}

func (s *Server) ReplaceAuthenticationService(svc authentication.Service) {
	s.virtual.AuthenticationService = svc
}

func (s *Server) AddInput(ctx context.Context, msg interface{}) error {
	return s.virtual.Conveyor.AddInput(ctx, s.GetPulseNumber(), msg)
}

func (s *Server) GlobalCaller() reference.Global {
	return s.caller
}

func (s *Server) RandomLocalWithPulse() reference.Local {
	return gen.UniqueLocalRefWithPulse(s.GetPulseNumber())
}

func (s *Server) BuildRandomOutgoingWithGivenPulse(pn pulse.Number) reference.Global {
	return reference.NewRecordOf(s.GlobalCaller(), gen.UniqueLocalRefWithPulse(pn))
}

func (s *Server) BuildRandomOutgoingWithPulse() reference.Global {
	return reference.NewRecordOf(s.GlobalCaller(), s.RandomLocalWithPulse())
}

func (s *Server) RandomGlobalWithPulse() reference.Global {
	return gen.UniqueGlobalRefWithPulse(s.GetPulseNumber())
}

func (s *Server) RandomLocalWithPrevPulse() reference.Local {
	return gen.UniqueLocalRefWithPulse(s.GetPrevPulseNumber())
}

func (s *Server) BuildRandomOutgoingWithPrevPulse() reference.Global {
	return reference.NewRecordOf(s.GlobalCaller(), s.RandomLocalWithPrevPulse())
}

func (s *Server) RandomGlobalWithPrevPulse() reference.Global {
	return gen.UniqueGlobalRefWithPulse(s.GetPrevPulseNumber())
}

func (s *Server) DelegationToken(outgoing reference.Global, to reference.Global, object reference.Global) rms.CallDelegationToken {
	return s.virtual.AuthenticationService.GetCallDelegationToken(outgoing, to, s.GetPulseNumber(), object)
}

func (s *Server) Stop() {
	defer close(s.fullStop)

	s.ctxCancelFn()
	s.virtual.Conveyor.Stop()
	_ = s.testWalletServer.Stop(context.Background())
	_ = s.messageSender.Close()
}

func (s *Server) WaitIdleConveyor() {
	s.waitIdleConveyor(false)
}

func (s *Server) WaitActiveThenIdleConveyor() {
	s.waitIdleConveyor(true)
}

func (s *Server) waitIdleConveyor(checkActive bool) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	s.setWaitCallback(func(c *conveyor.PulseConveyor, hadActive, isIdle bool) {
		if checkActive && !hadActive {
			return
		}
		if isIdle {
			s.setWaitCallback(nil)
			wg.Done()
		}
	})
	wg.Wait()
}

func (s *Server) SuspendConveyorNoWait() {
	s.debugLock.Lock()
	defer s.debugLock.Unlock()

	if s.suspend == nil {
		s.suspend = make(synckit.ClosableSignalChannel)
	}
}

func (s *Server) _suspendConveyorAndWait(wg *sync.WaitGroup) {
	s.debugLock.Lock()
	defer s.debugLock.Unlock()

	if s.cycleFn != nil {
		panic(throw.IllegalState())
	}

	if s.suspend == nil {
		s.suspend = make(synckit.ClosableSignalChannel)
	}

	flags := s.debugFlags.Load()
	if flags&isNotScanning != 0 {
		wg.Done()
		return
	}

	s.debugFlags.SetBits(callOnce)
	s.cycleFn = func(*conveyor.PulseConveyor, bool, bool) {
		wg.Done()
	}
}

func (s *Server) SuspendConveyorAndWait() {
	wg := sync.WaitGroup{}
	wg.Add(1)

	s._suspendConveyorAndWait(&wg)

	wg.Wait()
}

func (s *Server) SuspendConveyorAndWaitThenResetActive() {
	s.SuspendConveyorAndWait()
	s.ResetActiveConveyorFlag()
}

func (s *Server) ResumeConveyor() {
	s.debugLock.Lock()
	defer s.debugLock.Unlock()
	s._resumeConveyor()
}

func (s *Server) _resumeConveyor() {
	if ch := s.suspend; ch != nil {
		s.suspend = nil
		close(ch)
	}
}

func (s *Server) ResetActiveConveyorFlag() {
	s.debugLock.Lock()
	defer s.debugLock.Unlock()
	s.debugFlags.UnsetBits(hasActive)
}

const (
	hasActive     = 1
	isIdle        = 2
	isNotScanning = 4
	callOnce      = 8
)

func (s *Server) onCycleUpdate(fn func() uint32) synckit.SignalChannel {
	updateFn, ch := s._onCycleUpdate(fn)
	if updateFn != nil {
		updateFn()
	}
	return ch
}

func (s *Server) _onCycleUpdate(fn func() uint32) (func(), synckit.SignalChannel) {
	s.debugLock.Lock()
	defer s.debugLock.Unlock()

	// makes sure that calling Wait in a parallel thread will not lock up caller of Suspend
	if cs := s.debugFlags.Load(); cs&callOnce != 0 {
		s.debugFlags.UnsetBits(callOnce)
		if cycleFn := s.cycleFn; cycleFn != nil {
			s.cycleFn = nil
			go cycleFn(s.virtual.Conveyor, cs&hasActive != 0, cs&isIdle != 0)
		}
	}

	cs := fn()
	if cycleFn := s.cycleFn; cycleFn != nil {
		return func() {
			cycleFn(s.virtual.Conveyor, cs&hasActive != 0, cs&isIdle != 0)
		}, s.suspend
	}

	return nil, s.suspend
}

func (s *Server) onConveyorCycle(state conveyor.CycleState) {
	ch := s.onCycleUpdate(func() uint32 {
		switch state {
		case conveyor.ScanActive:
			return s.debugFlags.SetBits(hasActive | isNotScanning)
		case conveyor.ScanIdle:
			return s.debugFlags.SetBits(isIdle | isNotScanning)
		case conveyor.Scanning:
			return s.debugFlags.UnsetBits(isIdle | isNotScanning)
		default:
			panic(throw.Impossible())
		}
	})
	if ch != nil && state == conveyor.Scanning {
		<-ch
	}
}

func (s *Server) setWaitCallback(cycleFn ConveyorCycleFunc) {
	s.onCycleUpdate(func() uint32 {
		if cycleFn != nil {
			s._resumeConveyor()
		} else {
			s.debugFlags.UnsetBits(hasActive)
		}
		s.cycleFn = cycleFn
		return s.debugFlags.Load()
	})
}

func (s *Server) WrapPayload(pl rmsreg.GoGoSerializable) *RequestWrapper {
	return NewRequestWrapper(s.GetPulseNumber(), pl).SetSender(s.caller)
}

func (s *Server) SendPayload(ctx context.Context, pl rmsreg.GoGoSerializable) {
	msg := s.WrapPayload(pl).Finalize()
	s.SendMessage(ctx, msg)
}

func (s *Server) WrapPayloadAsFuture(pl rmsreg.GoGoSerializable) *RequestWrapper {
	return NewRequestWrapper(s.GetPulse().NextPulseNumber(), pl).SetSender(s.caller)
}

func (s *Server) SendPayloadAsFuture(ctx context.Context, pl rmsreg.GoGoSerializable) {
	msg := s.WrapPayloadAsFuture(pl).Finalize()
	s.SendMessage(ctx, msg)
}

func (s *Server) GetPulseTime() time.Duration {
	return time.Duration(s.pulseGenerator.GetDelta()) * time.Second
}
