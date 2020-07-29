// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package servicenetwork

import (
	"context"
	"strings"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/controller"
	"github.com/insolar/assured-ledger/ledger-core/reference"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
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
	serviceNetwork, err := NewServiceNetwork(cfg, component.NewManager(nil))
	require.NoError(t, err)

	nodeKeeper := networkUtils.NewNodeKeeperMock(t)
	nodeMock := networkUtils.NewNetworkNodeMock(t)
	nodeMock.GetReferenceMock.Return(gen.UniqueGlobalRef())
	nodeKeeper.GetOriginMock.Return(nodeMock)
	serviceNetwork.NodeKeeper = nodeKeeper

	return serviceNetwork
}

func TestSendMessageHandler_ReceiverNotSet(t *testing.T) {
	cfg := configuration.NewConfiguration()
	instestlogger.SetTestOutputWithErrorFilter(t, func(s string) bool {
		if strings.Contains(s, "failed to send message: Receiver in message metadata is not set") {
			return false
		}
		return true
	})

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
	svcNw, err := NewServiceNetwork(cfg, component.NewManager(nil))
	nodeRef := gen.UniqueGlobalRef()
	nodeN := networkUtils.NewNodeKeeperMock(t)
	nodeN.GetOriginMock.Set(func() (r nodeinfo.NetworkNode) {
		n := networkUtils.NewNetworkNodeMock(t)
		n.GetReferenceMock.Set(func() (r reference.Global) {
			return nodeRef
		})
		return n
	})
	pubMock := &PublisherMock{}
	pulseMock := beat.NewAccessorMock(t)
	pulseMock.LatestMock.Return(pulsestor.GenesisPulse, nil)
	svcNw.NodeKeeper = nodeN
	svcNw.Pub = pubMock

	p := []byte{1, 2, 3, 4, 5}
	meta := payload.Meta{
		Payload:  p,
		Receiver: nodeRef,
	}
	data, err := meta.Marshal()
	require.NoError(t, err)

	inMsg := message.NewMessage(watermill.NewUUID(), data)

	err = svcNw.SendMessageHandler(inMsg)
	require.NoError(t, err)
}

func TestSendMessageHandler_SendError(t *testing.T) {
	cfg := configuration.NewConfiguration()
	pubMock := &PublisherMock{}
	svcNw, err := NewServiceNetwork(cfg, component.NewManager(nil))
	require.NoError(t, err)
	svcNw.Pub = pubMock
	nodeN := networkUtils.NewNodeKeeperMock(t)
	nodeN.GetOriginMock.Set(func() (r nodeinfo.NetworkNode) {
		n := networkUtils.NewNetworkNodeMock(t)
		n.GetReferenceMock.Set(func() (r reference.Global) {
			return gen.UniqueGlobalRef()
		})
		return n
	})
	rpc := controller.NewRPCControllerMock(t)
	rpc.SendBytesMock.Set(func(p context.Context, p1 reference.Global, p2 string, p3 []byte) (r []byte, r1 error) {
		return nil, errors.New("test error")
	})
	pulseMock := beat.NewAccessorMock(t)
	pulseMock.LatestMock.Return(pulsestor.GenesisPulse, nil)
	svcNw.RPC = rpc
	svcNw.NodeKeeper = nodeN

	p := []byte{1, 2, 3, 4, 5}
	meta := payload.Meta{
		Payload:  p,
		Receiver: gen.UniqueGlobalRef(),
	}
	data, err := meta.Marshal()
	require.NoError(t, err)

	inMsg := message.NewMessage(watermill.NewUUID(), data)

	err = svcNw.SendMessageHandler(inMsg)
	require.NoError(t, err)
}

func TestSendMessageHandler_WrongReply(t *testing.T) {
	cfg := configuration.NewConfiguration()
	pubMock := &PublisherMock{}
	svcNw, err := NewServiceNetwork(cfg, component.NewManager(nil))
	require.NoError(t, err)
	svcNw.Pub = pubMock
	nodeN := networkUtils.NewNodeKeeperMock(t)
	nodeN.GetOriginMock.Set(func() (r nodeinfo.NetworkNode) {
		n := networkUtils.NewNetworkNodeMock(t)
		n.GetReferenceMock.Set(func() (r reference.Global) {
			return gen.UniqueGlobalRef()
		})
		return n
	})
	rpc := controller.NewRPCControllerMock(t)
	rpc.SendBytesMock.Set(func(p context.Context, p1 reference.Global, p2 string, p3 []byte) (r []byte, r1 error) {
		return nil, nil
	})
	pulseMock := beat.NewAccessorMock(t)
	pulseMock.LatestMock.Return(pulsestor.GenesisPulse, nil)
	svcNw.RPC = rpc
	svcNw.NodeKeeper = nodeN

	p := []byte{1, 2, 3, 4, 5}
	meta := payload.Meta{
		Payload:  p,
		Receiver: gen.UniqueGlobalRef(),
	}
	data, err := meta.Marshal()
	require.NoError(t, err)

	inMsg := message.NewMessage(watermill.NewUUID(), data)

	err = svcNw.SendMessageHandler(inMsg)
	require.NoError(t, err)
}

func TestSendMessageHandler(t *testing.T) {
	cfg := configuration.NewConfiguration()
	svcNw, err := NewServiceNetwork(cfg, component.NewManager(nil))
	require.NoError(t, err)
	nodeN := networkUtils.NewNodeKeeperMock(t)
	nodeN.GetOriginMock.Set(func() (r nodeinfo.NetworkNode) {
		n := networkUtils.NewNetworkNodeMock(t)
		n.GetReferenceMock.Set(func() (r reference.Global) {
			return gen.UniqueGlobalRef()
		})
		return n
	})
	rpc := controller.NewRPCControllerMock(t)
	rpc.SendBytesMock.Set(func(p context.Context, p1 reference.Global, p2 string, p3 []byte) (r []byte, r1 error) {
		return ack, nil
	})
	pulseMock := beat.NewAccessorMock(t)
	pulseMock.LatestMock.Return(pulsestor.GenesisPulse, nil)
	svcNw.RPC = rpc
	svcNw.NodeKeeper = nodeN

	p := []byte{1, 2, 3, 4, 5}
	meta := payload.Meta{
		Payload:  p,
		Receiver: gen.UniqueGlobalRef(),
	}
	data, err := meta.Marshal()
	require.NoError(t, err)

	inMsg := message.NewMessage(watermill.NewUUID(), data)

	err = svcNw.SendMessageHandler(inMsg)
	require.NoError(t, err)
}

type stater struct{}

func (s *stater) RequestNodeState(fn chorus.NodeStateFunc) {
	fn(api.UpstreamState{})
}

func (s *stater) CancelNodeState() {}

func TestServiceNetwork_StartStop(t *testing.T) {
	t.Skip("fixme")
	instestlogger.SetTestOutput(t)
	cm := component.NewManager(nil)
	cm.SetLogger(global.Logger())

	origin := gen.UniqueGlobalRef()
	nk := nodenetwork.NewNodeKeeper(node.NewNode(origin, member.PrimaryRoleUnknown, nil, "127.0.0.1:0"))
	cert := &mandates.Certificate{}
	cert.Reference = origin.String()
	certManager := mandates.NewCertificateManager(cert)
	svcNw, err := NewServiceNetwork(configuration.NewConfiguration(), cm)
	require.NoError(t, err)
	ctx := context.Background()
	defer svcNw.Stop(ctx)

	kp := platformpolicy.NewKeyProcessor()
	skey, err := kp.GeneratePrivateKey()
	require.NoError(t, err)

	kpm := testutils.NewKeyProcessorMock(t)
	kpm.ExportPublicKeyBinaryMock.Return([]byte{1}, nil)

	cm.Inject(svcNw, nk, certManager,
		keystore.NewInplaceKeyStore(skey),
		&platformpolicy.NodeCryptographyService{},
		beat.NewAccessorMock(t),
		/* testutils.NewTerminationHandlerMock(t), */
		chorus.NewConductorMock(t),
		&PublisherMock{}, &stater{},
		testutils.NewPlatformCryptographyScheme(), kpm)
	err = svcNw.Init(ctx)
	require.NoError(t, err)
	err = svcNw.Start(ctx)
	require.NoError(t, err)
}

type publisherMock struct {
	Error error
}

func (pm *publisherMock) Publish(topic string, messages ...*message.Message) error { return pm.Error }
func (pm *publisherMock) Close() error                                             { return nil }

func TestServiceNetwork_processIncoming(t *testing.T) {
	instestlogger.SetTestOutputWithErrorFilter(t, func(s string) bool {
		expectedError := strings.Contains(s, "error while deserialize msg from buffer") ||
			strings.Contains(s, "error while publish msg to TopicIncoming")
		return !expectedError
	})

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
