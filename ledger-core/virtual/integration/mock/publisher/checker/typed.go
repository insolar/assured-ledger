// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
// Code generated by TypePublishGenerator. DO NOT EDIT.

package checker

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

// ============================================================================

type VCallRequestDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VCallRequestHandler
}
type VCallRequestHandler func(*payload.VCallRequest) bool
type PubVCallRequestMock struct{ parent *Typed }

func (p PubVCallRequestMock) ExpectedCount(count int) PubVCallRequestMock {
	p.parent.Handlers.VCallRequest.touched = true
	p.parent.Handlers.VCallRequest.expectedCount = count
	return p
}

func (p PubVCallRequestMock) Set(handler VCallRequestHandler) PubVCallRequestMock {
	p.parent.Handlers.VCallRequest.touched = true
	p.parent.Handlers.VCallRequest.handler = handler
	return p
}

func (p PubVCallRequestMock) SetResend(resend bool) PubVCallRequestMock {
	p.parent.Handlers.VCallRequest.touched = true
	p.parent.Handlers.VCallRequest.handler = func(*payload.VCallRequest) bool { return resend }
	return p
}

func (p PubVCallRequestMock) Count() int {
	return p.parent.Handlers.VCallRequest.count.Load()
}

func (p PubVCallRequestMock) CountBefore() int {
	return p.parent.Handlers.VCallRequest.countBefore.Load()
}

func (p PubVCallRequestMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.VCallRequest.count, count)
}

// ============================================================================

type VCallResultDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VCallResultHandler
}
type VCallResultHandler func(*payload.VCallResult) bool
type PubVCallResultMock struct{ parent *Typed }

func (p PubVCallResultMock) ExpectedCount(count int) PubVCallResultMock {
	p.parent.Handlers.VCallResult.touched = true
	p.parent.Handlers.VCallResult.expectedCount = count
	return p
}

func (p PubVCallResultMock) Set(handler VCallResultHandler) PubVCallResultMock {
	p.parent.Handlers.VCallResult.touched = true
	p.parent.Handlers.VCallResult.handler = handler
	return p
}

func (p PubVCallResultMock) SetResend(resend bool) PubVCallResultMock {
	p.parent.Handlers.VCallResult.touched = true
	p.parent.Handlers.VCallResult.handler = func(*payload.VCallResult) bool { return resend }
	return p
}

func (p PubVCallResultMock) Count() int {
	return p.parent.Handlers.VCallResult.count.Load()
}

func (p PubVCallResultMock) CountBefore() int {
	return p.parent.Handlers.VCallResult.countBefore.Load()
}

func (p PubVCallResultMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.VCallResult.count, count)
}

// ============================================================================

type VDelegatedCallRequestDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VDelegatedCallRequestHandler
}
type VDelegatedCallRequestHandler func(*payload.VDelegatedCallRequest) bool
type PubVDelegatedCallRequestMock struct{ parent *Typed }

func (p PubVDelegatedCallRequestMock) ExpectedCount(count int) PubVDelegatedCallRequestMock {
	p.parent.Handlers.VDelegatedCallRequest.touched = true
	p.parent.Handlers.VDelegatedCallRequest.expectedCount = count
	return p
}

func (p PubVDelegatedCallRequestMock) Set(handler VDelegatedCallRequestHandler) PubVDelegatedCallRequestMock {
	p.parent.Handlers.VDelegatedCallRequest.touched = true
	p.parent.Handlers.VDelegatedCallRequest.handler = handler
	return p
}

func (p PubVDelegatedCallRequestMock) SetResend(resend bool) PubVDelegatedCallRequestMock {
	p.parent.Handlers.VDelegatedCallRequest.touched = true
	p.parent.Handlers.VDelegatedCallRequest.handler = func(*payload.VDelegatedCallRequest) bool { return resend }
	return p
}

func (p PubVDelegatedCallRequestMock) Count() int {
	return p.parent.Handlers.VDelegatedCallRequest.count.Load()
}

func (p PubVDelegatedCallRequestMock) CountBefore() int {
	return p.parent.Handlers.VDelegatedCallRequest.countBefore.Load()
}

func (p PubVDelegatedCallRequestMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.VDelegatedCallRequest.count, count)
}

// ============================================================================

type VDelegatedCallResponseDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VDelegatedCallResponseHandler
}
type VDelegatedCallResponseHandler func(*payload.VDelegatedCallResponse) bool
type PubVDelegatedCallResponseMock struct{ parent *Typed }

func (p PubVDelegatedCallResponseMock) ExpectedCount(count int) PubVDelegatedCallResponseMock {
	p.parent.Handlers.VDelegatedCallResponse.touched = true
	p.parent.Handlers.VDelegatedCallResponse.expectedCount = count
	return p
}

func (p PubVDelegatedCallResponseMock) Set(handler VDelegatedCallResponseHandler) PubVDelegatedCallResponseMock {
	p.parent.Handlers.VDelegatedCallResponse.touched = true
	p.parent.Handlers.VDelegatedCallResponse.handler = handler
	return p
}

func (p PubVDelegatedCallResponseMock) SetResend(resend bool) PubVDelegatedCallResponseMock {
	p.parent.Handlers.VDelegatedCallResponse.touched = true
	p.parent.Handlers.VDelegatedCallResponse.handler = func(*payload.VDelegatedCallResponse) bool { return resend }
	return p
}

func (p PubVDelegatedCallResponseMock) Count() int {
	return p.parent.Handlers.VDelegatedCallResponse.count.Load()
}

func (p PubVDelegatedCallResponseMock) CountBefore() int {
	return p.parent.Handlers.VDelegatedCallResponse.countBefore.Load()
}

func (p PubVDelegatedCallResponseMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.VDelegatedCallResponse.count, count)
}

// ============================================================================

type VDelegatedRequestFinishedDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VDelegatedRequestFinishedHandler
}
type VDelegatedRequestFinishedHandler func(*payload.VDelegatedRequestFinished) bool
type PubVDelegatedRequestFinishedMock struct{ parent *Typed }

func (p PubVDelegatedRequestFinishedMock) ExpectedCount(count int) PubVDelegatedRequestFinishedMock {
	p.parent.Handlers.VDelegatedRequestFinished.touched = true
	p.parent.Handlers.VDelegatedRequestFinished.expectedCount = count
	return p
}

func (p PubVDelegatedRequestFinishedMock) Set(handler VDelegatedRequestFinishedHandler) PubVDelegatedRequestFinishedMock {
	p.parent.Handlers.VDelegatedRequestFinished.touched = true
	p.parent.Handlers.VDelegatedRequestFinished.handler = handler
	return p
}

func (p PubVDelegatedRequestFinishedMock) SetResend(resend bool) PubVDelegatedRequestFinishedMock {
	p.parent.Handlers.VDelegatedRequestFinished.touched = true
	p.parent.Handlers.VDelegatedRequestFinished.handler = func(*payload.VDelegatedRequestFinished) bool { return resend }
	return p
}

func (p PubVDelegatedRequestFinishedMock) Count() int {
	return p.parent.Handlers.VDelegatedRequestFinished.count.Load()
}

func (p PubVDelegatedRequestFinishedMock) CountBefore() int {
	return p.parent.Handlers.VDelegatedRequestFinished.countBefore.Load()
}

func (p PubVDelegatedRequestFinishedMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.VDelegatedRequestFinished.count, count)
}

// ============================================================================

type VFindCallRequestDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VFindCallRequestHandler
}
type VFindCallRequestHandler func(*payload.VFindCallRequest) bool
type PubVFindCallRequestMock struct{ parent *Typed }

func (p PubVFindCallRequestMock) ExpectedCount(count int) PubVFindCallRequestMock {
	p.parent.Handlers.VFindCallRequest.touched = true
	p.parent.Handlers.VFindCallRequest.expectedCount = count
	return p
}

func (p PubVFindCallRequestMock) Set(handler VFindCallRequestHandler) PubVFindCallRequestMock {
	p.parent.Handlers.VFindCallRequest.touched = true
	p.parent.Handlers.VFindCallRequest.handler = handler
	return p
}

func (p PubVFindCallRequestMock) SetResend(resend bool) PubVFindCallRequestMock {
	p.parent.Handlers.VFindCallRequest.touched = true
	p.parent.Handlers.VFindCallRequest.handler = func(*payload.VFindCallRequest) bool { return resend }
	return p
}

func (p PubVFindCallRequestMock) Count() int {
	return p.parent.Handlers.VFindCallRequest.count.Load()
}

func (p PubVFindCallRequestMock) CountBefore() int {
	return p.parent.Handlers.VFindCallRequest.countBefore.Load()
}

func (p PubVFindCallRequestMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.VFindCallRequest.count, count)
}

// ============================================================================

type VFindCallResponseDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VFindCallResponseHandler
}
type VFindCallResponseHandler func(*payload.VFindCallResponse) bool
type PubVFindCallResponseMock struct{ parent *Typed }

func (p PubVFindCallResponseMock) ExpectedCount(count int) PubVFindCallResponseMock {
	p.parent.Handlers.VFindCallResponse.touched = true
	p.parent.Handlers.VFindCallResponse.expectedCount = count
	return p
}

func (p PubVFindCallResponseMock) Set(handler VFindCallResponseHandler) PubVFindCallResponseMock {
	p.parent.Handlers.VFindCallResponse.touched = true
	p.parent.Handlers.VFindCallResponse.handler = handler
	return p
}

func (p PubVFindCallResponseMock) SetResend(resend bool) PubVFindCallResponseMock {
	p.parent.Handlers.VFindCallResponse.touched = true
	p.parent.Handlers.VFindCallResponse.handler = func(*payload.VFindCallResponse) bool { return resend }
	return p
}

func (p PubVFindCallResponseMock) Count() int {
	return p.parent.Handlers.VFindCallResponse.count.Load()
}

func (p PubVFindCallResponseMock) CountBefore() int {
	return p.parent.Handlers.VFindCallResponse.countBefore.Load()
}

func (p PubVFindCallResponseMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.VFindCallResponse.count, count)
}

// ============================================================================

type VObjectTranscriptReportDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VObjectTranscriptReportHandler
}
type VObjectTranscriptReportHandler func(*payload.VObjectTranscriptReport) bool
type PubVObjectTranscriptReportMock struct{ parent *Typed }

func (p PubVObjectTranscriptReportMock) ExpectedCount(count int) PubVObjectTranscriptReportMock {
	p.parent.Handlers.VObjectTranscriptReport.touched = true
	p.parent.Handlers.VObjectTranscriptReport.expectedCount = count
	return p
}

func (p PubVObjectTranscriptReportMock) Set(handler VObjectTranscriptReportHandler) PubVObjectTranscriptReportMock {
	p.parent.Handlers.VObjectTranscriptReport.touched = true
	p.parent.Handlers.VObjectTranscriptReport.handler = handler
	return p
}

func (p PubVObjectTranscriptReportMock) SetResend(resend bool) PubVObjectTranscriptReportMock {
	p.parent.Handlers.VObjectTranscriptReport.touched = true
	p.parent.Handlers.VObjectTranscriptReport.handler = func(*payload.VObjectTranscriptReport) bool { return resend }
	return p
}

func (p PubVObjectTranscriptReportMock) Count() int {
	return p.parent.Handlers.VObjectTranscriptReport.count.Load()
}

func (p PubVObjectTranscriptReportMock) CountBefore() int {
	return p.parent.Handlers.VObjectTranscriptReport.countBefore.Load()
}

func (p PubVObjectTranscriptReportMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.VObjectTranscriptReport.count, count)
}

// ============================================================================

type VStateReportDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VStateReportHandler
}
type VStateReportHandler func(*payload.VStateReport) bool
type PubVStateReportMock struct{ parent *Typed }

func (p PubVStateReportMock) ExpectedCount(count int) PubVStateReportMock {
	p.parent.Handlers.VStateReport.touched = true
	p.parent.Handlers.VStateReport.expectedCount = count
	return p
}

func (p PubVStateReportMock) Set(handler VStateReportHandler) PubVStateReportMock {
	p.parent.Handlers.VStateReport.touched = true
	p.parent.Handlers.VStateReport.handler = handler
	return p
}

func (p PubVStateReportMock) SetResend(resend bool) PubVStateReportMock {
	p.parent.Handlers.VStateReport.touched = true
	p.parent.Handlers.VStateReport.handler = func(*payload.VStateReport) bool { return resend }
	return p
}

func (p PubVStateReportMock) Count() int {
	return p.parent.Handlers.VStateReport.count.Load()
}

func (p PubVStateReportMock) CountBefore() int {
	return p.parent.Handlers.VStateReport.countBefore.Load()
}

func (p PubVStateReportMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.VStateReport.count, count)
}

// ============================================================================

type VStateRequestDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VStateRequestHandler
}
type VStateRequestHandler func(*payload.VStateRequest) bool
type PubVStateRequestMock struct{ parent *Typed }

func (p PubVStateRequestMock) ExpectedCount(count int) PubVStateRequestMock {
	p.parent.Handlers.VStateRequest.touched = true
	p.parent.Handlers.VStateRequest.expectedCount = count
	return p
}

func (p PubVStateRequestMock) Set(handler VStateRequestHandler) PubVStateRequestMock {
	p.parent.Handlers.VStateRequest.touched = true
	p.parent.Handlers.VStateRequest.handler = handler
	return p
}

func (p PubVStateRequestMock) SetResend(resend bool) PubVStateRequestMock {
	p.parent.Handlers.VStateRequest.touched = true
	p.parent.Handlers.VStateRequest.handler = func(*payload.VStateRequest) bool { return resend }
	return p
}

func (p PubVStateRequestMock) Count() int {
	return p.parent.Handlers.VStateRequest.count.Load()
}

func (p PubVStateRequestMock) CountBefore() int {
	return p.parent.Handlers.VStateRequest.countBefore.Load()
}

func (p PubVStateRequestMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.VStateRequest.count, count)
}

// ============================================================================

type TypedHandlers struct {
	VCallRequest              VCallRequestDefinition
	VCallResult               VCallResultDefinition
	VDelegatedCallRequest     VDelegatedCallRequestDefinition
	VDelegatedCallResponse    VDelegatedCallResponseDefinition
	VDelegatedRequestFinished VDelegatedRequestFinishedDefinition
	VFindCallRequest          VFindCallRequestDefinition
	VFindCallResponse         VFindCallResponseDefinition
	VObjectTranscriptReport   VObjectTranscriptReportDefinition
	VStateReport              VStateReportDefinition
	VStateRequest             VStateRequestDefinition

	BaseMessage struct {
		handler func(message *message.Message)
	}
}

type Typed struct {
	t             minimock.Tester
	timeout       time.Duration
	ctx           context.Context
	defaultResend bool
	resend        func(ctx context.Context, msg *message.Message)

	Handlers TypedHandlers

	VCallRequest              PubVCallRequestMock
	VCallResult               PubVCallResultMock
	VDelegatedCallRequest     PubVDelegatedCallRequestMock
	VDelegatedCallResponse    PubVDelegatedCallResponseMock
	VDelegatedRequestFinished PubVDelegatedRequestFinishedMock
	VFindCallRequest          PubVFindCallRequestMock
	VFindCallResponse         PubVFindCallResponseMock
	VObjectTranscriptReport   PubVObjectTranscriptReportMock
	VStateReport              PubVStateReportMock
	VStateRequest             PubVStateRequestMock
}

func NewTyped(ctx context.Context, t minimock.Tester, sender Sender) *Typed {
	checker := &Typed{
		t:             t,
		ctx:           ctx,
		defaultResend: false,
		timeout:       10 * time.Second,
		resend:        sender.SendMessage,

		Handlers: TypedHandlers{
			VCallRequest:              VCallRequestDefinition{expectedCount: -1},
			VCallResult:               VCallResultDefinition{expectedCount: -1},
			VDelegatedCallRequest:     VDelegatedCallRequestDefinition{expectedCount: -1},
			VDelegatedCallResponse:    VDelegatedCallResponseDefinition{expectedCount: -1},
			VDelegatedRequestFinished: VDelegatedRequestFinishedDefinition{expectedCount: -1},
			VFindCallRequest:          VFindCallRequestDefinition{expectedCount: -1},
			VFindCallResponse:         VFindCallResponseDefinition{expectedCount: -1},
			VObjectTranscriptReport:   VObjectTranscriptReportDefinition{expectedCount: -1},
			VStateReport:              VStateReportDefinition{expectedCount: -1},
			VStateRequest:             VStateRequestDefinition{expectedCount: -1},
		},
	}

	checker.VCallRequest = PubVCallRequestMock{parent: checker}
	checker.VCallResult = PubVCallResultMock{parent: checker}
	checker.VDelegatedCallRequest = PubVDelegatedCallRequestMock{parent: checker}
	checker.VDelegatedCallResponse = PubVDelegatedCallResponseMock{parent: checker}
	checker.VDelegatedRequestFinished = PubVDelegatedRequestFinishedMock{parent: checker}
	checker.VFindCallRequest = PubVFindCallRequestMock{parent: checker}
	checker.VFindCallResponse = PubVFindCallResponseMock{parent: checker}
	checker.VObjectTranscriptReport = PubVObjectTranscriptReportMock{parent: checker}
	checker.VStateReport = PubVStateReportMock{parent: checker}
	checker.VStateRequest = PubVStateRequestMock{parent: checker}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(checker)
	}

	return checker
}

func (p *Typed) CheckMessages(topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		p.checkMessage(p.ctx, msg)
	}

	return nil
}

func (p *Typed) checkMessage(ctx context.Context, msg *message.Message) {
	basePayload, err := payload.UnmarshalFromMeta(msg.Payload)
	if err != nil {
		return
	}

	var resend bool

	switch payload := basePayload.(type) {
	case *payload.VCallRequest:
		hdlStruct := &p.Handlers.VCallRequest

		resend = p.defaultResend

		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.handler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resend = hdlStruct.handler(payload)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message VCallRequest (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VCallResult:
		hdlStruct := &p.Handlers.VCallResult

		resend = p.defaultResend

		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.handler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resend = hdlStruct.handler(payload)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message VCallResult (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VDelegatedCallRequest:
		hdlStruct := &p.Handlers.VDelegatedCallRequest

		resend = p.defaultResend

		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.handler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resend = hdlStruct.handler(payload)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message VDelegatedCallRequest (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VDelegatedCallResponse:
		hdlStruct := &p.Handlers.VDelegatedCallResponse

		resend = p.defaultResend

		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.handler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resend = hdlStruct.handler(payload)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message VDelegatedCallResponse (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VDelegatedRequestFinished:
		hdlStruct := &p.Handlers.VDelegatedRequestFinished

		resend = p.defaultResend

		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.handler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resend = hdlStruct.handler(payload)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message VDelegatedRequestFinished (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VFindCallRequest:
		hdlStruct := &p.Handlers.VFindCallRequest

		resend = p.defaultResend

		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.handler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resend = hdlStruct.handler(payload)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message VFindCallRequest (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VFindCallResponse:
		hdlStruct := &p.Handlers.VFindCallResponse

		resend = p.defaultResend

		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.handler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resend = hdlStruct.handler(payload)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message VFindCallResponse (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VObjectTranscriptReport:
		hdlStruct := &p.Handlers.VObjectTranscriptReport

		resend = p.defaultResend

		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.handler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resend = hdlStruct.handler(payload)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message VObjectTranscriptReport (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VStateReport:
		hdlStruct := &p.Handlers.VStateReport

		resend = p.defaultResend

		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.handler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resend = hdlStruct.handler(payload)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message VStateReport (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VStateRequest:
		hdlStruct := &p.Handlers.VStateRequest

		resend = p.defaultResend

		oldCount := hdlStruct.countBefore.Add(1)

		if hdlStruct.handler != nil {
			done := make(synckit.ClosableSignalChannel)

			go func() {
				defer func() { _ = synckit.SafeClose(done) }()

				resend = hdlStruct.handler(payload)
			}()

			select {
			case <-done:
			case <-time.After(p.timeout):
				p.t.Error("timeout: failed to check message VStateRequest (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	default:
		p.t.Fatalf("unexpected %T payload", basePayload)
		return
	}

	if resend {
		p.resend(ctx, msg)
	}
}

func (p *Typed) SetDefaultResend(flag bool) *Typed {
	p.defaultResend = flag
	return p
}

func (p *Typed) minimockDone() bool {
	ok := true

	{
		fn := func() bool {
			hdl := &p.Handlers.VCallRequest

			switch {
			case hdl.expectedCount < 0:
				return true
			case p.defaultResend:
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
			hdl := &p.Handlers.VCallResult

			switch {
			case hdl.expectedCount < 0:
				return true
			case p.defaultResend:
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
			hdl := &p.Handlers.VDelegatedCallRequest

			switch {
			case hdl.expectedCount < 0:
				return true
			case p.defaultResend:
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
			hdl := &p.Handlers.VDelegatedCallResponse

			switch {
			case hdl.expectedCount < 0:
				return true
			case p.defaultResend:
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
			hdl := &p.Handlers.VDelegatedRequestFinished

			switch {
			case hdl.expectedCount < 0:
				return true
			case p.defaultResend:
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
			hdl := &p.Handlers.VFindCallRequest

			switch {
			case hdl.expectedCount < 0:
				return true
			case p.defaultResend:
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
			hdl := &p.Handlers.VFindCallResponse

			switch {
			case hdl.expectedCount < 0:
				return true
			case p.defaultResend:
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
			hdl := &p.Handlers.VObjectTranscriptReport

			switch {
			case hdl.expectedCount < 0:
				return true
			case p.defaultResend:
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
			hdl := &p.Handlers.VStateReport

			switch {
			case hdl.expectedCount < 0:
				return true
			case p.defaultResend:
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
			hdl := &p.Handlers.VStateRequest

			switch {
			case hdl.expectedCount < 0:
				return true
			case p.defaultResend:
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
func (p *Typed) MinimockFinish() {
	if !p.minimockDone() {
		p.t.Fatal("failed conditions on Typed")
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (p *Typed) MinimockWait(timeout time.Duration) {
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
