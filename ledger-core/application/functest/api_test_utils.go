// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest

package functest

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var (
	httpClient   *http.Client
	defaultPorts = [2]string{"32302", "32304"}
)

const (
	requestTimeout = 30 * time.Second
	contentType    = "Content-Type"

	defaultHost = "127.0.0.1"
	walletPath  = "/wallet"

	walletCreatePath     = walletPath + "/create"
	walletGetBalancePath = walletPath + "/get_balance"
	walletAddAmountPath  = walletPath + "/add_amount"
	walletTransferPath   = walletPath + "/transfer"
	walletDeletePath     = walletPath + "/delete"
)

func init() {
	rand.Seed(time.Now().Unix())
	httpClient = createHTTPClient()
}

// createHTTPClient for connection re-use
func createHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{},
		Timeout:   requestTimeout,
	}

	return client
}

// Creates http.Request with all necessary fields.
func prepareReq(url string, body interface{}) (*http.Request, error) {
	jsonValue, err := jsoniter.Marshal(body)
	if err != nil {
		return nil, throw.W(err, "problem with marshaling params")
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonValue))
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
func getURL(path, host, port string) string {
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

func sendAPIRequest(url string, body interface{}) ([]byte, error) {
	req, err := prepareReq(url, body)
	if err != nil {
		return nil, throw.W(err, "problem with preparing request")
	}

	return doReq(req)
}

// Creates wallet and returns it's reference.
func createSimpleWallet() (string, error) {
	createURL := getURL(walletCreatePath, "", "")
	rawResp, err := sendAPIRequest(createURL, nil)
	if err != nil {
		return "", throw.W(err, "failed to send request or get response body")
	}

	resp, err := unmarshalWalletCreateResponse(rawResp)
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
func getWalletBalance(url, ref string) (uint, error) {
	rawResp, err := sendAPIRequest(url, walletGetBalanceRequestBody{Ref: ref})
	if err != nil {
		return 0, throw.W(err, "failed to send request or get response body")
	}

	resp, err := unmarshalWalletGetBalanceResponse(rawResp)
	if err != nil {
		return 0, throw.W(err, "failed to unmarshal response")
	}
	if resp.Err != "" {
		return 0, throw.W(throw.New(resp.Err), "problem during execute request", struct {
			TraceID string
		}{TraceID: resp.TraceID})
	}
	return resp.Amount, nil
}

// Adds amount to wallet.
func addAmountToWallet(url, ref string, amount uint) error {
	rawResp, err := sendAPIRequest(url, walletAddAmountRequestBody{To: ref, Amount: amount})
	if err != nil {
		return throw.W(err, "failed to send request or get response body")
	}

	resp, err := unmarshalWalletAddAmountResponse(rawResp)
	if err != nil {
		return throw.W(err, "failed to unmarshal response")
	}
	if resp.Err != "" {
		return throw.W(throw.New(resp.Err), "problem during execute request", struct {
			TraceID string
		}{TraceID: resp.TraceID})
	}
	return nil
}
