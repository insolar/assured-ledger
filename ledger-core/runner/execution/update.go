// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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

	Result   requestresult.RequestResult
	Outgoing RPC
}
