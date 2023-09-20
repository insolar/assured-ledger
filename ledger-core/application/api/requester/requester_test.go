package requester

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"

	"github.com/stretchr/testify/require"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
)

const TESTREFERENCE = "insolar:1MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI"
const TESTSEED = "VGVzdA=="

var testSeedResponse = seedResponse{Seed: "Test", TraceID: "testTraceID"}
var testInfoResponse = InfoResponse{RootMember: "root_member_ref", RootDomain: "root_domain_ref", NodeDomain: "node_domain_ref"}
var testStatusResponse = StatusResponse{NetworkState: "OK"}

func writeReponse(response http.ResponseWriter, answer interface{}) {
	serJSON, err := json.MarshalIndent(answer, "", "    ")
	if err != nil {
		global.Errorf("Can't serialize response\n")
	}
	var newLine byte = '\n'
	_, err = response.Write(append(serJSON, newLine))
	if err != nil {
		global.Errorf("Can't write response\n")
	}
}

type RPCResponse struct {
	Response
	Result interface{} `json:"result,omitempty"`
}

func FakeRPCHandler(response http.ResponseWriter, req *http.Request) {
	response.Header().Add("Content-Type", "application/json")
	rpcResponse := RPCResponse{}
	request := Request{}
	_, err := unmarshalRequest(req, &request)
	if err != nil {
		global.Errorf("Can't read request\n")
		return
	}

	switch request.Method {
	case "node.getStatus":
		rpcResponse.Result = testStatusResponse
	case "network.getInfo":
		rpcResponse.Result = testInfoResponse
	case "node.getSeed":
		rpcResponse.Result = testSeedResponse
	case "contract.call":
		rpcResponse.Result = TESTREFERENCE
	default:
		rpcResponse.Result = TESTSEED

	}
	writeReponse(response, rpcResponse)
}

const rpcLOCATION = "/admin-api/rpc"
const PORT = "12221"
const HOST = "127.0.0.1"
const URL = "http://" + HOST + ":" + PORT + rpcLOCATION

var server = &http.Server{Addr: ":" + PORT}

func waitForStart() error {
	numAttempts := 5

	for ; numAttempts > 0; numAttempts-- {
		conn, _ := net.DialTimeout("tcp", net.JoinHostPort(HOST, PORT), time.Millisecond*50)
		if conn != nil {
			conn.Close()
			break
		}
	}
	if numAttempts == 0 {
		return errors.New("Problem with launching test api: couldn't wait more")
	}

	return nil
}

func startServer() error {
	server := &http.Server{}
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12221})
	if err != nil {
		return errors.W(err, "error creating listener")
	}
	go server.Serve(listener)

	return nil
}

func setup() (teardownFn func(pass bool), err error) {

	teardownFn = instestlogger.SetTestOutputWithStub()

	fRPCh := FakeRPCHandler
	http.HandleFunc(rpcLOCATION, fRPCh)
	global.Info("Starting Test api server ...")

	err = startServer()
	if err != nil {
		global.Error("Problem with starting test server: ", err)
		return
	}

	err = waitForStart()
	if err != nil {
		global.Error("Can't start api: ", err)
		return
	}

	return
}

func teardown() {
	const timeOut = 2
	global.Infof("Shutting down test server gracefully ...(waiting for %d seconds)", timeOut)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeOut)*time.Second)
	defer cancel()
	err := server.Shutdown(ctx)
	if err != nil {
		fmt.Println("STOPPING TEST SERVER:", err)
	}
}

func testMainWrapper(m *testing.M) int {
	teardownFn, err := setup()
	code := 1 // error
	defer func() {
		defer teardownFn(code == 0)
		teardown()
	} ()

	if err == nil {
		code = m.Run()
	} else {
		fmt.Println("error while setup, skip tests: ", err)
	}

	return code
}

func TestMain(m *testing.M) {
	os.Exit(testMainWrapper(m))
}

func TestGetSeed(t *testing.T) {
	instestlogger.SetTestOutput(t)

	seed, err := GetSeed(URL)
	require.NoError(t, err)
	require.Equal(t, "Test", seed)
}

func TestGetResponseBodyEmpty(t *testing.T) {
	instestlogger.SetTestOutput(t)

	_, err := GetResponseBodyPlatform("test", "", nil)
	require.EqualError(t, err, "problem with sending request;\tPost \"test\": unsupported protocol scheme \"\"")
}

func TestGetResponseBodyBadHttpStatus(t *testing.T) {
	instestlogger.SetTestOutput(t)

	_, err := GetResponseBodyPlatform(URL+"TEST", "", nil)
	require.EqualError(t, err, "bad http response code: 404")
}

func TestGetResponseBody(t *testing.T) {
	instestlogger.SetTestOutput(t)

	data, err := GetResponseBodyContract(URL, ContractRequest{}, "")
	response := RPCResponse{}
	_ = json.Unmarshal(data, &response)
	require.NoError(t, err)
	require.Contains(t, response.Result, TESTSEED)
}

func TestSetVerbose(t *testing.T) {
	instestlogger.SetTestOutput(t)

	require.False(t, verbose)
	SetVerbose(true)
	require.True(t, verbose)
	// restore original value for future tests, if -count 10 flag is used
	SetVerbose(false)
}

func readConfigs(t *testing.T) (*UserConfigJSON, *Params) {
	userConf, err := ReadUserConfigFromFile("testdata/userConfig.json")
	require.NoError(t, err)
	reqConf, err := ReadRequestParamsFromFile("testdata/requestConfig.json")
	require.NoError(t, err)

	return userConf, reqConf
}

func TestSend(t *testing.T) {
	instestlogger.SetTestOutput(t)

	ctx := inslogger.ContextWithTrace(context.Background(), "TestSend")
	userConf, reqParams := readConfigs(t)
	reqParams.CallSite = "member.create"
	resp, err := Send(ctx, URL, userConf, reqParams)
	require.NoError(t, err)
	require.Contains(t, string(resp), TESTREFERENCE)
}

func TestSendWithSeed(t *testing.T) {
	instestlogger.SetTestOutput(t)

	ctx := inslogger.ContextWithTrace(context.Background(), "TestSendWithSeed")
	userConf, reqParams := readConfigs(t)
	reqParams.CallSite = "member.create"
	resp, err := SendWithSeed(ctx, URL, userConf, reqParams, TESTSEED)
	require.NoError(t, err)
	require.Contains(t, string(resp), TESTREFERENCE)
}

func TestSendWithSeed_WithBadUrl(t *testing.T) {
	instestlogger.SetTestOutput(t)

	ctx := inslogger.ContextWithTrace(context.Background(), "TestSendWithSeed_WithBadUrl")
	userConf, reqConf := readConfigs(t)
	_, err := SendWithSeed(ctx, URL+"TTT", userConf, reqConf, TESTSEED)
	require.EqualError(t, err, "[ SendWithSeed ] Problem with sending target request;\tbad http response code: 404")
}

func TestSendWithSeed_NilConfigs(t *testing.T) {
	instestlogger.SetTestOutput(t)

	ctx := inslogger.ContextWithTrace(context.Background(), "TestSendWithSeed_NilConfigs")
	_, err := SendWithSeed(ctx, URL, nil, nil, TESTSEED)
	require.EqualError(t, err, "[ SendWithSeed ] Problem with creating target request;\tconfigs must be initialized")
}

func TestInfo(t *testing.T) {
	instestlogger.SetTestOutput(t)

	resp, err := Info(URL)
	require.NoError(t, err)
	require.Equal(t, resp, &testInfoResponse)
}

func TestStatus(t *testing.T) {
	instestlogger.SetTestOutput(t)

	resp, err := Status(URL)
	require.NoError(t, err)
	require.Equal(t, resp, &testStatusResponse)
}

func TestMarshalSig(t *testing.T) {
	instestlogger.SetTestOutput(t)

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	msg := "test"
	hash := sha256.Sum256([]byte(msg))

	r1, s1, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	require.NoError(t, err)
	derString, err := marshalSig(r1, s1)
	require.NoError(t, err)

	sig, err := base64.StdEncoding.DecodeString(derString)

	r2, s2, err := foundation.UnmarshalSig(sig)
	require.NoError(t, err)

	require.Equal(t, r1, r2, errors.New("Invalid S number"))
	require.Equal(t, s1, s2, errors.New("Invalid R number"))
}

// unmarshalRequest unmarshals request to api
func unmarshalRequest(req *http.Request, params interface{}) ([]byte, error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, errors.W(err, "[ unmarshalRequest ] Can't read body. So strange")
	}
	if len(body) == 0 {
		return nil, errors.New("[ unmarshalRequest ] Empty body")
	}

	err = json.Unmarshal(body, &params)
	if err != nil {
		return body, errors.W(err, "[ unmarshalRequest ] Can't unmarshal input params")
	}
	return body, nil
}
