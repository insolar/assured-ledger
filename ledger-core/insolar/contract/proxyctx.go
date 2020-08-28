// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package contract

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common"
)

// ProxyHelper interface with methods that are needed by contract proxies
type ProxyHelper interface {
	common.SystemError
	common.Serializer
	CallMethod(
		ref reference.Global,
		tolerance isolation.InterferenceFlag, isolation isolation.StateFlag, saga bool,
		method string, args []byte, proxyClass reference.Global,
	) (result []byte, err error)
	CallConstructor(
		parentRef, classRef reference.Global, constructorName string, argsSerialized []byte,
	) (result []byte, err error)
	DeactivateObject(object reference.Global) error
	MakeErrorSerializable(error) error
}

// CurrentProxyCtx - hackish way to give proxies access to the current environment. Also,
// to avoid compiling in whole Insolar platform into every contract based on GoPlugin.
var (
	currentProxyCtxLock sync.RWMutex
	currentProxyCtx     ProxyHelper
)

func CurrentProxyCtx() ProxyHelper {
	currentProxyCtxLock.RLock()
	defer currentProxyCtxLock.RUnlock()

	return currentProxyCtx
}

func SetCurrentProxyCtx(ctx ProxyHelper) {
	currentProxyCtxLock.Lock()
	defer currentProxyCtxLock.Unlock()

	currentProxyCtx = ctx
}
