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

// StateIniterMock implements StateIniter
type StateIniterMock struct {
	t minimock.Tester

	funcPrepareState          func(ctx context.Context, pulse insolar.PulseNumber) (justJoined bool, jets []insolar.JetID, err error)
	inspectFuncPrepareState   func(ctx context.Context, pulse insolar.PulseNumber)
	afterPrepareStateCounter  uint64
	beforePrepareStateCounter uint64
	PrepareStateMock          mStateIniterMockPrepareState
}

// NewStateIniterMock returns a mock for StateIniter
func NewStateIniterMock(t minimock.Tester) *StateIniterMock {
	m := &StateIniterMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.PrepareStateMock = mStateIniterMockPrepareState{mock: m}
	m.PrepareStateMock.callArgs = []*StateIniterMockPrepareStateParams{}

	return m
}

type mStateIniterMockPrepareState struct {
	mock               *StateIniterMock
	defaultExpectation *StateIniterMockPrepareStateExpectation
	expectations       []*StateIniterMockPrepareStateExpectation

	callArgs []*StateIniterMockPrepareStateParams
	mutex    sync.RWMutex
}

// StateIniterMockPrepareStateExpectation specifies expectation struct of the StateIniter.PrepareState
type StateIniterMockPrepareStateExpectation struct {
	mock    *StateIniterMock
	params  *StateIniterMockPrepareStateParams
	results *StateIniterMockPrepareStateResults
	Counter uint64
}

// StateIniterMockPrepareStateParams contains parameters of the StateIniter.PrepareState
type StateIniterMockPrepareStateParams struct {
	ctx   context.Context
	pulse insolar.PulseNumber
}

// StateIniterMockPrepareStateResults contains results of the StateIniter.PrepareState
type StateIniterMockPrepareStateResults struct {
	justJoined bool
	jets       []insolar.JetID
	err        error
}

// Expect sets up expected params for StateIniter.PrepareState
func (mmPrepareState *mStateIniterMockPrepareState) Expect(ctx context.Context, pulse insolar.PulseNumber) *mStateIniterMockPrepareState {
	if mmPrepareState.mock.funcPrepareState != nil {
		mmPrepareState.mock.t.Fatalf("StateIniterMock.PrepareState mock is already set by Set")
	}

	if mmPrepareState.defaultExpectation == nil {
		mmPrepareState.defaultExpectation = &StateIniterMockPrepareStateExpectation{}
	}

	mmPrepareState.defaultExpectation.params = &StateIniterMockPrepareStateParams{ctx, pulse}
	for _, e := range mmPrepareState.expectations {
		if minimock.Equal(e.params, mmPrepareState.defaultExpectation.params) {
			mmPrepareState.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmPrepareState.defaultExpectation.params)
		}
	}

	return mmPrepareState
}

// Inspect accepts an inspector function that has same arguments as the StateIniter.PrepareState
func (mmPrepareState *mStateIniterMockPrepareState) Inspect(f func(ctx context.Context, pulse insolar.PulseNumber)) *mStateIniterMockPrepareState {
	if mmPrepareState.mock.inspectFuncPrepareState != nil {
		mmPrepareState.mock.t.Fatalf("Inspect function is already set for StateIniterMock.PrepareState")
	}

	mmPrepareState.mock.inspectFuncPrepareState = f

	return mmPrepareState
}

// Return sets up results that will be returned by StateIniter.PrepareState
func (mmPrepareState *mStateIniterMockPrepareState) Return(justJoined bool, jets []insolar.JetID, err error) *StateIniterMock {
	if mmPrepareState.mock.funcPrepareState != nil {
		mmPrepareState.mock.t.Fatalf("StateIniterMock.PrepareState mock is already set by Set")
	}

	if mmPrepareState.defaultExpectation == nil {
		mmPrepareState.defaultExpectation = &StateIniterMockPrepareStateExpectation{mock: mmPrepareState.mock}
	}
	mmPrepareState.defaultExpectation.results = &StateIniterMockPrepareStateResults{justJoined, jets, err}
	return mmPrepareState.mock
}

//Set uses given function f to mock the StateIniter.PrepareState method
func (mmPrepareState *mStateIniterMockPrepareState) Set(f func(ctx context.Context, pulse insolar.PulseNumber) (justJoined bool, jets []insolar.JetID, err error)) *StateIniterMock {
	if mmPrepareState.defaultExpectation != nil {
		mmPrepareState.mock.t.Fatalf("Default expectation is already set for the StateIniter.PrepareState method")
	}

	if len(mmPrepareState.expectations) > 0 {
		mmPrepareState.mock.t.Fatalf("Some expectations are already set for the StateIniter.PrepareState method")
	}

	mmPrepareState.mock.funcPrepareState = f
	return mmPrepareState.mock
}

// When sets expectation for the StateIniter.PrepareState which will trigger the result defined by the following
// Then helper
func (mmPrepareState *mStateIniterMockPrepareState) When(ctx context.Context, pulse insolar.PulseNumber) *StateIniterMockPrepareStateExpectation {
	if mmPrepareState.mock.funcPrepareState != nil {
		mmPrepareState.mock.t.Fatalf("StateIniterMock.PrepareState mock is already set by Set")
	}

	expectation := &StateIniterMockPrepareStateExpectation{
		mock:   mmPrepareState.mock,
		params: &StateIniterMockPrepareStateParams{ctx, pulse},
	}
	mmPrepareState.expectations = append(mmPrepareState.expectations, expectation)
	return expectation
}

// Then sets up StateIniter.PrepareState return parameters for the expectation previously defined by the When method
func (e *StateIniterMockPrepareStateExpectation) Then(justJoined bool, jets []insolar.JetID, err error) *StateIniterMock {
	e.results = &StateIniterMockPrepareStateResults{justJoined, jets, err}
	return e.mock
}

// PrepareState implements StateIniter
func (mmPrepareState *StateIniterMock) PrepareState(ctx context.Context, pulse insolar.PulseNumber) (justJoined bool, jets []insolar.JetID, err error) {
	mm_atomic.AddUint64(&mmPrepareState.beforePrepareStateCounter, 1)
	defer mm_atomic.AddUint64(&mmPrepareState.afterPrepareStateCounter, 1)

	if mmPrepareState.inspectFuncPrepareState != nil {
		mmPrepareState.inspectFuncPrepareState(ctx, pulse)
	}

	mm_params := &StateIniterMockPrepareStateParams{ctx, pulse}

	// Record call args
	mmPrepareState.PrepareStateMock.mutex.Lock()
	mmPrepareState.PrepareStateMock.callArgs = append(mmPrepareState.PrepareStateMock.callArgs, mm_params)
	mmPrepareState.PrepareStateMock.mutex.Unlock()

	for _, e := range mmPrepareState.PrepareStateMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.justJoined, e.results.jets, e.results.err
		}
	}

	if mmPrepareState.PrepareStateMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmPrepareState.PrepareStateMock.defaultExpectation.Counter, 1)
		mm_want := mmPrepareState.PrepareStateMock.defaultExpectation.params
		mm_got := StateIniterMockPrepareStateParams{ctx, pulse}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmPrepareState.t.Errorf("StateIniterMock.PrepareState got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmPrepareState.PrepareStateMock.defaultExpectation.results
		if mm_results == nil {
			mmPrepareState.t.Fatal("No results are set for the StateIniterMock.PrepareState")
		}
		return (*mm_results).justJoined, (*mm_results).jets, (*mm_results).err
	}
	if mmPrepareState.funcPrepareState != nil {
		return mmPrepareState.funcPrepareState(ctx, pulse)
	}
	mmPrepareState.t.Fatalf("Unexpected call to StateIniterMock.PrepareState. %v %v", ctx, pulse)
	return
}

// PrepareStateAfterCounter returns a count of finished StateIniterMock.PrepareState invocations
func (mmPrepareState *StateIniterMock) PrepareStateAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareState.afterPrepareStateCounter)
}

// PrepareStateBeforeCounter returns a count of StateIniterMock.PrepareState invocations
func (mmPrepareState *StateIniterMock) PrepareStateBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareState.beforePrepareStateCounter)
}

// Calls returns a list of arguments used in each call to StateIniterMock.PrepareState.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmPrepareState *mStateIniterMockPrepareState) Calls() []*StateIniterMockPrepareStateParams {
	mmPrepareState.mutex.RLock()

	argCopy := make([]*StateIniterMockPrepareStateParams, len(mmPrepareState.callArgs))
	copy(argCopy, mmPrepareState.callArgs)

	mmPrepareState.mutex.RUnlock()

	return argCopy
}

// MinimockPrepareStateDone returns true if the count of the PrepareState invocations corresponds
// the number of defined expectations
func (m *StateIniterMock) MinimockPrepareStateDone() bool {
	for _, e := range m.PrepareStateMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareStateMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareStateCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareState != nil && mm_atomic.LoadUint64(&m.afterPrepareStateCounter) < 1 {
		return false
	}
	return true
}

// MinimockPrepareStateInspect logs each unmet expectation
func (m *StateIniterMock) MinimockPrepareStateInspect() {
	for _, e := range m.PrepareStateMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to StateIniterMock.PrepareState with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareStateMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareStateCounter) < 1 {
		if m.PrepareStateMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to StateIniterMock.PrepareState")
		} else {
			m.t.Errorf("Expected call to StateIniterMock.PrepareState with params: %#v", *m.PrepareStateMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareState != nil && mm_atomic.LoadUint64(&m.afterPrepareStateCounter) < 1 {
		m.t.Error("Expected call to StateIniterMock.PrepareState")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *StateIniterMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockPrepareStateInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *StateIniterMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *StateIniterMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockPrepareStateDone()
}
