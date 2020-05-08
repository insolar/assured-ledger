// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package builtin

import (
	"reflect"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common/rpctypes"
)

type ProxyHelper struct {
	common.Serializer
	common.SystemError
	methods common.RunnerRPCStub
}

func NewProxyHelper(runner common.RunnerRPCStub) *ProxyHelper {
	return &ProxyHelper{
		Serializer:  common.NewCBORSerializer(),
		SystemError: common.NewSystemError(),
		methods:     runner,
	}
}

func (h *ProxyHelper) getUpBaseReq() rpctypes.UpBaseReq {
	callContext := foundation.GetLogicalContext()

	return rpctypes.UpBaseReq{
		Mode:            callContext.Mode,
		Callee:          callContext.Callee,
		CalleePrototype: callContext.CallerPrototype,
		Request:         callContext.Request,
		ID:              callContext.ID,
	}
}

func (h *ProxyHelper) CallMethod(ref reference.Global, unordered bool, _ bool, method string, args []byte,
	proxyPrototype reference.Global) ([]byte, error) {

	if h.GetSystemError() != nil {
		return nil, h.GetSystemError()
	}

	res := rpctypes.UpRouteResp{}
	req := rpctypes.UpCallMethodReq{
		UpBaseReq: h.getUpBaseReq(),

		Object:    ref,
		Unordered: unordered,
		Method:    method,
		Arguments: args,
		Prototype: proxyPrototype,
	}

	err := h.methods.CallMethod(req, &res)
	if err != nil {
		h.SetSystemError(err)
		return nil, err
	}

	return res.Result, nil
}

func (h *ProxyHelper) CallConstructor(
	parentRef, classRef reference.Global,
	constructorName string, argsSerialized []byte,
) (
	[]byte, error,
) {
	if !parentRef.IsObjectReference() {
		return nil, errors.Errorf("Failed to save AsChild: objRef should be ObjectReference; ref=%s", parentRef.String())
	}

	if h.GetSystemError() != nil {
		// There was a system error during execution of the contract.
		// Immediately return this error to the calling contract - any
		// results will not be registered on LME anyway.
		return nil, h.GetSystemError()
	}

	res := rpctypes.UpCallConstructorResp{}
	req := rpctypes.UpCallConstructorReq{
		UpBaseReq: h.getUpBaseReq(),

		Parent:          parentRef,
		Prototype:       classRef,
		ConstructorName: constructorName,
		ArgsSerialized:  argsSerialized,
	}

	err := h.methods.CallConstructor(req, &res)
	if err != nil {
		h.SetSystemError(err)
		return nil, err
	}

	return res.Result, nil
}

func (h *ProxyHelper) DeactivateObject(object reference.Global) error {
	if h.GetSystemError() != nil {
		return h.GetSystemError()
	}

	res := rpctypes.UpDeactivateObjectResp{}
	req := rpctypes.UpDeactivateObjectReq{
		UpBaseReq: h.getUpBaseReq(),
	}

	if err := h.methods.DeactivateObject(req, &res); err != nil {
		h.SetSystemError(err)
		return err
	}
	return nil
}

func (h *ProxyHelper) MakeErrorSerializable(err error) error {
	if err == nil || err == (*foundation.Error)(nil) || reflect.ValueOf(err).IsNil() {
		return nil
	}
	return &foundation.Error{S: err.Error()}
}
