// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package contract

import (
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
