//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package certificate

import (
	"strings"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManagerReadCertificate(t *testing.T) {
	cs, _ := cryptography.NewStorageBoundCryptographyService(TestKeys)
	kp := platformpolicy.NewKeyProcessor()
	pk, _ := cs.GetPublicKey()

	certManager, err := NewManagerReadCertificate(pk, kp, TestCert)
	assert.NoError(t, err)
	require.NotNil(t, certManager)
	cert := certManager.GetCertificate()
	require.NotNil(t, cert)
}

func newDiscovery() (*BootstrapNode, insolar.CryptographyService) {
	kp := platformpolicy.NewKeyProcessor()
	key, _ := kp.GeneratePrivateKey()
	cs := cryptography.NewKeyBoundCryptographyService(key)
	pk, _ := cs.GetPublicKey()
	pubKeyBuf, _ := kp.ExportPublicKeyPEM(pk)
	ref := gen.Reference().String()
	n := NewBootstrapNode(pk, string(pubKeyBuf), " ", ref, insolar.StaticRoleVirtual.String())
	return n, cs
}

func TestSignAndVerifyCertificate(t *testing.T) {
	cs, _ := cryptography.NewStorageBoundCryptographyService(TestKeys)
	pubKey, err := cs.GetPublicKey()
	require.NoError(t, err)

	// init certificate
	proc := platformpolicy.NewKeyProcessor()
	publicKey, err := proc.ExportPublicKeyPEM(pubKey)
	require.NoError(t, err)

	cert := &Certificate{}
	cert.PublicKey = string(publicKey[:])
	cert.Reference = gen.Reference().String()
	cert.Role = insolar.StaticRoleHeavyMaterial.String()
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

	valid, err := VerifyAuthorizationCertificate(otherDiscoveryCS, []insolar.DiscoveryNode{discovery}, cert2)
	require.NoError(t, err)
	require.True(t, valid)

	// bad cases
	valid, err = VerifyAuthorizationCertificate(otherDiscoveryCS, []insolar.DiscoveryNode{discovery, otherDiscovery}, cert2)
	require.NoError(t, err)
	require.False(t, valid)

	valid, err = VerifyAuthorizationCertificate(otherDiscoveryCS, []insolar.DiscoveryNode{otherDiscovery}, cert2)
	require.NoError(t, err)
	require.False(t, valid)
}
