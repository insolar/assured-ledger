// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// Package builtin is implementation of builtin contracts engine
package builtin

import (
	"context"
	"errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common/foundation"
)

// BuiltIn is a contract runner engine
type BuiltIn struct {
	CodeRegistry         map[string]insolar.ContractWrapper
	CodeRefRegistry      map[insolar.Reference]string
	PrototypeRefRegistry map[insolar.Reference]string
}

// NewBuiltIn is an constructor
func NewBuiltIn(_ interface{}, stub common.LogicRunnerRPCStub) *BuiltIn {
	common.CurrentProxyCtx = NewProxyHelper(stub)

	return &BuiltIn{
		CodeRefRegistry: builtin.InitializeCodeRefs(),
		CodeRegistry:    builtin.InitializeContractMethods(),
	}
}

func (b *BuiltIn) CallConstructor(
	_ context.Context,
	callCtx *insolar.LogicCallContext,
	codeRef insolar.Reference,
	name string,
	args insolar.Arguments,
) ([]byte, insolar.Arguments, error) {
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

	objRef := insolar.NewReference(*callCtx.Request.GetLocal())
	return constructorFunc(*objRef, args)
}

func (b *BuiltIn) CallMethod(
	_ context.Context,
	callCtx *insolar.LogicCallContext,
	codeRef insolar.Reference,
	data []byte,
	method string,
	args insolar.Arguments,
) ([]byte, insolar.Arguments, error) {
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

	return methodFunc(data, args)
}
