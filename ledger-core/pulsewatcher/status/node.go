package status

import (
	"github.com/insolar/assured-ledger/ledger-core/application/api/requester"
)

type Node struct {
	URL    string
	Reply  requester.StatusResponse
	ErrStr string
}
