package pulse

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

// ShifterMock implements Shifter
type ShifterMock struct {
	t minimock.Tester

	funcShift          func(ctx context.Context, pn pulse.Number) (err error)
	inspectFuncShift   func(ctx context.Context, pn pulse.Number)
	afterShiftCounter  uint64
	beforeShiftCounter uint64
	ShiftMock          mShifterMockShift
}

// NewShifterMock returns a mock for Shifter
func NewShifterMock(t minimock.Tester) *ShifterMock {
	m := &ShifterMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.ShiftMock = mShifterMockShift{mock: m}
	m.ShiftMock.callArgs = []*ShifterMockShiftParams{}

	return m
}

type mShifterMockShift struct {
	mock               *ShifterMock
	defaultExpectation *ShifterMockShiftExpectation
	expectations       []*ShifterMockShiftExpectation

	callArgs []*ShifterMockShiftParams
	mutex    sync.RWMutex
}

// ShifterMockShiftExpectation specifies expectation struct of the Shifter.Shift
type ShifterMockShiftExpectation struct {
	mock    *ShifterMock
	params  *ShifterMockShiftParams
	results *ShifterMockShiftResults
	Counter uint64
}

// ShifterMockShiftParams contains parameters of the Shifter.Shift
type ShifterMockShiftParams struct {
	ctx context.Context
	pn  pulse.Number
}

// ShifterMockShiftResults contains results of the Shifter.Shift
type ShifterMockShiftResults struct {
	err error
}

// Expect sets up expected params for Shifter.Shift
func (mmShift *mShifterMockShift) Expect(ctx context.Context, pn pulse.Number) *mShifterMockShift {
	if mmShift.mock.funcShift != nil {
		mmShift.mock.t.Fatalf("ShifterMock.Shift mock is already set by Set")
	}

	if mmShift.defaultExpectation == nil {
		mmShift.defaultExpectation = &ShifterMockShiftExpectation{}
	}

	mmShift.defaultExpectation.params = &ShifterMockShiftParams{ctx, pn}
	for _, e := range mmShift.expectations {
		if minimock.Equal(e.params, mmShift.defaultExpectation.params) {
			mmShift.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmShift.defaultExpectation.params)
		}
	}

	return mmShift
}

// Inspect accepts an inspector function that has same arguments as the Shifter.Shift
func (mmShift *mShifterMockShift) Inspect(f func(ctx context.Context, pn pulse.Number)) *mShifterMockShift {
	if mmShift.mock.inspectFuncShift != nil {
		mmShift.mock.t.Fatalf("Inspect function is already set for ShifterMock.Shift")
	}

	mmShift.mock.inspectFuncShift = f

	return mmShift
}

// Return sets up results that will be returned by Shifter.Shift
func (mmShift *mShifterMockShift) Return(err error) *ShifterMock {
	if mmShift.mock.funcShift != nil {
		mmShift.mock.t.Fatalf("ShifterMock.Shift mock is already set by Set")
	}

	if mmShift.defaultExpectation == nil {
		mmShift.defaultExpectation = &ShifterMockShiftExpectation{mock: mmShift.mock}
	}
	mmShift.defaultExpectation.results = &ShifterMockShiftResults{err}
	return mmShift.mock
}

//Set uses given function f to mock the Shifter.Shift method
func (mmShift *mShifterMockShift) Set(f func(ctx context.Context, pn pulse.Number) (err error)) *ShifterMock {
	if mmShift.defaultExpectation != nil {
		mmShift.mock.t.Fatalf("Default expectation is already set for the Shifter.Shift method")
	}

	if len(mmShift.expectations) > 0 {
		mmShift.mock.t.Fatalf("Some expectations are already set for the Shifter.Shift method")
	}

	mmShift.mock.funcShift = f
	return mmShift.mock
}

// When sets expectation for the Shifter.Shift which will trigger the result defined by the following
// Then helper
func (mmShift *mShifterMockShift) When(ctx context.Context, pn pulse.Number) *ShifterMockShiftExpectation {
	if mmShift.mock.funcShift != nil {
		mmShift.mock.t.Fatalf("ShifterMock.Shift mock is already set by Set")
	}

	expectation := &ShifterMockShiftExpectation{
		mock:   mmShift.mock,
		params: &ShifterMockShiftParams{ctx, pn},
	}
	mmShift.expectations = append(mmShift.expectations, expectation)
	return expectation
}

// Then sets up Shifter.Shift return parameters for the expectation previously defined by the When method
func (e *ShifterMockShiftExpectation) Then(err error) *ShifterMock {
	e.results = &ShifterMockShiftResults{err}
	return e.mock
}

// Shift implements Shifter
func (mmShift *ShifterMock) Shift(ctx context.Context, pn pulse.Number) (err error) {
	mm_atomic.AddUint64(&mmShift.beforeShiftCounter, 1)
	defer mm_atomic.AddUint64(&mmShift.afterShiftCounter, 1)

	if mmShift.inspectFuncShift != nil {
		mmShift.inspectFuncShift(ctx, pn)
	}

	mm_params := &ShifterMockShiftParams{ctx, pn}

	// Record call args
	mmShift.ShiftMock.mutex.Lock()
	mmShift.ShiftMock.callArgs = append(mmShift.ShiftMock.callArgs, mm_params)
	mmShift.ShiftMock.mutex.Unlock()

	for _, e := range mmShift.ShiftMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmShift.ShiftMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmShift.ShiftMock.defaultExpectation.Counter, 1)
		mm_want := mmShift.ShiftMock.defaultExpectation.params
		mm_got := ShifterMockShiftParams{ctx, pn}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmShift.t.Errorf("ShifterMock.Shift got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmShift.ShiftMock.defaultExpectation.results
		if mm_results == nil {
			mmShift.t.Fatal("No results are set for the ShifterMock.Shift")
		}
		return (*mm_results).err
	}
	if mmShift.funcShift != nil {
		return mmShift.funcShift(ctx, pn)
	}
	mmShift.t.Fatalf("Unexpected call to ShifterMock.Shift. %v %v", ctx, pn)
	return
}

// ShiftAfterCounter returns a count of finished ShifterMock.Shift invocations
func (mmShift *ShifterMock) ShiftAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmShift.afterShiftCounter)
}

// ShiftBeforeCounter returns a count of ShifterMock.Shift invocations
func (mmShift *ShifterMock) ShiftBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmShift.beforeShiftCounter)
}

// Calls returns a list of arguments used in each call to ShifterMock.Shift.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmShift *mShifterMockShift) Calls() []*ShifterMockShiftParams {
	mmShift.mutex.RLock()

	argCopy := make([]*ShifterMockShiftParams, len(mmShift.callArgs))
	copy(argCopy, mmShift.callArgs)

	mmShift.mutex.RUnlock()

	return argCopy
}

// MinimockShiftDone returns true if the count of the Shift invocations corresponds
// the number of defined expectations
func (m *ShifterMock) MinimockShiftDone() bool {
	for _, e := range m.ShiftMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.ShiftMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterShiftCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcShift != nil && mm_atomic.LoadUint64(&m.afterShiftCounter) < 1 {
		return false
	}
	return true
}

// MinimockShiftInspect logs each unmet expectation
func (m *ShifterMock) MinimockShiftInspect() {
	for _, e := range m.ShiftMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ShifterMock.Shift with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.ShiftMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterShiftCounter) < 1 {
		if m.ShiftMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ShifterMock.Shift")
		} else {
			m.t.Errorf("Expected call to ShifterMock.Shift with params: %#v", *m.ShiftMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcShift != nil && mm_atomic.LoadUint64(&m.afterShiftCounter) < 1 {
		m.t.Error("Expected call to ShifterMock.Shift")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *ShifterMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockShiftInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *ShifterMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *ShifterMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockShiftDone()
}
