package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"time"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/reference"
)

const (
	apiTimeout = 10 * time.Second
)

type WalletCreateResponse struct {
	Err     string `json:"error"`
	Ref     string `json:"reference"`
	TraceID string `json:"traceID"`
}

func UnmarshalWalletCreateResponse(resp []byte) (WalletCreateResponse, error) { // nolint:unused,deadcode
	result := WalletCreateResponse{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return WalletCreateResponse{}, errors.W(err, "problem with unmarshaling response")
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
		return WalletGetBalanceResponse{}, errors.W(err, "problem with unmarshaling response")
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
		return WalletAddAmountResponse{}, errors.W(err, "problem with unmarshaling response")
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
		return WalletTransferResponse{}, errors.W(err, "problem with unmarshaling response")
	}
	return result, nil
}

// nolint:unused,deadcode
type WalletDeleteRequestBody struct {
	Ref string `json:"walletRef"`
}

// nolint:unused,deadcode
type WalletDeleteResponse struct {
	Err     string `json:"error"`
	TraceID string `json:"traceID"`
}

func UnmarshalWalletDeleteResponse(resp []byte) (WalletDeleteResponse, error) { // nolint:unused,deadcode
	result := WalletDeleteResponse{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return WalletDeleteResponse{}, errors.W(err, "problem with unmarshaling response")
	}
	return result, nil
}

// Utility function RE wallet + HTTP API
func (s *Server) CallAPICreateWallet(ctx context.Context) (int, []byte) {
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	var (
		responseWriter = httptest.NewRecorder()
		httpRequest    = httptest.NewRequest("POST", "/wallet/create", strings.NewReader("")).WithContext(ctx)
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
func (s *Server) CallAPIAddAmount(ctx context.Context, wallet reference.Global, amount uint) (int, []byte) {
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

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
		httpRequest    = httptest.NewRequest("POST", "/wallet/add_amount", bodyBuffer).WithContext(ctx)
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
func (s *Server) CallAPITransfer(ctx context.Context, walletFrom reference.Global, walletTo reference.Global, amount uint) (int, []byte) {
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

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
		httpRequest    = httptest.NewRequest("POST", "/wallet/transfer", bodyBuffer).WithContext(ctx)
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
func (s *Server) CallAPIGetBalance(ctx context.Context, wallet reference.Global) (int, []byte) {
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

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
		httpRequest    = httptest.NewRequest("POST", "/wallet/get_balance", bodyBuffer).WithContext(ctx)
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

// nolint:interfacer
func (s *Server) CallAPIDelete(ctx context.Context, wallet reference.Global) (int, []byte) {
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	bodyBuffer := bytes.NewBuffer(nil)

	{
		err := json.NewEncoder(bodyBuffer).Encode(WalletDeleteRequestBody{
			Ref: wallet.String(),
		})
		if err != nil {
			panic(err)
		}
	}

	var (
		responseWriter = httptest.NewRecorder()
		httpRequest    = httptest.NewRequest("POST", "/wallet/delete", bodyBuffer).WithContext(ctx)
	)

	s.testWalletServer.Delete(responseWriter, httpRequest)

	var (
		statusCode = responseWriter.Result().StatusCode
		body, err  = ioutil.ReadAll(responseWriter.Result().Body)
	)

	if err != nil {
		panic(err)
	}

	return statusCode, body
}
