// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package benchs

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/google/gops/agent"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	jsoniter "github.com/json-iterator/go"
)

func init() {
	// starts gops agent https://github.com/google/gops on default addr (127.0.0.1:0)
	if err := agent.Listen(agent.Options{}); err != nil {
		panic(err)
	}
}

var (
	httpClient   *http.Client
	apiAddresses = []string{"127.0.0.1:32302", "127.0.0.1:32304"}
)

const (
	requestTimeout = 30 * time.Second
	contentType    = "Content-Type"

	walletPath = "/wallet"

	walletCreatePath     = walletPath + "/create"
	walletGetBalancePath = walletPath + "/get_balance"
	walletAddAmountPath  = walletPath + "/add_amount"
)

// setAPIAddresses is not thread safe, it is supposed to be called before bench run, right after launchnet is configured
func setAPIAddresses(addresses []string) {
	apiAddresses = addresses
}

func init() {
	rand.Seed(time.Now().Unix())
	httpClient = createHTTPClient()
}

// createHTTPClient for connection re-use
func createHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 10000,
		},
		Timeout: requestTimeout,
	}

	return client
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
	defer postResp.Body.Close()
	if http.StatusOK != postResp.StatusCode {
		return nil, throw.New("bad http response code: " + strconv.Itoa(postResp.StatusCode))
	}

	body, err := ioutil.ReadAll(postResp.Body)
	if err != nil {
		return nil, throw.W(err, "problem with reading body")
	}

	return body, nil
}

// Creates full URL for http request.
func getURL(path, apiAddr string) string {
	if apiAddr == "" {
		apiAddr = apiAddresses[rand.Intn(len(apiAddresses))]
	}
	res := "http://" + apiAddr + path
	return res
}

func sendAPIRequest(ctx context.Context, url string, body interface{}) ([]byte, error) {
	req, err := prepareReq(ctx, url, body)
	if err != nil {
		return nil, throw.W(err, "problem with preparing request")
	}

	return doReq(req)
}

// Creates wallet and returns it's reference.
func createSimpleWallet(ctx context.Context) (string, error) {
	createURL := getURL(walletCreatePath, "")
	rawResp, err := sendAPIRequest(ctx, createURL, nil)
	if err != nil {
		return "", throw.W(err, "failed to send request or get response body")
	}

	resp, err := unmarshalWalletCreateResponse(rawResp)
	if err != nil {
		return "", throw.W(err, "failed to unmarshal response")
	}
	if resp.Err != "" {
		return "", fmt.Errorf("problem during execute request: %s", resp.Err)
	}
	return resp.Ref, nil
}

// Returns wallet balance.
func getWalletBalance(ctx context.Context, url, ref string) (uint, error) {
	rawResp, err := sendAPIRequest(ctx, url, walletGetBalanceRequestBody{Ref: ref})
	if err != nil {
		return 0, throw.W(err, "failed to send request or get response body")
	}

	resp, err := unmarshalWalletGetBalanceResponse(rawResp)
	if err != nil {
		return 0, throw.W(err, "failed to unmarshal response")
	}
	if resp.Err != "" {
		return 0, fmt.Errorf("problem during execute request: %s", resp.Err)
	}
	return resp.Amount, nil
}

// Adds amount to wallet.
func addAmountToWallet(ctx context.Context, url, ref string, amount uint) error {
	rawResp, err := sendAPIRequest(ctx, url, walletAddAmountRequestBody{To: ref, Amount: amount})
	if err != nil {
		return throw.W(err, "failed to send request or get response body")
	}

	resp, err := unmarshalWalletAddAmountResponse(rawResp)
	if err != nil {
		return throw.W(err, "failed to unmarshal response")
	}
	if resp.Err != "" {
		return fmt.Errorf("problem during execute request: %s", resp.Err)
	}
	return nil
}

const startBalance uint = 1000000000 // nolint:unused,deadcode,varcheck

// nolint:unused
type walletCreateResponse struct {
	Err     string `json:"error"`
	Ref     string `json:"reference"`
	TraceID string `json:"traceID"`
}

func unmarshalWalletCreateResponse(resp []byte) (walletCreateResponse, error) { // nolint:unused,deadcode
	result := walletCreateResponse{}
	if err := jsoniter.Unmarshal(resp, &result); err != nil {
		return walletCreateResponse{}, throw.W(err, "problem with unmarshaling response")
	}
	return result, nil
}

// nolint:unused,deadcode
type walletGetBalanceRequestBody struct {
	Ref string `json:"walletRef"`
}

// nolint:unused,deadcode
type walletGetBalanceResponse struct {
	Err     string `json:"error"`
	Amount  uint   `json:"amount"`
	TraceID string `json:"traceID"`
}

func unmarshalWalletGetBalanceResponse(resp []byte) (walletGetBalanceResponse, error) { // nolint:unused,deadcode
	result := walletGetBalanceResponse{}
	if err := jsoniter.Unmarshal(resp, &result); err != nil {
		return walletGetBalanceResponse{}, throw.W(err, "problem with unmarshaling response")
	}
	return result, nil
}

// nolint:unused,deadcode,varcheck
type walletAddAmountRequestBody struct {
	To     string `json:"to"`
	Amount uint   `json:"amount"`
}

// nolint:unused
type walletAddAmountResponse struct {
	Err     string `json:"error"`
	TraceID string `json:"traceID"`
}

func unmarshalWalletAddAmountResponse(resp []byte) (walletAddAmountResponse, error) { // nolint:unused,deadcode
	result := walletAddAmountResponse{}
	if err := jsoniter.Unmarshal(resp, &result); err != nil {
		return walletAddAmountResponse{}, throw.W(err, "problem with unmarshaling response")
	}
	return result, nil
}
