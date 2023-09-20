package main

const (
	ReferenceBuilderMock = `
		// Code generated. DO NOT EDIT.

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

	{{ range $pos, $msg := .Messages }}
		type {{ $msg }}Definition struct {
			touched       bool
			count         atomickit.Int
			countBefore   atomickit.Int
			expectedCount int
			anticipatedRefFromBytesHandler    {{ $msg }}AnticipatedRefFromBytesHandler
			anticipatedRefFromWriterToHandler {{ $msg }}AnticipatedRefFromWriterToHandler
			anticipatedRefFromRecordHandler   {{ $msg }}AnticipatedRefFromRecordHandler
		}
		type {{ $msg }}AnticipatedRefFromBytesHandler func(object reference.Global, pn pulse.Number, record *rms.{{ $msg }}) reference.Global
		type {{ $msg }}AnticipatedRefFromWriterToHandler func(object reference.Global, pn pulse.Number, record *rms.{{ $msg }}) reference.Global
		type {{ $msg }}AnticipatedRefFromRecordHandler func(object reference.Global, pn pulse.Number, record *rms.{{ $msg }}) reference.Global
		type {{ $msg }}BuilderMock struct{ parent *TypedReferenceBuilder }

		func (p {{ $msg }}BuilderMock) ExpectedCount(count int) {{ $msg }}BuilderMock {
			p.parent.Handlers.{{ $msg }}.touched = true
			p.parent.Handlers.{{ $msg }}.expectedCount = count
			return p
		}

		func (p *{{ $msg }}BuilderMock) AnticipatedRefFromBytesMock(handler {{ $msg }}AnticipatedRefFromBytesHandler) *{{ $msg }}BuilderMock {
			p.parent.Handlers.{{ $msg }}.touched = true
			p.parent.Handlers.{{ $msg }}.anticipatedRefFromBytesHandler = handler
			return p
		}
		
		func (p *{{ $msg }}BuilderMock) AnticipatedRefFromWriterToMock(handler {{ $msg }}AnticipatedRefFromWriterToHandler) *{{ $msg }}BuilderMock {
			p.parent.Handlers.{{ $msg }}.touched = true
			p.parent.Handlers.{{ $msg }}.anticipatedRefFromWriterToHandler = handler
			return p
		}

		func (p *{{ $msg }}BuilderMock) AnticipatedRefFromRecordMock(handler {{ $msg }}AnticipatedRefFromRecordHandler) *{{ $msg }}BuilderMock {
			p.parent.Handlers.{{ $msg }}.touched = true
			p.parent.Handlers.{{ $msg }}.anticipatedRefFromRecordHandler = handler
			return p
		}

		func (p {{ $msg }}BuilderMock) Count() int {
			return p.parent.Handlers.{{ $msg }}.count.Load()
		}

		func (p {{ $msg }}BuilderMock) CountBefore() int {
			return p.parent.Handlers.{{ $msg }}.countBefore.Load()
		}

		func (p {{ $msg }}BuilderMock) Wait(ctx context.Context, count int) synckit.SignalChannel {
			return waitCounterIndefinitely(ctx, &p.parent.Handlers.{{ $msg }}.count, count)
		}

		// ============================================================================
	{{ end }}

		type TypedHandlers struct {
		{{ range $pos, $msg := .Messages -}}
			{{ $msg }} {{ $msg }}Definition
		{{ end }}
		}

		type TypedReferenceBuilder struct {
			t             minimock.Tester
			timeout       time.Duration
			ctx           context.Context
			
			Handlers TypedHandlers

		{{ range $pos, $msg := .Messages -}}
			{{ $msg }} {{ $msg }}BuilderMock
		{{ end }}
		}

		func NewTypedReferenceBuilder(ctx context.Context, t minimock.Tester) *TypedReferenceBuilder {
			checker := &TypedReferenceBuilder{
				t:             t,
				ctx:           ctx,
				timeout:       10 * time.Second,

				Handlers: TypedHandlers{
				{{ range $pos, $msg := .Messages -}}
					{{ $msg }}: {{ $msg }}Definition{expectedCount: -1},
				{{ end }}
				},
			}

			{{ range $pos, $msg := .Messages -}}
				checker.{{ $msg }} = {{ $msg }}BuilderMock{parent: checker}
			{{ end }}

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
		{{- range $pos, $msg := .Messages }}
			case *rms.{{ $msg }}:
				msg := rec.Get().(*rms.{{ $msg }})
				hdlStruct := &p.Handlers.{{ $msg }}
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
						p.t.Error("timeout: failed to check message {{ $msg }} (position: %s)", oldCount)
					}
				} else if !hdlStruct.touched {
					p.t.Fatalf("unexpected %T record", record)
					return reference.Global{}
				}
		
				hdlStruct.count.Add(1)
		{{ end }}
			
			default:
				p.t.Fatalf("unexpected %T record", record)
				return reference.Global{}
			}
		
			return resultRef
		}

		func (p *TypedReferenceBuilder) minimockDone() bool {
			ok := true

		{{ range $pos, $msg := .Messages -}}
			{
				fn := func () bool {
					hdl := &p.Handlers.{{ $msg }}

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
		{{ end }}

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
	`
)
