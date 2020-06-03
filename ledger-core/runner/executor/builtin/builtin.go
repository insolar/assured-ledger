// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// Package builtin is implementation of builtin contracts engine
package builtin

import (
	"context"

	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

// Runner is a contract runner engine
type Runner struct {
	DescriptorRegistry map[reference.Global]interface{}
	CodeRegistry       map[string]contract.Wrapper
	CodeRefRegistry    map[reference.Global]string
	ClassRefRegistry   map[reference.Global]string
}

// New is an constructor
func New(stub common.RunnerRPCStub) *Runner {
	common.SetCurrentProxyCtx(NewProxyHelper(stub))

	descriptorRegistry := make(map[reference.Global]interface{})

	for _, classDescriptor := range builtin.InitializeClassDescriptors() {
		descriptorRegistry[classDescriptor.HeadRef()] = classDescriptor
	}
	for _, codeDescriptor := range builtin.InitializeCodeDescriptors() {
		descriptorRegistry[codeDescriptor.Ref()] = codeDescriptor
	}

	return &Runner{
		DescriptorRegistry: descriptorRegistry,
		CodeRegistry:       builtin.InitializeContractMethods(),
		CodeRefRegistry:    builtin.InitializeCodeRefs(),
		ClassRefRegistry:   builtin.InitializeClassRefs(),
	}
}

func (r *Runner) CallConstructor(
	_ context.Context,
	callCtx *call.LogicContext,
	codeRef reference.Global,
	name string,
	args []byte,
) ([]byte, []byte, error) {
	foundation.SetLogicalContext(callCtx)
	defer foundation.ClearContext()

	contractName, ok := r.CodeRefRegistry[codeRef]
	if !ok {
		return nil, nil, errors.New("failed to find contract with reference")
	}
	contract := r.CodeRegistry[contractName]

	constructorFunc, ok := contract.Constructors[name]
	if !ok {
		return nil, nil, errors.New("failed to find contracts constructor")
	}

	return constructorFunc(callCtx.Callee, args)
}

func (r *Runner) CallMethod(
	_ context.Context,
	callCtx *call.LogicContext,
	codeRef reference.Global,
	data []byte,
	method string,
	args []byte,
) ([]byte, []byte, error) {
	foundation.SetLogicalContext(callCtx)
	defer foundation.ClearContext()

	contractName, ok := r.CodeRefRegistry[codeRef]
	if !ok {
		return nil, nil, errors.New("failed to find contract with reference")
	}
	contract := r.CodeRegistry[contractName]

	methodObject, ok := contract.Methods[method]
	if !ok {
		return nil, nil, errors.New("failed to find contracts method")
	}

	return methodObject.Func(data, args)
}

func (r *Runner) ClassifyMethod(_ context.Context,
	codeRef reference.Global,
	method string) (contract.MethodIsolation, error) {

	contractName, ok := r.CodeRefRegistry[codeRef]
	if !ok {
		errInfo := struct{ Reference reference.Global }{Reference: codeRef}
		return contract.MethodIsolation{}, throw.E("failed to find contract with reference", errInfo)
	}
	contractObj := r.CodeRegistry[contractName]

	methodObject, ok := contractObj.Methods[method]
	if !ok {
		return contract.MethodIsolation{}, throw.E("failed to find contracts method")
	}

	return methodObject.Isolation, nil
}

func (r *Runner) GetDescriptor(ref reference.Global) (interface{}, error) {
	return r.DescriptorRegistry[ref], nil
}
