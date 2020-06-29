// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
)

var (
	defaultPorts = [2]string{"32302", "32304"}
)

const (
	requestTimeout = 30 * time.Second
	contentType    = "Content-Type"

	defaultHost = "127.0.0.1"
	walletPath  = "/wallet"

	WalletCreatePath     = walletPath + "/create"
	WalletGetBalancePath = walletPath + "/get_balance"
	WalletAddAmountPath  = walletPath + "/add_amount"
	WalletTransferPath   = walletPath + "/transfer"
)

// CreateHTTPClient for connection re-use
func CreateHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{},
		Timeout:   requestTimeout,
	}

	return client
}

// Creates http.Request with all necessary fields.
func PrepareReq(url string, body interface{}) (*http.Request, error) {
	jsonValue, err := json.Marshal(body)
	if err != nil {
		return nil, errors.W(err, "problem with marshaling params")
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, errors.W(err, "problem with creating request")
	}
	req.Header.Set(contentType, "application/json")

	return req, nil
}

// Executes http.Request and returns response body.
func DoReq(client *http.Client, req *http.Request) ([]byte, error) {
	postResp, err := client.Do(req)
	if err != nil {
		return nil, errors.W(err, "problem with sending request")
	}

	if postResp == nil {
		return nil, errors.New("response is nil")
	}

	defer postResp.Body.Close()
	if http.StatusOK != postResp.StatusCode {
		return nil, errors.New("bad http response code: " + strconv.Itoa(postResp.StatusCode))
	}

	body, err := ioutil.ReadAll(postResp.Body)
	if err != nil {
		return nil, errors.W(err, "problem with reading body")
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

func SendAPIRequest(client *http.Client, url string, body interface{}) ([]byte, error) {
	req, err := PrepareReq(url, body)
	if err != nil {
		return nil, errors.W(err, "problem with preparing request")
	}

	return DoReq(client, req)
}

// Creates wallet and returns it's reference.
func CreateSimpleWallet(client *http.Client, url string) (string, error) {
	rawResp, err := SendAPIRequest(client, url, nil)
	if err != nil {
		return "", errors.W(err, "failed to send request or get response body")
	}

	resp, err := UnmarshalWalletCreateResponse(rawResp)
	if err != nil {
		return "", errors.W(err, "failed to unmarshal response")
	}
	if resp.Err != "" {
		return "", fmt.Errorf("problem during execute request: %s", resp.Err)
	}
	return resp.Ref, nil
}

// Returns wallet balance.
func GetWalletBalance(client *http.Client, url, ref string) (uint, error) {
	rawResp, err := SendAPIRequest(client, url, WalletGetBalanceRequestBody{Ref: ref})
	if err != nil {
		return 0, errors.W(err, "failed to send request or get response body")
	}

	resp, err := UnmarshalWalletGetBalanceResponse(rawResp)
	if err != nil {
		return 0, errors.W(err, "failed to unmarshal response")
	}
	if resp.Err != "" {
		return 0, fmt.Errorf("problem during execute request: %s", resp.Err)
	}
	return resp.Amount, nil
}

// Adds amount to wallet.
func AddAmountToWallet(client *http.Client, url, ref string, amount uint) error {
	rawResp, err := SendAPIRequest(client, url, WalletAddAmountRequestBody{To: ref, Amount: amount})
	if err != nil {
		return errors.W(err, "failed to send request or get response body")
	}

	resp, err := unmarshalWalletAddAmountResponse(rawResp)
	if err != nil {
		return errors.W(err, "failed to unmarshal response")
	}
	if resp.Err != "" {
		return fmt.Errorf("problem during execute request: %s", resp.Err)
	}
	return nil
}
