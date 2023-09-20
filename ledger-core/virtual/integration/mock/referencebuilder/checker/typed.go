package checker

import (
	"context"
	"io"
	"time"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

// ============================================================================

type RInboundResponseDefinition struct {
	touched                           bool
	count                             atomickit.Int
	countBefore                       atomickit.Int
	expectedCount                     int
	anticipatedRefFromBytesHandler    RInboundResponseAnticipatedRefFromBytesHandler
	anticipatedRefFromWriterToHandler RInboundResponseAnticipatedRefFromWriterToHandler
	anticipatedRefFromRecordHandler   RInboundResponseAnticipatedRefFromRecordHandler
}
type RInboundResponseAnticipatedRefFromBytesHandler func(object reference.Global, pn pulse.Number, record *rms.RInboundResponse) reference.Global
type RInboundResponseAnticipatedRefFromWriterToHandler func(object reference.Global, pn pulse.Number, record *rms.RInboundResponse) reference.Global
type RInboundResponseAnticipatedRefFromRecordHandler func(object reference.Global, pn pulse.Number, record *rms.RInboundResponse) reference.Global
type RInboundResponseBuilderMock struct{ parent *TypedReferenceBuilder }

func (p RInboundResponseBuilderMock) ExpectedCount(count int) RInboundResponseBuilderMock {
	p.parent.Handlers.RInboundResponse.touched = true
	p.parent.Handlers.RInboundResponse.expectedCount = count
	return p
}

func (p *RInboundResponseBuilderMock) AnticipatedRefFromBytesMock(handler RInboundResponseAnticipatedRefFromBytesHandler) *RInboundResponseBuilderMock {
	p.parent.Handlers.RInboundResponse.touched = true
	p.parent.Handlers.RInboundResponse.anticipatedRefFromBytesHandler = handler
	return p
}

func (p *RInboundResponseBuilderMock) AnticipatedRefFromWriterToMock(handler RInboundResponseAnticipatedRefFromWriterToHandler) *RInboundResponseBuilderMock {
	p.parent.Handlers.RInboundResponse.touched = true
	p.parent.Handlers.RInboundResponse.anticipatedRefFromWriterToHandler = handler
	return p
}

func (p *RInboundResponseBuilderMock) AnticipatedRefFromRecordMock(handler RInboundResponseAnticipatedRefFromRecordHandler) *RInboundResponseBuilderMock {
	p.parent.Handlers.RInboundResponse.touched = true
	p.parent.Handlers.RInboundResponse.anticipatedRefFromRecordHandler = handler
	return p
}

func (p RInboundResponseBuilderMock) Count() int {
	return p.parent.Handlers.RInboundResponse.count.Load()
}

func (p RInboundResponseBuilderMock) CountBefore() int {
	return p.parent.Handlers.RInboundResponse.countBefore.Load()
}

func (p RInboundResponseBuilderMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.RInboundResponse.count, count)
}

// ============================================================================

type RLineActivateDefinition struct {
	touched                           bool
	count                             atomickit.Int
	countBefore                       atomickit.Int
	expectedCount                     int
	anticipatedRefFromBytesHandler    RLineActivateAnticipatedRefFromBytesHandler
	anticipatedRefFromWriterToHandler RLineActivateAnticipatedRefFromWriterToHandler
	anticipatedRefFromRecordHandler   RLineActivateAnticipatedRefFromRecordHandler
}
type RLineActivateAnticipatedRefFromBytesHandler func(object reference.Global, pn pulse.Number, record *rms.RLineActivate) reference.Global
type RLineActivateAnticipatedRefFromWriterToHandler func(object reference.Global, pn pulse.Number, record *rms.RLineActivate) reference.Global
type RLineActivateAnticipatedRefFromRecordHandler func(object reference.Global, pn pulse.Number, record *rms.RLineActivate) reference.Global
type RLineActivateBuilderMock struct{ parent *TypedReferenceBuilder }

func (p RLineActivateBuilderMock) ExpectedCount(count int) RLineActivateBuilderMock {
	p.parent.Handlers.RLineActivate.touched = true
	p.parent.Handlers.RLineActivate.expectedCount = count
	return p
}

func (p *RLineActivateBuilderMock) AnticipatedRefFromBytesMock(handler RLineActivateAnticipatedRefFromBytesHandler) *RLineActivateBuilderMock {
	p.parent.Handlers.RLineActivate.touched = true
	p.parent.Handlers.RLineActivate.anticipatedRefFromBytesHandler = handler
	return p
}

func (p *RLineActivateBuilderMock) AnticipatedRefFromWriterToMock(handler RLineActivateAnticipatedRefFromWriterToHandler) *RLineActivateBuilderMock {
	p.parent.Handlers.RLineActivate.touched = true
	p.parent.Handlers.RLineActivate.anticipatedRefFromWriterToHandler = handler
	return p
}

func (p *RLineActivateBuilderMock) AnticipatedRefFromRecordMock(handler RLineActivateAnticipatedRefFromRecordHandler) *RLineActivateBuilderMock {
	p.parent.Handlers.RLineActivate.touched = true
	p.parent.Handlers.RLineActivate.anticipatedRefFromRecordHandler = handler
	return p
}

func (p RLineActivateBuilderMock) Count() int {
	return p.parent.Handlers.RLineActivate.count.Load()
}

func (p RLineActivateBuilderMock) CountBefore() int {
	return p.parent.Handlers.RLineActivate.countBefore.Load()
}

func (p RLineActivateBuilderMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.RLineActivate.count, count)
}

// ============================================================================

type RLineDeactivateDefinition struct {
	touched                           bool
	count                             atomickit.Int
	countBefore                       atomickit.Int
	expectedCount                     int
	anticipatedRefFromBytesHandler    RLineDeactivateAnticipatedRefFromBytesHandler
	anticipatedRefFromWriterToHandler RLineDeactivateAnticipatedRefFromWriterToHandler
	anticipatedRefFromRecordHandler   RLineDeactivateAnticipatedRefFromRecordHandler
}
type RLineDeactivateAnticipatedRefFromBytesHandler func(object reference.Global, pn pulse.Number, record *rms.RLineDeactivate) reference.Global
type RLineDeactivateAnticipatedRefFromWriterToHandler func(object reference.Global, pn pulse.Number, record *rms.RLineDeactivate) reference.Global
type RLineDeactivateAnticipatedRefFromRecordHandler func(object reference.Global, pn pulse.Number, record *rms.RLineDeactivate) reference.Global
type RLineDeactivateBuilderMock struct{ parent *TypedReferenceBuilder }

func (p RLineDeactivateBuilderMock) ExpectedCount(count int) RLineDeactivateBuilderMock {
	p.parent.Handlers.RLineDeactivate.touched = true
	p.parent.Handlers.RLineDeactivate.expectedCount = count
	return p
}

func (p *RLineDeactivateBuilderMock) AnticipatedRefFromBytesMock(handler RLineDeactivateAnticipatedRefFromBytesHandler) *RLineDeactivateBuilderMock {
	p.parent.Handlers.RLineDeactivate.touched = true
	p.parent.Handlers.RLineDeactivate.anticipatedRefFromBytesHandler = handler
	return p
}

func (p *RLineDeactivateBuilderMock) AnticipatedRefFromWriterToMock(handler RLineDeactivateAnticipatedRefFromWriterToHandler) *RLineDeactivateBuilderMock {
	p.parent.Handlers.RLineDeactivate.touched = true
	p.parent.Handlers.RLineDeactivate.anticipatedRefFromWriterToHandler = handler
	return p
}

func (p *RLineDeactivateBuilderMock) AnticipatedRefFromRecordMock(handler RLineDeactivateAnticipatedRefFromRecordHandler) *RLineDeactivateBuilderMock {
	p.parent.Handlers.RLineDeactivate.touched = true
	p.parent.Handlers.RLineDeactivate.anticipatedRefFromRecordHandler = handler
	return p
}

func (p RLineDeactivateBuilderMock) Count() int {
	return p.parent.Handlers.RLineDeactivate.count.Load()
}

func (p RLineDeactivateBuilderMock) CountBefore() int {
	return p.parent.Handlers.RLineDeactivate.countBefore.Load()
}

func (p RLineDeactivateBuilderMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.RLineDeactivate.count, count)
}

// ============================================================================

type RLineMemoryDefinition struct {
	touched                           bool
	count                             atomickit.Int
	countBefore                       atomickit.Int
	expectedCount                     int
	anticipatedRefFromBytesHandler    RLineMemoryAnticipatedRefFromBytesHandler
	anticipatedRefFromWriterToHandler RLineMemoryAnticipatedRefFromWriterToHandler
	anticipatedRefFromRecordHandler   RLineMemoryAnticipatedRefFromRecordHandler
}
type RLineMemoryAnticipatedRefFromBytesHandler func(object reference.Global, pn pulse.Number, record *rms.RLineMemory) reference.Global
type RLineMemoryAnticipatedRefFromWriterToHandler func(object reference.Global, pn pulse.Number, record *rms.RLineMemory) reference.Global
type RLineMemoryAnticipatedRefFromRecordHandler func(object reference.Global, pn pulse.Number, record *rms.RLineMemory) reference.Global
type RLineMemoryBuilderMock struct{ parent *TypedReferenceBuilder }

func (p RLineMemoryBuilderMock) ExpectedCount(count int) RLineMemoryBuilderMock {
	p.parent.Handlers.RLineMemory.touched = true
	p.parent.Handlers.RLineMemory.expectedCount = count
	return p
}

func (p *RLineMemoryBuilderMock) AnticipatedRefFromBytesMock(handler RLineMemoryAnticipatedRefFromBytesHandler) *RLineMemoryBuilderMock {
	p.parent.Handlers.RLineMemory.touched = true
	p.parent.Handlers.RLineMemory.anticipatedRefFromBytesHandler = handler
	return p
}

func (p *RLineMemoryBuilderMock) AnticipatedRefFromWriterToMock(handler RLineMemoryAnticipatedRefFromWriterToHandler) *RLineMemoryBuilderMock {
	p.parent.Handlers.RLineMemory.touched = true
	p.parent.Handlers.RLineMemory.anticipatedRefFromWriterToHandler = handler
	return p
}

func (p *RLineMemoryBuilderMock) AnticipatedRefFromRecordMock(handler RLineMemoryAnticipatedRefFromRecordHandler) *RLineMemoryBuilderMock {
	p.parent.Handlers.RLineMemory.touched = true
	p.parent.Handlers.RLineMemory.anticipatedRefFromRecordHandler = handler
	return p
}

func (p RLineMemoryBuilderMock) Count() int {
	return p.parent.Handlers.RLineMemory.count.Load()
}

func (p RLineMemoryBuilderMock) CountBefore() int {
	return p.parent.Handlers.RLineMemory.countBefore.Load()
}

func (p RLineMemoryBuilderMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.RLineMemory.count, count)
}

// ============================================================================

type RLineMemoryExpectedDefinition struct {
	touched                           bool
	count                             atomickit.Int
	countBefore                       atomickit.Int
	expectedCount                     int
	anticipatedRefFromBytesHandler    RLineMemoryExpectedAnticipatedRefFromBytesHandler
	anticipatedRefFromWriterToHandler RLineMemoryExpectedAnticipatedRefFromWriterToHandler
	anticipatedRefFromRecordHandler   RLineMemoryExpectedAnticipatedRefFromRecordHandler
}
type RLineMemoryExpectedAnticipatedRefFromBytesHandler func(object reference.Global, pn pulse.Number, record *rms.RLineMemoryExpected) reference.Global
type RLineMemoryExpectedAnticipatedRefFromWriterToHandler func(object reference.Global, pn pulse.Number, record *rms.RLineMemoryExpected) reference.Global
type RLineMemoryExpectedAnticipatedRefFromRecordHandler func(object reference.Global, pn pulse.Number, record *rms.RLineMemoryExpected) reference.Global
type RLineMemoryExpectedBuilderMock struct{ parent *TypedReferenceBuilder }

func (p RLineMemoryExpectedBuilderMock) ExpectedCount(count int) RLineMemoryExpectedBuilderMock {
	p.parent.Handlers.RLineMemoryExpected.touched = true
	p.parent.Handlers.RLineMemoryExpected.expectedCount = count
	return p
}

func (p *RLineMemoryExpectedBuilderMock) AnticipatedRefFromBytesMock(handler RLineMemoryExpectedAnticipatedRefFromBytesHandler) *RLineMemoryExpectedBuilderMock {
	p.parent.Handlers.RLineMemoryExpected.touched = true
	p.parent.Handlers.RLineMemoryExpected.anticipatedRefFromBytesHandler = handler
	return p
}

func (p *RLineMemoryExpectedBuilderMock) AnticipatedRefFromWriterToMock(handler RLineMemoryExpectedAnticipatedRefFromWriterToHandler) *RLineMemoryExpectedBuilderMock {
	p.parent.Handlers.RLineMemoryExpected.touched = true
	p.parent.Handlers.RLineMemoryExpected.anticipatedRefFromWriterToHandler = handler
	return p
}

func (p *RLineMemoryExpectedBuilderMock) AnticipatedRefFromRecordMock(handler RLineMemoryExpectedAnticipatedRefFromRecordHandler) *RLineMemoryExpectedBuilderMock {
	p.parent.Handlers.RLineMemoryExpected.touched = true
	p.parent.Handlers.RLineMemoryExpected.anticipatedRefFromRecordHandler = handler
	return p
}

func (p RLineMemoryExpectedBuilderMock) Count() int {
	return p.parent.Handlers.RLineMemoryExpected.count.Load()
}

func (p RLineMemoryExpectedBuilderMock) CountBefore() int {
	return p.parent.Handlers.RLineMemoryExpected.countBefore.Load()
}

func (p RLineMemoryExpectedBuilderMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.RLineMemoryExpected.count, count)
}

// ============================================================================

type RLineMemoryReuseDefinition struct {
	touched                           bool
	count                             atomickit.Int
	countBefore                       atomickit.Int
	expectedCount                     int
	anticipatedRefFromBytesHandler    RLineMemoryReuseAnticipatedRefFromBytesHandler
	anticipatedRefFromWriterToHandler RLineMemoryReuseAnticipatedRefFromWriterToHandler
	anticipatedRefFromRecordHandler   RLineMemoryReuseAnticipatedRefFromRecordHandler
}
type RLineMemoryReuseAnticipatedRefFromBytesHandler func(object reference.Global, pn pulse.Number, record *rms.RLineMemoryReuse) reference.Global
type RLineMemoryReuseAnticipatedRefFromWriterToHandler func(object reference.Global, pn pulse.Number, record *rms.RLineMemoryReuse) reference.Global
type RLineMemoryReuseAnticipatedRefFromRecordHandler func(object reference.Global, pn pulse.Number, record *rms.RLineMemoryReuse) reference.Global
type RLineMemoryReuseBuilderMock struct{ parent *TypedReferenceBuilder }

func (p RLineMemoryReuseBuilderMock) ExpectedCount(count int) RLineMemoryReuseBuilderMock {
	p.parent.Handlers.RLineMemoryReuse.touched = true
	p.parent.Handlers.RLineMemoryReuse.expectedCount = count
	return p
}

func (p *RLineMemoryReuseBuilderMock) AnticipatedRefFromBytesMock(handler RLineMemoryReuseAnticipatedRefFromBytesHandler) *RLineMemoryReuseBuilderMock {
	p.parent.Handlers.RLineMemoryReuse.touched = true
	p.parent.Handlers.RLineMemoryReuse.anticipatedRefFromBytesHandler = handler
	return p
}

func (p *RLineMemoryReuseBuilderMock) AnticipatedRefFromWriterToMock(handler RLineMemoryReuseAnticipatedRefFromWriterToHandler) *RLineMemoryReuseBuilderMock {
	p.parent.Handlers.RLineMemoryReuse.touched = true
	p.parent.Handlers.RLineMemoryReuse.anticipatedRefFromWriterToHandler = handler
	return p
}

func (p *RLineMemoryReuseBuilderMock) AnticipatedRefFromRecordMock(handler RLineMemoryReuseAnticipatedRefFromRecordHandler) *RLineMemoryReuseBuilderMock {
	p.parent.Handlers.RLineMemoryReuse.touched = true
	p.parent.Handlers.RLineMemoryReuse.anticipatedRefFromRecordHandler = handler
	return p
}

func (p RLineMemoryReuseBuilderMock) Count() int {
	return p.parent.Handlers.RLineMemoryReuse.count.Load()
}

func (p RLineMemoryReuseBuilderMock) CountBefore() int {
	return p.parent.Handlers.RLineMemoryReuse.countBefore.Load()
}

func (p RLineMemoryReuseBuilderMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.RLineMemoryReuse.count, count)
}

// ============================================================================

type RLineRecapDefinition struct {
	touched                           bool
	count                             atomickit.Int
	countBefore                       atomickit.Int
	expectedCount                     int
	anticipatedRefFromBytesHandler    RLineRecapAnticipatedRefFromBytesHandler
	anticipatedRefFromWriterToHandler RLineRecapAnticipatedRefFromWriterToHandler
	anticipatedRefFromRecordHandler   RLineRecapAnticipatedRefFromRecordHandler
}
type RLineRecapAnticipatedRefFromBytesHandler func(object reference.Global, pn pulse.Number, record *rms.RLineRecap) reference.Global
type RLineRecapAnticipatedRefFromWriterToHandler func(object reference.Global, pn pulse.Number, record *rms.RLineRecap) reference.Global
type RLineRecapAnticipatedRefFromRecordHandler func(object reference.Global, pn pulse.Number, record *rms.RLineRecap) reference.Global
type RLineRecapBuilderMock struct{ parent *TypedReferenceBuilder }

func (p RLineRecapBuilderMock) ExpectedCount(count int) RLineRecapBuilderMock {
	p.parent.Handlers.RLineRecap.touched = true
	p.parent.Handlers.RLineRecap.expectedCount = count
	return p
}

func (p *RLineRecapBuilderMock) AnticipatedRefFromBytesMock(handler RLineRecapAnticipatedRefFromBytesHandler) *RLineRecapBuilderMock {
	p.parent.Handlers.RLineRecap.touched = true
	p.parent.Handlers.RLineRecap.anticipatedRefFromBytesHandler = handler
	return p
}

func (p *RLineRecapBuilderMock) AnticipatedRefFromWriterToMock(handler RLineRecapAnticipatedRefFromWriterToHandler) *RLineRecapBuilderMock {
	p.parent.Handlers.RLineRecap.touched = true
	p.parent.Handlers.RLineRecap.anticipatedRefFromWriterToHandler = handler
	return p
}

func (p *RLineRecapBuilderMock) AnticipatedRefFromRecordMock(handler RLineRecapAnticipatedRefFromRecordHandler) *RLineRecapBuilderMock {
	p.parent.Handlers.RLineRecap.touched = true
	p.parent.Handlers.RLineRecap.anticipatedRefFromRecordHandler = handler
	return p
}

func (p RLineRecapBuilderMock) Count() int {
	return p.parent.Handlers.RLineRecap.count.Load()
}

func (p RLineRecapBuilderMock) CountBefore() int {
	return p.parent.Handlers.RLineRecap.countBefore.Load()
}

func (p RLineRecapBuilderMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.RLineRecap.count, count)
}

// ============================================================================

type ROutboundRequestDefinition struct {
	touched                           bool
	count                             atomickit.Int
	countBefore                       atomickit.Int
	expectedCount                     int
	anticipatedRefFromBytesHandler    ROutboundRequestAnticipatedRefFromBytesHandler
	anticipatedRefFromWriterToHandler ROutboundRequestAnticipatedRefFromWriterToHandler
	anticipatedRefFromRecordHandler   ROutboundRequestAnticipatedRefFromRecordHandler
}
type ROutboundRequestAnticipatedRefFromBytesHandler func(object reference.Global, pn pulse.Number, record *rms.ROutboundRequest) reference.Global
type ROutboundRequestAnticipatedRefFromWriterToHandler func(object reference.Global, pn pulse.Number, record *rms.ROutboundRequest) reference.Global
type ROutboundRequestAnticipatedRefFromRecordHandler func(object reference.Global, pn pulse.Number, record *rms.ROutboundRequest) reference.Global
type ROutboundRequestBuilderMock struct{ parent *TypedReferenceBuilder }

func (p ROutboundRequestBuilderMock) ExpectedCount(count int) ROutboundRequestBuilderMock {
	p.parent.Handlers.ROutboundRequest.touched = true
	p.parent.Handlers.ROutboundRequest.expectedCount = count
	return p
}

func (p *ROutboundRequestBuilderMock) AnticipatedRefFromBytesMock(handler ROutboundRequestAnticipatedRefFromBytesHandler) *ROutboundRequestBuilderMock {
	p.parent.Handlers.ROutboundRequest.touched = true
	p.parent.Handlers.ROutboundRequest.anticipatedRefFromBytesHandler = handler
	return p
}

func (p *ROutboundRequestBuilderMock) AnticipatedRefFromWriterToMock(handler ROutboundRequestAnticipatedRefFromWriterToHandler) *ROutboundRequestBuilderMock {
	p.parent.Handlers.ROutboundRequest.touched = true
	p.parent.Handlers.ROutboundRequest.anticipatedRefFromWriterToHandler = handler
	return p
}

func (p *ROutboundRequestBuilderMock) AnticipatedRefFromRecordMock(handler ROutboundRequestAnticipatedRefFromRecordHandler) *ROutboundRequestBuilderMock {
	p.parent.Handlers.ROutboundRequest.touched = true
	p.parent.Handlers.ROutboundRequest.anticipatedRefFromRecordHandler = handler
	return p
}

func (p ROutboundRequestBuilderMock) Count() int {
	return p.parent.Handlers.ROutboundRequest.count.Load()
}

func (p ROutboundRequestBuilderMock) CountBefore() int {
	return p.parent.Handlers.ROutboundRequest.countBefore.Load()
}

func (p ROutboundRequestBuilderMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.ROutboundRequest.count, count)
}

// ============================================================================

type ROutboundResponseDefinition struct {
	touched                           bool
	count                             atomickit.Int
	countBefore                       atomickit.Int
	expectedCount                     int
	anticipatedRefFromBytesHandler    ROutboundResponseAnticipatedRefFromBytesHandler
	anticipatedRefFromWriterToHandler ROutboundResponseAnticipatedRefFromWriterToHandler
	anticipatedRefFromRecordHandler   ROutboundResponseAnticipatedRefFromRecordHandler
}
type ROutboundResponseAnticipatedRefFromBytesHandler func(object reference.Global, pn pulse.Number, record *rms.ROutboundResponse) reference.Global
type ROutboundResponseAnticipatedRefFromWriterToHandler func(object reference.Global, pn pulse.Number, record *rms.ROutboundResponse) reference.Global
type ROutboundResponseAnticipatedRefFromRecordHandler func(object reference.Global, pn pulse.Number, record *rms.ROutboundResponse) reference.Global
type ROutboundResponseBuilderMock struct{ parent *TypedReferenceBuilder }

func (p ROutboundResponseBuilderMock) ExpectedCount(count int) ROutboundResponseBuilderMock {
	p.parent.Handlers.ROutboundResponse.touched = true
	p.parent.Handlers.ROutboundResponse.expectedCount = count
	return p
}

func (p *ROutboundResponseBuilderMock) AnticipatedRefFromBytesMock(handler ROutboundResponseAnticipatedRefFromBytesHandler) *ROutboundResponseBuilderMock {
	p.parent.Handlers.ROutboundResponse.touched = true
	p.parent.Handlers.ROutboundResponse.anticipatedRefFromBytesHandler = handler
	return p
}

func (p *ROutboundResponseBuilderMock) AnticipatedRefFromWriterToMock(handler ROutboundResponseAnticipatedRefFromWriterToHandler) *ROutboundResponseBuilderMock {
	p.parent.Handlers.ROutboundResponse.touched = true
	p.parent.Handlers.ROutboundResponse.anticipatedRefFromWriterToHandler = handler
	return p
}

func (p *ROutboundResponseBuilderMock) AnticipatedRefFromRecordMock(handler ROutboundResponseAnticipatedRefFromRecordHandler) *ROutboundResponseBuilderMock {
	p.parent.Handlers.ROutboundResponse.touched = true
	p.parent.Handlers.ROutboundResponse.anticipatedRefFromRecordHandler = handler
	return p
}

func (p ROutboundResponseBuilderMock) Count() int {
	return p.parent.Handlers.ROutboundResponse.count.Load()
}

func (p ROutboundResponseBuilderMock) CountBefore() int {
	return p.parent.Handlers.ROutboundResponse.countBefore.Load()
}

func (p ROutboundResponseBuilderMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.ROutboundResponse.count, count)
}

// ============================================================================

type TypedHandlers struct {
	RInboundResponse    RInboundResponseDefinition
	RLineActivate       RLineActivateDefinition
	RLineDeactivate     RLineDeactivateDefinition
	RLineMemory         RLineMemoryDefinition
	RLineMemoryExpected RLineMemoryExpectedDefinition
	RLineMemoryReuse    RLineMemoryReuseDefinition
	RLineRecap          RLineRecapDefinition
	ROutboundRequest    ROutboundRequestDefinition
	ROutboundResponse   ROutboundResponseDefinition
}

type TypedReferenceBuilder struct {
	t       minimock.Tester
	timeout time.Duration
	ctx     context.Context

	Handlers TypedHandlers

	RInboundResponse    RInboundResponseBuilderMock
	RLineActivate       RLineActivateBuilderMock
	RLineDeactivate     RLineDeactivateBuilderMock
	RLineMemory         RLineMemoryBuilderMock
	RLineMemoryExpected RLineMemoryExpectedBuilderMock
	RLineMemoryReuse    RLineMemoryReuseBuilderMock
	RLineRecap          RLineRecapBuilderMock
	ROutboundRequest    ROutboundRequestBuilderMock
	ROutboundResponse   ROutboundResponseBuilderMock
}

func NewTypedReferenceBuilder(ctx context.Context, t minimock.Tester) *TypedReferenceBuilder {
	checker := &TypedReferenceBuilder{
		t:       t,
		ctx:     ctx,
		timeout: 10 * time.Second,

		Handlers: TypedHandlers{
			RInboundResponse:    RInboundResponseDefinition{expectedCount: -1},
			RLineActivate:       RLineActivateDefinition{expectedCount: -1},
			RLineDeactivate:     RLineDeactivateDefinition{expectedCount: -1},
			RLineMemory:         RLineMemoryDefinition{expectedCount: -1},
			RLineMemoryExpected: RLineMemoryExpectedDefinition{expectedCount: -1},
			RLineMemoryReuse:    RLineMemoryReuseDefinition{expectedCount: -1},
			RLineRecap:          RLineRecapDefinition{expectedCount: -1},
			ROutboundRequest:    ROutboundRequestDefinition{expectedCount: -1},
			ROutboundResponse:   ROutboundResponseDefinition{expectedCount: -1},
		},
	}

	checker.RInboundResponse = RInboundResponseBuilderMock{parent: checker}
	checker.RLineActivate = RLineActivateBuilderMock{parent: checker}
	checker.RLineDeactivate = RLineDeactivateBuilderMock{parent: checker}
	checker.RLineMemory = RLineMemoryBuilderMock{parent: checker}
	checker.RLineMemoryExpected = RLineMemoryExpectedBuilderMock{parent: checker}
	checker.RLineMemoryReuse = RLineMemoryReuseBuilderMock{parent: checker}
	checker.RLineRecap = RLineRecapBuilderMock{parent: checker}
	checker.ROutboundRequest = ROutboundRequestBuilderMock{parent: checker}
	checker.ROutboundResponse = ROutboundResponseBuilderMock{parent: checker}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(checker)
	}

	return checker
}

func (p *TypedReferenceBuilder) AnticipatedRefFromWriterTo(object reference.Global, pn pulse.Number, to io.WriterTo) reference.Global {
	panic("implement me")
}

func (p *TypedReferenceBuilder) AnticipatedRefFromRecord(object reference.Global, pn pulse.Number, record rms.BasicRecord) reference.Global {
	panic("implement me")
}

func (p *TypedReferenceBuilder) AnticipatedRefFromBytes(object reference.Global, pn pulse.Number, data []byte) reference.Global {
	var rec rms.AnyRecord
	if err := rec.Unmarshal(data); err != nil {
		p.t.Fatalf("failed to unmarshal")
		return reference.Global{}
	}

	var resultRef reference.Global
	switch record := rec.Get().(interface{}).(type) {
	case *rms.RInboundResponse:
		msg := rec.Get().(*rms.RInboundResponse)
		hdlStruct := &p.Handlers.RInboundResponse
		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.anticipatedRefFromBytesHandler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resultRef = hdlStruct.anticipatedRefFromBytesHandler(object, pn, msg)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message RInboundResponse (position: %s)", oldCount)
			}
		} else if !hdlStruct.touched {
			p.t.Fatalf("unexpected %T record", record)
			return reference.Global{}
		}

		hdlStruct.count.Add(1)

	case *rms.RLineActivate:
		msg := rec.Get().(*rms.RLineActivate)
		hdlStruct := &p.Handlers.RLineActivate
		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.anticipatedRefFromBytesHandler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resultRef = hdlStruct.anticipatedRefFromBytesHandler(object, pn, msg)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message RLineActivate (position: %s)", oldCount)
			}
		} else if !hdlStruct.touched {
			p.t.Fatalf("unexpected %T record", record)
			return reference.Global{}
		}

		hdlStruct.count.Add(1)

	case *rms.RLineDeactivate:
		msg := rec.Get().(*rms.RLineDeactivate)
		hdlStruct := &p.Handlers.RLineDeactivate
		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.anticipatedRefFromBytesHandler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resultRef = hdlStruct.anticipatedRefFromBytesHandler(object, pn, msg)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message RLineDeactivate (position: %s)", oldCount)
			}
		} else if !hdlStruct.touched {
			p.t.Fatalf("unexpected %T record", record)
			return reference.Global{}
		}

		hdlStruct.count.Add(1)

	case *rms.RLineMemory:
		msg := rec.Get().(*rms.RLineMemory)
		hdlStruct := &p.Handlers.RLineMemory
		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.anticipatedRefFromBytesHandler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resultRef = hdlStruct.anticipatedRefFromBytesHandler(object, pn, msg)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message RLineMemory (position: %s)", oldCount)
			}
		} else if !hdlStruct.touched {
			p.t.Fatalf("unexpected %T record", record)
			return reference.Global{}
		}

		hdlStruct.count.Add(1)

	case *rms.RLineMemoryExpected:
		msg := rec.Get().(*rms.RLineMemoryExpected)
		hdlStruct := &p.Handlers.RLineMemoryExpected
		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.anticipatedRefFromBytesHandler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resultRef = hdlStruct.anticipatedRefFromBytesHandler(object, pn, msg)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message RLineMemoryExpected (position: %s)", oldCount)
			}
		} else if !hdlStruct.touched {
			p.t.Fatalf("unexpected %T record", record)
			return reference.Global{}
		}

		hdlStruct.count.Add(1)

	case *rms.RLineMemoryReuse:
		msg := rec.Get().(*rms.RLineMemoryReuse)
		hdlStruct := &p.Handlers.RLineMemoryReuse
		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.anticipatedRefFromBytesHandler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resultRef = hdlStruct.anticipatedRefFromBytesHandler(object, pn, msg)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message RLineMemoryReuse (position: %s)", oldCount)
			}
		} else if !hdlStruct.touched {
			p.t.Fatalf("unexpected %T record", record)
			return reference.Global{}
		}

		hdlStruct.count.Add(1)

	case *rms.RLineRecap:
		msg := rec.Get().(*rms.RLineRecap)
		hdlStruct := &p.Handlers.RLineRecap
		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.anticipatedRefFromBytesHandler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resultRef = hdlStruct.anticipatedRefFromBytesHandler(object, pn, msg)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message RLineRecap (position: %s)", oldCount)
			}
		} else if !hdlStruct.touched {
			p.t.Fatalf("unexpected %T record", record)
			return reference.Global{}
		}

		hdlStruct.count.Add(1)

	case *rms.ROutboundRequest:
		msg := rec.Get().(*rms.ROutboundRequest)
		hdlStruct := &p.Handlers.ROutboundRequest
		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.anticipatedRefFromBytesHandler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resultRef = hdlStruct.anticipatedRefFromBytesHandler(object, pn, msg)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message ROutboundRequest (position: %s)", oldCount)
			}
		} else if !hdlStruct.touched {
			p.t.Fatalf("unexpected %T record", record)
			return reference.Global{}
		}

		hdlStruct.count.Add(1)

	case *rms.ROutboundResponse:
		msg := rec.Get().(*rms.ROutboundResponse)
		hdlStruct := &p.Handlers.ROutboundResponse
		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.anticipatedRefFromBytesHandler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resultRef = hdlStruct.anticipatedRefFromBytesHandler(object, pn, msg)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message ROutboundResponse (position: %s)", oldCount)
			}
		} else if !hdlStruct.touched {
			p.t.Fatalf("unexpected %T record", record)
			return reference.Global{}
		}

		hdlStruct.count.Add(1)

	default:
		p.t.Fatalf("unexpected %T record", record)
		return reference.Global{}
	}

	return resultRef
}

func (p *TypedReferenceBuilder) minimockDone() bool {
	ok := true

	{
		fn := func() bool {
			hdl := &p.Handlers.RInboundResponse

			switch {
			case hdl.expectedCount < 0:
				return true
			case hdl.expectedCount == 0:
				return true
			}

			return hdl.count.Load() == hdl.expectedCount
		}

		ok = ok && fn()
	}
	{
		fn := func() bool {
			hdl := &p.Handlers.RLineActivate

			switch {
			case hdl.expectedCount < 0:
				return true
			case hdl.expectedCount == 0:
				return true
			}

			return hdl.count.Load() == hdl.expectedCount
		}

		ok = ok && fn()
	}
	{
		fn := func() bool {
			hdl := &p.Handlers.RLineDeactivate

			switch {
			case hdl.expectedCount < 0:
				return true
			case hdl.expectedCount == 0:
				return true
			}

			return hdl.count.Load() == hdl.expectedCount
		}

		ok = ok && fn()
	}
	{
		fn := func() bool {
			hdl := &p.Handlers.RLineMemory

			switch {
			case hdl.expectedCount < 0:
				return true
			case hdl.expectedCount == 0:
				return true
			}

			return hdl.count.Load() == hdl.expectedCount
		}

		ok = ok && fn()
	}
	{
		fn := func() bool {
			hdl := &p.Handlers.RLineMemoryExpected

			switch {
			case hdl.expectedCount < 0:
				return true
			case hdl.expectedCount == 0:
				return true
			}

			return hdl.count.Load() == hdl.expectedCount
		}

		ok = ok && fn()
	}
	{
		fn := func() bool {
			hdl := &p.Handlers.RLineMemoryReuse

			switch {
			case hdl.expectedCount < 0:
				return true
			case hdl.expectedCount == 0:
				return true
			}

			return hdl.count.Load() == hdl.expectedCount
		}

		ok = ok && fn()
	}
	{
		fn := func() bool {
			hdl := &p.Handlers.RLineRecap

			switch {
			case hdl.expectedCount < 0:
				return true
			case hdl.expectedCount == 0:
				return true
			}

			return hdl.count.Load() == hdl.expectedCount
		}

		ok = ok && fn()
	}
	{
		fn := func() bool {
			hdl := &p.Handlers.ROutboundRequest

			switch {
			case hdl.expectedCount < 0:
				return true
			case hdl.expectedCount == 0:
				return true
			}

			return hdl.count.Load() == hdl.expectedCount
		}

		ok = ok && fn()
	}
	{
		fn := func() bool {
			hdl := &p.Handlers.ROutboundResponse

			switch {
			case hdl.expectedCount < 0:
				return true
			case hdl.expectedCount == 0:
				return true
			}

			return hdl.count.Load() == hdl.expectedCount
		}

		ok = ok && fn()
	}

	return ok
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (p *TypedReferenceBuilder) MinimockFinish() {
	if !p.minimockDone() {
		p.t.Fatal("failed conditions on TypedReferenceBuilder")
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (p *TypedReferenceBuilder) MinimockWait(timeout time.Duration) {
	timeoutCh := time.After(timeout)
	for {
		if p.minimockDone() {
			return
		}
		select {
		case <-timeoutCh:
			p.MinimockFinish()
			return
		case <-time.After(10 * time.Millisecond):
		}
	}
}
