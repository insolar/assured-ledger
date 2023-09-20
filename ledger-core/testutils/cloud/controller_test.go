// +build slowtest,!longtest

package cloud_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/server"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/cloud"
)

func TestController_PartialDistribute(t *testing.T) {
	instestlogger.SetTestOutput(t)
	var (
		numVirtual        = uint(10)
		numLightMaterials = uint(0)
		numHeavyMaterials = uint(0)
	)

	cloudSettings := cloud.Settings{Running: cloud.NodeConfiguration{
		Virtual: numVirtual, LightMaterial: numLightMaterials, HeavyMaterial: numHeavyMaterials,
	},
		Prepared: cloud.NodeConfiguration{
			Virtual: numVirtual, LightMaterial: numLightMaterials, HeavyMaterial: numHeavyMaterials,
		},
		MinRoles: cloud.NodeConfiguration{
			Virtual: numVirtual, LightMaterial: numLightMaterials, HeavyMaterial: numHeavyMaterials,
		},
		MajorityRule: 10}

	confProvider := cloud.NewConfigurationProvider(cloudSettings)

	appConfigs := confProvider.GetAppConfigs()

	controller := cloud.NewController()
	s, _ := server.NewControlledMultiServer(controller, confProvider)
	go func() {
		s.Serve()
	}()

	s.WaitStarted()

	defer s.Stop()

	allNodesMap := confProvider.GetRunningNodesMap()

	allNodes := make(map[reference.Global]struct{}, len(allNodesMap))
	for nodeRef := range allNodesMap {
		allNodes[nodeRef] = struct{}{}
	}
	pulseGenerator := testutils.NewPulseGenerator(uint16(confProvider.PulsarConfig.Pulsar.NumberDelta), nil, nil)

	{ // Change pulse on all nodes
		for i := 0; i < 3; i++ {
			_ = pulseGenerator.Generate()
			packet := pulseGenerator.GetLastPulsePacket()
			controller.PartialDistribute(context.Background(), packet, allNodes)

			for _, conf := range appConfigs {

				body := getRPSResponseBody(t, getURL(conf.AdminAPIRunner.Address), makeStatusBody())

				response := rpcStatusResponse{}

				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				require.Equal(t, packet.PulseNumber, pulse.Number(response.Result.PulseNumber))
			}
		}
	}

	numElements := len(allNodes) / 2
	halfOfNodes := make(map[reference.Global]struct{}, numElements)
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

			controller.PartialDistribute(context.Background(), packet, halfOfNodes)
			for _, conf := range appConfigs {
				body := getRPSResponseBody(t, getURL(conf.AdminAPIRunner.Address), makeStatusBody())

				response := rpcStatusResponse{}

				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				// for all nodes CertificatePath == node reference string
				nodeRef, err := reference.GlobalFromString(conf.CertificatePath)
				require.NoError(t, err)
				if _, ok := halfOfNodes[nodeRef]; ok {
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
