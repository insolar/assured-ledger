package testutils

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	mm_insolar "github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

// PulseManagerMock implements insolar.PulseManager
type PulseManagerMock struct {
	t minimock.Tester

	funcSet          func(ctx context.Context, pulse mm_insolar.Pulse) (err error)
	inspectFuncSet   func(ctx context.Context, pulse mm_insolar.Pulse)
	afterSetCounter  uint64
	beforeSetCounter uint64
	SetMock          mPulseManagerMockSet
}

// NewPulseManagerMock returns a mock for insolar.PulseManager
func NewPulseManagerMock(t minimock.Tester) *PulseManagerMock {
	m := &PulseManagerMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.SetMock = mPulseManagerMockSet{mock: m}
	m.SetMock.callArgs = []*PulseManagerMockSetParams{}

	return m
}

type mPulseManagerMockSet struct {
	mock               *PulseManagerMock
	defaultExpectation *PulseManagerMockSetExpectation
	expectations       []*PulseManagerMockSetExpectation

	callArgs []*PulseManagerMockSetParams
	mutex    sync.RWMutex
}

// PulseManagerMockSetExpectation specifies expectation struct of the PulseManager.Set
type PulseManagerMockSetExpectation struct {
	mock    *PulseManagerMock
	params  *PulseManagerMockSetParams
	results *PulseManagerMockSetResults
	Counter uint64
}

// PulseManagerMockSetParams contains parameters of the PulseManager.Set
type PulseManagerMockSetParams struct {
	ctx   context.Context
	pulse mm_insolar.Pulse
}

// PulseManagerMockSetResults contains results of the PulseManager.Set
type PulseManagerMockSetResults struct {
	err error
}

// Expect sets up expected params for PulseManager.Set
func (mmSet *mPulseManagerMockSet) Expect(ctx context.Context, pulse mm_insolar.Pulse) *mPulseManagerMockSet {
	if mmSet.mock.funcSet != nil {
		mmSet.mock.t.Fatalf("PulseManagerMock.Set mock is already set by Set")
	}

	if mmSet.defaultExpectation == nil {
		mmSet.defaultExpectation = &PulseManagerMockSetExpectation{}
	}

	mmSet.defaultExpectation.params = &PulseManagerMockSetParams{ctx, pulse}
	for _, e := range mmSet.expectations {
		if minimock.Equal(e.params, mmSet.defaultExpectation.params) {
			mmSet.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmSet.defaultExpectation.params)
		}
	}

	return mmSet
}

// Inspect accepts an inspector function that has same arguments as the PulseManager.Set
func (mmSet *mPulseManagerMockSet) Inspect(f func(ctx context.Context, pulse mm_insolar.Pulse)) *mPulseManagerMockSet {
	if mmSet.mock.inspectFuncSet != nil {
		mmSet.mock.t.Fatalf("Inspect function is already set for PulseManagerMock.Set")
	}

	mmSet.mock.inspectFuncSet = f

	return mmSet
}

// Return sets up results that will be returned by PulseManager.Set
func (mmSet *mPulseManagerMockSet) Return(err error) *PulseManagerMock {
	if mmSet.mock.funcSet != nil {
		mmSet.mock.t.Fatalf("PulseManagerMock.Set mock is already set by Set")
	}

	if mmSet.defaultExpectation == nil {
		mmSet.defaultExpectation = &PulseManagerMockSetExpectation{mock: mmSet.mock}
	}
	mmSet.defaultExpectation.results = &PulseManagerMockSetResults{err}
	return mmSet.mock
}

//Set uses given function f to mock the PulseManager.Set method
func (mmSet *mPulseManagerMockSet) Set(f func(ctx context.Context, pulse mm_insolar.Pulse) (err error)) *PulseManagerMock {
	if mmSet.defaultExpectation != nil {
		mmSet.mock.t.Fatalf("Default expectation is already set for the PulseManager.Set method")
	}

	if len(mmSet.expectations) > 0 {
		mmSet.mock.t.Fatalf("Some expectations are already set for the PulseManager.Set method")
	}

	mmSet.mock.funcSet = f
	return mmSet.mock
}

// When sets expectation for the PulseManager.Set which will trigger the result defined by the following
// Then helper
func (mmSet *mPulseManagerMockSet) When(ctx context.Context, pulse mm_insolar.Pulse) *PulseManagerMockSetExpectation {
	if mmSet.mock.funcSet != nil {
		mmSet.mock.t.Fatalf("PulseManagerMock.Set mock is already set by Set")
	}

	expectation := &PulseManagerMockSetExpectation{
		mock:   mmSet.mock,
		params: &PulseManagerMockSetParams{ctx, pulse},
	}
	mmSet.expectations = append(mmSet.expectations, expectation)
	return expectation
}

// Then sets up PulseManager.Set return parameters for the expectation previously defined by the When method
func (e *PulseManagerMockSetExpectation) Then(err error) *PulseManagerMock {
	e.results = &PulseManagerMockSetResults{err}
	return e.mock
}

// Set implements insolar.PulseManager
func (mmSet *PulseManagerMock) Set(ctx context.Context, pulse mm_insolar.Pulse) (err error) {
	mm_atomic.AddUint64(&mmSet.beforeSetCounter, 1)
	defer mm_atomic.AddUint64(&mmSet.afterSetCounter, 1)

	if mmSet.inspectFuncSet != nil {
		mmSet.inspectFuncSet(ctx, pulse)
	}

	mm_params := &PulseManagerMockSetParams{ctx, pulse}

	// Record call args
	mmSet.SetMock.mutex.Lock()
	mmSet.SetMock.callArgs = append(mmSet.SetMock.callArgs, mm_params)
	mmSet.SetMock.mutex.Unlock()

	for _, e := range mmSet.SetMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmSet.SetMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmSet.SetMock.defaultExpectation.Counter, 1)
		mm_want := mmSet.SetMock.defaultExpectation.params
		mm_got := PulseManagerMockSetParams{ctx, pulse}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmSet.t.Errorf("PulseManagerMock.Set got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmSet.SetMock.defaultExpectation.results
		if mm_results == nil {
			mmSet.t.Fatal("No results are set for the PulseManagerMock.Set")
		}
		return (*mm_results).err
	}
	if mmSet.funcSet != nil {
		return mmSet.funcSet(ctx, pulse)
	}
	mmSet.t.Fatalf("Unexpected call to PulseManagerMock.Set. %v %v", ctx, pulse)
	return
}

// SetAfterCounter returns a count of finished PulseManagerMock.Set invocations
func (mmSet *PulseManagerMock) SetAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSet.afterSetCounter)
}

// SetBeforeCounter returns a count of PulseManagerMock.Set invocations
func (mmSet *PulseManagerMock) SetBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSet.beforeSetCounter)
}

// Calls returns a list of arguments used in each call to PulseManagerMock.Set.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmSet *mPulseManagerMockSet) Calls() []*PulseManagerMockSetParams {
	mmSet.mutex.RLock()

	argCopy := make([]*PulseManagerMockSetParams, len(mmSet.callArgs))
	copy(argCopy, mmSet.callArgs)

	mmSet.mutex.RUnlock()

	return argCopy
}

// MinimockSetDone returns true if the count of the Set invocations corresponds
// the number of defined expectations
func (m *PulseManagerMock) MinimockSetDone() bool {
	for _, e := range m.SetMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSet != nil && mm_atomic.LoadUint64(&m.afterSetCounter) < 1 {
		return false
	}
	return true
}

// MinimockSetInspect logs each unmet expectation
func (m *PulseManagerMock) MinimockSetInspect() {
	for _, e := range m.SetMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to PulseManagerMock.Set with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetCounter) < 1 {
		if m.SetMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to PulseManagerMock.Set")
		} else {
			m.t.Errorf("Expected call to PulseManagerMock.Set with params: %#v", *m.SetMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSet != nil && mm_atomic.LoadUint64(&m.afterSetCounter) < 1 {
		m.t.Error("Expected call to PulseManagerMock.Set")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *PulseManagerMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockSetInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *PulseManagerMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *PulseManagerMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockSetDone()
}
