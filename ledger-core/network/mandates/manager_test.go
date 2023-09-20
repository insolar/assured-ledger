package mandates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestNewManagerReadCertificate(t *testing.T) {
	cs, _ := platformpolicy.NewStorageBoundCryptographyService(TestKeys)
	kp := platformpolicy.NewKeyProcessor()
	pk, _ := cs.GetPublicKey()

	certManager, err := NewManagerReadCertificate(pk, kp, TestCert)
	assert.NoError(t, err)
	require.NotNil(t, certManager)
	cert := certManager.GetCertificate()
	require.NotNil(t, cert)
}

func newDiscovery() (*BootstrapNode, cryptography.Service) {
	kp := platformpolicy.NewKeyProcessor()
	key, _ := kp.GeneratePrivateKey()
	cs := platformpolicy.NewKeyBoundCryptographyService(key)
	pk, _ := cs.GetPublicKey()
	pubKeyBuf, _ := kp.ExportPublicKeyPEM(pk)
	ref := gen.UniqueGlobalRef().String()
	n := NewBootstrapNode(pk, string(pubKeyBuf), " ", ref, member.PrimaryRoleVirtual.String())
	return n, cs
}

func TestSignAndVerifyCertificate(t *testing.T) {
	cs, _ := platformpolicy.NewStorageBoundCryptographyService(TestKeys)
	pubKey, err := cs.GetPublicKey()
	require.NoError(t, err)

	// init certificate
	proc := platformpolicy.NewKeyProcessor()
	publicKey, err := proc.ExportPublicKeyPEM(pubKey)
	require.NoError(t, err)

	cert := &Certificate{}
	cert.PublicKey = string(publicKey[:])
	cert.Reference = gen.UniqueGlobalRef().String()
	cert.Role = member.PrimaryRoleHeavyMaterial.String()
	cert.MinRoles.HeavyMaterial = 1
	cert.MinRoles.Virtual = 4

	discovery, discoveryCS := newDiscovery()
	sign, err := SignCert(discoveryCS, cert.PublicKey, cert.Role, cert.Reference)
	require.NoError(t, err)
	discovery.NodeSign = sign.Bytes()
	cert.BootstrapNodes = []BootstrapNode{*discovery}

	jsonCert, err := cert.Dump()
	require.NoError(t, err)

	cert2, err := ReadCertificateFromReader(pubKey, proc, strings.NewReader(jsonCert))
	require.NoError(t, err)

	otherDiscovery, otherDiscoveryCS := newDiscovery()

	valid, err := VerifyAuthorizationCertificate(otherDiscoveryCS, []nodeinfo.DiscoveryNode{discovery}, cert2)
	require.NoError(t, err)
	require.True(t, valid)

	// bad cases
	valid, err = VerifyAuthorizationCertificate(otherDiscoveryCS, []nodeinfo.DiscoveryNode{discovery, otherDiscovery}, cert2)
	require.NoError(t, err)
	require.False(t, valid)

	valid, err = VerifyAuthorizationCertificate(otherDiscoveryCS, []nodeinfo.DiscoveryNode{otherDiscovery}, cert2)
	require.NoError(t, err)
	require.False(t, valid)
}
