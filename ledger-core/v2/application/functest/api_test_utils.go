// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest

package functest

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

var (
	httpClient *http.Client
	nodesPorts = [5]string{"32301", "32302", "32303", "32304", "32305"}
)

const (
	requestTimeout = 30 * time.Second
	contentType    = "Content-Type"

	defaultHost      = "127.0.0.1"
	createWalletPath = "/wallet/create"
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
	jsonValue, err := json.Marshal(body)
	if err != nil {
		return nil, errors.Wrap(err, "problem with marshaling params")
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, errors.Wrap(err, "problem with creating request")
	}
	req.Header.Set(contentType, "application/json")

	return req, nil
}

// Executes http.Request and returns response body.
func doReq(req *http.Request) ([]byte, error) {
	postResp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "problem with sending request")
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
		return nil, errors.Wrap(err, "problem with reading body")
	}

	return body, nil
}

// Creates full URL for http request.
func getURL(path, host, port string) string {
	if host == "" {
		host = defaultHost
	}
	if port == "" {
		port = nodesPorts[rand.Intn(len(nodesPorts))]
	}
	res := "http://" + host + ":" + port + path
	return res
}

func sendAPIRequest(url string, body interface{}) ([]byte, error) {
	req, err := prepareReq(url, body)
	if err != nil {
		return nil, errors.Wrap(err, "problem with preparing request")
	}

	return doReq(req)
}
