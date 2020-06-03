package network

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
)

// PulseAppenderMock implements storage.PulseAppender
type PulseAppenderMock struct {
	t minimock.Tester

	funcAppendPulse          func(ctx context.Context, pulse pulsestor.Pulse) (err error)
	inspectFuncAppendPulse   func(ctx context.Context, pulse pulsestor.Pulse)
	afterAppendPulseCounter  uint64
	beforeAppendPulseCounter uint64
	AppendPulseMock          mPulseAppenderMockAppendPulse
}

// NewPulseAppenderMock returns a mock for storage.PulseAppender
func NewPulseAppenderMock(t minimock.Tester) *PulseAppenderMock {
	m := &PulseAppenderMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.AppendPulseMock = mPulseAppenderMockAppendPulse{mock: m}
	m.AppendPulseMock.callArgs = []*PulseAppenderMockAppendPulseParams{}

	return m
}

type mPulseAppenderMockAppendPulse struct {
	mock               *PulseAppenderMock
	defaultExpectation *PulseAppenderMockAppendPulseExpectation
	expectations       []*PulseAppenderMockAppendPulseExpectation

	callArgs []*PulseAppenderMockAppendPulseParams
	mutex    sync.RWMutex
}

// PulseAppenderMockAppendPulseExpectation specifies expectation struct of the PulseAppender.AppendPulse
type PulseAppenderMockAppendPulseExpectation struct {
	mock    *PulseAppenderMock
	params  *PulseAppenderMockAppendPulseParams
	results *PulseAppenderMockAppendPulseResults
	Counter uint64
}

// PulseAppenderMockAppendPulseParams contains parameters of the PulseAppender.AppendPulse
type PulseAppenderMockAppendPulseParams struct {
	ctx   context.Context
	pulse pulsestor.Pulse
}

// PulseAppenderMockAppendPulseResults contains results of the PulseAppender.AppendPulse
type PulseAppenderMockAppendPulseResults struct {
	err error
}

// Expect sets up expected params for PulseAppender.AppendPulse
func (mmAppendPulse *mPulseAppenderMockAppendPulse) Expect(ctx context.Context, pulse pulsestor.Pulse) *mPulseAppenderMockAppendPulse {
	if mmAppendPulse.mock.funcAppendPulse != nil {
		mmAppendPulse.mock.t.Fatalf("PulseAppenderMock.AppendPulse mock is already set by Set")
	}

	if mmAppendPulse.defaultExpectation == nil {
		mmAppendPulse.defaultExpectation = &PulseAppenderMockAppendPulseExpectation{}
	}

	mmAppendPulse.defaultExpectation.params = &PulseAppenderMockAppendPulseParams{ctx, pulse}
	for _, e := range mmAppendPulse.expectations {
		if minimock.Equal(e.params, mmAppendPulse.defaultExpectation.params) {
			mmAppendPulse.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmAppendPulse.defaultExpectation.params)
		}
	}

	return mmAppendPulse
}

// Inspect accepts an inspector function that has same arguments as the PulseAppender.AppendPulse
func (mmAppendPulse *mPulseAppenderMockAppendPulse) Inspect(f func(ctx context.Context, pulse pulsestor.Pulse)) *mPulseAppenderMockAppendPulse {
	if mmAppendPulse.mock.inspectFuncAppendPulse != nil {
		mmAppendPulse.mock.t.Fatalf("Inspect function is already set for PulseAppenderMock.AppendPulse")
	}

	mmAppendPulse.mock.inspectFuncAppendPulse = f

	return mmAppendPulse
}

// Return sets up results that will be returned by PulseAppender.AppendPulse
func (mmAppendPulse *mPulseAppenderMockAppendPulse) Return(err error) *PulseAppenderMock {
	if mmAppendPulse.mock.funcAppendPulse != nil {
		mmAppendPulse.mock.t.Fatalf("PulseAppenderMock.AppendPulse mock is already set by Set")
	}

	if mmAppendPulse.defaultExpectation == nil {
		mmAppendPulse.defaultExpectation = &PulseAppenderMockAppendPulseExpectation{mock: mmAppendPulse.mock}
	}
	mmAppendPulse.defaultExpectation.results = &PulseAppenderMockAppendPulseResults{err}
	return mmAppendPulse.mock
}

//Set uses given function f to mock the PulseAppender.AppendPulse method
func (mmAppendPulse *mPulseAppenderMockAppendPulse) Set(f func(ctx context.Context, pulse pulsestor.Pulse) (err error)) *PulseAppenderMock {
	if mmAppendPulse.defaultExpectation != nil {
		mmAppendPulse.mock.t.Fatalf("Default expectation is already set for the PulseAppender.AppendPulse method")
	}

	if len(mmAppendPulse.expectations) > 0 {
		mmAppendPulse.mock.t.Fatalf("Some expectations are already set for the PulseAppender.AppendPulse method")
	}

	mmAppendPulse.mock.funcAppendPulse = f
	return mmAppendPulse.mock
}

// When sets expectation for the PulseAppender.AppendPulse which will trigger the result defined by the following
// Then helper
func (mmAppendPulse *mPulseAppenderMockAppendPulse) When(ctx context.Context, pulse pulsestor.Pulse) *PulseAppenderMockAppendPulseExpectation {
	if mmAppendPulse.mock.funcAppendPulse != nil {
		mmAppendPulse.mock.t.Fatalf("PulseAppenderMock.AppendPulse mock is already set by Set")
	}

	expectation := &PulseAppenderMockAppendPulseExpectation{
		mock:   mmAppendPulse.mock,
		params: &PulseAppenderMockAppendPulseParams{ctx, pulse},
	}
	mmAppendPulse.expectations = append(mmAppendPulse.expectations, expectation)
	return expectation
}

// Then sets up PulseAppender.AppendPulse return parameters for the expectation previously defined by the When method
func (e *PulseAppenderMockAppendPulseExpectation) Then(err error) *PulseAppenderMock {
	e.results = &PulseAppenderMockAppendPulseResults{err}
	return e.mock
}

// AppendPulse implements storage.PulseAppender
func (mmAppendPulse *PulseAppenderMock) AppendPulse(ctx context.Context, pulse pulsestor.Pulse) (err error) {
	mm_atomic.AddUint64(&mmAppendPulse.beforeAppendPulseCounter, 1)
	defer mm_atomic.AddUint64(&mmAppendPulse.afterAppendPulseCounter, 1)

	if mmAppendPulse.inspectFuncAppendPulse != nil {
		mmAppendPulse.inspectFuncAppendPulse(ctx, pulse)
	}

	mm_params := &PulseAppenderMockAppendPulseParams{ctx, pulse}

	// Record call args
	mmAppendPulse.AppendPulseMock.mutex.Lock()
	mmAppendPulse.AppendPulseMock.callArgs = append(mmAppendPulse.AppendPulseMock.callArgs, mm_params)
	mmAppendPulse.AppendPulseMock.mutex.Unlock()

	for _, e := range mmAppendPulse.AppendPulseMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmAppendPulse.AppendPulseMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmAppendPulse.AppendPulseMock.defaultExpectation.Counter, 1)
		mm_want := mmAppendPulse.AppendPulseMock.defaultExpectation.params
		mm_got := PulseAppenderMockAppendPulseParams{ctx, pulse}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmAppendPulse.t.Errorf("PulseAppenderMock.AppendPulse got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmAppendPulse.AppendPulseMock.defaultExpectation.results
		if mm_results == nil {
			mmAppendPulse.t.Fatal("No results are set for the PulseAppenderMock.AppendPulse")
		}
		return (*mm_results).err
	}
	if mmAppendPulse.funcAppendPulse != nil {
		return mmAppendPulse.funcAppendPulse(ctx, pulse)
	}
	mmAppendPulse.t.Fatalf("Unexpected call to PulseAppenderMock.AppendPulse. %v %v", ctx, pulse)
	return
}

// AppendPulseAfterCounter returns a count of finished PulseAppenderMock.AppendPulse invocations
func (mmAppendPulse *PulseAppenderMock) AppendPulseAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmAppendPulse.afterAppendPulseCounter)
}

// AppendPulseBeforeCounter returns a count of PulseAppenderMock.AppendPulse invocations
func (mmAppendPulse *PulseAppenderMock) AppendPulseBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmAppendPulse.beforeAppendPulseCounter)
}

// Calls returns a list of arguments used in each call to PulseAppenderMock.AppendPulse.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmAppendPulse *mPulseAppenderMockAppendPulse) Calls() []*PulseAppenderMockAppendPulseParams {
	mmAppendPulse.mutex.RLock()

	argCopy := make([]*PulseAppenderMockAppendPulseParams, len(mmAppendPulse.callArgs))
	copy(argCopy, mmAppendPulse.callArgs)

	mmAppendPulse.mutex.RUnlock()

	return argCopy
}

// MinimockAppendPulseDone returns true if the count of the AppendPulse invocations corresponds
// the number of defined expectations
func (m *PulseAppenderMock) MinimockAppendPulseDone() bool {
	for _, e := range m.AppendPulseMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.AppendPulseMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterAppendPulseCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcAppendPulse != nil && mm_atomic.LoadUint64(&m.afterAppendPulseCounter) < 1 {
		return false
	}
	return true
}

// MinimockAppendPulseInspect logs each unmet expectation
func (m *PulseAppenderMock) MinimockAppendPulseInspect() {
	for _, e := range m.AppendPulseMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to PulseAppenderMock.AppendPulse with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.AppendPulseMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterAppendPulseCounter) < 1 {
		if m.AppendPulseMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to PulseAppenderMock.AppendPulse")
		} else {
			m.t.Errorf("Expected call to PulseAppenderMock.AppendPulse with params: %#v", *m.AppendPulseMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcAppendPulse != nil && mm_atomic.LoadUint64(&m.afterAppendPulseCounter) < 1 {
		m.t.Error("Expected call to PulseAppenderMock.AppendPulse")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *PulseAppenderMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockAppendPulseInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *PulseAppenderMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *PulseAppenderMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockAppendPulseDone()
}
