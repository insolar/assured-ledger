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
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat/memstor"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/controller"
	"github.com/insolar/assured-ledger/ledger-core/reference"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func prepareNetwork(t *testing.T, cfg configuration.Configuration) (*ServiceNetwork, reference.Global) {
	serviceNetwork, err := NewServiceNetwork(cfg, component.NewManager(nil))
	require.NoError(t, err)

	nodeKeeper := beat.NewNodeKeeperMock(t)
	ref := gen.UniqueGlobalRef()
	nodeKeeper.GetLocalNodeReferenceMock.Return(ref)
	serviceNetwork.NodeKeeper = nodeKeeper

	return serviceNetwork, ref
}

func TestSendMessageHandler_ReceiverNotSet(t *testing.T) {
	instestlogger.SetTestOutputWithErrorFilter(t, func(s string) bool {
		if strings.Contains(s, "failed to send message: Receiver in message metadata is not set") {
			return false
		}
		return true
	})

	serviceNetwork, _ := prepareNetwork(t, configuration.NewConfiguration())

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
	svcNw, nodeRef := prepareNetwork(t, configuration.NewConfiguration())
	svcNw.router = svcNw.router.ReplacePublisher(&publisherMock{})

	pulseMock := beat.NewAppenderMock(t)
	pulseMock.LatestTimeBeatMock.Return(pulsestor.GenesisPulse, nil)

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
	svcNw, _ := prepareNetwork(t, configuration.NewConfiguration())
	svcNw.router = svcNw.router.ReplacePublisher(&publisherMock{})

	rpc := controller.NewRPCControllerMock(t)
	rpc.SendBytesMock.Set(func(p context.Context, p1 reference.Global, p2 string, p3 []byte) (r []byte, r1 error) {
		return nil, throw.New("test error")
	})
	pulseMock := beat.NewAppenderMock(t)
	pulseMock.LatestTimeBeatMock.Return(pulsestor.GenesisPulse, nil)
	svcNw.RPC = rpc

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
	svcNw, _ := prepareNetwork(t, configuration.NewConfiguration())
	svcNw.router = svcNw.router.ReplacePublisher(&publisherMock{})

	rpc := controller.NewRPCControllerMock(t)
	rpc.SendBytesMock.Set(func(p context.Context, p1 reference.Global, p2 string, p3 []byte) (r []byte, r1 error) {
		return nil, nil
	})
	pulseMock := beat.NewAppenderMock(t)
	pulseMock.LatestTimeBeatMock.Return(pulsestor.GenesisPulse, nil)
	svcNw.RPC = rpc

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
	svcNw, _ := prepareNetwork(t, configuration.NewConfiguration())
	svcNw.router = svcNw.router.ReplacePublisher(&publisherMock{})

	rpc := controller.NewRPCControllerMock(t)
	rpc.SendBytesMock.Set(func(p context.Context, p1 reference.Global, p2 string, p3 []byte) (r []byte, r1 error) {
		return ack, nil
	})
	pulseMock := beat.NewAppenderMock(t)
	pulseMock.LatestTimeBeatMock.Return(pulsestor.GenesisPulse, nil)
	svcNw.RPC = rpc

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

	originRef := gen.UniqueGlobalRef()
	nk := memstor.NewNodeKeeper(originRef, member.PrimaryRoleUnknown)

	cert := &mandates.Certificate{}
	cert.Reference = originRef.String()
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
		beat.NewAppenderMock(t),
		/* testutils.NewTerminationHandlerMock(t), */
		chorus.NewConductorMock(t),
		&publisherMock{}, &stater{},
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
	serviceNetwork.router = serviceNetwork.router.ReplacePublisher(pub)
	ctx := context.Background()
	_, err = serviceNetwork.processIncoming(ctx, []byte("ololo"))
	assert.Error(t, err)
	msg := message.NewMessage("1", nil)
	data, err := serializeMessage(msg)
	require.NoError(t, err)
	_, err = serviceNetwork.processIncoming(ctx, data)
	assert.NoError(t, err)
	pub.Error = throw.New("Failed to publish message")
	_, err = serviceNetwork.processIncoming(ctx, data)
	assert.Error(t, err)
}
