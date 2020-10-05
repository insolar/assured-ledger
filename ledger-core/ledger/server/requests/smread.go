// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package requests

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datareader"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datawriter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
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
	cataloger datawriter.LineCataloger
	reader    buildersvc.ReadAdapter

	// runtime
	sdl       datawriter.LineDataLink
	dropID    jet.DropID
	extractor datareader.SequenceExtractor
}

func (p *SMRead) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMRead) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&p.pulseSlot)
	injector.MustInject(&p.cataloger)
	injector.MustInject(&p.reader)
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
		panic(throw.NotImplemented()) // TODO alternative reader
	case p.request.Flags & (rms.ReadFlags_ExplicitRedirection|rms.ReadFlags_IgnoreRedirection) != 0:
		panic(throw.NotImplemented())
	}

	if tpn := p.request.TargetRef.Get().GetLocal().Pulse(); tpn != p.pulseSlot.PulseNumber() {
		return ctx.Error(throw.E("wrong target pulse", struct { TargetPN, SlotPN pulse.Number }{ tpn, p.pulseSlot.PulseNumber() }))
	}

	// p.request.LimitCount
	// p.request.LimitSize
	// p.request.LimitRef
	// p.request.Flags

	p.extractor = &datareader.WholeExtractor{ReadAll: true}

	return ctx.Jump(p.stepFindLine)
}

func (p *SMRead) stepFindLine(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.sdl.IsZero() {
		normTargetRef := reference.NormCopy(p.request.TargetRef.Get())
		lineRef := reference.NewSelf(normTargetRef.GetBase())

		p.sdl = p.cataloger.Get(ctx, lineRef)
		if p.sdl.IsZero() {
			// line is absent
			return ctx.Jump(p.stepSendResponse)
		}
	}

	var activator smachine.SyncLink
	switch p.sdl.TryAccess(ctx, func(sd *datawriter.LineSharedData) (wakeup bool) {
		activator = sd.GetActiveSync()
		return false
	}) {
	case smachine.Passed:
		//
	case smachine.NotPassed:
		return ctx.WaitShared(p.sdl.SharedDataLink).ThenRepeat()
	default:
		panic(throw.IllegalState())
	}

	if ctx.Acquire(activator) {
		ctx.ReleaseAll()
		return ctx.Jump(p.stepLineIsReady)
	}

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if ctx.Acquire(activator) {
			ctx.ReleaseAll()
			return ctx.Jump(p.stepLineIsReady)
		}
		return ctx.Sleep().ThenRepeat()
	})
}

func (p *SMRead) stepLineIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var future *buildersvc.Future

	switch p.sdl.TryAccess(ctx, func(sd *datawriter.LineSharedData) (wakeup bool) {
		if !sd.IsValid() {
			return
		}
		p.dropID = sd.DropID()

		sd.TrimStages()

		normTargetRef := reference.NormCopy(p.request.TargetRef.Get())
		_, future = sd.FindSequence(normTargetRef, p.extractor.AddLineRecord)

		return false
	}) {
	case smachine.Passed:
		//
	case smachine.NotPassed:
		return ctx.WaitShared(p.sdl.SharedDataLink).ThenRepeat()
	default:
		panic(throw.IllegalState())
	}

	if future == nil {
		return ctx.Jump(p.stepPrepareData)
	}

	ready := future.GetReadySync()

	if ctx.Acquire(ready) {
		ctx.ReleaseAll()
		return ctx.Jump(p.stepDataIsReady)
	}

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if ctx.Acquire(ready) {
			ctx.ReleaseAll()
			return ctx.Jump(p.stepDataIsReady)
		}
		return ctx.Sleep().ThenRepeat()
	})
}

func (p *SMRead) stepDataIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch p.sdl.TryAccess(ctx, func(sd *datawriter.LineSharedData) (wakeup bool) {
		sd.TrimStages()

		normTargetRef := reference.NormCopy(p.request.TargetRef.Get())
		if _, future := sd.FindSequence(normTargetRef, p.extractor.AddLineRecord); future != nil {
			panic(throw.Impossible())
		}

		return false
	}) {
	case smachine.Passed:
		//
	case smachine.NotPassed:
		return ctx.WaitShared(p.sdl.SharedDataLink).ThenRepeat()
	default:
		panic(throw.IllegalState())
	}

	return ctx.Jump(p.stepPrepareData)
}

func (p *SMRead) stepPrepareData(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.extractor.NeedsDirtyReader() {
		return ctx.Jump(p.stepPrepareDataWithReader)
	}

	if p.extractor.ExtractMoreRecords(100) {
		return ctx.Repeat(100)
	}

	return ctx.Jump(p.stepSendResponse)
}

func (p *SMRead) stepPrepareDataWithReader(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return p.reader.PrepareAsync(ctx, func(svc buildersvc.ReadService) smachine.AsyncResultFunc {
		err := svc.DropReadDirty(p.dropID, p.extractor.ExtractAllRecordsWithReader)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				panic(err)
			}
			ctx.WakeUp()
		}
	}).DelayedStart().Sleep().ThenJump(p.stepSendResponse)
}

func (p *SMRead) stepSendResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	response := &rms.LReadResponse{}

	response.Entries = p.extractor.GetExtractRecords()

	nextInfo := p.extractor.GetExtractedTail()
	response.NextRecordSize = uint32(nextInfo.NextRecordSize)
	response.NextRecordPayloadsSize = uint32(nextInfo.NextRecordPayloadsSize)

	ctx.SetTerminationResult(response)

	// TODO send response back

	return ctx.Stop()
}

func (p *SMRead) migrateOnce(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(p.migrateTwice)
	return ctx.Stay()
}

func (p *SMRead) migrateTwice(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Error(throw.E("expired"))
}

func (p *SMRead) handleError(ctx smachine.FailureContext) {
	// TODO implement failure send
}
