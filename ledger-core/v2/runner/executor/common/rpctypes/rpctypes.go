// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rpctypes

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/machine"
)

// Types for RPC requests and responses between goplugin and goinsider.
// Calls from goplugin to goinsider go "downwards" and names are
// prefixed with "Down". Reverse calls go "upwards", so "Up" prefix

// todo it may use foundation.Context
// DownCallMethodReq is a set of arguments for CallMethod RPC in the runner
type DownCallMethodReq struct {
	Context   *call.LogicContext
	Code      reference.Global
	Data      []byte
	Method    string
	Arguments Arguments
}

// DownCallMethodResp is response from CallMethod RPC in the runner
type DownCallMethodResp struct {
	Data []byte
	Ret  Arguments
}

// DownCallConstructorReq is a set of arguments for CallConstructor RPC
// in the runner
type DownCallConstructorReq struct {
	Context   *call.LogicContext
	Code      reference.Global
	Name      string
	Arguments Arguments
}

// DownCallConstructorResp is response from CallConstructor RPC in the runner
type DownCallConstructorResp struct {
	Data []byte
	Ret  Arguments
}

// UpBaseReq  is a base type for all insgorund -> logicrunner requests
type UpBaseReq struct {
	Mode            insolar.CallMode
	Callee          reference.Global
	CalleePrototype reference.Global
	Request         reference.Global
	ID              call.ID
}

// UpRespIface interface for UpBaseReq descendant responses
type UpRespIface interface{}

// UpGetCodeReq is a set of arguments for GetCode RPC in goplugin
type UpGetCodeReq struct {
	UpBaseReq
	MType machine.Type
	Code  reference.Global
}

// UpGetCodeResp is response from GetCode RPC in goplugin
type UpGetCodeResp struct {
	Code []byte
}

// UpCallMethodReq is a set of arguments for Send RPC in goplugin
type UpCallMethodReq struct {
	UpBaseReq
	Tolerance payload.ToleranceFlag
	Isolation payload.StateFlag
	Saga      bool
	Object    reference.Global
	Method    string
	Arguments Arguments
	Prototype reference.Global
}

// UpCallMethodResp is response from Send RPC in goplugin
type UpCallMethodResp struct {
	Result Arguments
}

// UpCallConstructorReq is a set of arguments for CallConstructor RPC in goplugin
type UpCallConstructorReq struct {
	UpBaseReq
	Parent          reference.Global
	Prototype       reference.Global
	ConstructorName string
	ArgsSerialized  []byte
}

// UpCallConstructorResp is a set of arguments for CallConstructor RPC in goplugin
type UpCallConstructorResp struct {
	Result Arguments
}

// UpDeactivateObjectReq is a set of arguments for DeactivateObject RPC in goplugin
type UpDeactivateObjectReq struct {
	UpBaseReq
}

// UpDeactivateObjectResp is response from DeactivateObject RPC in goplugin
type UpDeactivateObjectResp struct {
}
