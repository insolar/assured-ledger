package execution

import (
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
)

//go:generate stringer -type=UpdateType
type UpdateType int

const (
	Undefined UpdateType = iota

	Error
	Abort
	Done

	OutgoingCall
)

type Update struct {
	Type  UpdateType
	Error error

	Result   *requestresult.RequestResult
	Outgoing RPC
}
