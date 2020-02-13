// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package api

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/reply"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/builtin/foundation"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
)

const CallUrl = "http://localhost:19192/api/rpc"

func TestTimeoutSuite(t *testing.T) {

	ctx, _ := inslogger.WithTraceField(context.Background(), "APItests")
	mc := minimock.NewController(t)
	defer mc.Wait(17 * time.Second)
	defer mc.Finish()

	ks := platformpolicy.NewKeyProcessor()
	sKey, err := ks.GeneratePrivateKey()
	require.NoError(t, err)
	sKeyString, err := ks.ExportPrivateKeyPEM(sKey)
	require.NoError(t, err)
	pKey := ks.ExtractPublicKey(sKey)
	pKeyString, err := ks.ExportPublicKeyPEM(pKey)
	require.NoError(t, err)

	userRef := gen.Reference().String()
	user, err := requester.CreateUserConfig(userRef, string(sKeyString), string(pKeyString))

	cr := testutils.NewContractRequesterMock(mc)
	cr.CallMock.Set(func(p context.Context, p1 *insolar.Reference, method string, p3 []interface{}, p4 insolar.PulseNumber) (insolar.Reply, *insolar.Reference, error) {
		requestReference, _ := insolar.NewReferenceFromString("insolar:1MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI")
		switch method {
		case "Call":
			var result = "OK"
			data, _ := foundation.MarshalMethodResult(result, nil)
			return &reply.CallMethod{
				Result: data,
			}, requestReference, nil
		default:
			return nil, nil, errors.New("Unknown method: " + method)
		}
	})

	checker := testutils.NewAvailabilityCheckerMock(mc)
	checker.IsAvailableMock.Return(true)

	http.DefaultServeMux = new(http.ServeMux)
	cfg := configuration.NewAPIRunner(false)
	cfg.Address = "localhost:19192"
	cfg.SwaggerPath = "spec/api-exported.yaml"
	api, err := NewRunner(
		&cfg,
		nil,
		cr,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		checker,
	)
	require.NoError(t, err)
	seed, err := api.SeedGenerator.Next()
	require.NoError(t, err)

	api.SeedManager.Add(*seed, 0)

	seedString := base64.StdEncoding.EncodeToString(seed[:])

	requester.SetTimeout(25)
	req, err := requester.MakeRequestWithSeed(
		ctx,
		CallUrl,
		user,
		&requester.Params{CallSite: "member.create", CallParams: map[string]interface{}{}, PublicKey: user.PublicKey},
		seedString,
	)
	require.NoError(t, err, "make request with seed error")

	rr := httptest.NewRecorder()
	api.Handler().ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "got StatusOK http code")

	var result requester.ContractResponse
	// fmt.Println("response:", rr.Body.String())
	err = json.Unmarshal(rr.Body.Bytes(), &result)
	require.NoError(t, err, "json unmarshal error")
	require.Nil(t, result.Error, "error should be nil in result")
	require.Equal(t, "OK", result.Result.CallResult, "call result is OK")
}

func TestDigestParser(t *testing.T) {
	invalidDigest := ""
	_, err := parseDigest(invalidDigest)
	require.Error(t, err)

	validDigest := "SHA-256=foo"
	_, err = parseDigest(validDigest)
	require.NoError(t, err)
}

func TestSignatureParser(t *testing.T) {
	invalidSignature := ""
	_, err := parseSignature(invalidSignature)

	validSignature := `keyId="member-pub-key", algorithm="ecdsa", headers="digest", signature=bar`
	_, err = parseSignature(validSignature)
	require.NoError(t, err)
}

func TestValidateRequestHeaders(t *testing.T) {
	body := []byte("foobar")
	h := sha256.New()
	_, err := h.Write(body)
	require.NoError(t, err)

	digest := h.Sum(nil)
	calculatedDigest := `SHA-256=` + base64.URLEncoding.EncodeToString(digest)
	signature := `keyId="member-pub-key", algorithm="ecdsa", headers="digest", signature=bar`
	sig, err := validateRequestHeaders(calculatedDigest, signature, body)
	require.NoError(t, err)
	require.Equal(t, "bar", sig)
}
