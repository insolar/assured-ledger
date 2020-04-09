// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package executionupdate

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executionevent"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/requestresult"
)

//go:generate stringer -type=ContractExecutionStateUpdateType
type ContractExecutionStateUpdateType int

const (
	_ ContractExecutionStateUpdateType = iota

	ContractError
	ContractAborted
	ContractDone

	ContractOutgoingCall
)

type ContractExecutionStateUpdate struct {
	Type  ContractExecutionStateUpdateType
	Error error

	Result   *requestresult.RequestResult
	Outgoing descriptor.RPCEvent
}
