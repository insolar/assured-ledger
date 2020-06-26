// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"context"
	"sync"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodestorage"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/journal"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/convlog"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock"
	"github.com/insolar/assured-ledger/ledger-core/virtual/pulsemanager"
	"github.com/insolar/assured-ledger/ledger-core/virtual/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

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

	// testing components and Mocks
	PublisherMock      *mock.PublisherMock
	JetCoordinatorMock *jet.AffinityHelperMock
	pulseGenerator     *testutils.PulseGenerator
	pulseStorage       *pulsestor.StorageMem
	pulseManager       *pulsemanager.PulseManager
	Journal            *journal.Journal

	// wait and suspend operations

	// finalization
	fullStop synckit.ClosableSignalChannel

	// components for testing http api
	testWalletServer *testwalletapi.TestWalletServer

	// top-level caller ID
	caller reference.Global
}

type ConveyorCycleFunc func(c *conveyor.PulseConveyor, hasActive, isIdle bool)

func NewServer(ctx context.Context, t *testing.T) (*Server, context.Context) {
	return newServerExt(ctx, t, nil, true)
}

func NewServerWithErrorFilter(ctx context.Context, t *testing.T, errorFilterFn logcommon.ErrorFilterFunc) (*Server, context.Context) {
	return newServerExt(ctx, t, errorFilterFn, true)
}

func NewUninitializedServer(ctx context.Context, t *testing.T) (*Server, context.Context) {
	return newServerExt(ctx, t, nil, false)
}

func NewUninitializedServerWithErrorFilter(ctx context.Context, t *testing.T, errorFilterFn logcommon.ErrorFilterFunc) (*Server, context.Context) {
	return newServerExt(ctx, t, errorFilterFn, false)
}

func newServerExt(ctx context.Context, t *testing.T, errorFilterFn logcommon.ErrorFilterFunc, init bool) (*Server, context.Context) {
	instestlogger.SetTestOutputWithErrorFilter(t, errorFilterFn)

	if ctx == nil {
		ctx = instestlogger.TestContext(t)
	}

	s := Server{
		caller:   gen.UniqueGlobalRef(),
		fullStop: make(synckit.ClosableSignalChannel),
	}

	// Pulse-related components
	var (
		PulseManager *pulsemanager.PulseManager
		Pulses       *pulsestor.StorageMem
	)
	{
		networkNodeMock := network.NewNetworkNodeMock(t).
			IDMock.Return(gen.UniqueGlobalRef()).
			ShortIDMock.Return(node.ShortNodeID(0)).
			RoleMock.Return(node.StaticRoleVirtual).
			AddressMock.Return("").
			GetStateMock.Return(node.Ready).
			GetPowerMock.Return(1)
		networkNodeList := []node.NetworkNode{networkNodeMock}

		nodeNetworkAccessorMock := network.NewAccessorMock(t).GetWorkingNodesMock.Return(networkNodeList)
		nodeNetworkMock := network.NewNodeNetworkMock(t).GetAccessorMock.Return(nodeNetworkAccessorMock)
		nodeSetter := nodestorage.NewModifierMock(t).SetMock.Return(nil)

		Pulses = pulsestor.NewStorageMem()
		PulseManager = pulsemanager.NewPulseManager()
		PulseManager.NodeNet = nodeNetworkMock
		PulseManager.NodeSetter = nodeSetter
		PulseManager.PulseAccessor = Pulses
		PulseManager.PulseAppender = Pulses
	}

	s.pulseManager = PulseManager
	s.pulseStorage = Pulses
	s.pulseGenerator = testutils.NewPulseGenerator(10)

	s.JetCoordinatorMock = jet.NewAffinityHelperMock(t).
		MeMock.Return(s.caller).
		QueryRoleMock.Return([]reference.Global{s.caller}, nil)

	s.PublisherMock = mock.NewPublisherMock()
	s.PublisherMock.SetResenderMode(ctx, &s)

	runnerService := runner.NewService()
	if err := runnerService.Init(); err != nil {
		panic(err)
	}
	s.Runner = runnerService

	messageSender := messagesender.NewDefaultService(s.PublisherMock, s.JetCoordinatorMock, s.pulseStorage)
	s.messageSender = messageSender

	var machineLogger smachine.SlotMachineLogger

	if convlog.UseTextConvLog {
		machineLogger = convlog.MachineLogger{}
	} else {
		machineLogger = statemachine.ConveyorLoggerFactory{}
	}
	s.Journal = journal.New()
	machineLogger = s.Journal.InterceptSlotMachineLog(machineLogger, s.fullStop)

	virtualDispatcher := virtual.NewDispatcher()
	virtualDispatcher.Runner = runnerService
	virtualDispatcher.MessageSender = messageSender
	virtualDispatcher.Affinity = s.JetCoordinatorMock
	virtualDispatcher.AuthenticationService = authentication.NewService(ctx, virtualDispatcher.Affinity)

	virtualDispatcher.CycleFn = s.onConveyorCycle
	virtualDispatcher.EventlessSleep = -1 // disable EventlessSleep for proper WaitActiveThenIdleConveyor behavior
	virtualDispatcher.MachineLogger = machineLogger
	s.virtual = virtualDispatcher

	// re HTTP testing
	testWalletAPIConfig := configuration.TestWalletAPI{Address: "very naughty address"}
	s.testWalletServer = testwalletapi.NewTestWalletServer(testWalletAPIConfig, virtualDispatcher, Pulses)

	if init {
		s.Init(ctx)
	}

	return &s, ctx
}

func (s *Server) Init(ctx context.Context) {
	if err := s.virtual.Init(ctx); err != nil {
		panic(err)
	}

	s.pulseManager.AddDispatcher(s.virtual.FlowDispatcher)
	s.IncrementPulseAndWaitIdle(ctx)
}

func (s *Server) StartRecording() {
	s.StartRecordingExt(10_000, true)
}

func (s *Server) StartRecordingExt(limit int, discardOnOverflow bool) {
	s.Journal.StartRecording(limit, discardOnOverflow)
}

func (s *Server) GetPulse() pulsestor.Pulse {
	return s.pulseGenerator.GetLastPulseAsPulse()
}

func (s *Server) GetPrevPulse() pulsestor.Pulse {
	return s.pulseGenerator.GetPrevPulseAsPulse()
}

func (s *Server) incrementPulse(ctx context.Context) {
	s.pulseGenerator.Generate()

	if err := s.pulseManager.Set(ctx, s.GetPulse()); err != nil {
		panic(err)
	}
}

func (s *Server) IncrementPulse(ctx context.Context) {
	s.pulseLock.Lock()
	defer s.pulseLock.Unlock()

	s.incrementPulse(ctx)
}

func (s *Server) IncrementPulseAndWaitIdle(ctx context.Context) {
	s.IncrementPulse(ctx)

	s.WaitActiveThenIdleConveyor()
}

func (s *Server) SendMessage(_ context.Context, msg *message.Message) {
	if err := s.virtual.FlowDispatcher.Process(msg); err != nil {
		panic(err)
	}
}

func (s *Server) ReplaceRunner(svc runner.Service) {
	s.virtual.Runner = svc
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
	return s.virtual.Conveyor.AddInput(ctx, s.GetPulse().PulseNumber, msg)
}

func (s *Server) GlobalCaller() reference.Global {
	return s.caller
}

func (s *Server) RandomLocalWithPulse() reference.Local {
	return gen.UniqueLocalRefWithPulse(s.GetPulse().PulseNumber)
}

func (s *Server) RandomGlobalWithPulse() reference.Global {
	return gen.UniqueGlobalRefWithPulse(s.GetPulse().PulseNumber)
}

func (s *Server) Stop() {
	defer close(s.fullStop)

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

func (s *Server) WrapPayload(pl payload.Marshaler) *RequestWrapper {
	return NewRequestWrapper(s.GetPulse().PulseNumber, pl).SetSender(s.caller)
}

func (s *Server) SendPayload(ctx context.Context, pl payload.Marshaler) {
	msg := s.WrapPayload(pl).Finalize()
	s.SendMessage(ctx, msg)
}
