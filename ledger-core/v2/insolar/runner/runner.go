// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md.

package runner

import (
	"context"

	"github.com/google/uuid"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

type ContractExecutionStateUpdateType int

const (
	_ ContractExecutionStateUpdateType = iota
	ContractError
	ContractAborted
	ContractDone

	ContractOutgoingCall
)

type RPCEvent interface{}

type ContractExecutionStateUpdate struct {
	Type  ContractExecutionStateUpdateType
	Error error

	Result   *requestresult.RequestResult
	Outgoing RPCEvent
}

type Execution struct {
	Object     descriptor.ObjectDescriptor
	Context    context.Context
	Request    *record.IncomingRequest
	Nonce      int64
	Deactivate bool
	Pulse      insolar.Pulse

	LogicContext insolar.LogicCallContext
}

type Runner interface {
	ExecutionStart(ctx context.Context, execution Execution) (*ContractExecutionStateUpdate, uuid.UUID, error)
	ExecutionContinue(ctx context.Context, id uuid.UUID, result interface{}) (*ContractExecutionStateUpdate, error)
	ExecutionAbort(ctx context.Context, id uuid.UUID)
	ContractCompile(ctx context.Context, contract interface{})
}

type EventGetCode interface {
	CodeReference() insolar.Reference
}

type EventDeactivate interface {
	ParentObjectReference() insolar.Reference
	ParentRequestReference() insolar.Reference
}

type EventSaveAsChild interface {
	Prototype() insolar.Reference
	Arguments() []byte
	Constructor() string
	ParentObjectReference() insolar.Reference
	ParentRequestReference() insolar.Reference
	ConstructOutgoing(transcript common.Transcript) record.OutgoingRequest
}

type EventRouteCall interface {
	Saga() bool
	Immutable() bool
	Prototype() insolar.Reference
	Object() insolar.Reference
	Arguments() []byte
	Method() string
	ParentObjectReference() insolar.Reference
	ParentRequestReference() insolar.Reference
}
