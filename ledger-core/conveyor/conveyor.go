// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type InputEvent = interface{}

// PulseEventFactoryFunc should return pulse.Unknown or current.Pulse when SM doesn't need to be put into a different pulse slot.
// Arg (pulse.Range) can be nil for future slot.
type PulseEventFactoryFunc = func(pulse.Number, pulse.Range, InputEvent) (pulse.Number, smachine.CreateFunc, error)

type EventInputer interface {
	AddInput(ctx context.Context, pn pulse.Number, event InputEvent) error
	AddInputExt(ctx context.Context, pn pulse.Number, event InputEvent,
		createDefaults smachine.CreateDefaultValues,
	) error
}

type PreparedState = struct{}
type PreparePulseChangeChannel = chan<- PreparedState

type PulseChanger interface {
	PreparePulseChange(out PreparePulseChangeChannel) error
	CancelPulseChange() error
	CommitPulseChange(pr pulse.Range) error
}

type PulseConveyorConfig struct {
	ConveyorMachineConfig             smachine.SlotMachineConfig
	SlotMachineConfig                 smachine.SlotMachineConfig
	EventlessSleep                    time.Duration
	MinCachePulseAge, MaxPastPulseAge uint32
	PulseDataService                  PulseDataServicePrepareFunc
}

func NewPulseConveyor(
	ctx context.Context,
	config PulseConveyorConfig,
	factoryFn PulseEventFactoryFunc,
	registry injector.DependencyRegistry,
) *PulseConveyor {

	r := &PulseConveyor{
		workerCtx: ctx,
		slotConfig: PulseSlotConfig{
			config: config.SlotMachineConfig,
		},
		factoryFn:      factoryFn,
		eventlessSleep: config.EventlessSleep,
	}
	r.slotConfig.config.CleanupWeakOnMigrate = true
	r.slotMachine = smachine.NewSlotMachine(config.ConveyorMachineConfig,
		r.internalSignal.NextBroadcast,
		combineCallbacks(r.externalSignal.NextBroadcast, r.internalSignal.NextBroadcast),
		registry)

	r.slotConfig.eventCallback = r.internalSignal.NextBroadcast
	r.slotConfig.parentRegistry = r.slotMachine

	// shared SlotId sequence
	r.slotConfig.config.SlotIDGenerateFn = r.slotMachine.CopyConfig().SlotIDGenerateFn

	r.pdm.init(config.MinCachePulseAge, config.MaxPastPulseAge, 1, config.PulseDataService)

	return r
}

type PulseConveyor struct {
	// immutable, provided, set at construction
	slotConfig     PulseSlotConfig
	eventlessSleep time.Duration
	factoryFn      PulseEventFactoryFunc
	workerCtx      context.Context

	// immutable, set at construction
	externalSignal synckit.VersionedSignal
	internalSignal synckit.VersionedSignal

	slotMachine   *smachine.SlotMachine
	machineWorker smachine.AttachableSlotWorker

	pdm PulseDataManager

	// mutable, set under SlotMachine synchronization
	presentMachine *PulseSlotMachine
	unpublishPulse pulse.Number

	stoppingChan <-chan struct{}
}

func (p *PulseConveyor) SetFactoryFunc(factory PulseEventFactoryFunc) {
	p.factoryFn = factory
}

func (p *PulseConveyor) GetDataManager() *PulseDataManager {
	return &p.pdm
}

func (p *PulseConveyor) AddDependency(v interface{}) {
	p.slotMachine.AddDependency(v)
}

func (p *PulseConveyor) AddInterfaceDependency(v interface{}) {
	p.slotMachine.AddInterfaceDependency(v)
}

func (p *PulseConveyor) FindDependency(id string) (interface{}, bool) {
	return p.slotMachine.FindDependency(id)
}

func (p *PulseConveyor) PutDependency(id string, v interface{}) {
	p.slotMachine.PutDependency(id, v)
}

func (p *PulseConveyor) TryPutDependency(id string, v interface{}) bool {
	return p.slotMachine.TryPutDependency(id, v)
}

func (p *PulseConveyor) GetPublishedGlobalAliasAndBargeIn(key interface{}) (smachine.SlotLink, smachine.BargeInHolder) {
	return p.slotMachine.GetPublishedGlobalAliasAndBargeIn(key)
}

func (p *PulseConveyor) AddInput(ctx context.Context, pn pulse.Number, event InputEvent) error {
	return p.AddInputExt(ctx, pn, event, smachine.CreateDefaultValues{})
}

type errMissingPN struct {
	PN      pulse.Number
	RemapPN pulse.Number `opt:""`
}

func (p *PulseConveyor) AddInputExt(ctx context.Context, pn pulse.Number, event InputEvent,
	createDefaults smachine.CreateDefaultValues,
) error {
	pulseSlotMachine, targetPN, pulseState, err := p.mapToPulseSlotMachine(pn)
	switch {
	case err != nil:
		return err
	case pulseSlotMachine == nil || pulseState == 0:
		return throw.E("slotMachine is missing", errMissingPN{PN: pn})
	}

	var pr pulse.Range

	if pulseState != Antique {
		pr, _ = pulseSlotMachine.pulseSlot.PulseRange()
	} else if pulseSlot := p.pdm.getCachedPulseSlot(targetPN); pulseSlot != nil {
		pr, _ = pulseSlot.pulseData.PulseRange()
	}

	remapPN, createFn, err := p.factoryFn(targetPN, pr, event)
	switch {
	case createFn == nil || err != nil:
		return err

	case remapPN == targetPN || remapPN == pn || remapPN.IsUnknown():
		//
	case remapPN.IsTimePulse():
		if pulseSlotMachine, targetPN, pulseState, err = p.mapToPulseSlotMachine(remapPN); err != nil {
			return err
		}
		if pulseSlotMachine != nil && pulseState != 0 {
			break
		}
		fallthrough
	default:
		return throw.E("slotMachine remap is missing", errMissingPN{PN: pn, RemapPN: remapPN})
	}

	switch {
	case pulseState == Future:
		// event for future needs special handling - it must wait until the pulse will actually arrive
		pulseSlotMachine.innerMachine.AddNew(ctx,
			newFutureEventSM(targetPN, &pulseSlotMachine.pulseSlot, createFn), createDefaults)
		return nil

	case pulseState == Antique:
		createDefaults.InheritAllDependencies = true // ensure inheritance

		// Antique events have individual pulse slots, while being executed in a single SlotMachine
		if cps := p.pdm.getCachedPulseSlot(targetPN); cps != nil {
			createDefaults.PutOverride(injector.GetDefaultInjectionID(cps), cps)
			break // add SM
		}

		if !p.pdm.IsRecentPastRange(pn) {
			// for non-recent past HasPulseData() can be incorrect / incomplete
			// we must use a longer procedure to get PulseData and utilize SM for it
			pulseSlotMachine.innerMachine.AddNew(ctx,
				newAntiqueEventSM(targetPN, &pulseSlotMachine.pulseSlot, createFn), createDefaults)
			return nil
		}
		fallthrough

	case !p.pdm.TouchPulseData(targetPN): // make sure - for PAST and PRESENT we must always have the data ...
		return throw.E("unknown data for pulse", errMissingPN{PN: targetPN})
	}

	if _, ok := pulseSlotMachine.innerMachine.AddNewByFunc(ctx, createFn, createDefaults); !ok {
		return throw.E("ignored event", errMissingPN{PN: targetPN})
	}

	return nil
}

func (p *PulseConveyor) mapToPulseSlotMachine(pn pulse.Number) (*PulseSlotMachine, pulse.Number, PulseSlotState, error) {

	var presentPN, futurePN pulse.Number
	for {
		presentPN, futurePN = p.pdm.GetPresentPulse()

		if !presentPN.IsUnknown() {
			break
		}
		// when no present pulse - all pulses go to future
		switch {
		case futurePN != uninitializedFuture:
			// there should be only one futureSlot, so we won't create a new future slot even when futurePN != pn
			if pn.IsTimePulse() {
				break
			} else if pn.IsUnknown() {
				pn = futurePN
				break
			}
			fallthrough
		case !pn.IsTimePulse():
			return nil, 0, 0, fmt.Errorf("pulse number is invalid: pn=%v", pn)
		case p.pdm.setUninitializedFuturePulse(pn):
			futurePN = pn
		default:
			// get the updated pulse
			continue
		}
		return p.getFuturePulseSlotMachine(presentPN, futurePN), pn, Future, nil
	}

	switch {
	case pn.IsUnknownOrEqualTo(presentPN):
		if psm := p.getPulseSlotMachine(presentPN); psm != nil {
			return psm, presentPN, Present, nil
		}
		// present slot must be present
		panic("illegal state")
	case !pn.IsTimePulse():
		return nil, 0, 0, fmt.Errorf("pulse number is invalid: pn=%v", pn)
	case pn < presentPN:
		// this can be either be a past/antique slot, or a part of the present range
		if psm := p.getPulseSlotMachine(pn); psm != nil {
			return psm, pn, Past, nil
		}

		// check if the pulse is within PRESENT range (as it may include some skipped pulses)
		if psm := p.getPulseSlotMachine(presentPN); psm == nil {
			// present slot must be present
			panic("illegal state")
		} else {
			switch ps, ok := psm.pulseSlot._isAcceptedPresent(presentPN, pn); {
			case ps == Past:
				// pulse has changed - then we handle the packet as usual
				break
			case !ok:
				return nil, 0, 0, fmt.Errorf("pulse number is not allowed: pn=%v", pn)
			case ps != Present:
				panic("illegal state")
			}
		}

		if !p.pdm.isAllowedPastSpan(presentPN, pn) {
			return nil, 0, 0, fmt.Errorf("pulse number is too far in past: pn=%v, present=%v", pn, presentPN)
		}
		return p.getAntiquePulseSlotMachine(), pn, Antique, nil
	case pn < futurePN:
		return nil, 0, 0, fmt.Errorf("pulse number is unexpected: pn=%v", pn)
	default: // pn >= futurePN
		if !p.pdm.isAllowedFutureSpan(presentPN, futurePN, pn) {
			return nil, 0, 0, fmt.Errorf("pulse number is too far in future: pn=%v, expected=%v", pn, futurePN)
		}
		return p.getFuturePulseSlotMachine(presentPN, futurePN), pn, Future, nil
	}
}

func (p *PulseConveyor) getPulseSlotMachine(pn pulse.Number) *PulseSlotMachine {
	if psv, ok := p.slotMachine.GetPublished(pn); ok {
		if psm, ok := psv.(*PulseSlotMachine); ok {
			return psm
		}
		panic("illegal state")
	}
	return nil
}

func (p *PulseConveyor) getFuturePulseSlotMachine(presentPN, futurePN pulse.Number) *PulseSlotMachine {
	if psm := p.getPulseSlotMachine(futurePN); psm != nil {
		return psm
	}

	psm := p.newPulseSlotMachine()

	prevDelta := futurePN - presentPN
	switch {
	case presentPN.IsUnknown():
		prevDelta = 0
	case prevDelta > math.MaxUint16:
		prevDelta = math.MaxUint16
	}

	psm.setFuture(pulse.NewExpectedPulsarData(futurePN, uint16(prevDelta)))
	return p._publishPulseSlotMachine(futurePN, psm)
}

func (p *PulseConveyor) getAntiquePulseSlotMachine() *PulseSlotMachine {
	if psm := p.getPulseSlotMachine(0); psm != nil {
		return psm
	}
	psm := p.newPulseSlotMachine()
	psm.setAntique()
	return p._publishPulseSlotMachine(0, psm)
}

func (p *PulseConveyor) _publishPulseSlotMachine(pn pulse.Number, psm *PulseSlotMachine) *PulseSlotMachine {
	if psv, ok := p.slotMachine.TryPublish(pn, psm); !ok {
		psm = psv.(*PulseSlotMachine)
		if psm == nil {
			panic("illegal state")
		}
		return psm
	}
	psm.activate(p.workerCtx, p.slotMachine.AddNew)
	psm.setPulseForUnpublish(p.slotMachine, pn)

	return psm
}

func (p *PulseConveyor) _publishUninitializedPulseSlotMachine(pn pulse.Number) (*PulseSlotMachine, bool) {
	if psm := p.getPulseSlotMachine(pn); psm != nil {
		return psm, false
	}
	psm := p.newPulseSlotMachine()
	if psv, ok := p.slotMachine.TryPublish(pn, psm); !ok {
		psm = psv.(*PulseSlotMachine)
		if psm == nil {
			panic("illegal state")
		}
		return psm, false
	}
	return psm, true
}

func (p *PulseConveyor) newPulseSlotMachine() *PulseSlotMachine {
	return NewPulseSlotMachine(p.slotConfig, &p.pdm)
}

func (p *PulseConveyor) sendSignal(fn smachine.MachineCallFunc) error {
	result := make(chan error, 1)
	p.slotMachine.ScheduleCall(func(ctx smachine.MachineCallContext) {
		defer func() {
			result <- smachine.RecoverSlotPanicWithStack("signal", recover(), nil, 0)
			close(result)
		}()
		fn(ctx)
	}, true)
	return <-result
}

func (p *PulseConveyor) PreparePulseChange(out PreparePulseChangeChannel) error {
	return p.sendSignal(func(ctx smachine.MachineCallContext) {
		if p.presentMachine == nil {
			// wrong - first pulse can only be committed but not prepared
			panic("illegal state")
		}
		p.pdm.setPreparingPulse(out)
		if !ctx.CallDirectBargeIn(p.presentMachine.SlotLink().GetAnyStepLink(), func(ctx smachine.BargeInContext) smachine.StateUpdate {
			return p.presentMachine.preparePulseChange(ctx, out)
		}) {
			// TODO handle stuck PulseSlot - need to do p.pdm.unsetPreparingPulse(), then close(out) when relevant code is available on network side
			panic("present slot is busy")
		}
	})
}

func (p *PulseConveyor) CancelPulseChange() error {
	return p.sendSignal(func(ctx smachine.MachineCallContext) {
		if p.presentMachine == nil {
			// wrong - first pulse can only be committed but not prepared
			panic("illegal state")
		}
		p.pdm.unsetPreparingPulse()
		if !ctx.CallDirectBargeIn(p.presentMachine.SlotLink().GetAnyStepLink(), p.presentMachine.cancelPulseChange) {
			panic("present slot is busy")
		}
	})
}

func (p *PulseConveyor) CommitPulseChange(pr pulse.Range, pulseStart time.Time) error {
	pd := pr.RightBoundData()
	pd.EnsurePulsarData()

	return p.sendSignal(func(ctx smachine.MachineCallContext) {
		prevPresentPN, prevFuturePN := p.pdm.GetPresentPulse()

		if p.presentMachine == nil {
			switch {
			case p.getPulseSlotMachine(prevPresentPN) != nil:
				panic("illegal state")
			case prevFuturePN == uninitializedFuture:
				// ok
			case p.getPulseSlotMachine(prevFuturePN) == nil:
				panic("illegal state")
			}
		} else {
			switch {
			case p.getPulseSlotMachine(prevPresentPN) != p.presentMachine:
				panic("illegal state")
			case pr.LeftBoundNumber() != prevFuturePN:
				panic("illegal state")
			case prevPresentPN.Next(pr.LeftPrevDelta()) != pr.LeftBoundNumber():
				panic("illegal state")
			}
		}

		p.pdm.putPulseRange(pr)

		if p.presentMachine != nil {
			p.presentMachine.setPast()
		}

		p.pdm.unsetPreparingPulse()

		ctx.Migrate(func() {
			p._migratePulseSlots(ctx, pr, prevPresentPN, prevFuturePN, pulseStart.UTC())
		})
	})
}

func (p *PulseConveyor) _migratePulseSlots(ctx smachine.MachineCallContext, pr pulse.Range,
	_ /* prevPresentPN */, prevFuturePN pulse.Number, pulseStart time.Time,
) {
	if p.unpublishPulse.IsTimePulse() {
		// we know what we do - right!?
		p.slotMachine.TryUnsafeUnpublish(p.unpublishPulse)
		p.unpublishPulse = pulse.Unknown
	}

	pd := pr.RightBoundData()

	prevFuture, activatePresent := p._publishUninitializedPulseSlotMachine(prevFuturePN)
	prevFuture.setPresent(pr, pulseStart)
	p.presentMachine = prevFuture

	if prevFuturePN != pd.PulseNumber {
		// new pulse is different than expected at the previous cycle, so we have to remove the pulse number alias
		// to avoids unnecessary synchronization - the previous alias will be unpublished on commit of a next pulse
		p.unpublishPulse = prevFuturePN
		p.presentMachine.setPulseForUnpublish(p.slotMachine, pd.PulseNumber)

		if conflict, ok := p.slotMachine.TryPublish(pd.PulseNumber, p.presentMachine); !ok {
			panic(fmt.Sprintf("illegal state - conflict: key=%v existing=%v", pd.PulseNumber, conflict))
		}
	}

	if activatePresent {
		p.presentMachine.activate(p.workerCtx, ctx.AddNew)
	}
	p.pdm.setPresentPulse(pr) // reroutes incoming events
}

func (p *PulseConveyor) StopNoWait() {
	p.slotMachine.Stop()
}

func (p *PulseConveyor) Stop() {
	if p.stoppingChan == nil {
		panic("illegal state")
	}
	p.slotMachine.Stop()
	<-p.stoppingChan
}

func (p *PulseConveyor) StartWorker(emergencyStop <-chan struct{}, completedFn func()) {
	p.StartWorkerExt(emergencyStop, completedFn, nil)
}

type CycleState uint8

const (
	Scanning CycleState = iota
	ScanActive
	ScanIdle
)

type PulseConveyorCycleFunc = func(CycleState)

func (p *PulseConveyor) StartWorkerExt(emergencyStop <-chan struct{}, completedFn func(), cycleFn PulseConveyorCycleFunc) {
	if p.machineWorker != nil {
		panic("illegal state")
	}
	p.machineWorker = sworker.NewAttachableSimpleSlotWorker()
	ch := make(chan struct{})
	p.stoppingChan = ch
	go p.runWorker(emergencyStop, ch, completedFn, cycleFn)
}

func (p *PulseConveyor) runWorker(emergencyStop <-chan struct{}, closeOnStop chan<- struct{}, completedFn func(), cycleFn PulseConveyorCycleFunc) {
	if emergencyStop != nil {
		go func() {
			<-emergencyStop
			p.slotMachine.Stop()
			p.externalSignal.NextBroadcast()
		}()
	}

	if closeOnStop != nil {
		defer close(closeOnStop)
	}
	if completedFn != nil {
		defer completedFn()
	}

	for {
		var (
			repeatNow    bool
			nextPollTime time.Time
		)
		eventMark := p.internalSignal.Mark()

		if cycleFn != nil {
			cycleFn(Scanning)
		}

		_, callCount := p.machineWorker.AttachTo(p.slotMachine, p.externalSignal.Mark(), math.MaxUint32, func(worker smachine.AttachedSlotWorker) {
			repeatNow, nextPollTime = p.slotMachine.ScanOnce(smachine.ScanDefault, worker)
		})

		select {
		case <-emergencyStop:
			return
		default:
			// pass
		}

		if callCount > 0 && cycleFn != nil {
			cycleFn(ScanActive)
		}

		if !p.slotMachine.IsActive() {
			break
		}

		if repeatNow || eventMark.HasSignal() {
			continue
		}

		select {
		case <-emergencyStop:
			return
		case <-eventMark.Channel():
		case <-func() <-chan time.Time {
			if cycleFn != nil {
				cycleFn(ScanIdle)
			}

			switch {
			case !nextPollTime.IsZero():
				return time.After(time.Until(nextPollTime))
			case p.eventlessSleep > 0 && p.eventlessSleep < math.MaxInt64:
				return time.After(p.eventlessSleep)
			}
			return nil
		}():
		}
	}

	p.slotMachine.RunToStop(p.machineWorker, synckit.NewNeverSignal())
	p.presentMachine = nil
}

func (p *PulseConveyor) WakeUpWorker() {
	p.internalSignal.NextBroadcast()
}
