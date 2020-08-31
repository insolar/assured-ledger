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
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// BaseContract is a base embeddable struct for all insolar contracts
type BaseContract struct {
	//ProxyHelper contract.ProxyHelper
	contractFoundation *ContractFoundationImpl
}

// ProxyInterface interface any proxy of a contract implements
type ProxyInterface interface {
	GetReference(BaseContractInterface) reference.Global
	GetClass(BaseContractInterface) (reference.Global, error)
	GetCode(BaseContractInterface) (reference.Global, error)
}

// BaseContractInterface is an interface to deal with any contract same way
type BaseContractInterface interface {
	GetReference() reference.Global
	GetClass() reference.Global
	GetCode() reference.Global
	Foundation() ContractFoundation
	InitFoundation(proxyHelper contract.ProxyHelper)
	ResetFoundation()
}

// ------------ ContractFoundation

type ContractFoundation interface {
	CurrentProxyCtx() contract.ProxyHelper
}

type ContractFoundationImpl struct {
	proxyHelper contract.ProxyHelper
}

// CurrentProxyCtx returns foundation
func (cf *ContractFoundationImpl) CurrentProxyCtx() contract.ProxyHelper {
	return cf.proxyHelper
}

// InitFoundation initializes foundation
func (bc *BaseContract) InitFoundation(proxyHelper contract.ProxyHelper) {
	if bc.contractFoundation != nil {
		panic(throw.IllegalState())
	}
	bc.contractFoundation = &ContractFoundationImpl{
		proxyHelper: proxyHelper,
	}
}

// ResetFoundation nullifies foundation
func (bc *BaseContract) ResetFoundation() {
	bc.contractFoundation = nil
}

// Foundation returns instance of foundation
func (bc BaseContract) Foundation() ContractFoundation {
	return bc.contractFoundation
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
	return bc.Foundation().CurrentProxyCtx().DeactivateObject(bc.GetReference())
}
