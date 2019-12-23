package executor

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

// HeadTruncaterMock implements headTruncater
type HeadTruncaterMock struct {
	t minimock.Tester

	funcTruncateHead          func(ctx context.Context, from insolar.PulseNumber) (err error)
	inspectFuncTruncateHead   func(ctx context.Context, from insolar.PulseNumber)
	afterTruncateHeadCounter  uint64
	beforeTruncateHeadCounter uint64
	TruncateHeadMock          mHeadTruncaterMockTruncateHead
}

// NewHeadTruncaterMock returns a mock for headTruncater
func NewHeadTruncaterMock(t minimock.Tester) *HeadTruncaterMock {
	m := &HeadTruncaterMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.TruncateHeadMock = mHeadTruncaterMockTruncateHead{mock: m}
	m.TruncateHeadMock.callArgs = []*HeadTruncaterMockTruncateHeadParams{}

	return m
}

type mHeadTruncaterMockTruncateHead struct {
	mock               *HeadTruncaterMock
	defaultExpectation *HeadTruncaterMockTruncateHeadExpectation
	expectations       []*HeadTruncaterMockTruncateHeadExpectation

	callArgs []*HeadTruncaterMockTruncateHeadParams
	mutex    sync.RWMutex
}

// HeadTruncaterMockTruncateHeadExpectation specifies expectation struct of the headTruncater.TruncateHead
type HeadTruncaterMockTruncateHeadExpectation struct {
	mock    *HeadTruncaterMock
	params  *HeadTruncaterMockTruncateHeadParams
	results *HeadTruncaterMockTruncateHeadResults
	Counter uint64
}

// HeadTruncaterMockTruncateHeadParams contains parameters of the headTruncater.TruncateHead
type HeadTruncaterMockTruncateHeadParams struct {
	ctx  context.Context
	from insolar.PulseNumber
}

// HeadTruncaterMockTruncateHeadResults contains results of the headTruncater.TruncateHead
type HeadTruncaterMockTruncateHeadResults struct {
	err error
}

// Expect sets up expected params for headTruncater.TruncateHead
func (mmTruncateHead *mHeadTruncaterMockTruncateHead) Expect(ctx context.Context, from insolar.PulseNumber) *mHeadTruncaterMockTruncateHead {
	if mmTruncateHead.mock.funcTruncateHead != nil {
		mmTruncateHead.mock.t.Fatalf("HeadTruncaterMock.TruncateHead mock is already set by Set")
	}

	if mmTruncateHead.defaultExpectation == nil {
		mmTruncateHead.defaultExpectation = &HeadTruncaterMockTruncateHeadExpectation{}
	}

	mmTruncateHead.defaultExpectation.params = &HeadTruncaterMockTruncateHeadParams{ctx, from}
	for _, e := range mmTruncateHead.expectations {
		if minimock.Equal(e.params, mmTruncateHead.defaultExpectation.params) {
			mmTruncateHead.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmTruncateHead.defaultExpectation.params)
		}
	}

	return mmTruncateHead
}

// Inspect accepts an inspector function that has same arguments as the headTruncater.TruncateHead
func (mmTruncateHead *mHeadTruncaterMockTruncateHead) Inspect(f func(ctx context.Context, from insolar.PulseNumber)) *mHeadTruncaterMockTruncateHead {
	if mmTruncateHead.mock.inspectFuncTruncateHead != nil {
		mmTruncateHead.mock.t.Fatalf("Inspect function is already set for HeadTruncaterMock.TruncateHead")
	}

	mmTruncateHead.mock.inspectFuncTruncateHead = f

	return mmTruncateHead
}

// Return sets up results that will be returned by headTruncater.TruncateHead
func (mmTruncateHead *mHeadTruncaterMockTruncateHead) Return(err error) *HeadTruncaterMock {
	if mmTruncateHead.mock.funcTruncateHead != nil {
		mmTruncateHead.mock.t.Fatalf("HeadTruncaterMock.TruncateHead mock is already set by Set")
	}

	if mmTruncateHead.defaultExpectation == nil {
		mmTruncateHead.defaultExpectation = &HeadTruncaterMockTruncateHeadExpectation{mock: mmTruncateHead.mock}
	}
	mmTruncateHead.defaultExpectation.results = &HeadTruncaterMockTruncateHeadResults{err}
	return mmTruncateHead.mock
}

//Set uses given function f to mock the headTruncater.TruncateHead method
func (mmTruncateHead *mHeadTruncaterMockTruncateHead) Set(f func(ctx context.Context, from insolar.PulseNumber) (err error)) *HeadTruncaterMock {
	if mmTruncateHead.defaultExpectation != nil {
		mmTruncateHead.mock.t.Fatalf("Default expectation is already set for the headTruncater.TruncateHead method")
	}

	if len(mmTruncateHead.expectations) > 0 {
		mmTruncateHead.mock.t.Fatalf("Some expectations are already set for the headTruncater.TruncateHead method")
	}

	mmTruncateHead.mock.funcTruncateHead = f
	return mmTruncateHead.mock
}

// When sets expectation for the headTruncater.TruncateHead which will trigger the result defined by the following
// Then helper
func (mmTruncateHead *mHeadTruncaterMockTruncateHead) When(ctx context.Context, from insolar.PulseNumber) *HeadTruncaterMockTruncateHeadExpectation {
	if mmTruncateHead.mock.funcTruncateHead != nil {
		mmTruncateHead.mock.t.Fatalf("HeadTruncaterMock.TruncateHead mock is already set by Set")
	}

	expectation := &HeadTruncaterMockTruncateHeadExpectation{
		mock:   mmTruncateHead.mock,
		params: &HeadTruncaterMockTruncateHeadParams{ctx, from},
	}
	mmTruncateHead.expectations = append(mmTruncateHead.expectations, expectation)
	return expectation
}

// Then sets up headTruncater.TruncateHead return parameters for the expectation previously defined by the When method
func (e *HeadTruncaterMockTruncateHeadExpectation) Then(err error) *HeadTruncaterMock {
	e.results = &HeadTruncaterMockTruncateHeadResults{err}
	return e.mock
}

// TruncateHead implements headTruncater
func (mmTruncateHead *HeadTruncaterMock) TruncateHead(ctx context.Context, from insolar.PulseNumber) (err error) {
	mm_atomic.AddUint64(&mmTruncateHead.beforeTruncateHeadCounter, 1)
	defer mm_atomic.AddUint64(&mmTruncateHead.afterTruncateHeadCounter, 1)

	if mmTruncateHead.inspectFuncTruncateHead != nil {
		mmTruncateHead.inspectFuncTruncateHead(ctx, from)
	}

	mm_params := &HeadTruncaterMockTruncateHeadParams{ctx, from}

	// Record call args
	mmTruncateHead.TruncateHeadMock.mutex.Lock()
	mmTruncateHead.TruncateHeadMock.callArgs = append(mmTruncateHead.TruncateHeadMock.callArgs, mm_params)
	mmTruncateHead.TruncateHeadMock.mutex.Unlock()

	for _, e := range mmTruncateHead.TruncateHeadMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmTruncateHead.TruncateHeadMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmTruncateHead.TruncateHeadMock.defaultExpectation.Counter, 1)
		mm_want := mmTruncateHead.TruncateHeadMock.defaultExpectation.params
		mm_got := HeadTruncaterMockTruncateHeadParams{ctx, from}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmTruncateHead.t.Errorf("HeadTruncaterMock.TruncateHead got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmTruncateHead.TruncateHeadMock.defaultExpectation.results
		if mm_results == nil {
			mmTruncateHead.t.Fatal("No results are set for the HeadTruncaterMock.TruncateHead")
		}
		return (*mm_results).err
	}
	if mmTruncateHead.funcTruncateHead != nil {
		return mmTruncateHead.funcTruncateHead(ctx, from)
	}
	mmTruncateHead.t.Fatalf("Unexpected call to HeadTruncaterMock.TruncateHead. %v %v", ctx, from)
	return
}

// TruncateHeadAfterCounter returns a count of finished HeadTruncaterMock.TruncateHead invocations
func (mmTruncateHead *HeadTruncaterMock) TruncateHeadAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmTruncateHead.afterTruncateHeadCounter)
}

// TruncateHeadBeforeCounter returns a count of HeadTruncaterMock.TruncateHead invocations
func (mmTruncateHead *HeadTruncaterMock) TruncateHeadBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmTruncateHead.beforeTruncateHeadCounter)
}

// Calls returns a list of arguments used in each call to HeadTruncaterMock.TruncateHead.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmTruncateHead *mHeadTruncaterMockTruncateHead) Calls() []*HeadTruncaterMockTruncateHeadParams {
	mmTruncateHead.mutex.RLock()

	argCopy := make([]*HeadTruncaterMockTruncateHeadParams, len(mmTruncateHead.callArgs))
	copy(argCopy, mmTruncateHead.callArgs)

	mmTruncateHead.mutex.RUnlock()

	return argCopy
}

// MinimockTruncateHeadDone returns true if the count of the TruncateHead invocations corresponds
// the number of defined expectations
func (m *HeadTruncaterMock) MinimockTruncateHeadDone() bool {
	for _, e := range m.TruncateHeadMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.TruncateHeadMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterTruncateHeadCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcTruncateHead != nil && mm_atomic.LoadUint64(&m.afterTruncateHeadCounter) < 1 {
		return false
	}
	return true
}

// MinimockTruncateHeadInspect logs each unmet expectation
func (m *HeadTruncaterMock) MinimockTruncateHeadInspect() {
	for _, e := range m.TruncateHeadMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to HeadTruncaterMock.TruncateHead with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.TruncateHeadMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterTruncateHeadCounter) < 1 {
		if m.TruncateHeadMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to HeadTruncaterMock.TruncateHead")
		} else {
			m.t.Errorf("Expected call to HeadTruncaterMock.TruncateHead with params: %#v", *m.TruncateHeadMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcTruncateHead != nil && mm_atomic.LoadUint64(&m.afterTruncateHeadCounter) < 1 {
		m.t.Error("Expected call to HeadTruncaterMock.TruncateHead")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *HeadTruncaterMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockTruncateHeadInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *HeadTruncaterMock) MinimockWait(timeout mm_time.Duration) {
	timeoutCh := mm_time.After(timeout)
	for {
		if m.minimockDone() {
			return
		}
		select {
		case <-timeoutCh:
			m.MinimockFinish()
			return
		case <-mm_time.After(10 * mm_time.Millisecond):
		}
	}
}

func (m *HeadTruncaterMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockTruncateHeadDone()
}
