package api

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
)

// AvailabilityCheckerMock implements AvailabilityChecker
type AvailabilityCheckerMock struct {
	t minimock.Tester

	funcIsAvailable          func(ctx context.Context) (b1 bool)
	inspectFuncIsAvailable   func(ctx context.Context)
	afterIsAvailableCounter  uint64
	beforeIsAvailableCounter uint64
	IsAvailableMock          mAvailabilityCheckerMockIsAvailable
}

// NewAvailabilityCheckerMock returns a mock for AvailabilityChecker
func NewAvailabilityCheckerMock(t minimock.Tester) *AvailabilityCheckerMock {
	m := &AvailabilityCheckerMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.IsAvailableMock = mAvailabilityCheckerMockIsAvailable{mock: m}
	m.IsAvailableMock.callArgs = []*AvailabilityCheckerMockIsAvailableParams{}

	return m
}

type mAvailabilityCheckerMockIsAvailable struct {
	mock               *AvailabilityCheckerMock
	defaultExpectation *AvailabilityCheckerMockIsAvailableExpectation
	expectations       []*AvailabilityCheckerMockIsAvailableExpectation

	callArgs []*AvailabilityCheckerMockIsAvailableParams
	mutex    sync.RWMutex
}

// AvailabilityCheckerMockIsAvailableExpectation specifies expectation struct of the AvailabilityChecker.IsAvailable
type AvailabilityCheckerMockIsAvailableExpectation struct {
	mock    *AvailabilityCheckerMock
	params  *AvailabilityCheckerMockIsAvailableParams
	results *AvailabilityCheckerMockIsAvailableResults
	Counter uint64
}

// AvailabilityCheckerMockIsAvailableParams contains parameters of the AvailabilityChecker.IsAvailable
type AvailabilityCheckerMockIsAvailableParams struct {
	ctx context.Context
}

// AvailabilityCheckerMockIsAvailableResults contains results of the AvailabilityChecker.IsAvailable
type AvailabilityCheckerMockIsAvailableResults struct {
	b1 bool
}

// Expect sets up expected params for AvailabilityChecker.IsAvailable
func (mmIsAvailable *mAvailabilityCheckerMockIsAvailable) Expect(ctx context.Context) *mAvailabilityCheckerMockIsAvailable {
	if mmIsAvailable.mock.funcIsAvailable != nil {
		mmIsAvailable.mock.t.Fatalf("AvailabilityCheckerMock.IsAvailable mock is already set by Set")
	}

	if mmIsAvailable.defaultExpectation == nil {
		mmIsAvailable.defaultExpectation = &AvailabilityCheckerMockIsAvailableExpectation{}
	}

	mmIsAvailable.defaultExpectation.params = &AvailabilityCheckerMockIsAvailableParams{ctx}
	for _, e := range mmIsAvailable.expectations {
		if minimock.Equal(e.params, mmIsAvailable.defaultExpectation.params) {
			mmIsAvailable.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmIsAvailable.defaultExpectation.params)
		}
	}

	return mmIsAvailable
}

// Inspect accepts an inspector function that has same arguments as the AvailabilityChecker.IsAvailable
func (mmIsAvailable *mAvailabilityCheckerMockIsAvailable) Inspect(f func(ctx context.Context)) *mAvailabilityCheckerMockIsAvailable {
	if mmIsAvailable.mock.inspectFuncIsAvailable != nil {
		mmIsAvailable.mock.t.Fatalf("Inspect function is already set for AvailabilityCheckerMock.IsAvailable")
	}

	mmIsAvailable.mock.inspectFuncIsAvailable = f

	return mmIsAvailable
}

// Return sets up results that will be returned by AvailabilityChecker.IsAvailable
func (mmIsAvailable *mAvailabilityCheckerMockIsAvailable) Return(b1 bool) *AvailabilityCheckerMock {
	if mmIsAvailable.mock.funcIsAvailable != nil {
		mmIsAvailable.mock.t.Fatalf("AvailabilityCheckerMock.IsAvailable mock is already set by Set")
	}

	if mmIsAvailable.defaultExpectation == nil {
		mmIsAvailable.defaultExpectation = &AvailabilityCheckerMockIsAvailableExpectation{mock: mmIsAvailable.mock}
	}
	mmIsAvailable.defaultExpectation.results = &AvailabilityCheckerMockIsAvailableResults{b1}
	return mmIsAvailable.mock
}

//Set uses given function f to mock the AvailabilityChecker.IsAvailable method
func (mmIsAvailable *mAvailabilityCheckerMockIsAvailable) Set(f func(ctx context.Context) (b1 bool)) *AvailabilityCheckerMock {
	if mmIsAvailable.defaultExpectation != nil {
		mmIsAvailable.mock.t.Fatalf("Default expectation is already set for the AvailabilityChecker.IsAvailable method")
	}

	if len(mmIsAvailable.expectations) > 0 {
		mmIsAvailable.mock.t.Fatalf("Some expectations are already set for the AvailabilityChecker.IsAvailable method")
	}

	mmIsAvailable.mock.funcIsAvailable = f
	return mmIsAvailable.mock
}

// When sets expectation for the AvailabilityChecker.IsAvailable which will trigger the result defined by the following
// Then helper
func (mmIsAvailable *mAvailabilityCheckerMockIsAvailable) When(ctx context.Context) *AvailabilityCheckerMockIsAvailableExpectation {
	if mmIsAvailable.mock.funcIsAvailable != nil {
		mmIsAvailable.mock.t.Fatalf("AvailabilityCheckerMock.IsAvailable mock is already set by Set")
	}

	expectation := &AvailabilityCheckerMockIsAvailableExpectation{
		mock:   mmIsAvailable.mock,
		params: &AvailabilityCheckerMockIsAvailableParams{ctx},
	}
	mmIsAvailable.expectations = append(mmIsAvailable.expectations, expectation)
	return expectation
}

// Then sets up AvailabilityChecker.IsAvailable return parameters for the expectation previously defined by the When method
func (e *AvailabilityCheckerMockIsAvailableExpectation) Then(b1 bool) *AvailabilityCheckerMock {
	e.results = &AvailabilityCheckerMockIsAvailableResults{b1}
	return e.mock
}

// IsAvailable implements AvailabilityChecker
func (mmIsAvailable *AvailabilityCheckerMock) IsAvailable(ctx context.Context) (b1 bool) {
	mm_atomic.AddUint64(&mmIsAvailable.beforeIsAvailableCounter, 1)
	defer mm_atomic.AddUint64(&mmIsAvailable.afterIsAvailableCounter, 1)

	if mmIsAvailable.inspectFuncIsAvailable != nil {
		mmIsAvailable.inspectFuncIsAvailable(ctx)
	}

	mm_params := &AvailabilityCheckerMockIsAvailableParams{ctx}

	// Record call args
	mmIsAvailable.IsAvailableMock.mutex.Lock()
	mmIsAvailable.IsAvailableMock.callArgs = append(mmIsAvailable.IsAvailableMock.callArgs, mm_params)
	mmIsAvailable.IsAvailableMock.mutex.Unlock()

	for _, e := range mmIsAvailable.IsAvailableMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.b1
		}
	}

	if mmIsAvailable.IsAvailableMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmIsAvailable.IsAvailableMock.defaultExpectation.Counter, 1)
		mm_want := mmIsAvailable.IsAvailableMock.defaultExpectation.params
		mm_got := AvailabilityCheckerMockIsAvailableParams{ctx}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmIsAvailable.t.Errorf("AvailabilityCheckerMock.IsAvailable got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmIsAvailable.IsAvailableMock.defaultExpectation.results
		if mm_results == nil {
			mmIsAvailable.t.Fatal("No results are set for the AvailabilityCheckerMock.IsAvailable")
		}
		return (*mm_results).b1
	}
	if mmIsAvailable.funcIsAvailable != nil {
		return mmIsAvailable.funcIsAvailable(ctx)
	}
	mmIsAvailable.t.Fatalf("Unexpected call to AvailabilityCheckerMock.IsAvailable. %v", ctx)
	return
}

// IsAvailableAfterCounter returns a count of finished AvailabilityCheckerMock.IsAvailable invocations
func (mmIsAvailable *AvailabilityCheckerMock) IsAvailableAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmIsAvailable.afterIsAvailableCounter)
}

// IsAvailableBeforeCounter returns a count of AvailabilityCheckerMock.IsAvailable invocations
func (mmIsAvailable *AvailabilityCheckerMock) IsAvailableBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmIsAvailable.beforeIsAvailableCounter)
}

// Calls returns a list of arguments used in each call to AvailabilityCheckerMock.IsAvailable.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmIsAvailable *mAvailabilityCheckerMockIsAvailable) Calls() []*AvailabilityCheckerMockIsAvailableParams {
	mmIsAvailable.mutex.RLock()

	argCopy := make([]*AvailabilityCheckerMockIsAvailableParams, len(mmIsAvailable.callArgs))
	copy(argCopy, mmIsAvailable.callArgs)

	mmIsAvailable.mutex.RUnlock()

	return argCopy
}

// MinimockIsAvailableDone returns true if the count of the IsAvailable invocations corresponds
// the number of defined expectations
func (m *AvailabilityCheckerMock) MinimockIsAvailableDone() bool {
	for _, e := range m.IsAvailableMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.IsAvailableMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterIsAvailableCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcIsAvailable != nil && mm_atomic.LoadUint64(&m.afterIsAvailableCounter) < 1 {
		return false
	}
	return true
}

// MinimockIsAvailableInspect logs each unmet expectation
func (m *AvailabilityCheckerMock) MinimockIsAvailableInspect() {
	for _, e := range m.IsAvailableMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to AvailabilityCheckerMock.IsAvailable with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.IsAvailableMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterIsAvailableCounter) < 1 {
		if m.IsAvailableMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to AvailabilityCheckerMock.IsAvailable")
		} else {
			m.t.Errorf("Expected call to AvailabilityCheckerMock.IsAvailable with params: %#v", *m.IsAvailableMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcIsAvailable != nil && mm_atomic.LoadUint64(&m.afterIsAvailableCounter) < 1 {
		m.t.Error("Expected call to AvailabilityCheckerMock.IsAvailable")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *AvailabilityCheckerMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockIsAvailableInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *AvailabilityCheckerMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *AvailabilityCheckerMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockIsAvailableDone()
}
