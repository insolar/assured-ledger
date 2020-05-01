// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package common

import "github.com/insolar/assured-ledger/ledger-core/v2/reference"

// ProxyHelper interface with methods that are needed by contract proxies
type ProxyHelper interface {
	SystemError
	Serializer
	RouteCall(
		ref reference.Global,
		immutable bool, saga bool,
		method string, args []byte, proxyPrototype reference.Global,
	) (result []byte, err error)
	SaveAsChild(
		parentRef, classRef reference.Global, constructorName string, argsSerialized []byte,
	) (result []byte, err error)
	DeactivateObject(object reference.Global) error
	MakeErrorSerializable(error) error
}

// CurrentProxyCtx - hackish way to give proxies access to the current environment. Also,
// to avoid compiling in whole Insolar platform into every contract based on GoPlugin.
var CurrentProxyCtx ProxyHelper
