package machine

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/call"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/runner/machine.Executor -o ./ -s _mock.go -g

// Executor is an interface for implementers of one particular machine type
type Executor interface {
	CallMethod(
		ctx context.Context, callContext *call.LogicContext,
		code reference.Global, data []byte,
		method string, args []byte,
	) (
		newObjectState []byte, methodResults []byte, err error,
	)
	CallConstructor(
		ctx context.Context, callContext *call.LogicContext,
		code reference.Global, name string, args []byte,
	) (
		objectState []byte, result []byte, err error,
	)
	ClassifyMethod(ctx context.Context,
		codeRef reference.Global,
		method string) (contract.MethodIsolation, error)
}
