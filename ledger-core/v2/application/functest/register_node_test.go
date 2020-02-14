// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest

package functest

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/testutils/launchnet"
	"github.com/insolar/assured-ledger/ledger-core/v2/certificate"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
)

var scheme = platformpolicy.NewPlatformCryptographyScheme()
var keyProcessor = platformpolicy.NewKeyProcessor()

const TESTPUBLICKEY = "some_fancy_public_key"

func registerNodeSignedCall(t *testing.T, params map[string]interface{}) (string, error) {
	res, err := signedRequest(t, launchnet.TestRPCUrl, &launchnet.Root, "contract.registerNode", params)
	if err != nil {
		return "", err
	}
	return res.(string), nil
}

func TestRegisterNodeVirtual(t *testing.T) {
	const testRole = "virtual"
	ref, err := registerNodeSignedCall(t, map[string]interface{}{"publicKey": TESTPUBLICKEY, "role": testRole})
	require.NoError(t, err)

	require.NotNil(t, ref)
}

func TestRegisterNodeHeavyMaterial(t *testing.T) {
	const testRole = "heavy_material"
	ref, err := registerNodeSignedCall(t, map[string]interface{}{"publicKey": TESTPUBLICKEY, "role": testRole})
	require.NoError(t, err)

	require.NotNil(t, ref)
}

func TestRegisterNodeLightMaterial(t *testing.T) {
	const testRole = "light_material"
	ref, err := registerNodeSignedCall(t, map[string]interface{}{"publicKey": TESTPUBLICKEY, "role": testRole})
	require.NoError(t, err)

	require.NotNil(t, ref)
}

func TestRegisterNodeNotExistRole(t *testing.T) {
	_, err := signedRequestWithEmptyRequestRef(t, launchnet.TestRPCUrl, &launchnet.Root,
		"contract.registerNode", map[string]interface{}{"publicKey": TESTPUBLICKEY, "role": "some_not_fancy_role"})
	data := checkConvertRequesterError(t, err).Data
	require.Contains(t, data.Trace, "role is not supported")
}

func TestRegisterNodeByNoRoot(t *testing.T) {
	member := createMember(t)
	const testRole = "virtual"
	_, err := signedRequestWithEmptyRequestRef(t, launchnet.TestRPCUrl, member, "contract.registerNode",
		map[string]interface{}{"publicKey": TESTPUBLICKEY, "role": testRole})
	data := checkConvertRequesterError(t, err).Data
	require.Contains(t, data.Trace, "only root member can register node")
}

func TestReceiveNodeCert(t *testing.T) {
	const testRole = "virtual"
	ref, err := registerNodeSignedCall(t, map[string]interface{}{"publicKey": TESTPUBLICKEY, "role": testRole})
	require.NoError(t, err)

	body := getRPSResponseBody(t, launchnet.TestRPCUrl, postParams{
		"jsonrpc": "2.0",
		"method":  "cert.get",
		"id":      "",
		"params":  map[string]string{"ref": ref},
	})

	res := struct {
		Result struct {
			Cert certificate.Certificate
		}
	}{}

	err = json.Unmarshal(body, &res)
	require.NoError(t, err)

	networkPart := res.Result.Cert.SerializeNetworkPart()
	nodePart := res.Result.Cert.SerializeNodePart()

	for _, discoveryNode := range res.Result.Cert.BootstrapNodes {
		pKey, err := keyProcessor.ImportPublicKeyPEM([]byte(discoveryNode.PublicKey))
		require.NoError(t, err)

		t.Run("Verify network sign for "+discoveryNode.Host, func(t *testing.T) {
			verified := scheme.DataVerifier(pKey, scheme.IntegrityHasher()).Verify(insolar.SignatureFromBytes(discoveryNode.NetworkSign), networkPart)
			require.True(t, verified)
		})
		t.Run("Verify node sign for "+discoveryNode.Host, func(t *testing.T) {
			verified := scheme.DataVerifier(pKey, scheme.IntegrityHasher()).Verify(insolar.SignatureFromBytes(discoveryNode.NodeSign), nodePart)
			require.True(t, verified)
		})
	}
}
