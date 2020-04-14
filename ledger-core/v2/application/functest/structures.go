// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package functest

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// nolint:unused
type createWalletResponse struct {
	Err     error  `json:"error"`
	Ref     string `json:"reference"`
	TraceID string `json:"traceID"`
}

func unmarshalCreateWalletResponse(resp []byte) (createWalletResponse, error) { // nolint:unused,deadcode
	result := createWalletResponse{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return createWalletResponse{}, errors.Wrap(err, "problem with unmarshaling response")
	}
	return result, nil
}

type getWalletBalanceRequestBody struct {
	Ref string `json:"walletRef"`
}

type getWalletBalanceResponse struct {
	Err     error  `json:"error"`
	Amount  int    `json:"amount"`
	TraceID string `json:"traceID"`
}

func unmarshalGetWalletBalanceResponse(resp []byte) (getWalletBalanceResponse, error) {
	result := getWalletBalanceResponse{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return getWalletBalanceResponse{}, errors.Wrap(err, "problem with unmarshaling response")
	}
	return result, nil
}

type walletAddAmountRequestBody struct {
	To     string `json:"to"`
	Amount uint   `json:"amount"`
}

type walletAddAmountResponse struct {
	Err     error  `json:"error"`
	TraceID string `json:"traceID"`
}

func unmarshalWalletAddAmountResponse(resp []byte) (walletAddAmountResponse, error) {
	result := walletAddAmountResponse{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return walletAddAmountResponse{}, errors.Wrap(err, "problem with unmarshaling response")
	}
	return result, nil
}
