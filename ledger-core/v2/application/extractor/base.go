// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package extractor

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/builtin/foundation"

	"github.com/pkg/errors"
)

func stringResponse(data []byte) (string, error) {
	var result string
	var contractErr *foundation.Error
	err := foundation.UnmarshalMethodResultSimplified(data, &result, &contractErr)
	if err != nil {
		return "", errors.Wrap(err, "[ StringResponse ] Can't unmarshal response ")
	}
	if contractErr != nil {
		return "", errors.Wrap(contractErr, "[ StringResponse ] Has error in response")
	}

	return result, nil
}
