// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package extractor

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/builtin/foundation"

	"github.com/pkg/errors"
)

// NodeInfoResponse extracts response of GetNodeInfo
func NodeInfoResponse(data []byte) (string, string, error) {
	res := struct {
		PublicKey string
		Role      insolar.StaticRole
	}{}
	var contractErr *foundation.Error
	err := foundation.UnmarshalMethodResultSimplified(data, &res, &contractErr)
	if err != nil {
		return "", "", errors.Wrap(err, "[ NodeInfoResponse ] Can't unmarshal response")
	}
	if contractErr != nil {
		return "", "", errors.Wrap(contractErr, "[ NodeInfoResponse ] Has error in response")
	}

	return res.PublicKey, res.Role.String(), nil
}
