// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// Package builtin is implementation of builtin contracts engine
package builtin

import (
	"context"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common/foundation"
)

// Runner is a contract runner engine
type Runner struct {
	DescriptorRegistry   map[reference.Global]interface{}
	CodeRegistry         map[string]insolar.ContractWrapper
	CodeRefRegistry      map[reference.Global]string
	PrototypeRefRegistry map[reference.Global]string
}

// New is an constructor
func New(stub common.RunnerRPCStub) *Runner {
	common.CurrentProxyCtx = NewProxyHelper(stub)

	descriptorRegistry := make(map[reference.Global]interface{})

	for _, prototypeDescriptor := range builtin.InitializePrototypeDescriptors() {
		descriptorRegistry[prototypeDescriptor.HeadRef()] = prototypeDescriptor
	}
	for _, codeDescriptor := range builtin.InitializeCodeDescriptors() {
		descriptorRegistry[codeDescriptor.Ref()] = codeDescriptor
	}

	return &Runner{
		DescriptorRegistry:   descriptorRegistry,
		CodeRegistry:         builtin.InitializeContractMethods(),
		CodeRefRegistry:      builtin.InitializeCodeRefs(),
		PrototypeRefRegistry: builtin.InitializePrototypeRefs(),
	}
}

func (r *Runner) CallConstructor(
	_ context.Context,
	callCtx *insolar.LogicCallContext,
	codeRef reference.Global,
	name string,
	args insolar.Arguments,
) ([]byte, insolar.Arguments, error) {
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
	callCtx *insolar.LogicCallContext,
	codeRef reference.Global,
	data []byte,
	method string,
	args insolar.Arguments,
) ([]byte, insolar.Arguments, error) {
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

	if methodObject.Unordered != callCtx.Unordered {
		orderedUnordered := func(unordered bool) string {
			if unordered {
				return "Unordered"
			}
			return "Ordered"
		}

		return nil, nil, errors.Errorf("calling %s method as %s",
			orderedUnordered(methodObject.Unordered),
			orderedUnordered(callCtx.Unordered),
		)
	}

	return methodObject.Func(data, args)
}

func (r *Runner) GetDescriptor(ref reference.Global) (interface{}, error) {
	return r.DescriptorRegistry[ref], nil
}
