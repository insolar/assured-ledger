// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package builtin

import (
	"reflect"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/builtin/foundation"
	lrCommon "github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/goplugin/rpctypes"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type ProxyHelper struct {
	lrCommon.Serializer
	lrCommon.SystemError
	methods lrCommon.LogicRunnerRPCStub
}

func NewProxyHelper(runner lrCommon.LogicRunnerRPCStub) *ProxyHelper {
	return &ProxyHelper{
		Serializer:  lrCommon.NewCBORSerializer(),
		SystemError: lrCommon.NewSystemError(),
		methods:     runner,
	}
}

func (h *ProxyHelper) getUpBaseReq() rpctypes.UpBaseReq {
	callContext := foundation.GetLogicalContext()

	return rpctypes.UpBaseReq{
		Mode:            callContext.Mode,
		Callee:          *callContext.Callee,
		CalleePrototype: *callContext.CallerPrototype,
		Request:         *callContext.Request,
	}
}

func (h *ProxyHelper) RouteCall(ref reference.Global, immutable bool, saga bool, method string, args []byte,
	proxyPrototype reference.Global) ([]byte, error) {

	if h.GetSystemError() != nil {
		return nil, h.GetSystemError()
	}

	res := rpctypes.UpRouteResp{}
	req := rpctypes.UpRouteReq{
		UpBaseReq: h.getUpBaseReq(),

		Object:    ref,
		Immutable: immutable,
		Saga:      saga,
		Method:    method,
		Arguments: args,
		Prototype: proxyPrototype,
	}

	err := h.methods.RouteCall(req, &res)
	if err != nil {
		h.SetSystemError(err)
		return nil, err
	}

	return res.Result, nil
}

func (h *ProxyHelper) SaveAsChild(
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

	res := rpctypes.UpSaveAsChildResp{}
	req := rpctypes.UpSaveAsChildReq{
		UpBaseReq: h.getUpBaseReq(),

		Parent:          parentRef,
		Prototype:       classRef,
		ConstructorName: constructorName,
		ArgsSerialized:  argsSerialized,
	}

	err := h.methods.SaveAsChild(req, &res)
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

/*
func (h *ProxyHelper) Serialize(what interface{}, to *[]byte) error {
	panic("implement me")
}

func (h *ProxyHelper) Deserialize(from []byte, into interface{}) error {
	panic("implement me")
}
*/

func (h *ProxyHelper) MakeErrorSerializable(err error) error {
	if err == nil || err == (*foundation.Error)(nil) || reflect.ValueOf(err).IsNil() {
		return nil
	}
	return &foundation.Error{S: err.Error()}
}
