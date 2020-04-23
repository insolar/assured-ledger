// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"strings"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type WalletCreateResponse struct {
	Err     string `json:"error"`
	Ref     string `json:"reference"`
	TraceID string `json:"traceID"`
}

func UnmarshalWalletCreateResponse(resp []byte) (WalletCreateResponse, error) { // nolint:unused,deadcode
	result := WalletCreateResponse{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return WalletCreateResponse{}, errors.Wrap(err, "problem with unmarshaling response")
	}
	return result, nil
}

// nolint:unused,deadcode
type WalletGetBalanceRequestBody struct {
	Ref string `json:"walletRef"`
}

// nolint:unused,deadcode
type WalletGetBalanceResponse struct {
	Err     string `json:"error"`
	Amount  uint   `json:"amount"`
	TraceID string `json:"traceID"`
}

func UnmarshalWalletGetBalanceResponse(resp []byte) (WalletGetBalanceResponse, error) { // nolint:unused,deadcode
	result := WalletGetBalanceResponse{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return WalletGetBalanceResponse{}, errors.Wrap(err, "problem with unmarshaling response")
	}
	return result, nil
}

// nolint:unused,deadcode
type WalletAddAmountRequestBody struct {
	To     string `json:"to"`
	Amount uint   `json:"amount"`
}

// nolint:unused
type WalletAddAmountResponse struct {
	Err     string `json:"error"`
	TraceID string `json:"traceID"`
}

func UnmarshalWalletAddAmountResponse(resp []byte) (WalletAddAmountResponse, error) { // nolint:unused,deadcode
	result := WalletAddAmountResponse{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return WalletAddAmountResponse{}, errors.Wrap(err, "problem with unmarshaling response")
	}
	return result, nil
}

// nolint:unused,deadcode
type WalletTransferRequestBody struct {
	To     string `json:"to"`
	From   string `json:"from"`
	Amount uint   `json:"amount"`
}

// nolint:unused
type WalletTransferResponse struct {
	Err     string `json:"error"`
	TraceID string `json:"traceID"`
}

func UnmarshalWalletTransferResponse(resp []byte) (WalletTransferResponse, error) { // nolint:unused,deadcode
	result := WalletTransferResponse{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return WalletTransferResponse{}, errors.Wrap(err, "problem with unmarshaling response")
	}
	return result, nil
}

// Utility function RE wallet + HTTP API
func (s *Server) CallAPICreateWallet() (int, []byte) {
	var (
		responseWriter = httptest.NewRecorder()
		httpRequest    = httptest.NewRequest("POST", "/wallet/create", strings.NewReader(""))
	)

	s.testWalletServer.Create(responseWriter, httpRequest)

	var (
		statusCode = responseWriter.Result().StatusCode
		body, err  = ioutil.ReadAll(responseWriter.Result().Body)
	)

	if err != nil {
		panic(err)
	}

	return statusCode, body
}

// nolint:interfacer
func (s *Server) CallAPIAddAmount(wallet reference.Global, amount uint) (int, []byte) {
	bodyBuffer := bytes.NewBuffer(nil)

	{
		err := json.NewEncoder(bodyBuffer).Encode(WalletAddAmountRequestBody{
			To:     wallet.String(),
			Amount: amount,
		})
		if err != nil {
			panic(err)
		}
	}

	var (
		responseWriter = httptest.NewRecorder()
		httpRequest    = httptest.NewRequest("POST", "/wallet/add_amount", bodyBuffer)
	)

	s.testWalletServer.AddAmount(responseWriter, httpRequest)

	var (
		statusCode = responseWriter.Result().StatusCode
		body, err  = ioutil.ReadAll(responseWriter.Result().Body)
	)

	if err != nil {
		panic(err)
	}

	return statusCode, body
}

// nolint:interfacer
func (s *Server) CallAPITransfer(walletFrom reference.Global, walletTo reference.Global, amount uint) (int, []byte) {
	bodyBuffer := bytes.NewBuffer(nil)

	{
		err := json.NewEncoder(bodyBuffer).Encode(WalletTransferRequestBody{
			To:     walletTo.String(),
			From:   walletFrom.String(),
			Amount: amount,
		})
		if err != nil {
			panic(err)
		}
	}

	var (
		responseWriter = httptest.NewRecorder()
		httpRequest    = httptest.NewRequest("POST", "/wallet/transfer", bodyBuffer)
	)

	s.testWalletServer.Transfer(responseWriter, httpRequest)

	var (
		statusCode = responseWriter.Result().StatusCode
		body, err  = ioutil.ReadAll(responseWriter.Result().Body)
	)

	if err != nil {
		panic(err)
	}

	return statusCode, body
}

// nolint:interfacer
func (s *Server) CallAPIGetBalance(wallet reference.Global) (int, []byte) {
	bodyBuffer := bytes.NewBuffer(nil)

	{
		err := json.NewEncoder(bodyBuffer).Encode(WalletGetBalanceRequestBody{
			Ref: wallet.String(),
		})
		if err != nil {
			panic(err)
		}
	}

	var (
		responseWriter = httptest.NewRecorder()
		httpRequest    = httptest.NewRequest("POST", "/wallet/get_balance", bodyBuffer)
	)

	s.testWalletServer.GetBalance(responseWriter, httpRequest)

	var (
		statusCode = responseWriter.Result().StatusCode
		body, err  = ioutil.ReadAll(responseWriter.Result().Body)
	)

	if err != nil {
		panic(err)
	}

	return statusCode, body
}
