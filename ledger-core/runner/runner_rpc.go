// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/rpctypes"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func (r *DefaultService) CallMethod(in rpctypes.UpCallMethodReq, out *rpctypes.UpCallMethodResp) error {
	sink := r.getExecutionSink(in.ID)
	if sink == nil {
		panic(throw.E("failed to find ExecutionContext", nil))
	}

	event := execution.NewRPCBuilder(in.Request, in.Callee).
		CallMethod(in.Object, in.Class, in.Method, in.Arguments).
		SetInterference(in.Interference).SetIsolation(in.Isolation)
	if !sink.ExternalCall(event) {
		panic(throw.IllegalState())
	}
	r.awaitedRunFinish(in.ID, false)

	result := sink.WaitInput()
	if result.Error != nil {
		panic(throw.NotImplemented())
	}

	out.Result = result.ExecutionResult
	if out.Result == nil {
		panic(throw.E("CallMethod result unexpected type, got nil"))
	}

	return nil

}

func (r *DefaultService) CallConstructor(in rpctypes.UpCallConstructorReq, out *rpctypes.UpCallConstructorResp) error {
	sink := r.getExecutionSink(in.ID)
	if sink == nil {
		panic(throw.E("failed to find ExecutionContext", nil))
	}

	event := execution.NewRPCBuilder(in.Request, in.Callee).
		CallConstructor(in.Class, in.ConstructorName, in.ArgsSerialized)
	if !sink.ExternalCall(event) {
		panic(throw.IllegalState())
	}
	r.awaitedRunFinish(in.ID, false)

	result := sink.WaitInput()
	if result.Error != nil {
		panic(throw.NotImplemented())
	}

	out.Result = result.ExecutionResult
	if out.Result == nil {
		panic(throw.E("CallConstructor result unexpected type, got nil"))
	}

	return nil
}

func (r *DefaultService) DeactivateObject(in rpctypes.UpDeactivateObjectReq, _ *rpctypes.UpDeactivateObjectResp) error {
	sink := r.getExecutionSink(in.ID)
	if sink == nil {
		panic(throw.E("failed to find ExecutionContext", nil))
	}

	sink.ResultBuilder().Deactivate()

	return nil
}
