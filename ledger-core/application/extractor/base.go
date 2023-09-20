package extractor

import (
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func stringResponse(data []byte) (string, error) {
	var result string
	var contractErr *foundation.Error
	err := foundation.UnmarshalMethodResultSimplified(data, &result, &contractErr)
	if err != nil {
		return "", errors.W(err, "[ StringResponse ] Can't unmarshal response ")
	}
	if contractErr != nil {
		return "", errors.W(contractErr, "[ StringResponse ] Has error in response")
	}

	return result, nil
}
