// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// Package foundation server implementation of smartcontract functions
package foundation

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/call"
)

// BaseContract is a base embeddable struct for all insolar contracts
type BaseContract struct {
}

// ProxyInterface interface any proxy of a contract implements
type ProxyInterface interface {
	GetReference() reference.Global
	GetClass() (reference.Global, error)
	GetCode() (reference.Global, error)
}

// BaseContractInterface is an interface to deal with any contract same way
type BaseContractInterface interface {
	GetReference() reference.Global
	GetClass() reference.Global
	GetCode() reference.Global
}

// GetReference - Returns public reference of contract
func (bc *BaseContract) GetReference() reference.Global {
	ctx := bc.getContext()
	if ctx.Callee.IsEmpty() {
		panic("context has no callee set")
	}
	return ctx.Callee
}

// GetClass - Returns class of contract
func (bc *BaseContract) GetClass() reference.Global {
	return bc.getContext().Class
}

// GetCode - Returns code of contract
func (bc *BaseContract) GetCode() reference.Global {
	return bc.getContext().Code
}

// getContext returns current calling context
func (bc *BaseContract) getContext() *call.LogicContext {
	return GetLogicalContext()
}

// SelfDestruct contract will be marked as deleted
func (bc *BaseContract) SelfDestruct() error {
	return contract.CurrentProxyCtx().DeactivateObject(bc.GetReference())
}
