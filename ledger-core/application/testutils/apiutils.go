// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"bytes"
	"context"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var (
	httpClient   *http.Client
	defaultPorts = []string{"32302", "32304"}
)

func GetPorts() []string {
	return defaultPorts
}

// SetAPIPorts is not thread safe, it is supposed to be called before bench run, right after launchnet is configured
func SetAPIPorts(ports []string) {
	defaultPorts = ports
}

// SetAPIPortsByAddresses is not thread safe, it is supposed to be called before bench run, right after launchnet is configured
func SetAPIPortsByAddresses(apiAddresses []string) {
	ports := make([]string, 0, len(apiAddresses))
	for _, addr := range apiAddresses {
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			panic(throw.W(err, "failed to split host:port"))
		}
		ports = append(ports, port)
	}
	defaultPorts = ports
}

const (
	requestTimeout = 30 * time.Second
	contentType    = "Content-Type"

	defaultHost = "127.0.0.1"
	walletPath  = "/wallet"

	WalletCreatePath     = walletPath + "/create"
	WalletGetBalancePath = walletPath + "/get_balance"
	WalletAddAmountPath  = walletPath + "/add_amount"
	WalletTransferPath   = walletPath + "/transfer"
	WalletDeletePath     = walletPath + "/delete"
)

func init() {
	rand.Seed(time.Now().Unix())
	httpClient = createHTTPClient()
}

// createHTTPClient for connection re-use
func createHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 1000,
		},
		Timeout: requestTimeout,
	}

	return client
}

func ResetHTTPClient() {
	httpClient.CloseIdleConnections()
}

// Creates http.Request with all necessary fields.
func prepareReq(ctx context.Context, url string, body interface{}) (*http.Request, error) {
	jsonValue, err := jsoniter.Marshal(body)
	if err != nil {
		return nil, throw.W(err, "problem with marshaling params")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, throw.W(err, "problem with creating request")
	}
	req.Header.Set(contentType, "application/json")

	return req, nil
}

// Executes http.Request and returns response body.
func doReq(req *http.Request) ([]byte, error) {
	postResp, err := httpClient.Do(req)
	if err != nil {
		return nil, throw.W(err, "problem with sending request")
	}

	if postResp == nil {
		return nil, throw.New("response is nil")
	}

	defer postResp.Body.Close()
	if http.StatusOK != postResp.StatusCode {
		return nil, throw.New("bad http response: " + postResp.Status)
	}

	body, err := ioutil.ReadAll(postResp.Body)
	if err != nil {
		return nil, throw.W(err, "problem with reading body")
	}

	return body, nil
}

// Creates full URL for http request.
func GetURL(path, host, port string) string {
	if host == "" {
		host = defaultHost
	}
	if port == "" {
		port = defaultPorts[rand.Intn(len(defaultPorts))]
	}

	hostOverride := os.Getenv(launchnet.TestWalletHost)
	if hostOverride != "" {
		host = hostOverride
		port = "80"
	}
	res := "http://" + host + ":" + port + path
	return res
}

func SendAPIRequest(ctx context.Context, url string, body interface{}) ([]byte, error) {
	req, err := prepareReq(ctx, url, body)
	if err != nil {
		return nil, throw.W(err, "problem with preparing request")
	}

	return doReq(req)
}

// Creates wallet and returns it's reference.
func CreateSimpleWallet(ctx context.Context) (string, error) {
	createURL := GetURL(WalletCreatePath, "", "")
	rawResp, err := SendAPIRequest(ctx, createURL, nil)
	if err != nil {
		return "", throw.W(err, "failed to send request or get response body")
	}

	resp, err := UnmarshalWalletCreateResponse(rawResp)
	if err != nil {
		return "", throw.W(err, "failed to unmarshal response")
	}
	if resp.Err != "" {
		return "", throw.W(throw.New(resp.Err), "problem during execute request", struct {
			TraceID string
		}{TraceID: resp.TraceID})
	}
	return resp.Ref, nil
}

// Returns wallet balance.
func GetWalletBalance(ctx context.Context, url, ref string) (uint, error) {
	rawResp, err := SendAPIRequest(ctx, url, WalletGetBalanceRequestBody{Ref: ref})
	if err != nil {
		return 0, throw.W(err, "failed to send request or get response body")
	}

	resp, err := UnmarshalWalletGetBalanceResponse(rawResp)
	if err != nil {
		return 0, throw.W(err, "failed to unmarshal response")
	}
	if resp.Err != "" {
		return 0, throw.W(throw.New(resp.Err), "problem during execute request", struct {
			TraceID   string
			WalletRef string
		}{TraceID: resp.TraceID, WalletRef: ref})
	}
	return resp.Amount, nil
}

// Adds amount to wallet.
func AddAmountToWallet(ctx context.Context, url, ref string, amount uint) error {
	rawResp, err := SendAPIRequest(ctx, url, WalletAddAmountRequestBody{To: ref, Amount: amount})
	if err != nil {
		return throw.W(err, "failed to send request or get response body")
	}

	resp, err := UnmarshalWalletAddAmountResponse(rawResp)
	if err != nil {
		return throw.W(err, "failed to unmarshal response")
	}
	if resp.Err != "" {
		return throw.W(throw.New(resp.Err), "problem during execute request", struct {
			TraceID   string
			WalletRef string
		}{TraceID: resp.TraceID, WalletRef: ref})
	}
	return nil
}
