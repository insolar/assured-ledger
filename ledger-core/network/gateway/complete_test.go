// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	mock "github.com/insolar/assured-ledger/ledger-core/testutils/network"
)

func mockCryptographyService(t *testing.T, ok bool) cryptography.Service {
	cs := cryptography.NewServiceMock(t)
	cs.SignMock.Set(func(data []byte) (*cryptography.Signature, error) {
		if ok {
			sig := cryptography.SignatureFromBytes([]byte("test_sig"))
			return &sig, nil
		}
		return nil, errors.New("test_error")
	})
	return cs
}

func mockCertificateManager(t *testing.T, certNodeRef reference.Global, discoveryNodeRef reference.Global, unsignCertOk bool) *testutils.CertificateManagerMock {
	cm := testutils.NewCertificateManagerMock(t)
	cm.GetCertificateMock.Set(func() nodeinfo.Certificate {
		return &mandates.Certificate{
			AuthorizationCertificate: mandates.AuthorizationCertificate{
				PublicKey: "test_public_key",
				Reference: certNodeRef.String(),
				Role:      "virtual",
			},
			MajorityRule:     0,
			PulsarPublicKeys: []string{},
			BootstrapNodes: []mandates.BootstrapNode{
				{
					NodeRef:     discoveryNodeRef.String(),
					PublicKey:   "test_discovery_public_key",
					Host:        "test_discovery_host",
					NetworkSign: []byte("test_network_sign"),
				},
			},
		}
	})
	return cm
}

func mockReply(t *testing.T) []byte {
	res := struct {
		PublicKey string
		Role      member.PrimaryRole
	}{
		PublicKey: "test_node_public_key",
		Role:      member.PrimaryRoleVirtual,
	}
	node, err := foundation.MarshalMethodResult(res, nil)
	require.NoError(t, err)
	return node
}

func mockPulseManager(t *testing.T) chorus.Conductor {
	pm := chorus.NewConductorMock(t)
	return pm
}

func TestComplete_GetCert(t *testing.T) {
	t.Skip("fixme")

	nodeRef := gen.UniqueGlobalRef()
	certNodeRef := gen.UniqueGlobalRef()

	gatewayer := mock.NewGatewayerMock(t)
	nodekeeper := beat.NewNodeKeeperMock(t)
	hn := mock.NewHostNetworkMock(t)

	cm := mockCertificateManager(t, certNodeRef, certNodeRef, true)
	cs := mockCryptographyService(t, true)
	pm := mockPulseManager(t)
	pa := beat.NewAppenderMock(t)

	var ge network.Gateway
	ge = newNoNetwork(&Base{
		Gatewayer:           gatewayer,
		NodeKeeper:          nodekeeper,
		HostNetwork:         hn,
		CertificateManager:  cm,
		CryptographyService: cs,
		PulseManager:        pm,
	})
	ge = ge.NewGateway(context.Background(), network.CompleteNetworkState)
	ctx := context.Background()

	pa.LatestTimeBeatMock.Return(pulsestor.GenesisPulse, nil)

	result, err := ge.Auther().GetCert(ctx, nodeRef)
	require.NoError(t, err)

	cert := result.(*mandates.Certificate)
	assert.Equal(t, "test_node_public_key", cert.PublicKey)
	assert.Equal(t, nodeRef.String(), cert.Reference)
	assert.Equal(t, "virtual", cert.Role)
	assert.Equal(t, 0, cert.MajorityRule)
	assert.Equal(t, uint(0), cert.MinRoles.Virtual)
	assert.Equal(t, uint(0), cert.MinRoles.HeavyMaterial)
	assert.Equal(t, uint(0), cert.MinRoles.LightMaterial)
	assert.Equal(t, []string{}, cert.PulsarPublicKeys)
	assert.Equal(t, 1, len(cert.BootstrapNodes))
	assert.Equal(t, "test_discovery_public_key", cert.BootstrapNodes[0].PublicKey)
	assert.Equal(t, []byte("test_network_sign"), cert.BootstrapNodes[0].NetworkSign)
	assert.Equal(t, "test_discovery_host", cert.BootstrapNodes[0].Host)
	assert.Equal(t, []byte("test_sig"), cert.BootstrapNodes[0].NodeSign)
	assert.Equal(t, certNodeRef.String(), cert.BootstrapNodes[0].NodeRef)
}

func TestComplete_handler(t *testing.T) {
	t.Skip("fixme")
	nodeRef := gen.UniqueGlobalRef()
	certNodeRef := gen.UniqueGlobalRef()

	gatewayer := mock.NewGatewayerMock(t)
	nodekeeper := beat.NewNodeKeeperMock(t)

	cm := mockCertificateManager(t, certNodeRef, certNodeRef, true)
	cs := mockCryptographyService(t, true)
	pm := mockPulseManager(t)
	pa := beat.NewAppenderMock(t)

	hn := mock.NewHostNetworkMock(t)

	var ge network.Gateway
	ge = newNoNetwork(&Base{
		Gatewayer:           gatewayer,
		NodeKeeper:          nodekeeper,
		HostNetwork:         hn,
		CertificateManager:  cm,
		CryptographyService: cs,
		PulseManager:        pm,
	})

	ge = ge.NewGateway(context.Background(), network.CompleteNetworkState)
	ctx := context.Background()
	pa.LatestTimeBeatMock.Return(pulsestor.GenesisPulse, nil)

	p := packet.NewReceivedPacket(packet.NewPacket(nwapi.Address{}, nwapi.Address{}, types.SignCert, 1), nil)
	p.SetRequest(&rms.SignCertRequest{NodeRef: rms.NewReference(nodeRef)})

	hn.BuildResponseMock.Set(func(ctx context.Context, request network.Packet, responseData interface{}) (p1 network.Packet) {
		r := packet.NewPacket(nwapi.Address{}, nwapi.Address{}, types.SignCert, 1)
		r.SetResponse(&rms.SignCertResponse{Sign: []byte("test_sig")})
		return r
	})
	result, err := ge.(*Complete).signCertHandler(ctx, p)

	require.NoError(t, err)
	require.Equal(t, []byte("test_sig"), result.GetResponse().GetSignCert().Sign)
}
