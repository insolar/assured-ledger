// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package outgoing

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/common"
)

type RPCEvent interface{}

type RPCRouteCallEvent interface {
	RPCEvent

	SetSaga(isSaga bool) RPCRouteCallEvent
	SetImmutable(isImmutable bool) RPCRouteCallEvent
}

type RPCOutgoingConstructor interface {
	ConstructOutgoing(transcript common.Transcript) record.OutgoingRequest
}

type RPCEventParentFunc func(parentRequest insolar.Reference, parentObject insolar.Reference) RPCEventBuilder

type RPCEventBuilder interface {
	Deactivate() RPCEvent
	SaveAsChild(prototype insolar.Reference, constructor string, arguments []byte) RPCEvent
	RouteCall(object insolar.Reference, prototype insolar.Reference, method string, arguments []byte) RPCRouteCallEvent
	GetCode(code insolar.Reference) RPCEvent
}

type SaveAsChildEvent struct {
	ParentRequestReference insolar.Reference
	ParentObjectReference  insolar.Reference

	Constructor string
	Arguments   []byte
	Prototype   insolar.Reference
}

func (e SaveAsChildEvent) ConstructOutgoing(transcript common.Transcript) record.OutgoingRequest {
	return record.OutgoingRequest{
		Caller: *transcript.Request.Object,
		Nonce:  0,

		CallType:  record.CTSaveAsChild,
		Base:      &e.ParentObjectReference,
		Prototype: &e.Prototype,
		Method:    e.Constructor,
		Arguments: e.Arguments,

		APIRequestID: transcript.Request.APIRequestID,
		Reason:       e.ParentRequestReference,
	}
}

type DeactivateEvent struct {
	ParentRequestReference insolar.Reference
	ParentObjectReference  insolar.Reference
}

type RouteCallEvent struct {
	ParentRequestReference insolar.Reference
	ParentObjectReference  insolar.Reference

	Method    string
	Arguments []byte
	Object    insolar.Reference
	Prototype insolar.Reference
	Immutable bool
	Saga      bool
}

func (e RouteCallEvent) ConstructOutgoing(transcript common.Transcript) record.OutgoingRequest {
	returnMode := record.ReturnResult
	if e.Saga {
		returnMode = record.ReturnSaga
	}

	return record.OutgoingRequest{
		Caller: *transcript.Request.Object,
		Nonce:  0,

		CallType:   record.CTMethod,
		Base:       &e.ParentObjectReference,
		Object:     &e.Object,
		Prototype:  &e.Prototype,
		Method:     e.Method,
		Arguments:  e.Arguments,
		Immutable:  e.Immutable,
		ReturnMode: returnMode,

		APIRequestID: transcript.Request.APIRequestID,
		Reason:       e.ParentRequestReference,
	}
}

func (e RouteCallEvent) SetSaga(isSaga bool) RPCRouteCallEvent {
	e.Saga = isSaga
	return e
}

func (e RouteCallEvent) SetImmutable(isImmutable bool) RPCRouteCallEvent {
	e.Immutable = isImmutable
	return e
}

type GetCodeEvent struct {
	CodeReference insolar.Reference
}

type rpcBuilder struct {
	request insolar.Reference
	object  insolar.Reference
}

func (r rpcBuilder) Deactivate() RPCEvent {
	return DeactivateEvent{
		ParentObjectReference:  r.object,
		ParentRequestReference: r.request,
	}
}

func (r rpcBuilder) SaveAsChild(prototype insolar.Reference, constructor string, arguments []byte) RPCEvent {
	return SaveAsChildEvent{
		ParentRequestReference: r.request,
		ParentObjectReference:  r.object,

		Constructor: constructor,
		Arguments:   arguments,
		Prototype:   prototype,
	}
}

func (r rpcBuilder) RouteCall(
	object insolar.Reference,
	prototype insolar.Reference,
	method string,
	arguments []byte,
) RPCRouteCallEvent {
	return RouteCallEvent{
		ParentRequestReference: r.request,
		ParentObjectReference:  r.object,

		Object:    object,
		Method:    method,
		Arguments: arguments,
		Prototype: prototype,
		Immutable: false,
		Saga:      false,
	}
}

func (r rpcBuilder) GetCode(code insolar.Reference) RPCEvent {
	return &GetCodeEvent{
		CodeReference: code,
	}
}

func NewRPCBuilder(request insolar.Reference, object insolar.Reference) RPCEventBuilder {
	return &rpcBuilder{request: request, object: object}
}
