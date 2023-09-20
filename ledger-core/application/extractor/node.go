package extractor

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// NodeInfoResponse extracts response of GetNodeInfo
func NodeInfoResponse(data []byte) (string, string, error) {
	res := struct {
		PublicKey string
		Role      member.PrimaryRole
	}{}
	var contractErr *foundation.Error
	err := foundation.UnmarshalMethodResultSimplified(data, &res, &contractErr)
	if err != nil {
		return "", "", errors.W(err, "[ NodeInfoResponse ] Can't unmarshal response")
	}
	if contractErr != nil {
		return "", "", errors.W(contractErr, "[ NodeInfoResponse ] Has error in response")
	}

	return res.PublicKey, res.Role.String(), nil
}
