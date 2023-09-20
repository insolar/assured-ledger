package datareader

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/dataextractor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewReader(cfg dataextractor.Config) smachine.SubroutineStateMachine {
	cfg.Ensure()
	return &SMLineReader{cfg: cfg}
}

var _ smachine.SubroutineStateMachine = &SMLineReader{}
type SMLineReader struct {
	smachine.StateMachineDeclTemplate

	// input
	cfg dataextractor.Config

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
	cfg := p.cfg

	return p.readAdapter.PrepareAsync(ctx, func(svc readersvc.Service) smachine.AsyncResultFunc {
		cab, err := svc.FindCabinet(cfg.Target)
		switch {
		case err != nil:
			panic(err)
		case cab != nil:
			if err = cab.Open(); err != nil {
				panic(err)
			}
		}

		err = readDataFromCabinet(svc, cab, cfg, 0)

		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("read failed", err)
				panic(err)
			}
			p.ready = true
		}
	}).DelayedStart().Sleep().ThenJump(p.stepDone)
}

func readDataFromCabinet(svc readersvc.Service, cab readersvc.Cabinet, cfg dataextractor.Config, id jet.DropID) (err error) {
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

	if cfg.Selector.Direction.IsToPast() {
		return svc.ReadFromCabinet(cab, id, prevReader{ cfg }.ReadData)
	}
	return svc.ReadFromCabinet(cab, id, nextReader{ cfg }.ReadData)
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

func (p *SMLineReader) stepDone(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Stop()
}
