// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package mock

import (
	"context"
	mm_time "time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gojuno/minimock"

	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
)

// ============================================================================

type VCallRequestHandler func(*payload.VCallRequest) bool
type PubVCallRequestMock struct{ parent *TypePublishChecker }

func (p PubVCallRequestMock) ExpectedCount(count int) PubVCallRequestMock {
	p.parent.Handlers.VCallRequest.touched = true
	p.parent.Handlers.VCallRequest.expectedCount = count
	return p
}

func (p PubVCallRequestMock) Set(handler VCallRequestHandler) PubVCallRequestMock {
	p.parent.Handlers.VCallRequest.touched = true
	p.parent.Handlers.VCallRequest.handler = handler
	if p.parent.Handlers.VCallRequest.expectedCount == 0 {
		p.parent.Handlers.VCallRequest.expectedCount = -1
	}
	return p
}

func (p PubVCallRequestMock) SetResend() PubVCallRequestMock {
	p.parent.Handlers.VCallRequest.touched = true
	p.parent.Handlers.VCallRequest.handler = func(*payload.VCallRequest) bool { return true }
	return p
}

// ============================================================================

type VCallResultHandler func(*payload.VCallResult) bool
type PubVCallResultMock struct{ parent *TypePublishChecker }

func (p PubVCallResultMock) ExpectedCount(count int) PubVCallResultMock {
	p.parent.Handlers.VCallResult.touched = true
	p.parent.Handlers.VCallResult.expectedCount = count
	return p
}

func (p PubVCallResultMock) Set(handler VCallResultHandler) PubVCallResultMock {
	p.parent.Handlers.VCallResult.touched = true
	p.parent.Handlers.VCallResult.handler = handler
	if p.parent.Handlers.VCallResult.expectedCount == 0 {
		p.parent.Handlers.VCallResult.expectedCount = -1
	}
	return p
}

func (p PubVCallResultMock) SetResend() PubVCallResultMock {
	p.parent.Handlers.VCallResult.touched = true
	p.parent.Handlers.VCallResult.handler = func(*payload.VCallResult) bool { return true }
	return p
}

// ============================================================================

type VDelegatedCallRequestHandler func(*payload.VDelegatedCallRequest) bool
type PubVDelegatedCallRequestMock struct{ parent *TypePublishChecker }

func (p PubVDelegatedCallRequestMock) ExpectedCount(count int) PubVDelegatedCallRequestMock {
	p.parent.Handlers.VDelegatedCallRequest.touched = true
	p.parent.Handlers.VDelegatedCallRequest.expectedCount = count
	return p
}

func (p PubVDelegatedCallRequestMock) Set(handler VDelegatedCallRequestHandler) PubVDelegatedCallRequestMock {
	p.parent.Handlers.VDelegatedCallRequest.touched = true
	p.parent.Handlers.VDelegatedCallRequest.handler = handler
	if p.parent.Handlers.VDelegatedCallRequest.expectedCount == 0 {
		p.parent.Handlers.VDelegatedCallRequest.expectedCount = -1
	}
	return p
}

func (p PubVDelegatedCallRequestMock) SetResend() PubVDelegatedCallRequestMock {
	p.parent.Handlers.VDelegatedCallRequest.touched = true
	p.parent.Handlers.VDelegatedCallRequest.handler = func(*payload.VDelegatedCallRequest) bool { return true }
	return p
}

// ============================================================================

type VDelegatedCallResponseHandler func(*payload.VDelegatedCallResponse) bool
type PubVDelegatedCallResponseMock struct{ parent *TypePublishChecker }

func (p PubVDelegatedCallResponseMock) ExpectedCount(count int) PubVDelegatedCallResponseMock {
	p.parent.Handlers.VDelegatedCallResponse.touched = true
	p.parent.Handlers.VDelegatedCallResponse.expectedCount = count
	return p
}

func (p PubVDelegatedCallResponseMock) Set(handler VDelegatedCallResponseHandler) PubVDelegatedCallResponseMock {
	p.parent.Handlers.VDelegatedCallResponse.touched = true
	p.parent.Handlers.VDelegatedCallResponse.handler = handler
	if p.parent.Handlers.VDelegatedCallResponse.expectedCount == 0 {
		p.parent.Handlers.VDelegatedCallResponse.expectedCount = -1
	}
	return p
}

func (p PubVDelegatedCallResponseMock) SetResend() PubVDelegatedCallResponseMock {
	p.parent.Handlers.VDelegatedCallResponse.touched = true
	p.parent.Handlers.VDelegatedCallResponse.handler = func(*payload.VDelegatedCallResponse) bool { return true }
	return p
}

// ============================================================================

type VDelegatedRequestFinishedHandler func(*payload.VDelegatedRequestFinished) bool
type PubVDelegatedRequestFinishedMock struct{ parent *TypePublishChecker }

func (p PubVDelegatedRequestFinishedMock) ExpectedCount(count int) PubVDelegatedRequestFinishedMock {
	p.parent.Handlers.VDelegatedRequestFinished.touched = true
	p.parent.Handlers.VDelegatedRequestFinished.expectedCount = count
	return p
}

func (p PubVDelegatedRequestFinishedMock) Set(handler VDelegatedRequestFinishedHandler) PubVDelegatedRequestFinishedMock {
	p.parent.Handlers.VDelegatedRequestFinished.touched = true
	p.parent.Handlers.VDelegatedRequestFinished.handler = handler
	if p.parent.Handlers.VDelegatedRequestFinished.expectedCount == 0 {
		p.parent.Handlers.VDelegatedRequestFinished.expectedCount = -1
	}
	return p
}

func (p PubVDelegatedRequestFinishedMock) SetResend() PubVDelegatedRequestFinishedMock {
	p.parent.Handlers.VDelegatedRequestFinished.touched = true
	p.parent.Handlers.VDelegatedRequestFinished.handler = func(*payload.VDelegatedRequestFinished) bool { return true }
	return p
}

// ============================================================================

type VStateRequestHandler func(*payload.VStateRequest) bool
type PubVStateRequestMock struct{ parent *TypePublishChecker }

func (p PubVStateRequestMock) ExpectedCount(count int) PubVStateRequestMock {
	p.parent.Handlers.VStateRequest.touched = true
	p.parent.Handlers.VStateRequest.expectedCount = count
	return p
}

func (p PubVStateRequestMock) Set(handler VStateRequestHandler) PubVStateRequestMock {
	p.parent.Handlers.VStateRequest.touched = true
	p.parent.Handlers.VStateRequest.handler = handler
	if p.parent.Handlers.VStateRequest.expectedCount == 0 {
		p.parent.Handlers.VStateRequest.expectedCount = -1
	}
	return p
}

func (p PubVStateRequestMock) SetResend() PubVStateRequestMock {
	p.parent.Handlers.VStateRequest.touched = true
	p.parent.Handlers.VStateRequest.handler = func(*payload.VStateRequest) bool { return true }
	return p
}

// ============================================================================

type VStateReportHandler func(*payload.VStateReport) bool
type PubVStateReportMock struct{ parent *TypePublishChecker }

func (p PubVStateReportMock) ExpectedCount(count int) PubVStateReportMock {
	p.parent.Handlers.VStateReport.touched = true
	p.parent.Handlers.VStateReport.expectedCount = count
	return p
}

func (p PubVStateReportMock) Set(handler VStateReportHandler) PubVStateReportMock {
	p.parent.Handlers.VStateReport.touched = true
	p.parent.Handlers.VStateReport.handler = handler
	if p.parent.Handlers.VStateReport.expectedCount == 0 {
		p.parent.Handlers.VStateReport.expectedCount = -1
	}
	return p
}

func (p PubVStateReportMock) SetResend() PubVStateReportMock {
	p.parent.Handlers.VStateReport.touched = true
	p.parent.Handlers.VStateReport.handler = func(*payload.VStateReport) bool { return true }
	return p
}

// ============================================================================

type TypePublishCheckerHandlers struct {
	VCallRequest struct {
		touched       bool
		count         atomickit.Int
		expectedCount int
		handler       VCallRequestHandler
	}
	VCallResult struct {
		touched       bool
		count         atomickit.Int
		expectedCount int
		handler       VCallResultHandler
	}
	VDelegatedCallRequest struct {
		touched       bool
		count         atomickit.Int
		expectedCount int
		handler       VDelegatedCallRequestHandler
	}
	VDelegatedCallResponse struct {
		touched       bool
		count         atomickit.Int
		expectedCount int
		handler       VDelegatedCallResponseHandler
	}
	VDelegatedRequestFinished struct {
		touched       bool
		count         atomickit.Int
		expectedCount int
		handler       VDelegatedRequestFinishedHandler
	}
	VStateRequest struct {
		touched       bool
		count         atomickit.Int
		expectedCount int
		handler       VStateRequestHandler
	}
	VStateReport struct {
		touched       bool
		count         atomickit.Int
		expectedCount int
		handler       VStateReportHandler
	}

	BaseMessage struct {
		handler func(message *message.Message)
	}
}

type TypePublishChecker struct {
	t             minimock.Tester
	ctx           context.Context
	defaultResend bool
	resend        func(ctx context.Context, msg *message.Message)

	Handlers TypePublishCheckerHandlers

	VCallRequest              PubVCallRequestMock
	VCallResult               PubVCallResultMock
	VDelegatedCallRequest     PubVDelegatedCallRequestMock
	VDelegatedCallResponse    PubVDelegatedCallResponseMock
	VDelegatedRequestFinished PubVDelegatedRequestFinishedMock
	VStateRequest             PubVStateRequestMock
	VStateReport              PubVStateReportMock
}

func NewTypePublishChecker(ctx context.Context, t minimock.Tester, sender Sender) *TypePublishChecker {
	checker := &TypePublishChecker{
		t:             t,
		ctx:           ctx,
		defaultResend: false,
		resend:        sender.SendMessage,
	}

	checker.VCallRequest = PubVCallRequestMock{parent: checker}
	checker.VCallResult = PubVCallResultMock{parent: checker}
	checker.VDelegatedCallRequest = PubVDelegatedCallRequestMock{parent: checker}
	checker.VDelegatedCallResponse = PubVDelegatedCallResponseMock{parent: checker}
	checker.VDelegatedRequestFinished = PubVDelegatedRequestFinishedMock{parent: checker}
	checker.VStateRequest = PubVStateRequestMock{parent: checker}
	checker.VStateReport = PubVStateReportMock{parent: checker}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(checker)
	}

	return checker
}

func (p *TypePublishChecker) CheckMessages(topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		p.checkMessage(p.ctx, msg)
	}

	return nil
}

func (p *TypePublishChecker) checkMessage(ctx context.Context, msg *message.Message) {
	basePayload, err := payload.UnmarshalFromMeta(msg.Payload)
	if err != nil {
		return
	}

	var resend bool

	switch payload := basePayload.(type) {
	case *payload.VCallRequest:
		hdlStruct := &p.Handlers.VCallRequest

		resend = p.defaultResend

		if hdlStruct.handler != nil {
			resend = hdlStruct.handler(payload)
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VCallResult:
		hdlStruct := &p.Handlers.VCallResult

		resend = p.defaultResend

		if hdlStruct.handler != nil {
			resend = hdlStruct.handler(payload)
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VDelegatedCallRequest:
		hdlStruct := &p.Handlers.VDelegatedCallRequest

		resend = p.defaultResend

		if hdlStruct.handler != nil {
			resend = hdlStruct.handler(payload)
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VDelegatedCallResponse:
		hdlStruct := &p.Handlers.VDelegatedCallResponse

		resend = p.defaultResend

		if hdlStruct.handler != nil {
			resend = hdlStruct.handler(payload)
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VDelegatedRequestFinished:
		hdlStruct := &p.Handlers.VDelegatedRequestFinished

		resend = p.defaultResend

		if hdlStruct.handler != nil {
			resend = hdlStruct.handler(payload)
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VStateRequest:
		hdlStruct := &p.Handlers.VStateRequest

		resend = p.defaultResend

		if hdlStruct.handler != nil {
			resend = hdlStruct.handler(payload)
		} else if !p.defaultResend && !hdlStruct.touched {
			p.t.Fatalf("unexpected %T payload", payload)
			return
		}

		hdlStruct.count.Add(1)

	case *payload.VStateReport:
		hdlStruct := &p.Handlers.VStateReport

		resend = p.defaultResend

		if hdlStruct.handler != nil {
			resend = hdlStruct.handler(payload)
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

func (p *TypePublishChecker) SetDefaultResend(flag bool) *TypePublishChecker {
	p.defaultResend = flag
	return p
}

func (p *TypePublishChecker) minimockDone() bool {
	if hdl := p.Handlers.VCallRequest; hdl.expectedCount >= 0 && !(p.defaultResend && hdl.expectedCount == 0) {
		return hdl.count.Load() == hdl.expectedCount
	}
	if hdl := p.Handlers.VCallResult; hdl.expectedCount >= 0 && !(p.defaultResend && hdl.expectedCount == 0) {
		return hdl.count.Load() == hdl.expectedCount
	}
	if hdl := p.Handlers.VDelegatedCallRequest; hdl.expectedCount >= 0 && !(p.defaultResend && hdl.expectedCount == 0) {
		return hdl.count.Load() == hdl.expectedCount
	}
	if hdl := p.Handlers.VDelegatedCallResponse; hdl.expectedCount >= 0 && !(p.defaultResend && hdl.expectedCount == 0) {
		return hdl.count.Load() == hdl.expectedCount
	}
	if hdl := p.Handlers.VDelegatedRequestFinished; hdl.expectedCount >= 0 && !(p.defaultResend && hdl.expectedCount == 0) {
		return hdl.count.Load() == hdl.expectedCount
	}
	if hdl := p.Handlers.VStateRequest; hdl.expectedCount >= 0 && !(p.defaultResend && hdl.expectedCount == 0) {
		return hdl.count.Load() == hdl.expectedCount
	}
	if hdl := p.Handlers.VStateReport; hdl.expectedCount >= 0 && !(p.defaultResend && hdl.expectedCount == 0) {
		return hdl.count.Load() == hdl.expectedCount
	}
	return true
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (p *TypePublishChecker) MinimockFinish() {
	if !p.minimockDone() {
		p.t.Fatal("failed conditions on TypePublishChecker")
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (p *TypePublishChecker) MinimockWait(timeout mm_time.Duration) {
	timeoutCh := mm_time.After(timeout)
	for {
		if p.minimockDone() {
			return
		}
		select {
		case <-timeoutCh:
			p.MinimockFinish()
			return
		case <-mm_time.After(10 * mm_time.Millisecond):
		}
	}
}
