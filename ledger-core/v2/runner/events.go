// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	runner2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/runner"
)

type RPCEvent interface{}

type eventGetCode struct {
	codeReference insolar.Reference
}

func (e eventGetCode) CodeReference() insolar.Reference {
	return e.codeReference
}

type eventSaveAsChild struct {
	parentRequestReference insolar.Reference
	parentObjectReference  insolar.Reference

	constructor string
	arguments   []byte
	prototype   insolar.Reference
}

func (e eventSaveAsChild) Prototype() insolar.Reference {
	return e.prototype
}

func (e eventSaveAsChild) Arguments() []byte {
	return e.arguments
}

func (e eventSaveAsChild) Constructor() string {
	return e.constructor
}

func (e eventSaveAsChild) ParentObjectReference() insolar.Reference {
	return e.parentObjectReference
}

func (e eventSaveAsChild) ParentRequestReference() insolar.Reference {
	return e.parentRequestReference
}

type eventDeactivate struct {
	parentRequestReference insolar.Reference
	parentObjectReference  insolar.Reference
}

func (e eventDeactivate) ParentObjectReference() insolar.Reference {
	return e.parentObjectReference
}

func (e eventDeactivate) ParentRequestReference() insolar.Reference {
	return e.parentRequestReference
}

type eventRouteCall struct {
	parentRequestReference insolar.Reference
	parentObjectReference  insolar.Reference

	method    string
	arguments []byte
	object    insolar.Reference
	prototype insolar.Reference
	immutable bool
	saga      bool
}

func (e eventRouteCall) Saga() bool {
	return e.saga
}

func (e eventRouteCall) Immutable() bool {
	return e.immutable
}

func (e eventRouteCall) Prototype() insolar.Reference {
	return e.prototype
}

func (e eventRouteCall) Object() insolar.Reference {
	return e.object
}

func (e eventRouteCall) Arguments() []byte {
	return e.arguments
}

func (e eventRouteCall) Method() string {
	return e.method
}

func (e eventRouteCall) ParentObjectReference() insolar.Reference {
	return e.parentObjectReference
}

func (e eventRouteCall) ParentRequestReference() insolar.Reference {
	return e.parentRequestReference
}

type RPCRouteCallEvent interface {
	RPCEvent

	SetSaga(isSaga bool) RPCRouteCallEvent
	SetImmutable(isImmutable bool) RPCRouteCallEvent
}

type RPCOutgoingConstructor interface {
	ConstructOutgoing(transcript runner2.Execution) record.OutgoingRequest
}

type RPCEventParentFunc func(parentRequest insolar.Reference, parentObject insolar.Reference) RPCEventBuilder

type RPCEventBuilder interface {
	Deactivate() RPCEvent
	SaveAsChild(prototype insolar.Reference, constructor string, arguments []byte) RPCEvent
	RouteCall(object insolar.Reference, prototype insolar.Reference, method string, arguments []byte) RPCRouteCallEvent
	GetCode(code insolar.Reference) RPCEvent
}

func (e eventSaveAsChild) ConstructOutgoing(execution runner2.Execution) record.OutgoingRequest {
	return record.OutgoingRequest{
		Caller: *execution.Request.Object,
		Nonce:  0,

		CallType:  record.CTSaveAsChild,
		Base:      &e.parentObjectReference,
		Prototype: &e.prototype,
		Method:    e.constructor,
		Arguments: e.arguments,

		APIRequestID: execution.Request.APIRequestID,
		Reason:       e.parentRequestReference,
	}
}

func (e eventRouteCall) ConstructOutgoing(execution runner2.Execution) record.OutgoingRequest {
	returnMode := record.ReturnResult
	if e.saga {
		returnMode = record.ReturnSaga
	}

	return record.OutgoingRequest{
		Caller: *execution.Request.Object,
		Nonce:  0,

		CallType:   record.CTMethod,
		Base:       &e.parentObjectReference,
		Object:     &e.object,
		Prototype:  &e.prototype,
		Method:     e.method,
		Arguments:  e.arguments,
		Immutable:  e.immutable,
		ReturnMode: returnMode,

		APIRequestID: execution.Request.APIRequestID,
		Reason:       e.parentRequestReference,
	}
}

func (e eventRouteCall) SetSaga(isSaga bool) RPCRouteCallEvent {
	e.saga = isSaga
	return e
}

func (e eventRouteCall) SetImmutable(isImmutable bool) RPCRouteCallEvent {
	e.immutable = isImmutable
	return e
}

type rpcBuilder struct {
	request insolar.Reference
	object  insolar.Reference
}

func (r rpcBuilder) Deactivate() RPCEvent {
	return eventDeactivate{
		parentObjectReference:  r.object,
		parentRequestReference: r.request,
	}
}

func (r rpcBuilder) SaveAsChild(prototype insolar.Reference, constructor string, arguments []byte) RPCEvent {
	return eventSaveAsChild{
		parentRequestReference: r.request,
		parentObjectReference:  r.object,

		constructor: constructor,
		arguments:   arguments,
		prototype:   prototype,
	}
}

func (r rpcBuilder) RouteCall(
	object insolar.Reference,
	prototype insolar.Reference,
	method string,
	arguments []byte,
) RPCRouteCallEvent {
	return eventRouteCall{
		parentRequestReference: r.request,
		parentObjectReference:  r.object,

		object:    object,
		method:    method,
		arguments: arguments,
		prototype: prototype,
		immutable: false,
		saga:      false,
	}
}

func (r rpcBuilder) GetCode(code insolar.Reference) RPCEvent {
	return &eventGetCode{
		codeReference: code,
	}
}

func NewRPCBuilder(request insolar.Reference, object insolar.Reference) RPCEventBuilder {
	return &rpcBuilder{request: request, object: object}
}
