// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bootstrap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography/platformpolicy"
	node2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/hostnetwork/host"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
)

func createCryptographyService(t *testing.T) cryptography.Service {
	keyProcessor := platformpolicy.NewKeyProcessor()
	privateKey, err := keyProcessor.GeneratePrivateKey()
	require.NoError(t, err)
	return platformpolicy.NewKeyBoundCryptographyService(privateKey)
}

func TestCreateAndVerifyPermit(t *testing.T) {
	origin, err := host.NewHostN("127.0.0.1:123", gen.UniqueReference())
	assert.NoError(t, err)
	redirect, err := host.NewHostN("127.0.0.1:321", gen.UniqueReference())
	assert.NoError(t, err)

	cryptographyService := createCryptographyService(t)

	permit, err := CreatePermit(origin.NodeID, redirect, []byte{}, cryptographyService)
	assert.NoError(t, err)
	assert.NotNil(t, permit)

	cert := testutils.NewCertificateMock(t)
	cert.GetDiscoveryNodesMock.Set(func() (r []node2.DiscoveryNode) {
		pk, _ := cryptographyService.GetPublicKey()
		node := mandates.NewBootstrapNode(pk, "", origin.Address.String(), origin.NodeID.String(), node2.StaticRoleVirtual.String())
		return []node2.DiscoveryNode{node}
	})

	// validate
	err = ValidatePermit(permit, cert, createCryptographyService(t))
	assert.NoError(t, err)

	assert.Less(t, time.Now().Unix(), permit.Payload.ExpireTimestamp)
}
