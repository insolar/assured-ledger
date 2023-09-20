package example

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

type ContractCallType uint8

const (
	_ ContractCallType = iota
	ContractCallMutable
	ContractCallImmutable
	ContractCallSaga
)

type CallResult interface {
}

type ContractRunnerService interface {
	ClassifyCall(code ArtifactBinary, method string) ContractCallType
	CallImmutableMethod(code ArtifactBinary, method string, state ArtifactBinary) CallResult
}

type ContractRunnerServiceAdapter struct {
	svc  ContractRunnerService
	exec smachine.ExecutionAdapter
}

func (a *ContractRunnerServiceAdapter) PrepareSync(ctx smachine.ExecutionContext, fn func(svc ContractRunnerService)) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(context.Context, interface{}) smachine.AsyncResultFunc {
		fn(a.svc)
		return nil
	})
}

func (a *ContractRunnerServiceAdapter) PrepareAsync(ctx smachine.ExecutionContext, fn func(svc ContractRunnerService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(context.Context, interface{}) smachine.AsyncResultFunc {
		return fn(a.svc)
	})
}

// type contractRunnerService struct {
// }
//
// func (c contractRunnerService) ClassifyCall(code ArtifactBinary, method string) ContractCallType {
// 	panic("implement me")
// 	// if code.GetCacheId()
// }
//
// func (c contractRunnerService) CallImmutableMethod(code ArtifactBinary, method string, state ArtifactBinary) CallResult {
// 	panic("implement me")
// }
