// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package executionupdate

import (
	"github.com/insolar/assured-ledger/ledger-core/runner/executionevent"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
)

//go:generate stringer -type=StateUpdateType
type StateUpdateType int

const (
	Undefined StateUpdateType = iota

	Error
	Aborted
	Done

	OutgoingCall
)

type ContractExecutionStateUpdate struct {
	Type  StateUpdateType
	Error error

	Result   *requestresult.RequestResult
	Outgoing executionevent.RPC
}
