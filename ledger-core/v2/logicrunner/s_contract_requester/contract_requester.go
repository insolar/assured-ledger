// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package s_contract_requester // nolint: golint

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
)

type ContractRequesterService interface {
}

type ContractRequesterServiceAdapter struct {
	svc  ContractRequesterService
	exec smachine.ExecutionAdapter
}

func (a *ContractRequesterServiceAdapter) PrepareSync(ctx smachine.ExecutionContext, fn func(svc ContractRequesterService)) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(interface{}) smachine.AsyncResultFunc {
		fn(a.svc)
		return nil
	})
}

func (a *ContractRequesterServiceAdapter) PrepareAsync(ctx smachine.ExecutionContext, fn func(svc ContractRequesterService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(interface{}) smachine.AsyncResultFunc {
		return fn(a.svc)
	})
}

func (a *ContractRequesterServiceAdapter) PrepareNotify(ctx smachine.ExecutionContext, fn func(svc ContractRequesterService)) smachine.NotifyRequester {
	return a.exec.PrepareNotify(ctx, func(interface{}) { fn(a.svc) })
}
