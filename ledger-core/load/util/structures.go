// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
package util

import (
	"encoding/json"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const StartBalance uint = 1000000000 // nolint:unused,deadcode,varcheck

// nolint:unused
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

func unmarshalWalletAddAmountResponse(resp []byte) (WalletAddAmountResponse, error) { // nolint:unused,deadcode
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
