// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package executionevent

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type RPC interface{}

type GetCode interface {
	CodeReference() insolar.Reference
}

type Deactivate interface {
	ParentObjectReference() insolar.Reference
	ParentRequestReference() insolar.Reference
}

type SaveAsChild interface {
	Prototype() insolar.Reference
	Arguments() []byte
	Constructor() string
	ParentObjectReference() insolar.Reference
	ParentRequestReference() insolar.Reference
	ConstructOutgoing(transcript execution.Context) payload.VCallRequest
}

type RouteCall interface {
	Saga() bool
	Immutable() bool
	Prototype() insolar.Reference
	Object() insolar.Reference
	Arguments() []byte
	Method() string
	ParentObjectReference() insolar.Reference
	ParentRequestReference() insolar.Reference
}

type Builder interface {
	Deactivate() RPC
	SaveAsChild(prototype insolar.Reference, constructor string, arguments []byte) RPC
	RouteCall(object insolar.Reference, prototype insolar.Reference, method string, arguments []byte) RouteCallBuilder
	GetCode(code insolar.Reference) RPC
}

type builder struct {
	request insolar.Reference
	object  insolar.Reference
}

func (r builder) Deactivate() RPC {
	return deactivate{
		parentObjectReference:  r.object,
		parentRequestReference: r.request,
	}
}

func (r builder) SaveAsChild(prototype insolar.Reference, constructor string, arguments []byte) RPC {
	return saveAsChild{
		parentRequestReference: r.request,
		parentObjectReference:  r.object,

		constructor: constructor,
		arguments:   arguments,
		prototype:   prototype,
	}
}

func (r builder) RouteCall(
	object insolar.Reference,
	prototype insolar.Reference,
	method string,
	arguments []byte,
) RouteCallBuilder {
	return &routeCall{
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

func (r builder) GetCode(code insolar.Reference) RPC {
	return &getCode{
		codeReference: code,
	}
}

func NewRPCBuilder(request insolar.Reference, object insolar.Reference) Builder {
	return &builder{request: request, object: object}
}

type getCode struct {
	codeReference insolar.Reference
}

func (e getCode) CodeReference() insolar.Reference {
	return e.codeReference
}

type deactivate struct {
	parentRequestReference insolar.Reference
	parentObjectReference  insolar.Reference
}

func (e deactivate) ParentObjectReference() insolar.Reference {
	return e.parentObjectReference
}

func (e deactivate) ParentRequestReference() insolar.Reference {
	return e.parentRequestReference
}

type saveAsChild struct {
	parentRequestReference insolar.Reference
	parentObjectReference  insolar.Reference

	constructor string
	arguments   []byte
	prototype   insolar.Reference
}

func (e saveAsChild) Prototype() insolar.Reference {
	return e.prototype
}

func (e saveAsChild) Arguments() []byte {
	return e.arguments
}

func (e saveAsChild) Constructor() string {
	return e.constructor
}

func (e saveAsChild) ParentObjectReference() insolar.Reference {
	return e.parentObjectReference
}

func (e saveAsChild) ParentRequestReference() insolar.Reference {
	return e.parentRequestReference
}

func (e saveAsChild) ConstructOutgoing(execution execution.Context) payload.VCallRequest {
	panic(throw.NotImplemented())
}

type routeCall struct {
	parentRequestReference insolar.Reference
	parentObjectReference  insolar.Reference

	method    string
	arguments []byte
	object    insolar.Reference
	prototype insolar.Reference
	immutable bool
	saga      bool
}

func (e routeCall) Saga() bool {
	return e.saga
}

func (e routeCall) Immutable() bool {
	return e.immutable
}

func (e routeCall) Prototype() insolar.Reference {
	return e.prototype
}

func (e routeCall) Object() insolar.Reference {
	return e.object
}

func (e routeCall) Arguments() []byte {
	return e.arguments
}

func (e routeCall) Method() string {
	return e.method
}

func (e routeCall) ParentObjectReference() insolar.Reference {
	return e.parentObjectReference
}

func (e routeCall) ParentRequestReference() insolar.Reference {
	return e.parentRequestReference
}

type RouteCallBuilder interface {
	SetSaga(isSaga bool) RouteCallBuilder
	SetImmutable(isImmutable bool) RouteCallBuilder
}

func (e routeCall) ConstructOutgoing(execution execution.Context) payload.VCallRequest {
	panic(throw.NotImplemented())
}

func (e *routeCall) SetSaga(isSaga bool) RouteCallBuilder {
	e.saga = isSaga
	return e
}

func (e *routeCall) SetImmutable(isImmutable bool) RouteCallBuilder {
	e.immutable = isImmutable
	return e
}
