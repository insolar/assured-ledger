package common

import (
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/rpctypes"
)

type RunnerRPCStub interface {
	CallMethod(rpctypes.UpCallMethodReq, *rpctypes.UpCallMethodResp) error
	CallConstructor(rpctypes.UpCallConstructorReq, *rpctypes.UpCallConstructorResp) error
	DeactivateObject(rpctypes.UpDeactivateObjectReq, *rpctypes.UpDeactivateObjectResp) error
}
