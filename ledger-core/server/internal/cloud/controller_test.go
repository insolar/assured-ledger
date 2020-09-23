// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build slowtest

package cloud_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/server"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
)

func TestController_PartialDistribute(t *testing.T) {
	var (
		numVirtual        = 10
		numLightMaterials = 0
		numHeavyMaterials = 0
	)

	cloudSettings := launchnet.CloudSettings{Virtual: numVirtual, Light: numLightMaterials, Heavy: numHeavyMaterials}

	appConfigs, cloudBaseConf, certFactory, keyFactory := launchnet.PrepareCloudConfiguration(cloudSettings)
	baseConfig := configuration.Configuration{}
	baseConfig.Log = cloudBaseConf.Log

	confProvider := &server.CloudConfigurationProvider{
		PulsarConfig:       cloudBaseConf.PulsarConfiguration,
		BaseConfig:         baseConfig,
		CertificateFactory: certFactory,
		KeyFactory:         keyFactory,
		GetAppConfigs: func() []configuration.Configuration {
			return appConfigs
		},
	}

	s, pulseDistributor := server.NewMultiServerWithoutPulsar(confProvider)
	go func() {
		s.Serve()
	}()

	// wait for starting all components
	for !s.(*insapp.Server).Started() {
		time.Sleep(time.Millisecond)
	}

	defer s.(*insapp.Server).Stop()

	allNodes := make(map[reference.Global]struct{})
	for i := 0; i < len(appConfigs); i++ {
		cert, err := certFactory(nil, nil, appConfigs[i].CertificatePath)
		require.NoError(t, err)
		allNodes[cert.GetCertificate().GetNodeRef()] = struct{}{}
	}
	pulseGenerator := testutils.NewPulseGenerator(uint16(confProvider.PulsarConfig.Pulsar.NumberDelta), nil, nil)

	{ // Change pulse on all nodes
		for i := 0; i < 3; i++ {
			_ = pulseGenerator.Generate()
			packet := pulseGenerator.GetLastPulsePacket()
			pulseDistributor.PartialDistribute(context.Background(), packet, allNodes)

			for _, conf := range appConfigs {

				body := getRPSResponseBody(t, getURL(conf.AdminAPIRunner.Address), makeStatusBody())

				response := rpcStatusResponse{}

				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				require.Equal(t, packet.PulseNumber, pulse.Number(response.Result.PulseNumber))
			}
		}
	}

	halfOfNodes := make(map[reference.Global]struct{})
	numElements := len(allNodes) / 2
	for ref, _ := range allNodes {
		if numElements <= 0 {
			break
		}
		numElements--
		halfOfNodes[ref] = struct{}{}
	}

	{ // Partial pulse change
		prevPacket := pulseGenerator.GetLastPulsePacket()
		for i := 0; i < 3; i++ {
			_ = pulseGenerator.Generate()
			packet := pulseGenerator.GetLastPulsePacket()

			pulseDistributor.PartialDistribute(context.Background(), packet, halfOfNodes)
			for _, conf := range appConfigs {
				body := getRPSResponseBody(t, getURL(conf.AdminAPIRunner.Address), makeStatusBody())

				response := rpcStatusResponse{}

				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				cert, err := certFactory(nil, nil, conf.CertificatePath)
				require.NoError(t, err)
				if _, ok := halfOfNodes[cert.GetCertificate().GetNodeRef()]; ok {
					require.Equal(t, packet.PulseNumber, pulse.Number(response.Result.PulseNumber))
				} else {
					require.Equal(t, prevPacket.PulseNumber, pulse.Number(response.Result.PulseNumber))
				}
			}
		}
	}

}

func getURL(address string) string {
	urlTmpl := "http://%s%s"
	return fmt.Sprintf(urlTmpl, address, launchnet.TestAdminRPCUrl)
}

func makeStatusBody() map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "node.getStatus",
		"id":      "1",
	}
}

type rpcStatusResponse struct {
	Result requester.StatusResponse `json:"result"`
}

func getRPSResponseBody(t testing.TB, URL string, postParams map[string]interface{}) []byte {
	jsonValue, _ := json.Marshal(postParams)

	postResp, err := http.Post(URL, "application/json", bytes.NewBuffer(jsonValue))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, postResp.StatusCode)
	body, err := ioutil.ReadAll(postResp.Body)
	require.NoError(t, err)
	return body
}
