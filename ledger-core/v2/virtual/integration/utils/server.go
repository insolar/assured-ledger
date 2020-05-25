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

	"github.com/insolar/assured-ledger/ledger-core/v2/application/testwalletapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/nodestorage"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/network"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/convlog"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/mock"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/pulsemanager"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils"
)

type Server struct {
	lock sync.Mutex

	// real components
	virtual       *virtual.Dispatcher
	Runner        *runner.DefaultService
	messageSender *messagesender.DefaultService

	// testing components and Mocks
	PublisherMock      *mock.PublisherMock
	JetCoordinatorMock *jet.AffinityHelperMock
	pulseGenerator     *testutils.PulseGenerator
	pulseStorage       *pulsestor.StorageMem
	pulseManager       pulsestor.Manager

	// components for testing http api
	testWalletServer *testwalletapi.TestWalletServer

	// top-level caller ID
	caller reference.Global
}

func NewServer(ctx context.Context, t *testing.T) (*Server, context.Context) {
	return newServerExt(ctx, t, false)
}

func NewServerIgnoreLogErrors(ctx context.Context, t *testing.T) (*Server, context.Context) {
	return newServerExt(ctx, t, true)
}

func newServerExt(ctx context.Context, t *testing.T, suppressLogError bool) (*Server, context.Context) {
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
		MeMock.Return(gen.UniqueReference()).
		QueryRoleMock.Return([]reference.Global{gen.UniqueReference()}, nil)

	s.PublisherMock = &mock.PublisherMock{}

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

	if convlog.UseTextConvLog {
		virtualDispatcher.MachineLogger = convlog.MachineLogger{}
	}
	if err := virtualDispatcher.Init(ctx); err != nil {
		panic(err)
	}

	s.virtual = virtualDispatcher

	PulseManager.AddDispatcher(s.virtual.FlowDispatcher)
	s.IncrementPulse(ctx)

	// re HTTP testing
	testWalletAPIConfig := configuration.TestWalletAPI{Address: "very naughty address"}
	s.testWalletServer = testwalletapi.NewTestWalletServer(testWalletAPIConfig, virtualDispatcher, Pulses)

	return &s, ctx
}

func (s *Server) GetPulse() pulsestor.Pulse {
	return s.pulseGenerator.GetLastPulseAsPulse()
}

func (s *Server) incrementPulse(ctx context.Context) {
	s.pulseGenerator.Generate()

	if err := s.pulseManager.Set(ctx, s.GetPulse()); err != nil {
		panic(err)
	}
}

func (s *Server) IncrementPulse(ctx context.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.incrementPulse(ctx)
}

func (s *Server) SendMessage(_ context.Context, msg *message.Message) {
	if err := s.virtual.FlowDispatcher.Process(msg); err != nil {
		panic(err)
	}
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
