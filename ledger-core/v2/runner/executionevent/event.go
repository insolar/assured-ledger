// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package executionevent

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/execution"
)

type RPC interface{ rpc() }

type Builder struct {
	request reference.Global
	object  reference.Global
}

func (r Builder) Deactivate() Deactivate {
	return Deactivate{
		parentObjectReference:  r.object,
		parentRequestReference: r.request,
	}
}

func (r Builder) CallConstructor(class reference.Global, constructor string, arguments []byte) CallConstructor {
	return CallConstructor{
		parentRequestReference: r.request,
		parentObjectReference:  r.object,

		constructor: constructor,
		arguments:   arguments,
		class:       class,
	}
}

func (r Builder) CallMethod(
	object reference.Global,
	class reference.Global,
	method string,
	arguments []byte,
) CallMethod {
	return CallMethod{
		parentRequestReference: r.request,
		parentObjectReference:  r.object,

		object:    object,
		method:    method,
		arguments: arguments,
		class:     class,
	}
}

func (r Builder) GetCode(code reference.Global) GetCode {
	return GetCode{
		codeReference: code,
	}
}

func NewRPCBuilder(request reference.Global, object reference.Global) Builder {
	return Builder{request: request, object: object}
}

type GetCode struct {
	codeReference reference.Global
}

func (e GetCode) CodeReference() reference.Global {
	return e.codeReference
}

func (e GetCode) rpc() {}

type Deactivate struct {
	parentRequestReference reference.Global
	parentObjectReference  reference.Global
}

func (e Deactivate) ParentObjectReference() reference.Global {
	return e.parentObjectReference
}

func (e Deactivate) ParentRequestReference() reference.Global {
	return e.parentRequestReference
}

func (e Deactivate) rpc() {}

type CallConstructor struct {
	parentRequestReference reference.Global
	parentObjectReference  reference.Global

	constructor string
	arguments   []byte
	class       reference.Global
}

func (e CallConstructor) Class() reference.Global {
	return e.class
}

func (e CallConstructor) Arguments() []byte {
	return e.arguments
}

func (e CallConstructor) Constructor() string {
	return e.constructor
}

func (e CallConstructor) ParentObjectReference() reference.Global {
	return e.parentObjectReference
}

func (e CallConstructor) ParentRequestReference() reference.Global {
	return e.parentRequestReference
}

func (e CallConstructor) ConstructVCallRequest(execution execution.Context) *payload.VCallRequest {
	execution.Sequence++

	return &payload.VCallRequest{
		CallType:            payload.CTConstructor,
		CallFlags:           payload.BuildCallFlags(execution.Isolation.Interference, execution.Isolation.State),
		Caller:              e.parentObjectReference,
		Callee:              reference.Global{},
		CallSiteDeclaration: e.class,
		CallSiteMethod:      e.constructor,
		CallSequence:        execution.Sequence,
		CallReason:          e.parentRequestReference,
		KnownCalleeIncoming: reference.Global{},
		CallOutgoing:        reference.Local{}, // must be filled in the caller
		Arguments:           e.arguments,
	}
}

func (e CallConstructor) rpc() {}

type CallMethod struct {
	parentRequestReference reference.Global
	parentObjectReference  reference.Global

	method       string
	arguments    []byte
	object       reference.Global
	class        reference.Global
	interference contract.InterferenceFlag
	isolation    contract.StateFlag
}

func (e CallMethod) Interference() contract.InterferenceFlag {
	return e.interference
}

func (e CallMethod) Class() reference.Global {
	return e.class
}

func (e CallMethod) Object() reference.Global {
	return e.object
}

func (e CallMethod) Arguments() []byte {
	return e.arguments
}

func (e CallMethod) Method() string {
	return e.method
}

func (e CallMethod) ParentObjectReference() reference.Global {
	return e.parentObjectReference
}

func (e CallMethod) ParentRequestReference() reference.Global {
	return e.parentRequestReference
}

func (e CallMethod) ConstructVCallRequest(execution execution.Context) *payload.VCallRequest {
	execution.Sequence++

	return &payload.VCallRequest{
		CallType:            payload.CTMethod,
		CallFlags:           payload.BuildCallFlags(execution.Isolation.Interference, execution.Isolation.State),
		Caller:              e.parentObjectReference,
		Callee:              e.object,
		CallSiteDeclaration: e.class,
		CallSiteMethod:      e.method,
		CallSequence:        execution.Sequence,
		CallReason:          e.parentRequestReference,
		CallOutgoing:        reference.Local{}, // must be filled in the caller
		Arguments:           e.arguments,
	}
}

func (e CallMethod) SetInterference(interference contract.InterferenceFlag) CallMethod {
	e.interference = interference
	return e
}

func (e CallMethod) SetIsolation(isolation contract.StateFlag) CallMethod {
	e.isolation = isolation
	return e
}

func (e CallMethod) rpc() {}
