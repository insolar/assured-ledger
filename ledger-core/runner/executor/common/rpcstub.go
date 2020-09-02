// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package common

import (
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/rpctypes"
)

type RunnerRPCStub interface {
	CallMethod(rpctypes.UpCallMethodReq, *rpctypes.UpCallMethodResp) error
	CallConstructor(rpctypes.UpCallConstructorReq, *rpctypes.UpCallConstructorResp) error
	DeactivateObject(rpctypes.UpDeactivateObjectReq, *rpctypes.UpDeactivateObjectResp) error
}
