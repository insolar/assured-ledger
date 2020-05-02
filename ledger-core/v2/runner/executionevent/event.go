// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package executionevent

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type RPC interface{}

type GetCode interface {
	CodeReference() reference.Global
}

type Deactivate interface {
	ParentObjectReference() reference.Global
	ParentRequestReference() reference.Global
}

type SaveAsChild interface {
	Prototype() reference.Global
	Arguments() []byte
	Constructor() string
	ParentObjectReference() reference.Global
	ParentRequestReference() reference.Global
	ConstructOutgoing(transcript execution.Context) payload.VCallRequest
}

type RouteCall interface {
	Saga() bool
	Immutable() bool
	Prototype() reference.Global
	Object() reference.Global
	Arguments() []byte
	Method() string
	ParentObjectReference() reference.Global
	ParentRequestReference() reference.Global
}

type Builder interface {
	Deactivate() RPC
	SaveAsChild(prototype reference.Global, constructor string, arguments []byte) RPC
	RouteCall(object reference.Global, prototype reference.Global, method string, arguments []byte) RouteCallBuilder
	GetCode(code reference.Global) RPC
}

type builder struct {
	request reference.Global
	object  reference.Global
}

func (r builder) Deactivate() RPC {
	return deactivate{
		parentObjectReference:  r.object,
		parentRequestReference: r.request,
	}
}

func (r builder) SaveAsChild(prototype reference.Global, constructor string, arguments []byte) RPC {
	return saveAsChild{
		parentRequestReference: r.request,
		parentObjectReference:  r.object,

		constructor: constructor,
		arguments:   arguments,
		prototype:   prototype,
	}
}

func (r builder) RouteCall(
	object reference.Global,
	prototype reference.Global,
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

func (r builder) GetCode(code reference.Global) RPC {
	return &getCode{
		codeReference: code,
	}
}

func NewRPCBuilder(request reference.Global, object reference.Global) Builder {
	return &builder{request: request, object: object}
}

type getCode struct {
	codeReference reference.Global
}

func (e getCode) CodeReference() reference.Global {
	return e.codeReference
}

type deactivate struct {
	parentRequestReference reference.Global
	parentObjectReference  reference.Global
}

func (e deactivate) ParentObjectReference() reference.Global {
	return e.parentObjectReference
}

func (e deactivate) ParentRequestReference() reference.Global {
	return e.parentRequestReference
}

type saveAsChild struct {
	parentRequestReference reference.Global
	parentObjectReference  reference.Global

	constructor string
	arguments   []byte
	prototype   reference.Global
}

func (e saveAsChild) Prototype() reference.Global {
	return e.prototype
}

func (e saveAsChild) Arguments() []byte {
	return e.arguments
}

func (e saveAsChild) Constructor() string {
	return e.constructor
}

func (e saveAsChild) ParentObjectReference() reference.Global {
	return e.parentObjectReference
}

func (e saveAsChild) ParentRequestReference() reference.Global {
	return e.parentRequestReference
}

func (e saveAsChild) ConstructOutgoing(execution execution.Context) payload.VCallRequest {
	panic(throw.NotImplemented())
}

type routeCall struct {
	parentRequestReference reference.Global
	parentObjectReference  reference.Global

	method    string
	arguments []byte
	object    reference.Global
	prototype reference.Global
	immutable bool
	saga      bool
}

func (e routeCall) Saga() bool {
	return e.saga
}

func (e routeCall) Immutable() bool {
	return e.immutable
}

func (e routeCall) Prototype() reference.Global {
	return e.prototype
}

func (e routeCall) Object() reference.Global {
	return e.object
}

func (e routeCall) Arguments() []byte {
	return e.arguments
}

func (e routeCall) Method() string {
	return e.method
}

func (e routeCall) ParentObjectReference() reference.Global {
	return e.parentObjectReference
}

func (e routeCall) ParentRequestReference() reference.Global {
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
