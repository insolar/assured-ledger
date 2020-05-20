package network

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

// PulseAccessorMock implements storage.PulseAccessor
type PulseAccessorMock struct {
	t minimock.Tester

	funcGetLatestPulse          func(ctx context.Context) (p1 pulsestor.Pulse, err error)
	inspectFuncGetLatestPulse   func(ctx context.Context)
	afterGetLatestPulseCounter  uint64
	beforeGetLatestPulseCounter uint64
	GetLatestPulseMock          mPulseAccessorMockGetLatestPulse

	funcGetPulse          func(ctx context.Context, n1 pulse.Number) (p1 pulsestor.Pulse, err error)
	inspectFuncGetPulse   func(ctx context.Context, n1 pulse.Number)
	afterGetPulseCounter  uint64
	beforeGetPulseCounter uint64
	GetPulseMock          mPulseAccessorMockGetPulse
}

// NewPulseAccessorMock returns a mock for storage.PulseAccessor
func NewPulseAccessorMock(t minimock.Tester) *PulseAccessorMock {
	m := &PulseAccessorMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.GetLatestPulseMock = mPulseAccessorMockGetLatestPulse{mock: m}
	m.GetLatestPulseMock.callArgs = []*PulseAccessorMockGetLatestPulseParams{}

	m.GetPulseMock = mPulseAccessorMockGetPulse{mock: m}
	m.GetPulseMock.callArgs = []*PulseAccessorMockGetPulseParams{}

	return m
}

type mPulseAccessorMockGetLatestPulse struct {
	mock               *PulseAccessorMock
	defaultExpectation *PulseAccessorMockGetLatestPulseExpectation
	expectations       []*PulseAccessorMockGetLatestPulseExpectation

	callArgs []*PulseAccessorMockGetLatestPulseParams
	mutex    sync.RWMutex
}

// PulseAccessorMockGetLatestPulseExpectation specifies expectation struct of the PulseAccessor.GetLatestPulse
type PulseAccessorMockGetLatestPulseExpectation struct {
	mock    *PulseAccessorMock
	params  *PulseAccessorMockGetLatestPulseParams
	results *PulseAccessorMockGetLatestPulseResults
	Counter uint64
}

// PulseAccessorMockGetLatestPulseParams contains parameters of the PulseAccessor.GetLatestPulse
type PulseAccessorMockGetLatestPulseParams struct {
	ctx context.Context
}

// PulseAccessorMockGetLatestPulseResults contains results of the PulseAccessor.GetLatestPulse
type PulseAccessorMockGetLatestPulseResults struct {
	p1  pulsestor.Pulse
	err error
}

// Expect sets up expected params for PulseAccessor.GetLatestPulse
func (mmGetLatestPulse *mPulseAccessorMockGetLatestPulse) Expect(ctx context.Context) *mPulseAccessorMockGetLatestPulse {
	if mmGetLatestPulse.mock.funcGetLatestPulse != nil {
		mmGetLatestPulse.mock.t.Fatalf("PulseAccessorMock.GetLatestPulse mock is already set by Set")
	}

	if mmGetLatestPulse.defaultExpectation == nil {
		mmGetLatestPulse.defaultExpectation = &PulseAccessorMockGetLatestPulseExpectation{}
	}

	mmGetLatestPulse.defaultExpectation.params = &PulseAccessorMockGetLatestPulseParams{ctx}
	for _, e := range mmGetLatestPulse.expectations {
		if minimock.Equal(e.params, mmGetLatestPulse.defaultExpectation.params) {
			mmGetLatestPulse.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmGetLatestPulse.defaultExpectation.params)
		}
	}

	return mmGetLatestPulse
}

// Inspect accepts an inspector function that has same arguments as the PulseAccessor.GetLatestPulse
func (mmGetLatestPulse *mPulseAccessorMockGetLatestPulse) Inspect(f func(ctx context.Context)) *mPulseAccessorMockGetLatestPulse {
	if mmGetLatestPulse.mock.inspectFuncGetLatestPulse != nil {
		mmGetLatestPulse.mock.t.Fatalf("Inspect function is already set for PulseAccessorMock.GetLatestPulse")
	}

	mmGetLatestPulse.mock.inspectFuncGetLatestPulse = f

	return mmGetLatestPulse
}

// Return sets up results that will be returned by PulseAccessor.GetLatestPulse
func (mmGetLatestPulse *mPulseAccessorMockGetLatestPulse) Return(p1 pulsestor.Pulse, err error) *PulseAccessorMock {
	if mmGetLatestPulse.mock.funcGetLatestPulse != nil {
		mmGetLatestPulse.mock.t.Fatalf("PulseAccessorMock.GetLatestPulse mock is already set by Set")
	}

	if mmGetLatestPulse.defaultExpectation == nil {
		mmGetLatestPulse.defaultExpectation = &PulseAccessorMockGetLatestPulseExpectation{mock: mmGetLatestPulse.mock}
	}
	mmGetLatestPulse.defaultExpectation.results = &PulseAccessorMockGetLatestPulseResults{p1, err}
	return mmGetLatestPulse.mock
}

//Set uses given function f to mock the PulseAccessor.GetLatestPulse method
func (mmGetLatestPulse *mPulseAccessorMockGetLatestPulse) Set(f func(ctx context.Context) (p1 pulsestor.Pulse, err error)) *PulseAccessorMock {
	if mmGetLatestPulse.defaultExpectation != nil {
		mmGetLatestPulse.mock.t.Fatalf("Default expectation is already set for the PulseAccessor.GetLatestPulse method")
	}

	if len(mmGetLatestPulse.expectations) > 0 {
		mmGetLatestPulse.mock.t.Fatalf("Some expectations are already set for the PulseAccessor.GetLatestPulse method")
	}

	mmGetLatestPulse.mock.funcGetLatestPulse = f
	return mmGetLatestPulse.mock
}

// When sets expectation for the PulseAccessor.GetLatestPulse which will trigger the result defined by the following
// Then helper
func (mmGetLatestPulse *mPulseAccessorMockGetLatestPulse) When(ctx context.Context) *PulseAccessorMockGetLatestPulseExpectation {
	if mmGetLatestPulse.mock.funcGetLatestPulse != nil {
		mmGetLatestPulse.mock.t.Fatalf("PulseAccessorMock.GetLatestPulse mock is already set by Set")
	}

	expectation := &PulseAccessorMockGetLatestPulseExpectation{
		mock:   mmGetLatestPulse.mock,
		params: &PulseAccessorMockGetLatestPulseParams{ctx},
	}
	mmGetLatestPulse.expectations = append(mmGetLatestPulse.expectations, expectation)
	return expectation
}

// Then sets up PulseAccessor.GetLatestPulse return parameters for the expectation previously defined by the When method
func (e *PulseAccessorMockGetLatestPulseExpectation) Then(p1 pulsestor.Pulse, err error) *PulseAccessorMock {
	e.results = &PulseAccessorMockGetLatestPulseResults{p1, err}
	return e.mock
}

// GetLatestPulse implements storage.PulseAccessor
func (mmGetLatestPulse *PulseAccessorMock) GetLatestPulse(ctx context.Context) (p1 pulsestor.Pulse, err error) {
	mm_atomic.AddUint64(&mmGetLatestPulse.beforeGetLatestPulseCounter, 1)
	defer mm_atomic.AddUint64(&mmGetLatestPulse.afterGetLatestPulseCounter, 1)

	if mmGetLatestPulse.inspectFuncGetLatestPulse != nil {
		mmGetLatestPulse.inspectFuncGetLatestPulse(ctx)
	}

	mm_params := &PulseAccessorMockGetLatestPulseParams{ctx}

	// Record call args
	mmGetLatestPulse.GetLatestPulseMock.mutex.Lock()
	mmGetLatestPulse.GetLatestPulseMock.callArgs = append(mmGetLatestPulse.GetLatestPulseMock.callArgs, mm_params)
	mmGetLatestPulse.GetLatestPulseMock.mutex.Unlock()

	for _, e := range mmGetLatestPulse.GetLatestPulseMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.p1, e.results.err
		}
	}

	if mmGetLatestPulse.GetLatestPulseMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetLatestPulse.GetLatestPulseMock.defaultExpectation.Counter, 1)
		mm_want := mmGetLatestPulse.GetLatestPulseMock.defaultExpectation.params
		mm_got := PulseAccessorMockGetLatestPulseParams{ctx}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmGetLatestPulse.t.Errorf("PulseAccessorMock.GetLatestPulse got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmGetLatestPulse.GetLatestPulseMock.defaultExpectation.results
		if mm_results == nil {
			mmGetLatestPulse.t.Fatal("No results are set for the PulseAccessorMock.GetLatestPulse")
		}
		return (*mm_results).p1, (*mm_results).err
	}
	if mmGetLatestPulse.funcGetLatestPulse != nil {
		return mmGetLatestPulse.funcGetLatestPulse(ctx)
	}
	mmGetLatestPulse.t.Fatalf("Unexpected call to PulseAccessorMock.GetLatestPulse. %v", ctx)
	return
}

// GetLatestPulseAfterCounter returns a count of finished PulseAccessorMock.GetLatestPulse invocations
func (mmGetLatestPulse *PulseAccessorMock) GetLatestPulseAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetLatestPulse.afterGetLatestPulseCounter)
}

// GetLatestPulseBeforeCounter returns a count of PulseAccessorMock.GetLatestPulse invocations
func (mmGetLatestPulse *PulseAccessorMock) GetLatestPulseBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetLatestPulse.beforeGetLatestPulseCounter)
}

// Calls returns a list of arguments used in each call to PulseAccessorMock.GetLatestPulse.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmGetLatestPulse *mPulseAccessorMockGetLatestPulse) Calls() []*PulseAccessorMockGetLatestPulseParams {
	mmGetLatestPulse.mutex.RLock()

	argCopy := make([]*PulseAccessorMockGetLatestPulseParams, len(mmGetLatestPulse.callArgs))
	copy(argCopy, mmGetLatestPulse.callArgs)

	mmGetLatestPulse.mutex.RUnlock()

	return argCopy
}

// MinimockGetLatestPulseDone returns true if the count of the GetLatestPulse invocations corresponds
// the number of defined expectations
func (m *PulseAccessorMock) MinimockGetLatestPulseDone() bool {
	for _, e := range m.GetLatestPulseMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetLatestPulseMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetLatestPulseCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetLatestPulse != nil && mm_atomic.LoadUint64(&m.afterGetLatestPulseCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetLatestPulseInspect logs each unmet expectation
func (m *PulseAccessorMock) MinimockGetLatestPulseInspect() {
	for _, e := range m.GetLatestPulseMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to PulseAccessorMock.GetLatestPulse with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetLatestPulseMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetLatestPulseCounter) < 1 {
		if m.GetLatestPulseMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to PulseAccessorMock.GetLatestPulse")
		} else {
			m.t.Errorf("Expected call to PulseAccessorMock.GetLatestPulse with params: %#v", *m.GetLatestPulseMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetLatestPulse != nil && mm_atomic.LoadUint64(&m.afterGetLatestPulseCounter) < 1 {
		m.t.Error("Expected call to PulseAccessorMock.GetLatestPulse")
	}
}

type mPulseAccessorMockGetPulse struct {
	mock               *PulseAccessorMock
	defaultExpectation *PulseAccessorMockGetPulseExpectation
	expectations       []*PulseAccessorMockGetPulseExpectation

	callArgs []*PulseAccessorMockGetPulseParams
	mutex    sync.RWMutex
}

// PulseAccessorMockGetPulseExpectation specifies expectation struct of the PulseAccessor.GetPulse
type PulseAccessorMockGetPulseExpectation struct {
	mock    *PulseAccessorMock
	params  *PulseAccessorMockGetPulseParams
	results *PulseAccessorMockGetPulseResults
	Counter uint64
}

// PulseAccessorMockGetPulseParams contains parameters of the PulseAccessor.GetPulse
type PulseAccessorMockGetPulseParams struct {
	ctx context.Context
	n1  pulse.Number
}

// PulseAccessorMockGetPulseResults contains results of the PulseAccessor.GetPulse
type PulseAccessorMockGetPulseResults struct {
	p1  pulsestor.Pulse
	err error
}

// Expect sets up expected params for PulseAccessor.GetPulse
func (mmGetPulse *mPulseAccessorMockGetPulse) Expect(ctx context.Context, n1 pulse.Number) *mPulseAccessorMockGetPulse {
	if mmGetPulse.mock.funcGetPulse != nil {
		mmGetPulse.mock.t.Fatalf("PulseAccessorMock.GetPulse mock is already set by Set")
	}

	if mmGetPulse.defaultExpectation == nil {
		mmGetPulse.defaultExpectation = &PulseAccessorMockGetPulseExpectation{}
	}

	mmGetPulse.defaultExpectation.params = &PulseAccessorMockGetPulseParams{ctx, n1}
	for _, e := range mmGetPulse.expectations {
		if minimock.Equal(e.params, mmGetPulse.defaultExpectation.params) {
			mmGetPulse.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmGetPulse.defaultExpectation.params)
		}
	}

	return mmGetPulse
}

// Inspect accepts an inspector function that has same arguments as the PulseAccessor.GetPulse
func (mmGetPulse *mPulseAccessorMockGetPulse) Inspect(f func(ctx context.Context, n1 pulse.Number)) *mPulseAccessorMockGetPulse {
	if mmGetPulse.mock.inspectFuncGetPulse != nil {
		mmGetPulse.mock.t.Fatalf("Inspect function is already set for PulseAccessorMock.GetPulse")
	}

	mmGetPulse.mock.inspectFuncGetPulse = f

	return mmGetPulse
}

// Return sets up results that will be returned by PulseAccessor.GetPulse
func (mmGetPulse *mPulseAccessorMockGetPulse) Return(p1 pulsestor.Pulse, err error) *PulseAccessorMock {
	if mmGetPulse.mock.funcGetPulse != nil {
		mmGetPulse.mock.t.Fatalf("PulseAccessorMock.GetPulse mock is already set by Set")
	}

	if mmGetPulse.defaultExpectation == nil {
		mmGetPulse.defaultExpectation = &PulseAccessorMockGetPulseExpectation{mock: mmGetPulse.mock}
	}
	mmGetPulse.defaultExpectation.results = &PulseAccessorMockGetPulseResults{p1, err}
	return mmGetPulse.mock
}

//Set uses given function f to mock the PulseAccessor.GetPulse method
func (mmGetPulse *mPulseAccessorMockGetPulse) Set(f func(ctx context.Context, n1 pulse.Number) (p1 pulsestor.Pulse, err error)) *PulseAccessorMock {
	if mmGetPulse.defaultExpectation != nil {
		mmGetPulse.mock.t.Fatalf("Default expectation is already set for the PulseAccessor.GetPulse method")
	}

	if len(mmGetPulse.expectations) > 0 {
		mmGetPulse.mock.t.Fatalf("Some expectations are already set for the PulseAccessor.GetPulse method")
	}

	mmGetPulse.mock.funcGetPulse = f
	return mmGetPulse.mock
}

// When sets expectation for the PulseAccessor.GetPulse which will trigger the result defined by the following
// Then helper
func (mmGetPulse *mPulseAccessorMockGetPulse) When(ctx context.Context, n1 pulse.Number) *PulseAccessorMockGetPulseExpectation {
	if mmGetPulse.mock.funcGetPulse != nil {
		mmGetPulse.mock.t.Fatalf("PulseAccessorMock.GetPulse mock is already set by Set")
	}

	expectation := &PulseAccessorMockGetPulseExpectation{
		mock:   mmGetPulse.mock,
		params: &PulseAccessorMockGetPulseParams{ctx, n1},
	}
	mmGetPulse.expectations = append(mmGetPulse.expectations, expectation)
	return expectation
}

// Then sets up PulseAccessor.GetPulse return parameters for the expectation previously defined by the When method
func (e *PulseAccessorMockGetPulseExpectation) Then(p1 pulsestor.Pulse, err error) *PulseAccessorMock {
	e.results = &PulseAccessorMockGetPulseResults{p1, err}
	return e.mock
}

// GetPulse implements storage.PulseAccessor
func (mmGetPulse *PulseAccessorMock) GetPulse(ctx context.Context, n1 pulse.Number) (p1 pulsestor.Pulse, err error) {
	mm_atomic.AddUint64(&mmGetPulse.beforeGetPulseCounter, 1)
	defer mm_atomic.AddUint64(&mmGetPulse.afterGetPulseCounter, 1)

	if mmGetPulse.inspectFuncGetPulse != nil {
		mmGetPulse.inspectFuncGetPulse(ctx, n1)
	}

	mm_params := &PulseAccessorMockGetPulseParams{ctx, n1}

	// Record call args
	mmGetPulse.GetPulseMock.mutex.Lock()
	mmGetPulse.GetPulseMock.callArgs = append(mmGetPulse.GetPulseMock.callArgs, mm_params)
	mmGetPulse.GetPulseMock.mutex.Unlock()

	for _, e := range mmGetPulse.GetPulseMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.p1, e.results.err
		}
	}

	if mmGetPulse.GetPulseMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetPulse.GetPulseMock.defaultExpectation.Counter, 1)
		mm_want := mmGetPulse.GetPulseMock.defaultExpectation.params
		mm_got := PulseAccessorMockGetPulseParams{ctx, n1}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmGetPulse.t.Errorf("PulseAccessorMock.GetPulse got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmGetPulse.GetPulseMock.defaultExpectation.results
		if mm_results == nil {
			mmGetPulse.t.Fatal("No results are set for the PulseAccessorMock.GetPulse")
		}
		return (*mm_results).p1, (*mm_results).err
	}
	if mmGetPulse.funcGetPulse != nil {
		return mmGetPulse.funcGetPulse(ctx, n1)
	}
	mmGetPulse.t.Fatalf("Unexpected call to PulseAccessorMock.GetPulse. %v %v", ctx, n1)
	return
}

// GetPulseAfterCounter returns a count of finished PulseAccessorMock.GetPulse invocations
func (mmGetPulse *PulseAccessorMock) GetPulseAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetPulse.afterGetPulseCounter)
}

// GetPulseBeforeCounter returns a count of PulseAccessorMock.GetPulse invocations
func (mmGetPulse *PulseAccessorMock) GetPulseBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetPulse.beforeGetPulseCounter)
}

// Calls returns a list of arguments used in each call to PulseAccessorMock.GetPulse.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmGetPulse *mPulseAccessorMockGetPulse) Calls() []*PulseAccessorMockGetPulseParams {
	mmGetPulse.mutex.RLock()

	argCopy := make([]*PulseAccessorMockGetPulseParams, len(mmGetPulse.callArgs))
	copy(argCopy, mmGetPulse.callArgs)

	mmGetPulse.mutex.RUnlock()

	return argCopy
}

// MinimockGetPulseDone returns true if the count of the GetPulse invocations corresponds
// the number of defined expectations
func (m *PulseAccessorMock) MinimockGetPulseDone() bool {
	for _, e := range m.GetPulseMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetPulseMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetPulseCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetPulse != nil && mm_atomic.LoadUint64(&m.afterGetPulseCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetPulseInspect logs each unmet expectation
func (m *PulseAccessorMock) MinimockGetPulseInspect() {
	for _, e := range m.GetPulseMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to PulseAccessorMock.GetPulse with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetPulseMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetPulseCounter) < 1 {
		if m.GetPulseMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to PulseAccessorMock.GetPulse")
		} else {
			m.t.Errorf("Expected call to PulseAccessorMock.GetPulse with params: %#v", *m.GetPulseMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetPulse != nil && mm_atomic.LoadUint64(&m.afterGetPulseCounter) < 1 {
		m.t.Error("Expected call to PulseAccessorMock.GetPulse")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *PulseAccessorMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockGetLatestPulseInspect()

		m.MinimockGetPulseInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *PulseAccessorMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *PulseAccessorMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockGetLatestPulseDone() &&
		m.MinimockGetPulseDone()
}
