// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package machine

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/call"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/runner/machine.Executor -o ./ -s _mock.go -g

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
}
