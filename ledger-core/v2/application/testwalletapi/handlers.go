// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testwalletapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/application/testwalletapi/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/builtin/foundation"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type logIncomingRequest struct {
	*log.Msg `txt:"Incoming request"`

	URL string
}

const (
	traceIDField = "traceID"
)

type TestWalletServerCreateResult struct {
	Reference string `json:"reference"`
	TraceID   string `json:"traceID"`
	Error     string `json:"error"`
}

type TestWalletServerTransferResult struct {
	TraceID string `json:"traceID"`
	Error   string `json:"error"`
}

type TestWalletServerGetBalanceResult struct {
	Amount  uint   `json:"amount"`
	TraceID string `json:"traceID"`
	Error   string `json:"error"`
}

type TestWalletServerAddAmountResult struct {
	TraceID string `json:"traceID"`
	Error   string `json:"error"`
}

func (s *TestWalletServer) Create(w http.ResponseWriter, req *http.Request) {
	var (
		ctx     = context.Background()
		traceID = utils.RandTraceID()
		logger  = inslogger.FromContext(ctx)
	)

	logger.Infom(logIncomingRequest{URL: req.URL.String()})

	result := TestWalletServerCreateResult{
		Reference: "",
		TraceID:   traceID,
		Error:     "",
	}
	defer func() { s.mustWriteResult(w, result) }()

	empty, _ := insolar.Serialize(struct{}{})

	walletReq := payload.VCallRequest{
		CallType:            payload.CTConstructor,
		Callee:              gen.Reference(),
		Arguments:           empty,
		CallSiteDeclaration: testwallet.GetPrototype(),
		CallSiteMethod:      create,
	}

	walletRes, err := s.runWalletRequest(ctx, walletReq)
	if err != nil {
		result.Error = err.Error()
		return
	}

	var (
		ref             insolar.Reference
		contractCallErr *foundation.Error
	)
	err = foundation.UnmarshalMethodResultSimplified(walletRes.ReturnArguments, &ref, &contractCallErr)
	switch {
	case err != nil:
		result.Error = errors.Wrap(err, "Failed to unmarshal response").Error()
	case contractCallErr != nil:
		result.Error = contractCallErr.S
	default:
		result.Reference = ref.String()
	}
}

type TransferParams struct {
	From   string
	To     string
	Amount uint
}

func (t *TransferParams) isValid() bool {
	return len(t.To) > 0 && len(t.From) > 0 && t.Amount > 0
}

const badRequestErrorPattern = "%s. " + traceIDField + ": %s"

func (s *TestWalletServer) Transfer(w http.ResponseWriter, req *http.Request) {
	var (
		ctx     = context.Background()
		traceID = utils.RandTraceID()
		logger  = inslogger.FromContext(ctx)
	)
	logger.Infom(logIncomingRequest{URL: req.URL.String()})

	params := TransferParams{}
	err := json.NewDecoder(req.Body).Decode(&params)
	if err != nil {
		http.Error(w, fmt.Sprintf(badRequestErrorPattern, "Can't parse boby: "+err.Error(), traceID), http.StatusBadRequest)
		return
	}
	if !params.isValid() {
		http.Error(w, fmt.Sprintf(badRequestErrorPattern, "invalid input params", traceID), http.StatusBadRequest)
		return
	}

	result := TestWalletServerTransferResult{
		TraceID: traceID,
		Error:   "",
	}
	defer func() { s.mustWriteResult(w, result) }()
}

type GetBalanceParams struct {
	WalletRef string
}

func (g *GetBalanceParams) isValid() bool {
	return len(g.WalletRef) > 0
}

func (s *TestWalletServer) GetBalance(w http.ResponseWriter, req *http.Request) {
	var (
		ctx     = context.Background()
		traceID = utils.RandTraceID()
		logger  = inslogger.FromContext(ctx)
	)

	logger.Infom(logIncomingRequest{URL: req.URL.String()})

	params := GetBalanceParams{}
	err := json.NewDecoder(req.Body).Decode(&params)
	if err != nil {
		http.Error(w, fmt.Sprintf(badRequestErrorPattern, "Can't parse boby: "+err.Error(), traceID), http.StatusBadRequest)
		return
	}
	if !params.isValid() {
		http.Error(w, fmt.Sprintf(badRequestErrorPattern, "invalid input params", traceID), http.StatusBadRequest)
		return
	}

	result := TestWalletServerGetBalanceResult{
		Amount:  1000,
		TraceID: traceID,
		Error:   "",
	}

	defer func() { s.mustWriteResult(w, result) }()

	ref, err := insolar.NewReferenceFromString(params.WalletRef)

	if err != nil {
		result.Error = fmt.Sprintf("Failed to create reference from string (%s)", params.WalletRef)
		return
	}

	walletReq := payload.VCallRequest{
		CallType:       payload.CTMethod,
		Callee:         *ref,
		CallSiteMethod: getBalance,
	}

	walletRes, err := s.runWalletRequest(ctx, walletReq)

	if err != nil {
		result.Error = err.Error()
		return
	}

	var (
		amount          uint32
		contractCallErr *foundation.Error
	)

	err = foundation.UnmarshalMethodResultSimplified(walletRes.ReturnArguments, &amount, &contractCallErr)
	switch {
	case err != nil:
		result.Error = errors.Wrap(err, "Failed to unmarshal response").Error()
	case contractCallErr != nil:
		result.Error = contractCallErr.S
	default:
		result.Amount = uint(amount)
	}
}

type AddAmountParams struct {
	To     string
	Amount uint
}

func (a *AddAmountParams) isValid() bool {
	return a.Amount > 0 && len(a.To) > 0
}

func (s *TestWalletServer) AddAmount(w http.ResponseWriter, req *http.Request) {
	var (
		ctx     = context.Background()
		traceID = utils.RandTraceID()
		logger  = inslogger.FromContext(ctx)
	)

	logger.Infom(logIncomingRequest{URL: req.URL.String()})

	params := AddAmountParams{}
	err := json.NewDecoder(req.Body).Decode(&params)
	if err != nil {
		http.Error(w, fmt.Sprintf(badRequestErrorPattern, "Can't parse boby: "+err.Error(), traceID), http.StatusBadRequest)
		return
	}
	if !params.isValid() {
		http.Error(w, fmt.Sprintf(badRequestErrorPattern, "invalid input params", traceID), http.StatusBadRequest)
		return
	}

	result := TestWalletServerAddAmountResult{
		TraceID: traceID,
		Error:   "",
	}
	defer func() { s.mustWriteResult(w, result) }()

	ref, err := insolar.NewReferenceFromString(params.To)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create reference from string (%s)", params.To)
		return
	}

	param, err := insolar.Serialize(params.Amount)
	if err != nil {
		result.Error = "Failed to marshall amount"
		return
	}

	walletReq := payload.VCallRequest{
		CallType:       payload.CTMethod,
		Callee:         *ref,
		Arguments:      param,
		CallSiteMethod: addAmount,
	}

	walletRes, err := s.runWalletRequest(ctx, walletReq)
	if err != nil {
		result.Error = err.Error()
		return
	}

	var contractCallErr *foundation.Error
	err = foundation.UnmarshalMethodResultSimplified(walletRes.ReturnArguments, &contractCallErr)

	switch {
	case err != nil:
		result.Error = errors.Wrap(err, "Failed to unmarshal response").Error()
	case contractCallErr != nil:
		result.Error = contractCallErr.S
	default:

	}
}

func (s *TestWalletServer) runWalletRequest(ctx context.Context, req payload.VCallRequest) (*payload.VCallResult, error) {
	latestPulse, err := s.accessor.Latest(ctx)
	if err != nil {
		return nil, throw.W(err, "Failed to get latest pulse", nil)
	}

	call := &statemachine.TestAPICall{
		Payload:  req,
		Response: make(chan payload.VCallResult),
	}

	err = s.feeder.AddInput(ctx, latestPulse.PulseNumber, call)
	if err != nil {
		return nil, throw.W(err, "Failed to add call to conveyor", nil)
	}

	res := <-call.Response
	return &res, nil
}

func (s *TestWalletServer) mustWriteResult(w http.ResponseWriter, res interface{}) { // nolint:interfacer
	resultString, err := s.jsonCodec.Marshal(res)
	if err != nil {
		panic(err)
	}
	_, err = w.Write(resultString)
	if err != nil {
		panic(err)
	}
}
