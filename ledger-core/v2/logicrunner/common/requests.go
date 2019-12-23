package common

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
)

type OutgoingRequest struct {
	Request  record.IncomingRequest
	Response []byte
	Error    error
}
