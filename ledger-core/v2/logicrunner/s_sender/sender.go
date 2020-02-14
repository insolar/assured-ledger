// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package s_sender

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
)

// TODO[bigbes]: port it to state machine
type SenderService interface {
	bus.Sender
}

type SenderServiceAdapter struct {
	svc  SenderService
	exec smachine.ExecutionAdapter
}

func (a *SenderServiceAdapter) PrepareSync(ctx smachine.ExecutionContext, fn func(svc SenderService)) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(interface{}) smachine.AsyncResultFunc {
		fn(a.svc)
		return nil
	})
}

func (a *SenderServiceAdapter) PrepareAsync(ctx smachine.ExecutionContext, fn func(svc SenderService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(interface{}) smachine.AsyncResultFunc {
		return fn(a.svc)
	})
}

func (a *SenderServiceAdapter) PrepareNotify(ctx smachine.ExecutionContext, fn func(svc SenderService)) smachine.NotifyRequester {
	return a.exec.PrepareNotify(ctx, func(interface{}) { fn(a.svc) })
}

type senderService struct {
	bus.Sender
	Accessor pulse.Accessor
}

func CreateSenderService(sender bus.Sender, accessor pulse.Accessor) *SenderServiceAdapter {
	ctx := context.Background()
	ae, ch := smachine.NewCallChannelExecutor(ctx, -1, false, 16)
	smachine.StartChannelWorkerParallelCalls(ctx, 0, ch, nil)

	return &SenderServiceAdapter{
		svc: senderService{
			Sender:   sender,
			Accessor: accessor,
		},
		exec: smachine.NewExecutionAdapter("Sender", ae),
	}
}
