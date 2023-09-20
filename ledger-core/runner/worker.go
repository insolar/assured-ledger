package runner

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type worker struct {
	svc      *DefaultService
	ctx      context.Context
	cancelFn context.CancelFunc
}

func newWorker(ctx context.Context, svc *DefaultService) *worker {
	cancellableCtx, cancelFn := context.WithCancel(ctx)

	return &worker{
		svc:      svc,
		ctx:      cancellableCtx,
		cancelFn: cancelFn,
	}
}

func (w *worker) Run(calls <-chan smachine.AdapterCall) *worker {
	go func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			case call, ok := <-calls:
				if !ok {
					return
				}
				w.runnerAdapterWorkerIteration(call)
			}
		}
	}()

	return w
}

func (w worker) runnerAdapterWorkerIteration(call smachine.AdapterCall) {
	intercept := &runnerServiceInterceptor{svc: w.svc}
	delegationFn := func(ctx context.Context, callFn smachine.AdapterCallFunc, _ smachine.NestedCallFunc, _ *synckit.ChainedCancel) (smachine.AsyncResultFunc, error) {
		return callFn(ctx, intercept), nil
	}

	switch fn, err := call.DelegateAndSendResult(nil, delegationFn); {
	case err != nil:
		call.ReportError(err)
		if fn != nil {
			fn()
		}
	case fn == nil:
		panic(throw.NotImplemented())
	default:
		switch intercept.last.mode {
		case Start:
			w.svc.runStart(intercept.last.state, fn)
		case Continue:
			w.svc.runContinue(intercept.last.state, fn)
		case Abort:
			w.svc.runAbort(intercept.last.state, fn)
		default:
			panic(throw.IllegalValue())
		}
	}
}

func (w worker) Stop() {
	w.cancelFn()
}
