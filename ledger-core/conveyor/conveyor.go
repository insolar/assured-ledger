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

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type InputEvent = interface{}

// PulseEventFactoryFunc should return pulse.Unknown or current.Pulse when SM doesn't need to be put into a different pulse slot.
// Arg (pulse.Range) can be nil for future slot.
type PulseEventFactoryFunc = func(context.Context, InputEvent, InputContext) (InputSetup, error)

type InputContext struct {
	PulseNumber pulse.Number
	PulseRange  pulse.Range
}

type InputSetup struct {
	TargetPulse pulse.Number
	CreateFn    smachine.CreateFunc
	PreInitFn   smachine.PreInitHandlerFunc
}

type EventInputer interface {
	AddInput(ctx context.Context, pn pulse.Number, event InputEvent) error
	AddInputExt(pn pulse.Number, event InputEvent, createDefaults smachine.CreateDefaultValues) error
}

type PreparedState = beat.AckData
type PreparePulseChangeFunc = func(PreparedState)

type PulseChanger interface {
	PreparePulseChange(out PreparePulseChangeFunc) error
	CancelPulseChange() error
	CommitPulseChange(pr pulse.Range) error
}

// PulseSlotPostMigrateFunc is called on migration and on creation of the slot. For creation (prevState) will be zero.
type PulseSlotPostMigrateFunc = func(prevState PulseSlotState, slot *PulseSlot, h smachine.SlotMachineHolder)

type PulseConveyorConfig struct {
	ConveyorMachineConfig             smachine.SlotMachineConfig
	SlotMachineConfig                 smachine.SlotMachineConfig
	EventlessSleep                    time.Duration
	MinCachePulseAge, MaxPastPulseAge uint32
	PulseDataService                  PulseDataServicePrepareFunc
	PulseSlotMigration                PulseSlotPostMigrateFunc
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
	if registry != nil {
		r.sharedRegistry = *injector.NewDynamicContainer(registry)
	}

	r.slotConfig.config.CleanupWeakOnMigrate = true
	r.slotMachine = smachine.NewSlotMachine(config.ConveyorMachineConfig,
		r.internalSignal.NextBroadcast,
		combineCallbacks(r.externalSignal.NextBroadcast, r.internalSignal.NextBroadcast),
		&r.sharedRegistry)

	r.slotConfig.eventCallback = r.internalSignal.NextBroadcast
	r.slotConfig.parentRegistry = &r.sharedRegistry

	// shared SlotId sequence
	r.slotConfig.config.SlotIDGenerateFn = r.slotMachine.CopyConfig().SlotIDGenerateFn

	r.pdm.initCache(config.MinCachePulseAge, config.MaxPastPulseAge, 1)
	r.pdm.pulseDataAdapterFn = config.PulseDataService
	r.pdm.pulseMigrateFn = config.PulseSlotMigration

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

	sharedRegistry injector.DynamicContainer
	slotMachine   *smachine.SlotMachine
	machineWorker smachine.AttachableSlotWorker

	pdm   PulseDataManager
	comps componentManager

	// mutable, set under SlotMachine synchronization
	presentMachine *PulseSlotMachine
	unpublishPulse pulse.Number

	stoppingChan <-chan struct{}
}

func (p *PulseConveyor) SetFactoryFunc(factory PulseEventFactoryFunc) {
	if p.machineWorker != nil {
		panic(throw.IllegalState())
	}
	p.factoryFn = factory
}

func (p *PulseConveyor) GetDataManager() *PulseDataManager {
	return &p.pdm
}

func (p *PulseConveyor) AddManagedComponent(c managed.Component) {
	p.comps.Add(p, c)
}

func (p *PulseConveyor) FindDependency(id string) (interface{}, bool) {
	return p.sharedRegistry.FindDependency(id)
}

func (p *PulseConveyor) AddDependency(v interface{}) {
	p.sharedRegistry.AddDependency(v)
}

func (p *PulseConveyor) AddInterfaceDependency(v interface{}) {
	p.sharedRegistry.AddInterfaceDependency(v)
}

func (p *PulseConveyor) PutDependency(id string, v interface{}) {
	p.sharedRegistry.PutDependency(id, v)
}

func (p *PulseConveyor) TryPutDependency(id string, v interface{}) bool {
	return p.sharedRegistry.TryPutDependency(id, v)
}

func (p *PulseConveyor) GetPublishedGlobalAliasAndBargeIn(key interface{}) (smachine.SlotLink, smachine.BargeInHolder) {
	return p.slotMachine.GetPublishedGlobalAliasAndBargeIn(key)
}

func (p *PulseConveyor) AddInput(ctx context.Context, pn pulse.Number, event InputEvent) error {
	return p.AddInputExt(pn, event, smachine.CreateDefaultValues{Context: ctx})
}

type errMissingPN struct {
	PN      pulse.Number
	RemapPN pulse.Number   `opt:""`
	State   PulseSlotState `opt:""`
}

func (p *PulseConveyor) AddInputExt(pn pulse.Number, event InputEvent,
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
		bd, _ := pulseSlot.pulseData.BeatData()
		pr = bd.Range
	}

	setup, err := p.factoryFn(createDefaults.Context, event, InputContext{targetPN, pr})
	switch {
	case err != nil || setup.CreateFn == nil:
		return err

	case setup.TargetPulse == targetPN || setup.TargetPulse == pn || setup.TargetPulse.IsUnknown():
		//

	case setup.TargetPulse.IsTimePulse():
		if pulseSlotMachine, targetPN, pulseState, err = p.mapToPulseSlotMachine(setup.TargetPulse); err != nil {
			return err
		}
		if pulseSlotMachine != nil && pulseState != 0 {
			break
		}
		fallthrough
	default:
		return throw.E("slotMachine remap is missing", errMissingPN{PN: pn, RemapPN: setup.TargetPulse})
	}

	createFn := setup.CreateFn
	if setup.PreInitFn != nil {
		createDefaults.PreInitializationHandler = setup.PreInitFn
	}

	addedOk := false
	switch {
	case pulseState == Future:
		// event for future needs special handling - it must wait until the pulse will actually arrive
		//nolint:staticcheck
		_, addedOk = pulseSlotMachine.innerMachine.AddNew(nil,
			newFutureEventSM(targetPN, &pulseSlotMachine.pulseSlot, createFn), createDefaults)

	case pulseState == Past:
		if !p.pdm.TouchPulseData(targetPN) { // make sure - for PAST must always have the data
			return throw.E("unknown data for pulse", errMissingPN{PN: targetPN, State: pulseState})
		}

		_, addedOk = pulseSlotMachine.innerMachine.AddNewByFunc(nil, createFn, createDefaults) //nolint:staticcheck
		if addedOk || pulseSlotMachine.innerMachine.IsActive() {
			break
		}

		// This is a dying past slot - redirect addition to antique
		pulseSlotMachine = p.getAntiquePulseSlotMachine()
		pulseState = Antique
		fallthrough

	case pulseState == Antique:
		createDefaults.InheritAllDependencies = true // ensure inheritance

		// Antique events have individual pulse slots, while being executed in a single SlotMachine
		if cps := p.pdm.getCachedPulseSlot(targetPN); cps != nil {
			createDefaults.PutOverride(injector.GetDefaultInjectionID(cps), cps)
			//nolint:staticcheck
			_, addedOk = pulseSlotMachine.innerMachine.AddNewByFunc(nil, createFn, createDefaults)
			break // add SM
		}

		if !p.pdm.IsRecentPastRange(pn) {
			// for non-recent past HasPulseData() can be incorrect / incomplete
			// we must use a longer procedure to get PulseData and utilize SM for it
			//nolint:staticcheck
			_, addedOk = pulseSlotMachine.innerMachine.AddNew(nil,
				newAntiqueEventSM(targetPN, &pulseSlotMachine.pulseSlot, createFn), createDefaults)
			break
		}
		fallthrough

	case !p.pdm.TouchPulseData(targetPN): // make sure - for PRESENT we must always have the data
		return throw.E("unknown data for pulse", errMissingPN{PN: targetPN, State: pulseState})
	default:
		//nolint:staticcheck
		_, addedOk = pulseSlotMachine.innerMachine.AddNewByFunc(nil, createFn, createDefaults)
	}

	if !addedOk {
		return throw.E("ignored event", errMissingPN{PN: targetPN, State: pulseState})
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
		panic(throw.IllegalState())
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
			panic(throw.IllegalState())
		} else {
			switch ps, ok := psm.pulseSlot._isAcceptedPresent(presentPN, pn); {
			case ps == Past:
				// pulse has changed - then we handle the packet as usual
				break
			case !ok:
				return nil, 0, 0, fmt.Errorf("pulse number is not allowed: pn=%v", pn)
			case ps != Present:
				panic(throw.IllegalState())
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
		panic(throw.IllegalState())
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
			panic(throw.IllegalState())
		}
		return psm
	}
	psm.activate(p.workerCtx, p.slotMachine.AddNew)
	psm.setPulseForUnpublish(p.slotMachine, pn)

	return psm
}

func (p *PulseConveyor) _getOrPublishSlotMachineAsPresent(pn pulse.Number, bd BeatData, pulseStart time.Time) (*PulseSlotMachine, bool) {
	if psm := p.getPulseSlotMachine(pn); psm != nil {
		psm.pulseSlot.pulseData.MakePresent(bd, pulseStart)
		return psm, false
	}

	psm := p.newPulseSlotMachine()
	psm.pulseSlot.pulseData = &presentPulseDataHolder{bd: bd, at: pulseStart}

	psv, ok := p.slotMachine.TryPublish(pn, psm)
	if ok {
		return psm, true
	}

	psm = psv.(*PulseSlotMachine)
	psm.pulseSlot.pulseData.MakePresent(bd, pulseStart)
	return psm, false
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

func (p *PulseConveyor) PreparePulseChange(out PreparePulseChangeFunc) error {
	p.pdm.awaitPreparingPulse()
	return p.sendSignal(func(ctx smachine.MachineCallContext) {
		if p.presentMachine == nil {
			// wrong - first pulse can only be committed but not prepared
			panic(throw.IllegalState())
		}
		p.pdm.setPreparingPulse(out)
		if !ctx.CallDirectBargeIn(p.presentMachine.SlotLink().GetAnyStepLink(), func(ctx smachine.BargeInContext) smachine.StateUpdate {
			return p.presentMachine.preparePulseChange(ctx, out)
		}) {
			// TODO handle stuck PulseSlot - need to do p.pdm.unsetPreparingPulse(), then close(out) when relevant code is available on network side
			panic(throw.FailHere("present slot is busy"))
		}
	})
}

func (p *PulseConveyor) CancelPulseChange() error {
	return p.sendSignal(func(ctx smachine.MachineCallContext) {
		if p.presentMachine == nil {
			// wrong - first pulse can only be committed but not prepared
			panic(throw.IllegalState())
		}
		p.pdm.unsetPreparingPulse()
		if !ctx.CallDirectBargeIn(p.presentMachine.SlotLink().GetAnyStepLink(), p.presentMachine.cancelPulseChange) {
			panic(throw.FailHere("present slot is busy"))
		}
	})
}

func (p *PulseConveyor) CommitPulseChange(pr pulse.Range, pulseStart time.Time, online census.OnlinePopulation) error {
	pd := pr.RightBoundData()
	pd.EnsurePulsarData()

	return p.sendSignal(func(ctx smachine.MachineCallContext) {
		prevPresentPN, prevFuturePN := p.pdm.GetPresentPulse()

		if p.presentMachine == nil {
			switch {
			case p.getPulseSlotMachine(prevPresentPN) != nil:
				panic(throw.IllegalState())
			case prevFuturePN == uninitializedFuture:
				// ok
			case p.getPulseSlotMachine(prevFuturePN) == nil:
				panic(throw.IllegalState())
			}
		} else {
			switch {
			case p.getPulseSlotMachine(prevPresentPN) != p.presentMachine:
				panic(throw.IllegalState())
			case pr.LeftBoundNumber() != prevFuturePN:
				panic(throw.IllegalState())
			case prevPresentPN.Next(pr.LeftPrevDelta()) != pr.LeftBoundNumber():
				panic(throw.IllegalState())
			}
		}

		bd := BeatData{Range: pr, Online: online}
		p.pdm.putPulseUpdate(bd)

		if p.presentMachine != nil {
			p.presentMachine.setPast()
		}

		p.pdm.unsetPreparingPulse()

		p.comps.PulseChanged(p, pr)

		ctx.Migrate(func() {
			p._migratePulseSlots(ctx, bd, pulseStart.UTC(), prevPresentPN, prevFuturePN)
		})
	})
}

func (p *PulseConveyor) _migratePulseSlots(ctx smachine.MachineCallContext, bd BeatData, pulseStart time.Time,
	_ /* prevPresentPN */, prevFuturePN pulse.Number,
) {
	if p.unpublishPulse.IsTimePulse() {
		// we know what we do - right!?
		p.slotMachine.TryUnsafeUnpublish(p.unpublishPulse)
		p.unpublishPulse = pulse.Unknown
	}

	pd := bd.Range.RightBoundData()
	pd.EnsurePulsarData()

	slotIsNew := false
	p.presentMachine, slotIsNew = p._getOrPublishSlotMachineAsPresent(prevFuturePN, bd, pulseStart)

	if prevFuturePN != pd.PulseNumber {
		// new pulse is different than expected at the previous cycle, so we have to remove the pulse number alias
		// to avoids unnecessary synchronization - the previous alias will be unpublished on commit of a next pulse
		p.unpublishPulse = prevFuturePN
		p.presentMachine.setPulseForUnpublish(p.slotMachine, pd.PulseNumber)

		if conflict, ok := p.slotMachine.TryPublish(pd.PulseNumber, p.presentMachine); !ok {
			panic(fmt.Sprintf("illegal state - conflict: key=%v existing=%v", pd.PulseNumber, conflict))
		}
	}

	if slotIsNew {
		p.presentMachine.activate(p.workerCtx, ctx.AddNew)
	}
	p.pdm.setPresentPulse(bd.Range) // reroutes incoming events
}

func (p *PulseConveyor) StopNoWait() {
	p.slotMachine.Stop()
}

func (p *PulseConveyor) Stop() {
	if p.stoppingChan == nil {
		panic(throw.IllegalState())
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
		panic(throw.IllegalState())
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

	defer p.comps.Stopped(p)
	p.comps.Started(p)

	for {
		eventMark := p.internalSignal.Mark()

		if cycleFn != nil {
			cycleFn(Scanning)
		}

		var (
			repeatNow    bool
			nextPollTime time.Time
		)
		_, callCount := p.machineWorker.AttachTo(p.slotMachine, p.externalSignal.Mark(), math.MaxUint32, func(worker smachine.AttachedSlotWorker) {

			scanMode := smachine.ScanDefault
			if p.pdm.isPriorityWorkOnly() {
				scanMode = smachine.ScanPriorityOnly
			}

			repeatNow, nextPollTime = p.slotMachine.ScanOnce(scanMode, worker)
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
