package utils

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type StateReportBuilder struct {
	res rms.VStateReport

	class reference.Global

	statusCalled bool
	pulseCalled  bool
	objectCalled bool
	classCalled  bool
	memoryCalled bool
}

func NewStateReportBuilder() *StateReportBuilder {
	return &StateReportBuilder{
		class: gen.UniqueGlobalRef(),
	}
}

func (b *StateReportBuilder) Pulse(pn pulse.Number) *StateReportBuilder {
	if b.pulseCalled {
		panic(throw.IllegalState())
	}
	b.pulseCalled = true
	b.res.AsOf = pn
	b.res.Object.Set(gen.UniqueGlobalRefWithPulse(pn))

	return b
}

func (b *StateReportBuilder) Object(object reference.Global) *StateReportBuilder {
	if !b.pulseCalled {
		panic(throw.IllegalState())
	}
	if b.objectCalled {
		panic(throw.IllegalState())
	}
	b.objectCalled = true
	b.res.Object.Set(object)

	b.changeContentMaybe(func(e *rms.ObjectState) {
		local := e.Reference.GetValue().GetLocal()
		newRef := reference.NewRecordOf(object, local)
		e.Reference.Set(newRef)
	})

	return b
}

func (b *StateReportBuilder) Missing() *StateReportBuilder {
	b.status(rms.StateStatusMissing)

	return b
}

func (b *StateReportBuilder) Ready() *StateReportBuilder {
	b.status(rms.StateStatusReady)
	if !b.pulseCalled {
		panic(throw.IllegalState())
	}

	stateRef := reference.NewRecordOf(b.res.Object.GetValue(), gen.UniqueLocalRefWithPulse(b.res.AsOf))
	dState := rms.ObjectState{
		Reference: rms.NewReference(stateRef),
		Class:     rms.NewReference(b.class),
		Memory:    rms.NewBytes([]byte("some memory")),
	}
	vState := dState

	b.res.ProvidedContent = &rms.VStateReport_ProvidedContentBody{
		LatestDirtyState:     &dState,
		LatestValidatedState: &vState,
	}

	return b
}

func (b *StateReportBuilder) Inactive() *StateReportBuilder {
	b.status(rms.StateStatusInactive)

	return b
}

func (b *StateReportBuilder) Empty() *StateReportBuilder {
	b.status(rms.StateStatusEmpty)

	b.res.OrderedPendingCount = 1
	b.res.OrderedPendingEarliestPulse = b.res.AsOf

	return b
}

func (b *StateReportBuilder) status(s rms.VStateReport_StateStatus) {
	if b.statusCalled {
		panic(throw.IllegalState())
	}
	b.statusCalled = true
	b.res.Status = s
}

func (b *StateReportBuilder) Class(class reference.Global) *StateReportBuilder {
	if b.classCalled {
		panic(throw.IllegalState())
	}
	b.classCalled = true

	b.changeContent(func(e *rms.ObjectState) {
		e.Class.Set(class)
	})

	return b
}

func (b *StateReportBuilder) Memory(mem []byte) *StateReportBuilder {
	if b.memoryCalled {
		panic(throw.IllegalState())
	}
	b.memoryCalled = true

	b.changeContent(func(e *rms.ObjectState) {
		e.Memory.SetBytes(mem)
	})

	return b
}
func (b *StateReportBuilder) DirtyMemory(mem []byte) *StateReportBuilder {
	b.changeParticularContent(selectorTypeDirty, func(e *rms.ObjectState) {
		e.Memory.SetBytes(mem)
	})

	return b
}

func (b *StateReportBuilder) ValidatedMemory(mem []byte) *StateReportBuilder {
	b.changeParticularContent(selectorTypeValidated, func(e *rms.ObjectState) {
		e.Memory.SetBytes(mem)
	})

	return b
}

func (b *StateReportBuilder) NoValidated() *StateReportBuilder {
	switch {
	case b.res.Status != rms.StateStatusReady:
		panic(throw.IllegalState())
	case b.res.ProvidedContent == nil:
		panic(throw.IllegalState())
	case b.res.ProvidedContent.LatestValidatedState == nil:
		panic(throw.IllegalState())
	}

	b.res.ProvidedContent.LatestValidatedState = nil

	return b
}

func (b *StateReportBuilder) DirtyInactive() *StateReportBuilder {
	if !b.statusCalled {
		b.Ready()
	}

	b.changeParticularContent(selectorTypeDirty, func(e *rms.ObjectState) {
		e.Deactivated = true
	})

	return b
}

func (b *StateReportBuilder) StateRef(ref reference.Global) *StateReportBuilder {
	b.changeContent(func(e *rms.ObjectState) {
		e.Reference.Set(ref)
	})

	return b
}


func (b *StateReportBuilder) OrderedPendings(num int) *StateReportBuilder {
	if b.res.OrderedPendingCount > 0 {
		panic(throw.IllegalState())
	}
	if num < 1 || num > 127 {
		panic(throw.IllegalValue())
	}

	b.res.OrderedPendingCount = int32(num)
	b.res.OrderedPendingEarliestPulse = b.res.AsOf

	return b
}

func (b *StateReportBuilder) UnorderedPendings(num int) *StateReportBuilder {
	if b.res.UnorderedPendingCount > 0 {
		panic(throw.IllegalState())
	}
	if num < 1 || num > 127 {
		panic(throw.IllegalValue())
	}

	b.res.UnorderedPendingCount = int32(num)
	b.res.UnorderedPendingEarliestPulse = b.res.AsOf

	return b
}

func (b *StateReportBuilder) PendingsPulse(pn pulse.Number) *StateReportBuilder {
	if b.res.UnorderedPendingCount == 0 && b.res.OrderedPendingCount == 0 {
		panic(throw.IllegalState())
	}

	if b.res.UnorderedPendingCount > 0 {
		b.res.UnorderedPendingEarliestPulse = pn
	}
	if b.res.OrderedPendingCount > 0 {
		b.res.OrderedPendingEarliestPulse = pn
	}

	return b
}

func (b *StateReportBuilder) Report() rms.VStateReport {
	b.ensureReady()
	return b.res
}

func (b *StateReportBuilder) ReportPtr() *rms.VStateReport {
	b.ensureReady()
	return &b.res
}

func (b *StateReportBuilder) ensureReady() {
	switch {
	case !b.statusCalled:
		panic(throw.IllegalState())
	case !b.pulseCalled:
		panic(throw.IllegalState())
	}
}

type selectorType int

const (
	selectorTypeValidated selectorType = iota
	selectorTypeDirty
)

func (b *StateReportBuilder) changeContent(changer func(e *rms.ObjectState)) {
	if !b.statusCalled || b.res.Status != rms.StateStatusReady {
		panic(throw.IllegalState())
	}

	if b.res.ProvidedContent == nil {
		panic(throw.IllegalState())
	}

	b.changeContentMaybe(changer)
}

func (b *StateReportBuilder) changeContentMaybe(changer func(e *rms.ObjectState)) {
	content := b.res.ProvidedContent
	if content == nil {
		return
	}

	tmpList := []*rms.ObjectState{content.LatestValidatedState, content.LatestDirtyState}
	for _, e := range tmpList {
		if e == nil {
			continue
		}
		changer(e)
	}
}

func (b *StateReportBuilder) changeParticularContent(
	selector selectorType, changer func(e *rms.ObjectState),
) {
	content := b.res.ProvidedContent
	if content == nil {
		panic(throw.IllegalState())
	}

	var e *rms.ObjectState
	switch selector {
	case selectorTypeDirty:
		e = content.LatestDirtyState
	case selectorTypeValidated:
		e = content.LatestValidatedState
	default:
		panic(throw.IllegalValue())
	}

	if e == nil {
		panic(throw.IllegalState())
	}

	changer(e)
}
