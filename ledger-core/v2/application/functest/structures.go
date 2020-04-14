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
