// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md.

package common

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common/rpctypes"
)

type LogicRunnerRPCStub interface {
	GetCode(rpctypes.UpGetCodeReq, *rpctypes.UpGetCodeResp) error
	RouteCall(rpctypes.UpRouteReq, *rpctypes.UpRouteResp) error
	SaveAsChild(rpctypes.UpSaveAsChildReq, *rpctypes.UpSaveAsChildResp) error
	DeactivateObject(rpctypes.UpDeactivateObjectReq, *rpctypes.UpDeactivateObjectResp) error
}

// ProxyHelper interface with methods that are needed by contract proxies
type ProxyHelper interface {
	SystemError
	Serializer
	RouteCall(
		ref insolar.Reference,
		immutable bool, saga bool,
		method string, args []byte, proxyPrototype insolar.Reference,
	) (result []byte, err error)
	SaveAsChild(
		parentRef, classRef insolar.Reference, constructorName string, argsSerialized []byte,
	) (result []byte, err error)
	DeactivateObject(object insolar.Reference) error
	MakeErrorSerializable(error) error
}

// CurrentProxyCtx - hackish way to give proxies access to the current environment. Also,
// to avoid compiling in whole Insolar platform into every contract based on GoPlugin.
var CurrentProxyCtx ProxyHelper
