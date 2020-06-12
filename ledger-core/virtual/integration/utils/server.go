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
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodestorage"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/virtual"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/convlog"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock"
	"github.com/insolar/assured-ledger/ledger-core/virtual/pulsemanager"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/token"
)

type Server struct {
	pulseLock sync.Mutex
	dataLock  sync.Mutex

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

	cycleFn     ConveyorCycleFunc
	activeCount atomickit.Uint32

	// components for testing http api
	testWalletServer *testwalletapi.TestWalletServer

	// top-level caller ID
	caller reference.Global
}

type ConveyorCycleFunc func(c *conveyor.PulseConveyor, idle bool)

func NewServer(ctx context.Context, t *testing.T) (*Server, context.Context) {
	return newServerExt(ctx, t, false, true)
}

func NewServerIgnoreLogErrors(ctx context.Context, t *testing.T) (*Server, context.Context) {
	return newServerExt(ctx, t, true, true)
}

func NewUninitializedServer(ctx context.Context, t *testing.T) (*Server, context.Context) {
	return newServerExt(ctx, t, false, false)
}

func NewUninitializedServerIgnoreLogErrors(ctx context.Context, t *testing.T) (*Server, context.Context) {
	return newServerExt(ctx, t, true, false)
}

func newServerExt(ctx context.Context, t *testing.T, suppressLogError bool, init bool) (*Server, context.Context) {
	inslogger.SetTestOutput(t, suppressLogError)

	if ctx == nil {
		ctx = inslogger.TestContext(t)
	}

	s := Server{
		caller: gen.UniqueReference(),
	}

	// Pulse-related components
	var (
		PulseManager *pulsemanager.PulseManager
		Pulses       *pulsestor.StorageMem
	)
	{
		networkNodeMock := network.NewNetworkNodeMock(t).
			IDMock.Return(gen.UniqueReference()).
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
		QueryRoleMock.Return([]reference.Global{gen.UniqueReference()}, nil)

	s.PublisherMock = mock.NewPublisherMock()
	s.PublisherMock.SetResenderMode(ctx, &s)

	runnerService := runner.NewService()
	if err := runnerService.Init(); err != nil {
		panic(err)
	}
	s.Runner = runnerService

	messageSender := messagesender.NewDefaultService(s.PublisherMock, s.JetCoordinatorMock, s.pulseStorage)
	s.messageSender = messageSender

	virtualDispatcher := virtual.NewDispatcher()
	virtualDispatcher.Runner = runnerService
	virtualDispatcher.MessageSender = messageSender
	virtualDispatcher.TokenService = token.NewService(ctx, s.caller)
	virtualDispatcher.CycleFn = s.onCycle
	s.virtual = virtualDispatcher

	if convlog.UseTextConvLog {
		virtualDispatcher.MachineLogger = convlog.MachineLogger{}
	}

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

func (s *Server) SetCycleCallback(cycleFn ConveyorCycleFunc) {
	s.dataLock.Lock()
	defer s.dataLock.Unlock()

	s.cycleFn = cycleFn
}

func (s *Server) onCycle(idle bool) {
	s.dataLock.Lock()
	cycleFn := s.cycleFn
	s.dataLock.Unlock()

	if !idle {
		s.activeCount.Store(1)
	}
	if cycleFn == nil {
		return
	}
	cycleFn(s.virtual.Conveyor, idle)
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

func (s *Server) AddInput(ctx context.Context, msg interface{}) error {
	return s.virtual.Conveyor.AddInput(ctx, s.GetPulse().PulseNumber, msg)
}

func (s *Server) GlobalCaller() reference.Global {
	return s.caller
}

func (s *Server) RandomLocalWithPulse() reference.Local {
	return gen.UniqueIDWithPulse(s.GetPulse().PulseNumber)
}

func (s *Server) Stop() {
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
	s.SetCycleCallback(func(c *conveyor.PulseConveyor, idle bool) {
		if idle && (!checkActive || s.activeCount.Load() > 0) {
			s.activeCount.Store(0)
			s.SetCycleCallback(nil)
			wg.Done()
		}
	})
	s.virtual.Conveyor.WakeUpWorker()
	wg.Wait()
}

func (s *Server) ResetActiveConveyorFlag() {
	s.activeCount.Store(0)
}
