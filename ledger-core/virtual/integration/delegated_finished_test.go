// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func makeEmptyResult() []byte {
	var ret0 error

	return insolar.MustSerialize(foundation.Result{
		Error:   nil,
		Returns: []interface{}{ret0},
	})
}

type callMethodFunc = func(ctx context.Context, callContext *call.LogicContext, code reference.Global, data []byte, method string, args []byte) (newObjectState []byte, methodResults []byte, err error)
type callConstructorFunc = func(ctx context.Context, callContext *call.LogicContext, code reference.Global, name string, args []byte) (objectState []byte, result []byte, err error)

func mockExecutor(t *testing.T, server *utils.Server, callMethod callMethodFunc, constructorFunc callConstructorFunc) {
	builtinExecutor, err := server.Runner.Manager.GetExecutor(machine.Builtin)
	require.NoError(t, err)

	cache := server.Runner.Cache
	_, walletCodeRef, err := cache.ByClassRef(context.Background(), testwallet.GetClass())
	require.NoError(t, err)

	executorMock := machine.NewExecutorMock(t)
	if callMethod != nil {
		executorMock.CallMethodMock.Set(callMethod)
	}
	if constructorFunc != nil {
		executorMock.CallConstructorMock.Set(constructorFunc)
	}
	executorMock.ClassifyMethodMock.Set(builtinExecutor.ClassifyMethod)
	manager := machine.NewManager()
	err = manager.RegisterExecutor(machine.Builtin, executorMock)
	require.NoError(t, err)
	server.ReplaceMachinesManager(manager)

	cacheMock := descriptor.NewCacheMock(t)
	server.ReplaceCache(cacheMock)
	cacheMock.ByClassRefMock.Return(
		descriptor.NewClass(gen.UniqueGlobalRef(), gen.UniqueLocalRef(), testwallet.GetClass()),
		descriptor.NewCode(nil, machine.Builtin, walletCodeRef.Ref()),
		nil,
	)
}
