// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// Package builtin is implementation of builtin contracts engine
package builtin

import (
	"context"
	"errors"
	"time"

	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/insmetrics"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/builtin/foundation"
	lrCommon "github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/goplugin/rpctypes"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/metrics"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type LogicRunnerRPCStub interface {
	GetCode(rpctypes.UpGetCodeReq, *rpctypes.UpGetCodeResp) error
	RouteCall(rpctypes.UpRouteReq, *rpctypes.UpRouteResp) error
	SaveAsChild(rpctypes.UpSaveAsChildReq, *rpctypes.UpSaveAsChildResp) error
	DeactivateObject(rpctypes.UpDeactivateObjectReq, *rpctypes.UpDeactivateObjectResp) error
}

// BuiltIn is a contract runner engine
type BuiltIn struct {
	// Prototype -> Code + Versions
	// PrototypeRegistry    map[string]preprocessor.ContractWrapper
	// PrototypeRefRegistry map[reference.Global]string
	// Code ->
	CodeRegistry         map[string]insolar.ContractWrapper
	CodeRefRegistry      map[reference.Global]string
	PrototypeRefRegistry map[reference.Global]string
}

// NewBuiltIn is an constructor
func NewBuiltIn(stub LogicRunnerRPCStub) *BuiltIn {
	lrCommon.CurrentProxyCtx = NewProxyHelper(stub)

	return &BuiltIn{
		CodeRefRegistry: builtin.InitializeCodeRefs(),
		CodeRegistry:    builtin.InitializeContractMethods(),
	}
}

func (b *BuiltIn) CallConstructor(
	ctx context.Context, callCtx *insolar.LogicCallContext, codeRef reference.Global,
	name string, args insolar.Arguments,
) ([]byte, insolar.Arguments, error) {
	executeStart := time.Now()
	ctx = insmetrics.InsertTag(ctx, metrics.TagContractPrototype, b.PrototypeRefRegistry[codeRef])
	ctx = insmetrics.InsertTag(ctx, metrics.TagContractMethodName, "Constructor")
	defer func(ctx context.Context) {

		stats.Record(ctx, metrics.ContractExecutionTime.M(float64(time.Since(executeStart).Nanoseconds())/1e6))
	}(ctx)

	ctx, span := instracer.StartSpan(ctx, "builtin.CallConstructor")
	defer span.Finish()

	foundation.SetLogicalContext(callCtx)
	defer foundation.ClearContext()

	contractName, ok := b.CodeRefRegistry[codeRef]
	if !ok {
		return nil, nil, errors.New("failed to find contract with reference")
	}
	contract := b.CodeRegistry[contractName]

	constructorFunc, ok := contract.Constructors[name]
	if !ok {
		return nil, nil, errors.New("failed to find contracts method")
	}

	objRef := reference.NewSelf(callCtx.Request.GetLocal())
	return constructorFunc(objRef, args)
}

func (b *BuiltIn) CallMethod(ctx context.Context, callCtx *insolar.LogicCallContext, codeRef reference.Global,
	data []byte, method string, args insolar.Arguments) ([]byte, insolar.Arguments, error) {
	executeStart := time.Now()
	ctx = insmetrics.InsertTag(ctx, metrics.TagContractPrototype, b.PrototypeRefRegistry[codeRef])
	ctx = insmetrics.InsertTag(ctx, metrics.TagContractMethodName, method)
	defer func(ctx context.Context) {
		stats.Record(ctx, metrics.ContractExecutionTime.M(float64(time.Since(executeStart).Nanoseconds())/1e6))
	}(ctx)

	ctx, span := instracer.StartSpan(ctx, "builtin.CallMethod")
	defer span.Finish()

	foundation.SetLogicalContext(callCtx)
	defer foundation.ClearContext()

	contractName, ok := b.CodeRefRegistry[codeRef]
	if !ok {
		return nil, nil, errors.New("failed to find contract with reference")
	}
	contract := b.CodeRegistry[contractName]

	methodFunc, ok := contract.Methods[method]
	if !ok {
		return nil, nil, errors.New("failed to find contracts method")
	}

	return methodFunc.Func(data, args)
}
