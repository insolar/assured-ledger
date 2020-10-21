// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package requests

import (
	"runtime"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datareader"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datawriter"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewSMRead(request *rms.LReadRequest) *SMRead {
	return &SMRead{
		request: request,
	}
}

var _ smachine.StateMachine = &SMRead{}
type SMRead struct {
	smachine.StateMachineDeclTemplate

	// input
	request *rms.LReadRequest

	// injected
	pulseSlot *conveyor.PulseSlot
}

func (p *SMRead) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMRead) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&p.pulseSlot)
}

func (p *SMRead) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMRead) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(p.migrateOnce)
	ctx.SetDefaultErrorHandler(p.handleError)

	switch ps := p.pulseSlot.State(); {
	case p.request == nil:
		return ctx.Error(throw.E("missing request"))
	case p.request.TargetRef.IsEmpty():
		return ctx.Error(throw.E("missing target"))
	case ps == conveyor.Antique:
		return ctx.Jump(p.stepCleanRead)
	default:
		return ctx.Jump(p.stepDirtyRead)
	}
}

func (p *SMRead) stepCleanRead(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.CallSubroutine(datareader.NewReader(p.request), nil, p.subrReadDone)
}

func (p *SMRead) stepDirtyRead(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.CallSubroutine(datawriter.NewDirtyReader(p.request, p.pulseSlot.PulseNumber()), nil, p.subrReadDone)
}

func (p *SMRead) subrReadDone(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
	if err := ctx.GetError(); err != nil {
		return ctx.Error(err)
	}
	if ctx.EventParam() == nil {
		return ctx.Error(throw.FailHere("missing response"))
	}
	return ctx.Jump(p.stepSendResponse)
}

func (p *SMRead) stepSendResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	response := ctx.GetTerminationResult().(*rms.LReadResponse)
	runtime.KeepAlive(response)

	// TODO send response back

	return ctx.Stop()
}

func (p *SMRead) handleError(smachine.FailureContext) {
	// TODO implement failure send
}

func (p *SMRead) migrateOnce(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(p.migrateTwice)
	return ctx.Stay()
}

func (p *SMRead) migrateTwice(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Error(throw.FailHere("expired"))
}

