package testutils

import (
	jsoniter "github.com/json-iterator/go"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
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
	if err := jsoniter.Unmarshal(resp, &result); err != nil {
		return WalletCreateResponse{}, throw.W(err, "problem with unmarshaling response")
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
	if err := jsoniter.Unmarshal(resp, &result); err != nil {
		return WalletGetBalanceResponse{}, throw.W(err, "problem with unmarshaling response")
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
	if err := jsoniter.Unmarshal(resp, &result); err != nil {
		return WalletAddAmountResponse{}, throw.W(err, "problem with unmarshaling response")
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
	if err := jsoniter.Unmarshal(resp, &result); err != nil {
		return WalletTransferResponse{}, throw.W(err, "problem with unmarshaling response")
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
	if err := jsoniter.Unmarshal(resp, &result); err != nil {
		return WalletDeleteResponse{}, throw.W(err, "problem with unmarshaling response")
	}
	return result, nil
}
