// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/dataextractor"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewDirtyReader(request *rms.LReadRequest, pn pulse.Number) smachine.SubroutineStateMachine {
	if request == nil {
		panic(throw.IllegalValue())
	}
	return &SubSMDirtyReader{request: request, pn: pn}
}

var _ smachine.SubroutineStateMachine = &SubSMDirtyReader{}
type SubSMDirtyReader struct {
	smachine.StateMachineDeclTemplate

	// input
	request *rms.LReadRequest
	pn      pulse.Number

	// injected
	cataloger LineCataloger
	reader    buildersvc.ReadAdapter

	// runtime
	sdl       LineDataLink
	dropID    jet.DropID
	extractor dataextractor.SequenceExtractor
}

func (p *SubSMDirtyReader) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SubSMDirtyReader) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&p.cataloger)
	injector.MustInject(&p.reader)
}

func (p *SubSMDirtyReader) GetSubroutineInitState(smachine.SubroutineStartContext) smachine.InitFunc {
	return p.stepInit
}

func (p *SubSMDirtyReader) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SubSMDirtyReader) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	switch {
	case p.request.TargetStartRef.IsEmpty():
		return ctx.Error(throw.E("missing target"))
	case p.request.Flags != 0:
		panic(throw.NotImplemented())
	}

	if tpn := p.request.TargetStartRef.Get().GetLocal().Pulse(); tpn != p.pn {
		return ctx.Error(throw.E("wrong target pulse", struct { TargetPN, SlotPN pulse.Number }{ tpn, p.pn}))
	}

	// p.request.LimitCount
	// p.request.LimitSize
	// p.request.LimitRef
	// p.request.Flags

	p.extractor = &dataextractor.WholeExtractor{ReadAll: true}

	return ctx.Jump(p.stepFindLine)
}

func (p *SubSMDirtyReader) stepFindLine(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.sdl.IsZero() {
		normTargetRef := reference.NormCopy(p.request.TargetStartRef.Get())
		lineRef := reference.NewSelf(normTargetRef.GetBase())

		p.sdl = p.cataloger.Get(ctx, lineRef)
		if p.sdl.IsZero() {
			// line is absent
			return ctx.Jump(p.stepResponse)
		}
	}

	var activator smachine.SyncLink
	switch p.sdl.TryAccess(ctx, func(sd *LineSharedData) (wakeup bool) {
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

func (p *SubSMDirtyReader) stepLineIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var future *buildersvc.Future
	valid := false

	switch p.sdl.TryAccess(ctx, func(sd *LineSharedData) (wakeup bool) {
		if !sd.IsValid() {
			return
		}
		valid = true
		p.dropID = sd.DropID()

		sd.TrimStages()

		normTargetRef := reference.NormCopy(p.request.TargetStartRef.Get())
		// TODO do a few sub-cycles when too many record
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

	switch {
	case !valid:
		return ctx.Jump(p.stepResponse)
	case future == nil:
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

func (p *SubSMDirtyReader) stepDataIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch p.sdl.TryAccess(ctx, func(sd *LineSharedData) (wakeup bool) {
		sd.TrimStages()

		normTargetRef := reference.NormCopy(p.request.TargetStartRef.Get())
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

func (p *SubSMDirtyReader) stepPrepareData(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.extractor.NeedsDirtyReader() {
		return ctx.Jump(p.stepPrepareDataWithReader)
	}

	if p.extractor.ExtractMoreRecords(100) {
		return ctx.Repeat(100)
	}

	return ctx.Jump(p.stepResponse)
}

func (p *SubSMDirtyReader) stepPrepareDataWithReader(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return p.reader.PrepareAsync(ctx, func(svc buildersvc.ReadService) smachine.AsyncResultFunc {
		err := svc.DropReadDirty(p.dropID, p.extractor.ExtractAllRecordsWithReader)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				panic(err)
			}
			ctx.WakeUp()
		}
	}).DelayedStart().Sleep().ThenJump(p.stepResponse)
}

func (p *SubSMDirtyReader) stepResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	response := &rms.LReadResponse{}

	response.Entries = p.extractor.GetExtractRecords()

	nextInfo := p.extractor.GetExtractedTail()
	response.NextRecordSize = uint32(nextInfo.NextRecordSize)
	response.NextRecordPayloadsSize = uint32(nextInfo.NextRecordPayloadsSize)

	ctx.SetTerminationResult(response)
	return ctx.Stop()
}
