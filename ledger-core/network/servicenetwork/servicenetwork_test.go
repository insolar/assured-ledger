// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package servicenetwork

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/controller"
	"github.com/insolar/assured-ledger/ledger-core/reference"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/node"
	"github.com/insolar/assured-ledger/ledger-core/network/nodenetwork"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	networkUtils "github.com/insolar/assured-ledger/ledger-core/testutils/network"
)

type PublisherMock struct{}

func (p *PublisherMock) Publish(topic string, messages ...*message.Message) error {
	return nil
}

func (p *PublisherMock) Close() error {
	return nil
}

func prepareNetwork(t *testing.T, cfg configuration.Configuration) *ServiceNetwork {
	instestlogger.SetTestOutputWithIgnoreAllErrors(t)

	serviceNetwork, err := NewServiceNetwork(cfg, component.NewManager(nil))
	require.NoError(t, err)

	nodeKeeper := networkUtils.NewNodeKeeperMock(t)
	nodeMock := networkUtils.NewNetworkNodeMock(t)
	nodeMock.IDMock.Return(gen.UniqueGlobalRef())
	nodeKeeper.GetOriginMock.Return(nodeMock)
	serviceNetwork.NodeKeeper = nodeKeeper

	return serviceNetwork
}

func TestSendMessageHandler_ReceiverNotSet(t *testing.T) {
	cfg := configuration.NewConfiguration()

	serviceNetwork := prepareNetwork(t, cfg)

	p := []byte{1, 2, 3, 4, 5}
	meta := payload.Meta{
		Payload: p,
	}
	data, err := meta.Marshal()
	require.NoError(t, err)

	inMsg := message.NewMessage(watermill.NewUUID(), data)

	err = serviceNetwork.SendMessageHandler(inMsg)
	require.NoError(t, err)
}

func TestSendMessageHandler_SameNode(t *testing.T) {
	cfg := configuration.NewConfiguration()
	serviceNetwork, err := NewServiceNetwork(cfg, component.NewManager(nil))
	nodeRef := gen.UniqueGlobalRef()
	nodeN := networkUtils.NewNodeKeeperMock(t)
	nodeN.GetOriginMock.Set(func() (r node2.NetworkNode) {
		n := networkUtils.NewNetworkNodeMock(t)
		n.IDMock.Set(func() (r reference.Global) {
			return nodeRef
		})
		return n
	})
	pubMock := &PublisherMock{}
	pulseMock := networkUtils.NewPulseAccessorMock(t)
	pulseMock.GetLatestPulseMock.Return(*pulsestor.GenesisPulse, nil)
	serviceNetwork.PulseAccessor = pulseMock
	serviceNetwork.NodeKeeper = nodeN
	serviceNetwork.Pub = pubMock

	p := []byte{1, 2, 3, 4, 5}
	meta := payload.Meta{
		Payload:  p,
		Receiver: nodeRef,
	}
	data, err := meta.Marshal()
	require.NoError(t, err)

	inMsg := message.NewMessage(watermill.NewUUID(), data)

	err = serviceNetwork.SendMessageHandler(inMsg)
	require.NoError(t, err)
}

func TestSendMessageHandler_SendError(t *testing.T) {
	cfg := configuration.NewConfiguration()
	pubMock := &PublisherMock{}
	serviceNetwork, err := NewServiceNetwork(cfg, component.NewManager(nil))
	serviceNetwork.Pub = pubMock
	nodeN := networkUtils.NewNodeKeeperMock(t)
	nodeN.GetOriginMock.Set(func() (r node2.NetworkNode) {
		n := networkUtils.NewNetworkNodeMock(t)
		n.IDMock.Set(func() (r reference.Global) {
			return gen.UniqueGlobalRef()
		})
		return n
	})
	rpc := controller.NewRPCControllerMock(t)
	rpc.SendBytesMock.Set(func(p context.Context, p1 reference.Global, p2 string, p3 []byte) (r []byte, r1 error) {
		return nil, errors.New("test error")
	})
	pulseMock := networkUtils.NewPulseAccessorMock(t)
	pulseMock.GetLatestPulseMock.Return(*pulsestor.GenesisPulse, nil)
	serviceNetwork.PulseAccessor = pulseMock
	serviceNetwork.RPC = rpc
	serviceNetwork.NodeKeeper = nodeN

	p := []byte{1, 2, 3, 4, 5}
	meta := payload.Meta{
		Payload:  p,
		Receiver: gen.UniqueGlobalRef(),
	}
	data, err := meta.Marshal()
	require.NoError(t, err)

	inMsg := message.NewMessage(watermill.NewUUID(), data)

	err = serviceNetwork.SendMessageHandler(inMsg)
	require.NoError(t, err)
}

func TestSendMessageHandler_WrongReply(t *testing.T) {
	cfg := configuration.NewConfiguration()
	pubMock := &PublisherMock{}
	serviceNetwork, err := NewServiceNetwork(cfg, component.NewManager(nil))
	serviceNetwork.Pub = pubMock
	nodeN := networkUtils.NewNodeKeeperMock(t)
	nodeN.GetOriginMock.Set(func() (r node2.NetworkNode) {
		n := networkUtils.NewNetworkNodeMock(t)
		n.IDMock.Set(func() (r reference.Global) {
			return gen.UniqueGlobalRef()
		})
		return n
	})
	rpc := controller.NewRPCControllerMock(t)
	rpc.SendBytesMock.Set(func(p context.Context, p1 reference.Global, p2 string, p3 []byte) (r []byte, r1 error) {
		return nil, nil
	})
	pulseMock := networkUtils.NewPulseAccessorMock(t)
	pulseMock.GetLatestPulseMock.Return(*pulsestor.GenesisPulse, nil)
	serviceNetwork.PulseAccessor = pulseMock
	serviceNetwork.RPC = rpc
	serviceNetwork.NodeKeeper = nodeN

	p := []byte{1, 2, 3, 4, 5}
	meta := payload.Meta{
		Payload:  p,
		Receiver: gen.UniqueGlobalRef(),
	}
	data, err := meta.Marshal()
	require.NoError(t, err)

	inMsg := message.NewMessage(watermill.NewUUID(), data)

	err = serviceNetwork.SendMessageHandler(inMsg)
	require.NoError(t, err)
}

func TestSendMessageHandler(t *testing.T) {
	cfg := configuration.NewConfiguration()
	serviceNetwork, err := NewServiceNetwork(cfg, component.NewManager(nil))
	nodeN := networkUtils.NewNodeKeeperMock(t)
	nodeN.GetOriginMock.Set(func() (r node2.NetworkNode) {
		n := networkUtils.NewNetworkNodeMock(t)
		n.IDMock.Set(func() (r reference.Global) {
			return gen.UniqueGlobalRef()
		})
		return n
	})
	rpc := controller.NewRPCControllerMock(t)
	rpc.SendBytesMock.Set(func(p context.Context, p1 reference.Global, p2 string, p3 []byte) (r []byte, r1 error) {
		return ack, nil
	})
	pulseMock := networkUtils.NewPulseAccessorMock(t)
	pulseMock.GetLatestPulseMock.Return(*pulsestor.GenesisPulse, nil)
	serviceNetwork.PulseAccessor = pulseMock
	serviceNetwork.RPC = rpc
	serviceNetwork.NodeKeeper = nodeN

	p := []byte{1, 2, 3, 4, 5}
	meta := payload.Meta{
		Payload:  p,
		Receiver: gen.UniqueGlobalRef(),
	}
	data, err := meta.Marshal()
	require.NoError(t, err)

	inMsg := message.NewMessage(watermill.NewUUID(), data)

	err = serviceNetwork.SendMessageHandler(inMsg)
	require.NoError(t, err)
}

type stater struct{}

func (s *stater) State() []byte {
	return []byte("123")
}

func TestServiceNetwork_StartStop(t *testing.T) {
	t.Skip("fixme")
	cm := component.NewManager(nil)
	cm.SetLogger(global.Logger())

	origin := gen.UniqueGlobalRef()
	nk := nodenetwork.NewNodeKeeper(node.NewNode(origin, node2.StaticRoleUnknown, nil, "127.0.0.1:0", ""))
	cert := &mandates.Certificate{}
	cert.Reference = origin.String()
	certManager := mandates.NewCertificateManager(cert)
	serviceNetwork, err := NewServiceNetwork(configuration.NewConfiguration(), cm)
	require.NoError(t, err)
	ctx := context.Background()
	defer serviceNetwork.Stop(ctx)

	cm.Inject(serviceNetwork, nk, certManager, cryptography.NewServiceMock(t), pulsestor.NewAccessorMock(t),
		testutils.NewTerminationHandlerMock(t), pulsestor.NewManagerMock(t), &PublisherMock{}, &stater{},
		testutils.NewPlatformCryptographyScheme(), testutils.NewKeyProcessorMock(t))
	err = serviceNetwork.Init(ctx)
	require.NoError(t, err)
	err = serviceNetwork.Start(ctx)
	require.NoError(t, err)
}

type publisherMock struct {
	Error error
}

func (pm *publisherMock) Publish(topic string, messages ...*message.Message) error { return pm.Error }
func (pm *publisherMock) Close() error                                             { return nil }

func TestServiceNetwork_processIncoming(t *testing.T) {
	serviceNetwork, err := NewServiceNetwork(configuration.NewConfiguration(), component.NewManager(nil))
	require.NoError(t, err)
	pub := &publisherMock{}
	serviceNetwork.Pub = pub
	ctx := context.Background()
	_, err = serviceNetwork.processIncoming(ctx, []byte("ololo"))
	assert.Error(t, err)
	msg := message.NewMessage("1", nil)
	data, err := serializeMessage(msg)
	require.NoError(t, err)
	_, err = serviceNetwork.processIncoming(ctx, data)
	assert.NoError(t, err)
	pub.Error = errors.New("Failed to publish message")
	_, err = serviceNetwork.processIncoming(ctx, data)
	assert.Error(t, err)
}
