// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodeset

import (
	"crypto"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/reference"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestNewNodeNetwork(t *testing.T) {
	instestlogger.SetTestOutput(t)

	cfg := configuration.Transport{Address: "invalid"}
	certMock := testutils.NewCertificateMock(t)
	certMock.GetRoleMock.Set(func() member.PrimaryRole { return member.PrimaryRoleLightMaterial })
	certMock.GetPublicKeyMock.Set(func() crypto.PublicKey { return nil })
	certMock.GetNodeRefMock.Set(func() reference.Global { ref := gen.UniqueGlobalRef(); return ref })
	certMock.GetDiscoveryNodesMock.Set(func() []nodeinfo.DiscoveryNode { return nil })
	_, err := NewNodeNetwork(cfg, certMock)
	assert.Error(t, err)
	cfg.Address = "127.0.0.1:3355"
	_, err = NewNodeNetwork(cfg, certMock)
	assert.NoError(t, err)
}

func newNodeKeeper(t *testing.T, service cryptography.Service) network.NodeKeeper {
	cfg := configuration.Transport{Address: "127.0.0.1:3355"}
	certMock := testutils.NewCertificateMock(t)
	keyProcessor := platformpolicy.NewKeyProcessor()
	secret, err := keyProcessor.GeneratePrivateKey()
	require.NoError(t, err)
	pk := keyProcessor.ExtractPublicKey(secret)
	if service == nil {
		service = platformpolicy.NewKeyBoundCryptographyService(secret)
	}
	require.NoError(t, err)
	certMock.GetRoleMock.Set(func() member.PrimaryRole { return member.PrimaryRoleVirtual })
	certMock.GetPublicKeyMock.Set(func() crypto.PublicKey { return pk })
	certMock.GetNodeRefMock.Set(func() reference.Global { ref := gen.UniqueGlobalRef(); return ref })
	certMock.GetDiscoveryNodesMock.Set(func() []nodeinfo.DiscoveryNode { return nil })
	nw, err := NewNodeNetwork(cfg, certMock)
	require.NoError(t, err)
	nw.(*nodekeeper).snapshotStorage = NewMemoryStorage()
	return nw.(network.NodeKeeper)
}
