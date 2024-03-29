package checker

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

// ============================================================================

type LRegisterRequestDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       LRegisterRequestHandler
}
type LRegisterRequestHandler func(*rms.LRegisterRequest) bool
type PubLRegisterRequestMock struct{ parent *Typed }

func (p PubLRegisterRequestMock) ExpectedCount(count int) PubLRegisterRequestMock {
	p.parent.Handlers.LRegisterRequest.touched = true
	p.parent.Handlers.LRegisterRequest.expectedCount = count
	return p
}

func (p PubLRegisterRequestMock) Set(handler LRegisterRequestHandler) PubLRegisterRequestMock {
	p.parent.Handlers.LRegisterRequest.touched = true
	p.parent.Handlers.LRegisterRequest.handler = handler
	return p
}

func (p PubLRegisterRequestMock) SetResend(resend bool) PubLRegisterRequestMock {
	p.parent.Handlers.LRegisterRequest.touched = true
	p.parent.Handlers.LRegisterRequest.handler = func(*rms.LRegisterRequest) bool { return resend }
	return p
}

func (p PubLRegisterRequestMock) Count() int {
	return p.parent.Handlers.LRegisterRequest.count.Load()
}

func (p PubLRegisterRequestMock) CountBefore() int {
	return p.parent.Handlers.LRegisterRequest.countBefore.Load()
}

func (p PubLRegisterRequestMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.LRegisterRequest.count, count)
}

// ============================================================================

type LRegisterResponseDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       LRegisterResponseHandler
}
type LRegisterResponseHandler func(*rms.LRegisterResponse) bool
type PubLRegisterResponseMock struct{ parent *Typed }

func (p PubLRegisterResponseMock) ExpectedCount(count int) PubLRegisterResponseMock {
	p.parent.Handlers.LRegisterResponse.touched = true
	p.parent.Handlers.LRegisterResponse.expectedCount = count
	return p
}

func (p PubLRegisterResponseMock) Set(handler LRegisterResponseHandler) PubLRegisterResponseMock {
	p.parent.Handlers.LRegisterResponse.touched = true
	p.parent.Handlers.LRegisterResponse.handler = handler
	return p
}

func (p PubLRegisterResponseMock) SetResend(resend bool) PubLRegisterResponseMock {
	p.parent.Handlers.LRegisterResponse.touched = true
	p.parent.Handlers.LRegisterResponse.handler = func(*rms.LRegisterResponse) bool { return resend }
	return p
}

func (p PubLRegisterResponseMock) Count() int {
	return p.parent.Handlers.LRegisterResponse.count.Load()
}

func (p PubLRegisterResponseMock) CountBefore() int {
	return p.parent.Handlers.LRegisterResponse.countBefore.Load()
}

func (p PubLRegisterResponseMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.LRegisterResponse.count, count)
}

// ============================================================================

type VCachedMemoryRequestDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VCachedMemoryRequestHandler
}
type VCachedMemoryRequestHandler func(*rms.VCachedMemoryRequest) bool
type PubVCachedMemoryRequestMock struct{ parent *Typed }

func (p PubVCachedMemoryRequestMock) ExpectedCount(count int) PubVCachedMemoryRequestMock {
	p.parent.Handlers.VCachedMemoryRequest.touched = true
	p.parent.Handlers.VCachedMemoryRequest.expectedCount = count
	return p
}

func (p PubVCachedMemoryRequestMock) Set(handler VCachedMemoryRequestHandler) PubVCachedMemoryRequestMock {
	p.parent.Handlers.VCachedMemoryRequest.touched = true
	p.parent.Handlers.VCachedMemoryRequest.handler = handler
	return p
}

func (p PubVCachedMemoryRequestMock) SetResend(resend bool) PubVCachedMemoryRequestMock {
	p.parent.Handlers.VCachedMemoryRequest.touched = true
	p.parent.Handlers.VCachedMemoryRequest.handler = func(*rms.VCachedMemoryRequest) bool { return resend }
	return p
}

func (p PubVCachedMemoryRequestMock) Count() int {
	return p.parent.Handlers.VCachedMemoryRequest.count.Load()
}

func (p PubVCachedMemoryRequestMock) CountBefore() int {
	return p.parent.Handlers.VCachedMemoryRequest.countBefore.Load()
}

func (p PubVCachedMemoryRequestMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.VCachedMemoryRequest.count, count)
}

// ============================================================================

type VCachedMemoryResponseDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VCachedMemoryResponseHandler
}
type VCachedMemoryResponseHandler func(*rms.VCachedMemoryResponse) bool
type PubVCachedMemoryResponseMock struct{ parent *Typed }

func (p PubVCachedMemoryResponseMock) ExpectedCount(count int) PubVCachedMemoryResponseMock {
	p.parent.Handlers.VCachedMemoryResponse.touched = true
	p.parent.Handlers.VCachedMemoryResponse.expectedCount = count
	return p
}

func (p PubVCachedMemoryResponseMock) Set(handler VCachedMemoryResponseHandler) PubVCachedMemoryResponseMock {
	p.parent.Handlers.VCachedMemoryResponse.touched = true
	p.parent.Handlers.VCachedMemoryResponse.handler = handler
	return p
}

func (p PubVCachedMemoryResponseMock) SetResend(resend bool) PubVCachedMemoryResponseMock {
	p.parent.Handlers.VCachedMemoryResponse.touched = true
	p.parent.Handlers.VCachedMemoryResponse.handler = func(*rms.VCachedMemoryResponse) bool { return resend }
	return p
}

func (p PubVCachedMemoryResponseMock) Count() int {
	return p.parent.Handlers.VCachedMemoryResponse.count.Load()
}

func (p PubVCachedMemoryResponseMock) CountBefore() int {
	return p.parent.Handlers.VCachedMemoryResponse.countBefore.Load()
}

func (p PubVCachedMemoryResponseMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.VCachedMemoryResponse.count, count)
}

// ============================================================================

type VCallRequestDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VCallRequestHandler
}
type VCallRequestHandler func(*rms.VCallRequest) bool
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
	p.parent.Handlers.VCallRequest.handler = func(*rms.VCallRequest) bool { return resend }
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
type VCallResultHandler func(*rms.VCallResult) bool
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
	p.parent.Handlers.VCallResult.handler = func(*rms.VCallResult) bool { return resend }
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
type VDelegatedCallRequestHandler func(*rms.VDelegatedCallRequest) bool
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
	p.parent.Handlers.VDelegatedCallRequest.handler = func(*rms.VDelegatedCallRequest) bool { return resend }
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
type VDelegatedCallResponseHandler func(*rms.VDelegatedCallResponse) bool
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
	p.parent.Handlers.VDelegatedCallResponse.handler = func(*rms.VDelegatedCallResponse) bool { return resend }
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
type VDelegatedRequestFinishedHandler func(*rms.VDelegatedRequestFinished) bool
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
	p.parent.Handlers.VDelegatedRequestFinished.handler = func(*rms.VDelegatedRequestFinished) bool { return resend }
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
type VFindCallRequestHandler func(*rms.VFindCallRequest) bool
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
	p.parent.Handlers.VFindCallRequest.handler = func(*rms.VFindCallRequest) bool { return resend }
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
type VFindCallResponseHandler func(*rms.VFindCallResponse) bool
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
	p.parent.Handlers.VFindCallResponse.handler = func(*rms.VFindCallResponse) bool { return resend }
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
type VObjectTranscriptReportHandler func(*rms.VObjectTranscriptReport) bool
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
	p.parent.Handlers.VObjectTranscriptReport.handler = func(*rms.VObjectTranscriptReport) bool { return resend }
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

type VObjectValidationReportDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VObjectValidationReportHandler
}
type VObjectValidationReportHandler func(*rms.VObjectValidationReport) bool
type PubVObjectValidationReportMock struct{ parent *Typed }

func (p PubVObjectValidationReportMock) ExpectedCount(count int) PubVObjectValidationReportMock {
	p.parent.Handlers.VObjectValidationReport.touched = true
	p.parent.Handlers.VObjectValidationReport.expectedCount = count
	return p
}

func (p PubVObjectValidationReportMock) Set(handler VObjectValidationReportHandler) PubVObjectValidationReportMock {
	p.parent.Handlers.VObjectValidationReport.touched = true
	p.parent.Handlers.VObjectValidationReport.handler = handler
	return p
}

func (p PubVObjectValidationReportMock) SetResend(resend bool) PubVObjectValidationReportMock {
	p.parent.Handlers.VObjectValidationReport.touched = true
	p.parent.Handlers.VObjectValidationReport.handler = func(*rms.VObjectValidationReport) bool { return resend }
	return p
}

func (p PubVObjectValidationReportMock) Count() int {
	return p.parent.Handlers.VObjectValidationReport.count.Load()
}

func (p PubVObjectValidationReportMock) CountBefore() int {
	return p.parent.Handlers.VObjectValidationReport.countBefore.Load()
}

func (p PubVObjectValidationReportMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
	return waitCounterIndefinitely(ctx, &p.parent.Handlers.VObjectValidationReport.count, count)
}

// ============================================================================

type VStateReportDefinition struct {
	touched       bool
	count         atomickit.Int
	countBefore   atomickit.Int
	expectedCount int
	handler       VStateReportHandler
}
type VStateReportHandler func(*rms.VStateReport) bool
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
	p.parent.Handlers.VStateReport.handler = func(*rms.VStateReport) bool { return resend }
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
type VStateRequestHandler func(*rms.VStateRequest) bool
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
	p.parent.Handlers.VStateRequest.handler = func(*rms.VStateRequest) bool { return resend }
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
	LRegisterRequest          LRegisterRequestDefinition
	LRegisterResponse         LRegisterResponseDefinition
	VCachedMemoryRequest      VCachedMemoryRequestDefinition
	VCachedMemoryResponse     VCachedMemoryResponseDefinition
	VCallRequest              VCallRequestDefinition
	VCallResult               VCallResultDefinition
	VDelegatedCallRequest     VDelegatedCallRequestDefinition
	VDelegatedCallResponse    VDelegatedCallResponseDefinition
	VDelegatedRequestFinished VDelegatedRequestFinishedDefinition
	VFindCallRequest          VFindCallRequestDefinition
	VFindCallResponse         VFindCallResponseDefinition
	VObjectTranscriptReport   VObjectTranscriptReportDefinition
	VObjectValidationReport   VObjectValidationReportDefinition
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

	LRegisterRequest          PubLRegisterRequestMock
	LRegisterResponse         PubLRegisterResponseMock
	VCachedMemoryRequest      PubVCachedMemoryRequestMock
	VCachedMemoryResponse     PubVCachedMemoryResponseMock
	VCallRequest              PubVCallRequestMock
	VCallResult               PubVCallResultMock
	VDelegatedCallRequest     PubVDelegatedCallRequestMock
	VDelegatedCallResponse    PubVDelegatedCallResponseMock
	VDelegatedRequestFinished PubVDelegatedRequestFinishedMock
	VFindCallRequest          PubVFindCallRequestMock
	VFindCallResponse         PubVFindCallResponseMock
	VObjectTranscriptReport   PubVObjectTranscriptReportMock
	VObjectValidationReport   PubVObjectValidationReportMock
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
			LRegisterRequest:          LRegisterRequestDefinition{expectedCount: -1},
			LRegisterResponse:         LRegisterResponseDefinition{expectedCount: -1},
			VCachedMemoryRequest:      VCachedMemoryRequestDefinition{expectedCount: -1},
			VCachedMemoryResponse:     VCachedMemoryResponseDefinition{expectedCount: -1},
			VCallRequest:              VCallRequestDefinition{expectedCount: -1},
			VCallResult:               VCallResultDefinition{expectedCount: -1},
			VDelegatedCallRequest:     VDelegatedCallRequestDefinition{expectedCount: -1},
			VDelegatedCallResponse:    VDelegatedCallResponseDefinition{expectedCount: -1},
			VDelegatedRequestFinished: VDelegatedRequestFinishedDefinition{expectedCount: -1},
			VFindCallRequest:          VFindCallRequestDefinition{expectedCount: -1},
			VFindCallResponse:         VFindCallResponseDefinition{expectedCount: -1},
			VObjectTranscriptReport:   VObjectTranscriptReportDefinition{expectedCount: -1},
			VObjectValidationReport:   VObjectValidationReportDefinition{expectedCount: -1},
			VStateReport:              VStateReportDefinition{expectedCount: -1},
			VStateRequest:             VStateRequestDefinition{expectedCount: -1},
		},
	}

	checker.LRegisterRequest = PubLRegisterRequestMock{parent: checker}
	checker.LRegisterResponse = PubLRegisterResponseMock{parent: checker}
	checker.VCachedMemoryRequest = PubVCachedMemoryRequestMock{parent: checker}
	checker.VCachedMemoryResponse = PubVCachedMemoryResponseMock{parent: checker}
	checker.VCallRequest = PubVCallRequestMock{parent: checker}
	checker.VCallResult = PubVCallResultMock{parent: checker}
	checker.VDelegatedCallRequest = PubVDelegatedCallRequestMock{parent: checker}
	checker.VDelegatedCallResponse = PubVDelegatedCallResponseMock{parent: checker}
	checker.VDelegatedRequestFinished = PubVDelegatedRequestFinishedMock{parent: checker}
	checker.VFindCallRequest = PubVFindCallRequestMock{parent: checker}
	checker.VFindCallResponse = PubVFindCallResponseMock{parent: checker}
	checker.VObjectTranscriptReport = PubVObjectTranscriptReportMock{parent: checker}
	checker.VObjectValidationReport = PubVObjectValidationReportMock{parent: checker}
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
	var meta rms.Meta

	if err := meta.Unmarshal(msg.Payload); err != nil {
		p.t.Fatalf("failed to unmarshal %T", msg.Payload)
		return
	}

	var resend bool

	switch payload := meta.Payload.Get().(type) {
	case *rms.LRegisterRequest:
		hdlStruct := &p.Handlers.LRegisterRequest

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
				p.t.Error("timeout: failed to check message LRegisterRequest (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *rms.LRegisterResponse:
		hdlStruct := &p.Handlers.LRegisterResponse

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
				p.t.Error("timeout: failed to check message LRegisterResponse (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *rms.VCachedMemoryRequest:
		hdlStruct := &p.Handlers.VCachedMemoryRequest

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
				p.t.Error("timeout: failed to check message VCachedMemoryRequest (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *rms.VCachedMemoryResponse:
		hdlStruct := &p.Handlers.VCachedMemoryResponse

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
				p.t.Error("timeout: failed to check message VCachedMemoryResponse (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *rms.VCallRequest:
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

	case *rms.VCallResult:
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

	case *rms.VDelegatedCallRequest:
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

	case *rms.VDelegatedCallResponse:
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

	case *rms.VDelegatedRequestFinished:
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

	case *rms.VFindCallRequest:
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

	case *rms.VFindCallResponse:
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

	case *rms.VObjectTranscriptReport:
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

	case *rms.VObjectValidationReport:
		hdlStruct := &p.Handlers.VObjectValidationReport

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
				p.t.Error("timeout: failed to check message VObjectValidationReport (position: %s)", oldCount)
			}
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *rms.VStateReport:
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

	case *rms.VStateRequest:
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
		p.t.Fatalf("unexpected %T payload", meta.Payload.Get())
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
			hdl := &p.Handlers.LRegisterRequest

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
			hdl := &p.Handlers.LRegisterResponse

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
			hdl := &p.Handlers.VCachedMemoryRequest

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
			hdl := &p.Handlers.VCachedMemoryResponse

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
			hdl := &p.Handlers.VObjectValidationReport

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
