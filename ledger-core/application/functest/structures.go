// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package functest

import (
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	jsoniter "github.com/json-iterator/go"
)

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
		return walletCreateResponse{}, errors.W(err, "problem with unmarshaling response")
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
		return walletGetBalanceResponse{}, errors.W(err, "problem with unmarshaling response")
	}
	return result, nil
}

// nolint:unused,deadcode
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
		return walletAddAmountResponse{}, errors.W(err, "problem with unmarshaling response")
	}
	return result, nil
}

// nolint:unused,deadcode
type walletTransferRequestBody struct {
	To     string `json:"to"`
	From   string `json:"from"`
	Amount uint   `json:"amount"`
}

// nolint:unused
type walletTransferResponse struct {
	Err     string `json:"error"`
	TraceID string `json:"traceID"`
}

func unmarshalWalletTransferResponse(resp []byte) (walletTransferResponse, error) { // nolint:unused,deadcode
	result := walletTransferResponse{}
	if err := jsoniter.Unmarshal(resp, &result); err != nil {
		return walletTransferResponse{}, errors.W(err, "problem with unmarshaling response")
	}
	return result, nil
}

// nolint:unused,deadcode
type walletDeleteRequestBody struct {
	Ref string `json:"walletRef"`
}

// nolint:unused,deadcode
type walletDeleteResponse struct {
	Err     string `json:"error"`
	TraceID string `json:"traceID"`
}

func unmarshalWalletDeleteResponse(resp []byte) (walletDeleteResponse, error) { // nolint:unused,deadcode
	result := walletDeleteResponse{}
	if err := jsoniter.Unmarshal(resp, &result); err != nil {
		return walletDeleteResponse{}, errors.W(err, "problem with unmarshaling response")
	}
	return result, nil
}
