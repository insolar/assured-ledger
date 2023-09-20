package example

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smadapter"
)

/* Actual service */
type ServiceA interface {
	DoSomething(param string) string
	DoSomethingElse(param0 string, param1 int) (bool, string)
}

type implA struct {
}

func (implA) DoSomething(param string) string {
	return param
}

func (implA) DoSomethingElse(param0 string, param1 int) (bool, string) {
	return param1 != 0, param0
}

/* generated or provided adapter */
type ServiceAdapterA struct {
	svc  ServiceA
	exec smachine.ExecutionAdapter
}

func (a *ServiceAdapterA) PrepareSync(ctx smachine.ExecutionContext, fn func(svc ServiceA)) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(context.Context, interface{}) smachine.AsyncResultFunc {
		fn(a.svc)
		return nil
	})
}

func (a *ServiceAdapterA) PrepareAsync(ctx smachine.ExecutionContext, fn func(svc ServiceA) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(context.Context, interface{}) smachine.AsyncResultFunc {
		return fn(a.svc)
	})
}

func CreateServiceAdapterA() *ServiceAdapterA {
	ctx := context.Background()
	ae, ch := smadapter.NewCallChannelExecutor(ctx, 0, false, 5)
	ea := smachine.NewExecutionAdapter("ServiceA", ae)

	smachine.StartChannelWorker(ctx, ch, nil)

	return &ServiceAdapterA{implA{}, ea}
}
