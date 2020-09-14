// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package hostnetwork

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms/legacyhost"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

var id1, id2, id3, idunknown string

func init() {
	id1 = gen.UniqueGlobalRef().String()
	id2 = gen.UniqueGlobalRef().String()
	id3 = gen.UniqueGlobalRef().String()
	idunknown = gen.UniqueGlobalRef().String()
}

type MockResolver struct {
	mu       sync.RWMutex
	mapping  map[reference.Global]*legacyhost.Host
	smapping map[node.ShortNodeID]*legacyhost.Host
}

func (m *MockResolver) ResolveConsensus(id node.ShortNodeID) (*legacyhost.Host, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result, exist := m.smapping[id]
	if !exist {
		return nil, errors.New("failed to resolve")
	}
	return result, nil
}

func (m *MockResolver) ResolveConsensusRef(nodeID reference.Global) (*legacyhost.Host, error) {
	return m.Resolve(nodeID)
}

func (m *MockResolver) Resolve(nodeID reference.Global) (*legacyhost.Host, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result, exist := m.mapping[nodeID]
	if !exist {
		return nil, errors.New("failed to resolve")
	}
	return result, nil
}

func (m *MockResolver) addMapping(key, value string) error {
	k, err := reference.GlobalFromString(key)
	if err != nil {
		return err
	}
	h, err := legacyhost.NewHostN(value, k)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.mapping[k] = h
	return nil
}

func (m *MockResolver) addMappingHost(h *legacyhost.Host) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.mapping[h.NodeID] = h
	m.smapping[h.ShortID] = h
}

func newMockResolver() *MockResolver {
	return &MockResolver{
		mapping:  make(map[reference.Global]*legacyhost.Host),
		smapping: make(map[node.ShortNodeID]*legacyhost.Host),
	}
}

func TestNewHostNetwork_InvalidReference(t *testing.T) {
	n, err := NewHostNetwork("invalid reference")
	require.Error(t, err)
	require.Nil(t, n)
}

type hostSuite struct {
	t          *testing.T
	ctx1, ctx2 context.Context
	id1, id2   string
	n1, n2     network.HostNetwork
	resolver   *MockResolver
	cm1, cm2   *component.Manager
}

func newHostSuite(t *testing.T) *hostSuite {
	ctx := instestlogger.TestContext(t)

	ctx1 := inslogger.ContextWithTrace(ctx, "AAA")
	ctx2 := inslogger.ContextWithTrace(ctx, "BBB")
	resolver := newMockResolver()

	cm1 := component.NewManager(nil)
	cm1.SetLogger(global.Logger())

	cfg1 := configuration.NewHostNetwork().Transport
	cfg1.Address = "127.0.0.1:8088"
	f1 := transport.NewFakeFactory(cfg1)
	n1, err := NewHostNetwork(id1)
	require.NoError(t, err)
	cm1.Inject(f1, n1, resolver)

	cm2 := component.NewManager(nil)
	cm2.SetLogger(global.Logger())

	cfg2 := configuration.NewHostNetwork().Transport
	cfg2.Address = "127.0.0.1:8087"
	f2 := transport.NewFakeFactory(cfg2)
	n2, err := NewHostNetwork(id2)
	require.NoError(t, err)
	cm2.Inject(f2, n2, resolver)

	err = cm1.Init(ctx1)
	require.NoError(t, err)
	err = cm2.Init(ctx2)
	require.NoError(t, err)

	return &hostSuite{
		t: t, ctx1: ctx1, ctx2: ctx2, id1: id1, id2: id2, n1: n1, n2: n2, resolver: resolver, cm1: cm1, cm2: cm2,
	}
}

func (s *hostSuite) Start() {
	// start the second hostNetwork before the first because most test cases perform sending packets first -> second,
	// so the second hostNetwork should be ready to receive packets when the first starts to send
	err := s.cm1.Start(s.ctx1)
	require.NoError(s.t, err)
	err = s.cm2.Start(s.ctx2)
	require.NoError(s.t, err)

	require.NoError(s.t, transport.WaitFakeListeners(2, time.Second * 5))

	err = s.resolver.addMapping(s.id1, s.n1.PublicAddress())
	require.NoError(s.t, err, "failed to add mapping %s -> %s: %s", s.id1, s.n1.PublicAddress(), err)
	err = s.resolver.addMapping(s.id2, s.n2.PublicAddress())
	require.NoError(s.t, err, "failed to add mapping %s -> %s: %s", s.id2, s.n2.PublicAddress(), err)
}

func (s *hostSuite) Stop() {
	// stop hostNetworks in the reverse order of their start
	err := s.cm1.Stop(s.ctx1)
	assert.NoError(s.t, err)
	err = s.cm2.Stop(s.ctx2)
	assert.NoError(s.t, err)
}

func TestNewHostNetwork(t *testing.T) {
	defer testutils.LeakTester(t)
	instestlogger.SetTestOutputWithErrorFilter(t, func(s string) bool {
		return !strings.Contains(s, "Failed to send response")
	})

	s := newHostSuite(t)
	defer s.Stop()

	count := 10

	s.n2.RegisterRequestHandler(types.RPC, func(ctx context.Context, request network.ReceivedPacket) (network.Packet, error) {
		inslogger.FromContext(ctx).Info("handler triggered")
		return s.n2.BuildResponse(ctx, request, &rms.RPCResponse{}), nil
	})

	s.Start()

	responses := make([]network.Future, count)
	for i := 0; i < count; i++ {
		ref, err := reference.GlobalFromString(id2)
		require.NoError(t, err)
		responses[i], err = s.n1.SendRequest(s.ctx1, types.RPC, &rms.RPCRequest{}, ref)
		require.NoError(t, err)
	}

	for i := len(responses) - 1; i >= 0; i-- {
		_, err := responses[i].WaitResponse(time.Minute)
		require.NoError(t, err)
	}
}

func TestHostNetwork_SendRequestPacket(t *testing.T) {
	defer testutils.LeakTester(t)

	m := newMockResolver()
	ctx := instestlogger.TestContext(t)

	n1, err := NewHostNetwork(id1)
	require.NoError(t, err)

	cm := component.NewManager(nil)
	cm.SetLogger(global.Logger())
	cm.Register(m, n1, transport.NewFactory(configuration.NewHostNetwork().Transport))
	cm.Inject()
	err = cm.Init(ctx)
	require.NoError(t, err)
	err = cm.Start(ctx)
	require.NoError(t, err)

	defer func() {
		err = cm.Stop(ctx)
		assert.NoError(t, err)
	}()

	unknownID, err := reference.GlobalFromString(idunknown)
	require.NoError(t, err)

	// should return error because cannot resolve NodeID -> Address
	f, err := n1.SendRequest(ctx, types.Pulse, &rms.PulseRequest{}, unknownID)
	require.Error(t, err)
	assert.Nil(t, f)

	err = m.addMapping(id2, "abirvalg")
	require.Error(t, err)
	err = m.addMapping(id3, "127.0.0.1:7654")
	require.NoError(t, err)

	ref, err := reference.GlobalFromString(id3)
	require.NoError(t, err)
	// should return error because resolved address is invalid
	f, err = n1.SendRequest(ctx, types.Pulse, &rms.PulseRequest{}, ref)
	require.Error(t, err)
	assert.Nil(t, f)
}

func TestHostNetwork_SendRequestPacket3(t *testing.T) {
	defer testutils.LeakTester(t)

	instestlogger.SetTestOutput(t)
	s := newHostSuite(t)
	defer s.Stop()

	handler := func(ctx context.Context, r network.ReceivedPacket) (network.Packet, error) {
		inslogger.FromContext(ctx).Info("handler triggered")
		return s.n2.BuildResponse(ctx, r, &rms.BasicResponse{Error: "Error"}), nil
	}
	s.n2.RegisterRequestHandler(types.Pulse, handler)

	s.Start()

	request := &rms.PulseRequest{}
	ref, err := reference.GlobalFromString(id2)
	require.NoError(t, err)
	f, err := s.n1.SendRequest(s.ctx1, types.Pulse, request, ref)
	require.NoError(t, err)

	r, err := f.WaitResponse(time.Minute)
	require.NoError(t, err)

	d := r.GetResponse().GetBasic().Error
	require.Equal(t, "Error", d)

	request = &rms.PulseRequest{}
	f, err = s.n1.SendRequest(s.ctx1, types.Pulse, request, ref)
	require.NoError(t, err)

	r, err = f.WaitResponse(time.Second)
	assert.NoError(t, err)
	d = r.GetResponse().GetBasic().Error
	require.Equal(t, d, "Error")
}

func TestHostNetwork_SendRequestPacket_errors(t *testing.T) {
	defer testutils.LeakTester(t)

	instestlogger.SetTestOutputWithErrorFilter(t, func(s string) bool {
		return !strings.Contains(s, "Failed to send response")
	})
	s := newHostSuite(t)
	defer s.Stop()

	handler := func(ctx context.Context, r network.ReceivedPacket) (network.Packet, error) {
		inslogger.FromContext(ctx).Info("handler triggered")
		time.Sleep(time.Millisecond * 100)
		return s.n2.BuildResponse(ctx, r, &rms.RPCResponse{}), nil
	}
	s.n2.RegisterRequestHandler(types.RPC, handler)

	s.Start()

	ref, err := reference.GlobalFromString(id2)
	require.NoError(t, err)
	f, err := s.n1.SendRequest(s.ctx1, types.RPC, &rms.RPCRequest{}, ref)
	require.NoError(t, err)

	_, err = f.WaitResponse(time.Microsecond * 10)
	require.Error(t, err)

	f, err = s.n1.SendRequest(s.ctx1, types.RPC, &rms.RPCRequest{}, ref)
	require.NoError(t, err)

	_, err = f.WaitResponse(time.Minute)
	require.NoError(t, err)
}

func TestHostNetwork_WrongHandler(t *testing.T) {
	defer testutils.LeakTester(t)
	instestlogger.SetTestOutput(t)
	s := newHostSuite(t)
	defer s.Stop()

	handler := func(ctx context.Context, r network.ReceivedPacket) (network.Packet, error) {
		inslogger.FromContext(ctx).Info("handler triggered")
		require.Fail(t, "shouldn't be called")
		return s.n2.BuildResponse(ctx, r, nil), nil
	}
	s.n2.RegisterRequestHandler(types.Unknown, handler)

	s.Start()

	ref, err := reference.GlobalFromString(id2)
	require.NoError(t, err)
	f, err := s.n1.SendRequest(s.ctx1, types.Pulse, &rms.PulseRequest{}, ref)
	require.NoError(t, err)

	r, err := f.WaitResponse(time.Minute)
	require.NoError(t, err)

	d := r.GetResponse().GetError().Error
	require.NotEmpty(t, d)
}

func TestStartStopSend(t *testing.T) {
	defer testutils.LeakTester(t)
	instestlogger.SetTestOutput(t)
	s := newHostSuite(t)
	defer s.Stop()

	wg := sync.WaitGroup{}
	wg.Add(2)

	handler := func(ctx context.Context, r network.ReceivedPacket) (network.Packet, error) {
		inslogger.FromContext(ctx).Info("handler triggered")
		wg.Done()
		return s.n2.BuildResponse(ctx, r, &rms.RPCResponse{}), nil
	}
	s.n2.RegisterRequestHandler(types.RPC, handler)

	s.Start()

	send := func() {
		ref, err := reference.GlobalFromString(id2)
		require.NoError(t, err)
		f, err := s.n1.SendRequest(s.ctx1, types.RPC, &rms.RPCRequest{}, ref)
		require.NoError(t, err)
		_, err = f.WaitResponse(time.Second)
		assert.NoError(t, err)
	}

	send()

	err := s.cm1.Stop(s.ctx1)
	require.NoError(t, err)
	<-time.After(time.Millisecond * 10)

	s.ctx1 = instestlogger.TestContext(t)
	err = s.cm1.Start(s.ctx1)
	require.NoError(t, err)

	send()
	wg.Wait()
}

func TestHostNetwork_SendRequestToHost_NotStarted(t *testing.T) {
	defer testutils.LeakTester(t)

	ctx := instestlogger.TestContext(t)

	hn, err := NewHostNetwork(id1)
	require.NoError(t, err)

	f, err := hn.SendRequestToHost(ctx, types.Unknown, nil, nil)
	require.EqualError(t, err, "host network is not started")
	assert.Nil(t, f)
}
