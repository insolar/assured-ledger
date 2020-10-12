// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package referencebuilder

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

type TypedHandlers struct {
	RInboundResponse RInboundResponseDefinition
	ROutboundRequest ROutboundRequestDefinition
	RLineMemory      RLineMemoryDefinition
	RLineActivate    RLineActivateDefinition
}

type TypedReferenceBuilder struct {
	t       minimock.Tester
	timeout time.Duration
	ctx     context.Context

	Handlers TypedHandlers

	RInboundResponse RInboundResponseBuilderMock
	ROutboundRequest ROutboundRequestBuilderMock
	RLineMemory      RLineMemoryBuilderMock
	RLineActivate    RLineActivateBuilderMock
}

func NewTypedReferenceBuilder(ctx context.Context, t minimock.Tester) *TypedReferenceBuilder {
	checker := &TypedReferenceBuilder{
		t:       t,
		ctx:     ctx,
		timeout: 10 * time.Second,

		Handlers: TypedHandlers{
			RInboundResponse: RInboundResponseDefinition{expectedCount: -1},
			ROutboundRequest: ROutboundRequestDefinition{expectedCount: -1},
			RLineMemory:      RLineMemoryDefinition{expectedCount: -1},
			RLineActivate:    RLineActivateDefinition{expectedCount: -1},
		},
	}

	checker.RInboundResponse = RInboundResponseBuilderMock{parent: checker}
	checker.ROutboundRequest = ROutboundRequestBuilderMock{parent: checker}
	checker.RLineMemory = RLineMemoryBuilderMock{parent: checker}
	checker.RLineActivate = RLineActivateBuilderMock{parent: checker}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(checker)
	}

	return checker
}

func (p *TypedReferenceBuilder) AnticipatedRefFromWriterTo(object reference.Global, pn pulse.Number, to io.WriterTo) reference.Global {
	panic("implement me")
}

func (p *TypedReferenceBuilder) AnticipatedRefFromBytes(object reference.Global, pn pulse.Number, data []byte) reference.Global {
	var rec rms.AnyRecord
	if err := rec.Unmarshal(data); err != nil {
		p.t.Fatalf("failed to unmarshal")
		return reference.Global{}
	}

	var resultRef reference.Global
	switch record := rec.Get().(type) {
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
	default:
		p.t.Fatalf("unexpected %T record", record)
		return reference.Global{}
	}

	return resultRef
}

func (p *TypedReferenceBuilder) AnticipatedRefFromRecord(object reference.Global, pn pulse.Number, record rms.BasicRecord) reference.Global {
	panic("implement me")
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

	return ok
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (p *TypedReferenceBuilder) MinimockFinish() {
	if !p.minimockDone() {
		p.t.Fatal("failed conditions on Typed")
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

func waitCounterIndefinitely(ctx context.Context, counter *atomickit.Int, count int) synckit.SignalChannel {
	if counter.Load() >= count {
		return synckit.ClosedChannel()
	}

	ch := make(synckit.ClosableSignalChannel)
	go func() {
		defer close(ch)

		for {
			if counter.Load() >= count {
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Millisecond):
			}
		}
	}()

	return ch
}
