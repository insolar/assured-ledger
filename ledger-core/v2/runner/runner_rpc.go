// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executionevent"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common/rpctypes"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func (r *DefaultService) GetCode(_ rpctypes.UpGetCodeReq, _ *rpctypes.UpGetCodeResp) error {
	panic(throw.Unsupported())
}

func (r *DefaultService) CallMethod(in rpctypes.UpCallMethodReq, out *rpctypes.UpCallMethodResp) error {
	sink := r.getExecutionSink(in.ID)
	if sink == nil {
		panic(throw.E("failed to find ExecutionContext", nil))
	}

	event := executionevent.NewRPCBuilder(in.Request, in.Callee).
		CallMethod(in.Object, in.Prototype, in.Method, in.Arguments).
		SetTolerance(in.Tolerance).SetIsolation(in.Isolation)
	sink.ExternalCall(event)

	out.Result = <-sink.input

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

	event := executionevent.NewRPCBuilder(in.Request, in.Callee).
		CallConstructor(in.Prototype, in.ConstructorName, in.ArgsSerialized)
	sink.ExternalCall(event)

	out.Result = <-sink.input

	if out.Result == nil {
		panic(throw.E("CallConstructor result unexpected type, got nil"))
	}

	return nil
}

func (r *DefaultService) DeactivateObject(in rpctypes.UpDeactivateObjectReq, out *rpctypes.UpDeactivateObjectResp) error {
	sink := r.getExecutionSink(in.ID)
	if sink == nil {
		panic(throw.E("failed to find ExecutionContext", nil))
	}

	event := executionevent.NewRPCBuilder(in.Request, in.Callee).Deactivate()
	sink.ExternalCall(event)

	rawValue := <-sink.input

	if rawValue == nil {
		return throw.E("Deactivate result unexpected type, expected nil")
	}

	return nil
}
