package testutils

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
)

// PulseDistributorMock implements node.PulseDistributor
type PulseDistributorMock struct {
	t minimock.Tester

	funcDistribute          func(ctx context.Context, p1 pulsestor.Pulse)
	inspectFuncDistribute   func(ctx context.Context, p1 pulsestor.Pulse)
	afterDistributeCounter  uint64
	beforeDistributeCounter uint64
	DistributeMock          mPulseDistributorMockDistribute
}

// NewPulseDistributorMock returns a mock for node.PulseDistributor
func NewPulseDistributorMock(t minimock.Tester) *PulseDistributorMock {
	m := &PulseDistributorMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.DistributeMock = mPulseDistributorMockDistribute{mock: m}
	m.DistributeMock.callArgs = []*PulseDistributorMockDistributeParams{}

	return m
}

type mPulseDistributorMockDistribute struct {
	mock               *PulseDistributorMock
	defaultExpectation *PulseDistributorMockDistributeExpectation
	expectations       []*PulseDistributorMockDistributeExpectation

	callArgs []*PulseDistributorMockDistributeParams
	mutex    sync.RWMutex
}

// PulseDistributorMockDistributeExpectation specifies expectation struct of the PulseDistributor.Distribute
type PulseDistributorMockDistributeExpectation struct {
	mock   *PulseDistributorMock
	params *PulseDistributorMockDistributeParams

	Counter uint64
}

// PulseDistributorMockDistributeParams contains parameters of the PulseDistributor.Distribute
type PulseDistributorMockDistributeParams struct {
	ctx context.Context
	p1  pulsestor.Pulse
}

// Expect sets up expected params for PulseDistributor.Distribute
func (mmDistribute *mPulseDistributorMockDistribute) Expect(ctx context.Context, p1 pulsestor.Pulse) *mPulseDistributorMockDistribute {
	if mmDistribute.mock.funcDistribute != nil {
		mmDistribute.mock.t.Fatalf("PulseDistributorMock.Distribute mock is already set by Set")
	}

	if mmDistribute.defaultExpectation == nil {
		mmDistribute.defaultExpectation = &PulseDistributorMockDistributeExpectation{}
	}

	mmDistribute.defaultExpectation.params = &PulseDistributorMockDistributeParams{ctx, p1}
	for _, e := range mmDistribute.expectations {
		if minimock.Equal(e.params, mmDistribute.defaultExpectation.params) {
			mmDistribute.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmDistribute.defaultExpectation.params)
		}
	}

	return mmDistribute
}

// Inspect accepts an inspector function that has same arguments as the PulseDistributor.Distribute
func (mmDistribute *mPulseDistributorMockDistribute) Inspect(f func(ctx context.Context, p1 pulsestor.Pulse)) *mPulseDistributorMockDistribute {
	if mmDistribute.mock.inspectFuncDistribute != nil {
		mmDistribute.mock.t.Fatalf("Inspect function is already set for PulseDistributorMock.Distribute")
	}

	mmDistribute.mock.inspectFuncDistribute = f

	return mmDistribute
}

// Return sets up results that will be returned by PulseDistributor.Distribute
func (mmDistribute *mPulseDistributorMockDistribute) Return() *PulseDistributorMock {
	if mmDistribute.mock.funcDistribute != nil {
		mmDistribute.mock.t.Fatalf("PulseDistributorMock.Distribute mock is already set by Set")
	}

	if mmDistribute.defaultExpectation == nil {
		mmDistribute.defaultExpectation = &PulseDistributorMockDistributeExpectation{mock: mmDistribute.mock}
	}

	return mmDistribute.mock
}

//Set uses given function f to mock the PulseDistributor.Distribute method
func (mmDistribute *mPulseDistributorMockDistribute) Set(f func(ctx context.Context, p1 pulsestor.Pulse)) *PulseDistributorMock {
	if mmDistribute.defaultExpectation != nil {
		mmDistribute.mock.t.Fatalf("Default expectation is already set for the PulseDistributor.Distribute method")
	}

	if len(mmDistribute.expectations) > 0 {
		mmDistribute.mock.t.Fatalf("Some expectations are already set for the PulseDistributor.Distribute method")
	}

	mmDistribute.mock.funcDistribute = f
	return mmDistribute.mock
}

// Distribute implements node.PulseDistributor
func (mmDistribute *PulseDistributorMock) Distribute(ctx context.Context, p1 pulsestor.Pulse) {
	mm_atomic.AddUint64(&mmDistribute.beforeDistributeCounter, 1)
	defer mm_atomic.AddUint64(&mmDistribute.afterDistributeCounter, 1)

	if mmDistribute.inspectFuncDistribute != nil {
		mmDistribute.inspectFuncDistribute(ctx, p1)
	}

	mm_params := &PulseDistributorMockDistributeParams{ctx, p1}

	// Record call args
	mmDistribute.DistributeMock.mutex.Lock()
	mmDistribute.DistributeMock.callArgs = append(mmDistribute.DistributeMock.callArgs, mm_params)
	mmDistribute.DistributeMock.mutex.Unlock()

	for _, e := range mmDistribute.DistributeMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return
		}
	}

	if mmDistribute.DistributeMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmDistribute.DistributeMock.defaultExpectation.Counter, 1)
		mm_want := mmDistribute.DistributeMock.defaultExpectation.params
		mm_got := PulseDistributorMockDistributeParams{ctx, p1}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmDistribute.t.Errorf("PulseDistributorMock.Distribute got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		return

	}
	if mmDistribute.funcDistribute != nil {
		mmDistribute.funcDistribute(ctx, p1)
		return
	}
	mmDistribute.t.Fatalf("Unexpected call to PulseDistributorMock.Distribute. %v %v", ctx, p1)

}

// DistributeAfterCounter returns a count of finished PulseDistributorMock.Distribute invocations
func (mmDistribute *PulseDistributorMock) DistributeAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmDistribute.afterDistributeCounter)
}

// DistributeBeforeCounter returns a count of PulseDistributorMock.Distribute invocations
func (mmDistribute *PulseDistributorMock) DistributeBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmDistribute.beforeDistributeCounter)
}

// Calls returns a list of arguments used in each call to PulseDistributorMock.Distribute.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmDistribute *mPulseDistributorMockDistribute) Calls() []*PulseDistributorMockDistributeParams {
	mmDistribute.mutex.RLock()

	argCopy := make([]*PulseDistributorMockDistributeParams, len(mmDistribute.callArgs))
	copy(argCopy, mmDistribute.callArgs)

	mmDistribute.mutex.RUnlock()

	return argCopy
}

// MinimockDistributeDone returns true if the count of the Distribute invocations corresponds
// the number of defined expectations
func (m *PulseDistributorMock) MinimockDistributeDone() bool {
	for _, e := range m.DistributeMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.DistributeMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterDistributeCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcDistribute != nil && mm_atomic.LoadUint64(&m.afterDistributeCounter) < 1 {
		return false
	}
	return true
}

// MinimockDistributeInspect logs each unmet expectation
func (m *PulseDistributorMock) MinimockDistributeInspect() {
	for _, e := range m.DistributeMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to PulseDistributorMock.Distribute with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.DistributeMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterDistributeCounter) < 1 {
		if m.DistributeMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to PulseDistributorMock.Distribute")
		} else {
			m.t.Errorf("Expected call to PulseDistributorMock.Distribute with params: %#v", *m.DistributeMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcDistribute != nil && mm_atomic.LoadUint64(&m.afterDistributeCounter) < 1 {
		m.t.Error("Expected call to PulseDistributorMock.Distribute")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *PulseDistributorMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockDistributeInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *PulseDistributorMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *PulseDistributorMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockDistributeDone()
}
