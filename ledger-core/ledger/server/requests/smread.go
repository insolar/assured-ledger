// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package requests

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/dataextractor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datareader"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datawriter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const DefReadResponseSizeLimit = 1<<17
const MaxReadResponseSizeLimit = 1<<24

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

	// runtime
	output outputToLReadResponse
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

	p.output.wrapSize = 5 // approximation: 3 bytes per field tag and 2 bytes data length

	switch ps := p.pulseSlot.State(); {
	case p.request == nil:
		return ctx.Error(throw.E("missing request"))
	case p.request.TargetStartRef.IsEmpty():
		return ctx.Error(throw.E("missing target"))
	case ps == conveyor.Antique:
		return ctx.Jump(p.stepCleanRead)
	default:
		return ctx.Jump(p.stepDirtyRead)
	}
}

func reqValue(v, def uint32) uint32 {
	if v == 0 {
		return def
	}
	return v - 1
}

func (p *SMRead) getExtractorSelector() (selector dataextractor.Selector, err error) {
	selector = dataextractor.Selector{
		RootRef:   p.request.TargetRootRef.Get(),
		ReasonRef: p.request.TargetReasonRef.Get(),
		StartRef:  p.request.TargetStartRef.Get(),
		StopRef:   p.request.TargetStopRef.Get(),
	}

	flags := p.request.Flags
	switch {
	case flags & rms.ReadFlags_IncludeBranch != 0:
		selector.BranchMode = dataextractor.IncludeBranch
	case flags & rms.ReadFlags_FollowBranch != 0:
		selector.BranchMode = dataextractor.FollowBranch
	case flags & rms.ReadFlags_IgnoreBranch != 0:
		selector.BranchMode = dataextractor.IgnoreBranch
	}

	if flags & rms.ReadFlags_PresentToPast != 0 {
		selector.Direction = dataextractor.ToPast
	} else {
		selector.Direction = dataextractor.ToPresent
	}

	// TODO check refs
	// normRoot := reference.NormCopy(selector.RootRef)
	return selector, nil
}

func (p *SMRead) getExtractorConfig(selector dataextractor.Selector) dataextractor.Config {
	flags := p.request.Flags
	limits := dataextractor.Limits{
		StopRef:       reference.Copy(selector.StopRef),
		TotalCount:    reqValue(p.request.LimitCount, math.MaxUint32),
		Excerpts:      reqValue(p.request.LimitRecordWithExcerptCount, math.MaxUint32),
		Bodies:        reqValue(p.request.LimitRecordWithBodyCount, math.MaxUint32),
		Payloads:      reqValue(p.request.LimitRecordWithPayloadCount, 0),
		Extensions:    reqValue(p.request.LimitRecordWithExtensionsCount, 0),
		// RecTypeFilter: nil, // TODO
		// ExtTypeFilter: nil, // TODO
		ExcludeStart:  flags & rms.ReadFlags_ExcludeStart != 0,
		ExcludeStop:   flags & rms.ReadFlags_ExcludeStop != 0,
	}

	switch {
	case p.request.LimitSize == 0:
		limits.TotalSize = DefReadResponseSizeLimit
	case p.request.LimitSize > MaxReadResponseSizeLimit:
		limits.TotalSize = MaxReadResponseSizeLimit
	default:
		limits.TotalSize = p.request.LimitSize
	}

	cfg := dataextractor.Config{
		Selector:     selector,
		Limiter:      dataextractor.NewLimiter(limits),
		Output:       &p.output,
		Target:       p.request.TargetPulse,
	}

	return cfg
}

func (p *SMRead) stepCleanRead(ctx smachine.ExecutionContext) smachine.StateUpdate {
	selector, err := p.getExtractorSelector()
	if err != nil {
		return ctx.Error(err)
	}
	cfg := p.getExtractorConfig(selector)

	return ctx.CallSubroutine(datareader.NewReader(cfg), nil, p.subrReadDone)
}

func (p *SMRead) stepDirtyRead(ctx smachine.ExecutionContext) smachine.StateUpdate {
	selector, err := p.getExtractorSelector()
	if err != nil {
		return ctx.Error(err)
	}
	cfg := p.getExtractorConfig(selector)

	switch {
	case cfg.Target.IsUnknown():
		cfg.Target = p.pulseSlot.PulseNumber()
	case cfg.Target != p.pulseSlot.PulseNumber():
		return ctx.Error(throw.FailHere("wrong pulse for dirty read"))
	}

	return ctx.CallSubroutine(datawriter.NewDirtyReader(cfg), nil, p.subrReadDone)
}

func (p *SMRead) subrReadDone(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
	if err := ctx.GetError(); err != nil {
		return ctx.Error(err)
	}
	if ctx.EventParam() != nil {
		return ctx.Error(throw.FailHere("unexpected response"))
	}
	if p.output.hasDraft() {
		return ctx.Error(throw.FailHere("incomplete output"))
	}

	return ctx.Jump(p.stepSendResponse)
}

func (p *SMRead) stepSendResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	response := &rms.LReadResponse{}
	response.Entries = p.output.entries
	ctx.SetTerminationResult(response)

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

