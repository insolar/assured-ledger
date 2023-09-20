package bootstrap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/rms/legacyhost"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func createCryptographyService(t *testing.T) cryptography.Service {
	keyProcessor := platformpolicy.NewKeyProcessor()
	privateKey, err := keyProcessor.GeneratePrivateKey()
	require.NoError(t, err)
	return platformpolicy.NewKeyBoundCryptographyService(privateKey)
}

func TestCreateAndVerifyPermit(t *testing.T) {
	origin, err := legacyhost.NewHostN("127.0.0.1:123", gen.UniqueGlobalRef())
	assert.NoError(t, err)
	redirect, err := legacyhost.NewHostN("127.0.0.1:321", gen.UniqueGlobalRef())
	assert.NoError(t, err)

	cryptographyService := createCryptographyService(t)

	permit, err := CreatePermit(origin.NodeID, redirect, []byte{}, cryptographyService)
	assert.NoError(t, err)
	assert.NotNil(t, permit)

	cert := testutils.NewCertificateMock(t)
	cert.GetDiscoveryNodesMock.Set(func() (r []nodeinfo.DiscoveryNode) {
		pk, _ := cryptographyService.GetPublicKey()
		node := mandates.NewBootstrapNode(pk, "", origin.Address.String(), origin.NodeID.String(), member.PrimaryRoleVirtual.String())
		return []nodeinfo.DiscoveryNode{node}
	})

	// validate
	err = ValidatePermit(permit, cert, createCryptographyService(t))
	assert.NoError(t, err)

	assert.Less(t, time.Now().Unix(), permit.Payload.ExpireTimestamp)
}
