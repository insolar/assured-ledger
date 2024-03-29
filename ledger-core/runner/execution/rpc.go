package execution

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type RPC interface{ rpc() }

type Builder struct {
	request reference.Global
	object  reference.Global
}

func NewRPCBuilder(request reference.Global, object reference.Global) Builder {
	return Builder{request: request, object: object}
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

func (e CallConstructor) ConstructVCallRequest(execution Context) *rms.VCallRequest {
	return &rms.VCallRequest{
		CallType:            rms.CallTypeConstructor,
		CallFlags:           rms.BuildCallFlags(execution.Isolation.Interference, execution.Isolation.State),
		Caller:              rms.NewReference(e.parentObjectReference),
		Callee:              rms.NewReference(e.class),
		CallSiteMethod:      e.constructor,
		KnownCalleeIncoming: rms.NewReference(reference.Global{}),
		CallOutgoing:        rms.NewReference(reference.Global{}), // must be filled by the caller
		CallSequence:        0,                                    // must be filled by the caller
		Arguments:           rms.NewBytes(e.arguments),
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
	interference isolation.InterferenceFlag
	isolation    isolation.StateFlag
}

func (e CallMethod) Interference() isolation.InterferenceFlag {
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

func (e CallMethod) ConstructVCallRequest(execution Context) *rms.VCallRequest {
	return &rms.VCallRequest{
		CallType:       rms.CallTypeMethod,
		CallFlags:      rms.BuildCallFlags(execution.Isolation.Interference, execution.Isolation.State),
		Caller:         rms.NewReference(e.parentObjectReference),
		Callee:         rms.NewReference(e.object),
		CallSiteMethod: e.method,
		CallSequence:   0,                                    // must be filled by the caller
		CallOutgoing:   rms.NewReference(reference.Global{}), // must be filled by the caller
		Arguments:      rms.NewBytes(e.arguments),
	}
}

func (e CallMethod) SetInterference(interference isolation.InterferenceFlag) CallMethod {
	e.interference = interference
	return e
}

func (e CallMethod) SetIsolation(isolation isolation.StateFlag) CallMethod {
	e.isolation = isolation
	return e
}

func (e CallMethod) rpc() {}
