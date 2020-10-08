// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datareader

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewReader(request *rms.LReadRequest) smachine.SubroutineStateMachine {
	if request == nil {
		panic(throw.IllegalValue())
	}
	return &SMLineReader{ request: request}
}

var _ smachine.SubroutineStateMachine = &SMLineReader{}
type SMLineReader struct {
	smachine.StateMachineDeclTemplate

	// input
	request *rms.LReadRequest

	// inject
	readAdapter readersvc.Adapter

	// runtime - batched read
	ready   bool
	// cabinet readersvc.Cabinet
}

func (p *SMLineReader) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMLineReader) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	// injector.MustInject(&p.pulseSlot)
	injector.MustInject(&p.readAdapter)
	// injector.MustInject(&p.reader)
}

func (p *SMLineReader) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMLineReader) GetSubroutineInitState(smachine.SubroutineStartContext) smachine.InitFunc {
	return p.stepInit
}

func (p *SMLineReader) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	if p.readAdapter.ServiceNeedsBatching() {
		p.readAdapter = readersvc.Adapter{}
		return ctx.Jump(p.stepBatchRead)
	}
	return ctx.Jump(p.stepDirectRead)
}

func (p *SMLineReader) stepDirectRead(ctx smachine.ExecutionContext) smachine.StateUpdate {
	pn := p.request.TargetStartRef.GetPulseOfLocal()

	return p.readAdapter.PrepareAsync(ctx, func(svc readersvc.Service) smachine.AsyncResultFunc {
		cab, err := svc.FindCabinet(pn)
		switch {
		case err != nil:
			panic(err)
		case cab != nil:
			if err = cab.Open(); err != nil {
				panic(err)
			}
		}

		if err = readDataFromCabinet(svc, cab, 0); err != nil {
			panic(err)
		}

		return func(ctx smachine.AsyncResultContext) {
			p.ready = true
		}
	}).DelayedStart().Sleep().ThenJump(p.stepDone)
}

func readDataFromCabinet(svc readersvc.Service, cab readersvc.Cabinet, id jet.DropID) (err error) {
	defer func() {
		switch closeErr := cab.Close(); {
		case closeErr == nil:
		case err == nil:
			err = closeErr
		default:
			err = throw.WithDetails(closeErr, err)
		}

		err = throw.RW(recover(), err, "readDataFromCabinet")
	}()

	return svc.ReadFromCabinet(cab, id, func(reader readbundle.Reader) error {
		panic(throw.NotImplemented()) // TODO
	})
}

func (p *SMLineReader) stepBatchRead(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// ctx.SetFinalizer(func(ctx smachine.FinalizationContext) {
	// 	if p.cabinet != nil {
	// 		if err := p.cabinet.Close(); err != nil {
	// 			ctx.Log().Error("reader close", err)
	// 		}
	// 	}
	// })
	//

	panic(throw.NotImplemented()) // TODO
}

// func (p *SMLineReader) stepReadData(ctx smachine.ExecutionContext) smachine.StateUpdate {
// 	switch {
// 	case !p.ready:
// 		return ctx.Sleep().ThenRepeat()
// 	case p.cabinet == nil:
// 		return ctx.Error(throw.E("cabinet is not available"))
// 	}
//
// 	// // ledger.ControlSection
// 	// loc, err := p.reader.FindDirectoryEntry(ledger.DefaultEntrySection, p.request.TargetRef.Get())
// 	// switch {
// 	// case err != nil:
// 	// 	return ctx.Error(err)
// 	// case loc == 0:
// 	// 	return ctx.Error(throw.E("filament not found"))
// 	// }
//
// 	panic(throw.NotImplemented()) // TODO
// }

func (p *SMLineReader) stepDone(ctx smachine.ExecutionContext) smachine.StateUpdate {
	panic(throw.NotImplemented()) // TODO
}
