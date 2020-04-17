// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package small

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/testwalletapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/network"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/mimic"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/pulsemanager"
)

type PublisherMock struct {
	Checker func(topic string, messages ...*message.Message) error
}

func (p *PublisherMock) Publish(topic string, messages ...*message.Message) error {
	if err := p.Checker(topic, messages...); err != nil {
		panic(err)
	}
	return nil
}

func (*PublisherMock) Close() error { return nil }

var _ message.Publisher = &PublisherMock{}

type Server struct {
	lock sync.Mutex

	// real components
	virtual       *virtual.Dispatcher
	runner        *runner.DefaultService
	messageSender *messagesender.DefaultService

	// testing components and Mocks
	PublisherMock      *PublisherMock
	JetCoordinatorMock *jet.CoordinatorMock
	pulseGenerator     *mimic.PulseGenerator
	pulseStorage       *pulse.StorageMem
	pulseManager       insolar.PulseManager

	// components for testing http api
	testWalletServer *testwalletapi.TestWalletServer
}

func NewServer(t *testing.T) *Server {
	ctx := context.Background()

	s := Server{}

	// Pulse-related components
	var (
		PulseManager *pulsemanager.PulseManager
		Pulses       *pulse.StorageMem
	)
	{
		networkNodeMock := network.NewNetworkNodeMock(t).
			IDMock.Return(gen.Reference()).
			ShortIDMock.Return(insolar.ShortNodeID(0)).
			RoleMock.Return(insolar.StaticRoleVirtual).
			AddressMock.Return("").
			GetStateMock.Return(insolar.NodeReady).
			GetPowerMock.Return(1)
		networkNodeList := []insolar.NetworkNode{networkNodeMock}

		nodeNetworkAccessorMock := network.NewAccessorMock(t).GetWorkingNodesMock.Return(networkNodeList)
		nodeNetworkMock := network.NewNodeNetworkMock(t).GetAccessorMock.Return(nodeNetworkAccessorMock)
		nodeSetter := node.NewModifierMock(t).SetMock.Return(nil)
		jetModifier := jet.NewModifierMock(t).CloneMock.Return(nil)

		Pulses = pulse.NewStorageMem()
		PulseManager = pulsemanager.NewPulseManager()
		PulseManager.NodeNet = nodeNetworkMock
		PulseManager.NodeSetter = nodeSetter
		PulseManager.JetModifier = jetModifier
		PulseManager.PulseAccessor = Pulses
		PulseManager.PulseAppender = Pulses
	}

	s.pulseManager = PulseManager
	s.pulseStorage = Pulses
	s.pulseGenerator = mimic.NewPulseGenerator(10)

	s.JetCoordinatorMock = jet.NewCoordinatorMock(t).
		MeMock.Return(gen.Reference()).
		QueryRoleMock.Return([]insolar.Reference{gen.Reference()}, nil)

	s.PublisherMock = &PublisherMock{}

	runnerService := runner.NewService()
	if err := runnerService.Init(); err != nil {
		panic(err)
	}
	s.runner = runnerService

	messageSender := messagesender.NewDefaultService(s.PublisherMock, s.JetCoordinatorMock, s.pulseStorage)
	s.messageSender = messageSender

	virtualDispatcher := virtual.NewDispatcher()
	virtualDispatcher.Runner = runnerService
	virtualDispatcher.MessageSender = messageSender

	if err := virtualDispatcher.Init(ctx); err != nil {
		panic(err)
	}

	s.virtual = virtualDispatcher

	testWalletAPIConfig := configuration.TestWalletAPI{Address: "very naughty address"}
	s.testWalletServer = testwalletapi.NewTestWalletServer(testWalletAPIConfig, virtualDispatcher, Pulses)

	PulseManager.AddDispatcher(s.virtual.FlowDispatcher)
	s.IncrementPulse(ctx)

	return &s
}

func (s *Server) GetPulse() insolar.Pulse {
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

func (s *Server) ReplaceMachinesManager(manager executor.Manager) {
	s.runner.Manager = manager
}

func (s *Server) ReplaceCache(cache descriptor.Cache) {
	s.runner.Cache = cache
}

func (s *Server) AddInput(msg interface{}) error {
	return s.virtual.AddInput(context.Background(), s.GetPulse().PulseNumber, msg)
}

// Utility function RE wallet creation

func (s *Server) CallCreateWallet() (int, []byte) {
	var (
		responseWriter = httptest.NewRecorder()
		httpRequest    = httptest.NewRequest("POST", "/wallet/request", strings.NewReader(""))
	)

	s.testWalletServer.Create(responseWriter, httpRequest)

	var (
		statusCode = responseWriter.Result().StatusCode
		body, err  = ioutil.ReadAll(responseWriter.Result().Body)
	)

	if err != nil {
		panic(err)
	}

	return statusCode, body
}
