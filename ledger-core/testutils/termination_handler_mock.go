package testutils

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

// TerminationHandlerMock implements network.TerminationHandler
type TerminationHandlerMock struct {
	t minimock.Tester

	funcLeave          func(ctx context.Context, n1 pulse.Number)
	inspectFuncLeave   func(ctx context.Context, n1 pulse.Number)
	afterLeaveCounter  uint64
	beforeLeaveCounter uint64
	LeaveMock          mTerminationHandlerMockLeave

	funcOnLeaveApproved          func(ctx context.Context)
	inspectFuncOnLeaveApproved   func(ctx context.Context)
	afterOnLeaveApprovedCounter  uint64
	beforeOnLeaveApprovedCounter uint64
	OnLeaveApprovedMock          mTerminationHandlerMockOnLeaveApproved

	funcTerminating          func() (b1 bool)
	inspectFuncTerminating   func()
	afterTerminatingCounter  uint64
	beforeTerminatingCounter uint64
	TerminatingMock          mTerminationHandlerMockTerminating
}

// NewTerminationHandlerMock returns a mock for network.TerminationHandler
func NewTerminationHandlerMock(t minimock.Tester) *TerminationHandlerMock {
	m := &TerminationHandlerMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.LeaveMock = mTerminationHandlerMockLeave{mock: m}
	m.LeaveMock.callArgs = []*TerminationHandlerMockLeaveParams{}

	m.OnLeaveApprovedMock = mTerminationHandlerMockOnLeaveApproved{mock: m}
	m.OnLeaveApprovedMock.callArgs = []*TerminationHandlerMockOnLeaveApprovedParams{}

	m.TerminatingMock = mTerminationHandlerMockTerminating{mock: m}

	return m
}

type mTerminationHandlerMockLeave struct {
	mock               *TerminationHandlerMock
	defaultExpectation *TerminationHandlerMockLeaveExpectation
	expectations       []*TerminationHandlerMockLeaveExpectation

	callArgs []*TerminationHandlerMockLeaveParams
	mutex    sync.RWMutex
}

// TerminationHandlerMockLeaveExpectation specifies expectation struct of the TerminationHandler.Leave
type TerminationHandlerMockLeaveExpectation struct {
	mock   *TerminationHandlerMock
	params *TerminationHandlerMockLeaveParams

	Counter uint64
}

// TerminationHandlerMockLeaveParams contains parameters of the TerminationHandler.Leave
type TerminationHandlerMockLeaveParams struct {
	ctx context.Context
	n1  pulse.Number
}

// Expect sets up expected params for TerminationHandler.Leave
func (mmLeave *mTerminationHandlerMockLeave) Expect(ctx context.Context, n1 pulse.Number) *mTerminationHandlerMockLeave {
	if mmLeave.mock.funcLeave != nil {
		mmLeave.mock.t.Fatalf("TerminationHandlerMock.Leave mock is already set by Set")
	}

	if mmLeave.defaultExpectation == nil {
		mmLeave.defaultExpectation = &TerminationHandlerMockLeaveExpectation{}
	}

	mmLeave.defaultExpectation.params = &TerminationHandlerMockLeaveParams{ctx, n1}
	for _, e := range mmLeave.expectations {
		if minimock.Equal(e.params, mmLeave.defaultExpectation.params) {
			mmLeave.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmLeave.defaultExpectation.params)
		}
	}

	return mmLeave
}

// Inspect accepts an inspector function that has same arguments as the TerminationHandler.Leave
func (mmLeave *mTerminationHandlerMockLeave) Inspect(f func(ctx context.Context, n1 pulse.Number)) *mTerminationHandlerMockLeave {
	if mmLeave.mock.inspectFuncLeave != nil {
		mmLeave.mock.t.Fatalf("Inspect function is already set for TerminationHandlerMock.Leave")
	}

	mmLeave.mock.inspectFuncLeave = f

	return mmLeave
}

// Return sets up results that will be returned by TerminationHandler.Leave
func (mmLeave *mTerminationHandlerMockLeave) Return() *TerminationHandlerMock {
	if mmLeave.mock.funcLeave != nil {
		mmLeave.mock.t.Fatalf("TerminationHandlerMock.Leave mock is already set by Set")
	}

	if mmLeave.defaultExpectation == nil {
		mmLeave.defaultExpectation = &TerminationHandlerMockLeaveExpectation{mock: mmLeave.mock}
	}

	return mmLeave.mock
}

//Set uses given function f to mock the TerminationHandler.Leave method
func (mmLeave *mTerminationHandlerMockLeave) Set(f func(ctx context.Context, n1 pulse.Number)) *TerminationHandlerMock {
	if mmLeave.defaultExpectation != nil {
		mmLeave.mock.t.Fatalf("Default expectation is already set for the TerminationHandler.Leave method")
	}

	if len(mmLeave.expectations) > 0 {
		mmLeave.mock.t.Fatalf("Some expectations are already set for the TerminationHandler.Leave method")
	}

	mmLeave.mock.funcLeave = f
	return mmLeave.mock
}

// Leave implements network.TerminationHandler
func (mmLeave *TerminationHandlerMock) Leave(ctx context.Context, n1 pulse.Number) {
	mm_atomic.AddUint64(&mmLeave.beforeLeaveCounter, 1)
	defer mm_atomic.AddUint64(&mmLeave.afterLeaveCounter, 1)

	if mmLeave.inspectFuncLeave != nil {
		mmLeave.inspectFuncLeave(ctx, n1)
	}

	mm_params := &TerminationHandlerMockLeaveParams{ctx, n1}

	// Record call args
	mmLeave.LeaveMock.mutex.Lock()
	mmLeave.LeaveMock.callArgs = append(mmLeave.LeaveMock.callArgs, mm_params)
	mmLeave.LeaveMock.mutex.Unlock()

	for _, e := range mmLeave.LeaveMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return
		}
	}

	if mmLeave.LeaveMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmLeave.LeaveMock.defaultExpectation.Counter, 1)
		mm_want := mmLeave.LeaveMock.defaultExpectation.params
		mm_got := TerminationHandlerMockLeaveParams{ctx, n1}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmLeave.t.Errorf("TerminationHandlerMock.Leave got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		return

	}
	if mmLeave.funcLeave != nil {
		mmLeave.funcLeave(ctx, n1)
		return
	}
	mmLeave.t.Fatalf("Unexpected call to TerminationHandlerMock.Leave. %v %v", ctx, n1)

}

// LeaveAfterCounter returns a count of finished TerminationHandlerMock.Leave invocations
func (mmLeave *TerminationHandlerMock) LeaveAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmLeave.afterLeaveCounter)
}

// LeaveBeforeCounter returns a count of TerminationHandlerMock.Leave invocations
func (mmLeave *TerminationHandlerMock) LeaveBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmLeave.beforeLeaveCounter)
}

// Calls returns a list of arguments used in each call to TerminationHandlerMock.Leave.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmLeave *mTerminationHandlerMockLeave) Calls() []*TerminationHandlerMockLeaveParams {
	mmLeave.mutex.RLock()

	argCopy := make([]*TerminationHandlerMockLeaveParams, len(mmLeave.callArgs))
	copy(argCopy, mmLeave.callArgs)

	mmLeave.mutex.RUnlock()

	return argCopy
}

// MinimockLeaveDone returns true if the count of the Leave invocations corresponds
// the number of defined expectations
func (m *TerminationHandlerMock) MinimockLeaveDone() bool {
	for _, e := range m.LeaveMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.LeaveMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterLeaveCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcLeave != nil && mm_atomic.LoadUint64(&m.afterLeaveCounter) < 1 {
		return false
	}
	return true
}

// MinimockLeaveInspect logs each unmet expectation
func (m *TerminationHandlerMock) MinimockLeaveInspect() {
	for _, e := range m.LeaveMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to TerminationHandlerMock.Leave with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.LeaveMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterLeaveCounter) < 1 {
		if m.LeaveMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to TerminationHandlerMock.Leave")
		} else {
			m.t.Errorf("Expected call to TerminationHandlerMock.Leave with params: %#v", *m.LeaveMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcLeave != nil && mm_atomic.LoadUint64(&m.afterLeaveCounter) < 1 {
		m.t.Error("Expected call to TerminationHandlerMock.Leave")
	}
}

type mTerminationHandlerMockOnLeaveApproved struct {
	mock               *TerminationHandlerMock
	defaultExpectation *TerminationHandlerMockOnLeaveApprovedExpectation
	expectations       []*TerminationHandlerMockOnLeaveApprovedExpectation

	callArgs []*TerminationHandlerMockOnLeaveApprovedParams
	mutex    sync.RWMutex
}

// TerminationHandlerMockOnLeaveApprovedExpectation specifies expectation struct of the TerminationHandler.OnLeaveApproved
type TerminationHandlerMockOnLeaveApprovedExpectation struct {
	mock   *TerminationHandlerMock
	params *TerminationHandlerMockOnLeaveApprovedParams

	Counter uint64
}

// TerminationHandlerMockOnLeaveApprovedParams contains parameters of the TerminationHandler.OnLeaveApproved
type TerminationHandlerMockOnLeaveApprovedParams struct {
	ctx context.Context
}

// Expect sets up expected params for TerminationHandler.OnLeaveApproved
func (mmOnLeaveApproved *mTerminationHandlerMockOnLeaveApproved) Expect(ctx context.Context) *mTerminationHandlerMockOnLeaveApproved {
	if mmOnLeaveApproved.mock.funcOnLeaveApproved != nil {
		mmOnLeaveApproved.mock.t.Fatalf("TerminationHandlerMock.OnLeaveApproved mock is already set by Set")
	}

	if mmOnLeaveApproved.defaultExpectation == nil {
		mmOnLeaveApproved.defaultExpectation = &TerminationHandlerMockOnLeaveApprovedExpectation{}
	}

	mmOnLeaveApproved.defaultExpectation.params = &TerminationHandlerMockOnLeaveApprovedParams{ctx}
	for _, e := range mmOnLeaveApproved.expectations {
		if minimock.Equal(e.params, mmOnLeaveApproved.defaultExpectation.params) {
			mmOnLeaveApproved.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmOnLeaveApproved.defaultExpectation.params)
		}
	}

	return mmOnLeaveApproved
}

// Inspect accepts an inspector function that has same arguments as the TerminationHandler.OnLeaveApproved
func (mmOnLeaveApproved *mTerminationHandlerMockOnLeaveApproved) Inspect(f func(ctx context.Context)) *mTerminationHandlerMockOnLeaveApproved {
	if mmOnLeaveApproved.mock.inspectFuncOnLeaveApproved != nil {
		mmOnLeaveApproved.mock.t.Fatalf("Inspect function is already set for TerminationHandlerMock.OnLeaveApproved")
	}

	mmOnLeaveApproved.mock.inspectFuncOnLeaveApproved = f

	return mmOnLeaveApproved
}

// Return sets up results that will be returned by TerminationHandler.OnLeaveApproved
func (mmOnLeaveApproved *mTerminationHandlerMockOnLeaveApproved) Return() *TerminationHandlerMock {
	if mmOnLeaveApproved.mock.funcOnLeaveApproved != nil {
		mmOnLeaveApproved.mock.t.Fatalf("TerminationHandlerMock.OnLeaveApproved mock is already set by Set")
	}

	if mmOnLeaveApproved.defaultExpectation == nil {
		mmOnLeaveApproved.defaultExpectation = &TerminationHandlerMockOnLeaveApprovedExpectation{mock: mmOnLeaveApproved.mock}
	}

	return mmOnLeaveApproved.mock
}

//Set uses given function f to mock the TerminationHandler.OnLeaveApproved method
func (mmOnLeaveApproved *mTerminationHandlerMockOnLeaveApproved) Set(f func(ctx context.Context)) *TerminationHandlerMock {
	if mmOnLeaveApproved.defaultExpectation != nil {
		mmOnLeaveApproved.mock.t.Fatalf("Default expectation is already set for the TerminationHandler.OnLeaveApproved method")
	}

	if len(mmOnLeaveApproved.expectations) > 0 {
		mmOnLeaveApproved.mock.t.Fatalf("Some expectations are already set for the TerminationHandler.OnLeaveApproved method")
	}

	mmOnLeaveApproved.mock.funcOnLeaveApproved = f
	return mmOnLeaveApproved.mock
}

// OnLeaveApproved implements network.TerminationHandler
func (mmOnLeaveApproved *TerminationHandlerMock) OnLeaveApproved(ctx context.Context) {
	mm_atomic.AddUint64(&mmOnLeaveApproved.beforeOnLeaveApprovedCounter, 1)
	defer mm_atomic.AddUint64(&mmOnLeaveApproved.afterOnLeaveApprovedCounter, 1)

	if mmOnLeaveApproved.inspectFuncOnLeaveApproved != nil {
		mmOnLeaveApproved.inspectFuncOnLeaveApproved(ctx)
	}

	mm_params := &TerminationHandlerMockOnLeaveApprovedParams{ctx}

	// Record call args
	mmOnLeaveApproved.OnLeaveApprovedMock.mutex.Lock()
	mmOnLeaveApproved.OnLeaveApprovedMock.callArgs = append(mmOnLeaveApproved.OnLeaveApprovedMock.callArgs, mm_params)
	mmOnLeaveApproved.OnLeaveApprovedMock.mutex.Unlock()

	for _, e := range mmOnLeaveApproved.OnLeaveApprovedMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return
		}
	}

	if mmOnLeaveApproved.OnLeaveApprovedMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmOnLeaveApproved.OnLeaveApprovedMock.defaultExpectation.Counter, 1)
		mm_want := mmOnLeaveApproved.OnLeaveApprovedMock.defaultExpectation.params
		mm_got := TerminationHandlerMockOnLeaveApprovedParams{ctx}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmOnLeaveApproved.t.Errorf("TerminationHandlerMock.OnLeaveApproved got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		return

	}
	if mmOnLeaveApproved.funcOnLeaveApproved != nil {
		mmOnLeaveApproved.funcOnLeaveApproved(ctx)
		return
	}
	mmOnLeaveApproved.t.Fatalf("Unexpected call to TerminationHandlerMock.OnLeaveApproved. %v", ctx)

}

// OnLeaveApprovedAfterCounter returns a count of finished TerminationHandlerMock.OnLeaveApproved invocations
func (mmOnLeaveApproved *TerminationHandlerMock) OnLeaveApprovedAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmOnLeaveApproved.afterOnLeaveApprovedCounter)
}

// OnLeaveApprovedBeforeCounter returns a count of TerminationHandlerMock.OnLeaveApproved invocations
func (mmOnLeaveApproved *TerminationHandlerMock) OnLeaveApprovedBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmOnLeaveApproved.beforeOnLeaveApprovedCounter)
}

// Calls returns a list of arguments used in each call to TerminationHandlerMock.OnLeaveApproved.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmOnLeaveApproved *mTerminationHandlerMockOnLeaveApproved) Calls() []*TerminationHandlerMockOnLeaveApprovedParams {
	mmOnLeaveApproved.mutex.RLock()

	argCopy := make([]*TerminationHandlerMockOnLeaveApprovedParams, len(mmOnLeaveApproved.callArgs))
	copy(argCopy, mmOnLeaveApproved.callArgs)

	mmOnLeaveApproved.mutex.RUnlock()

	return argCopy
}

// MinimockOnLeaveApprovedDone returns true if the count of the OnLeaveApproved invocations corresponds
// the number of defined expectations
func (m *TerminationHandlerMock) MinimockOnLeaveApprovedDone() bool {
	for _, e := range m.OnLeaveApprovedMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.OnLeaveApprovedMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterOnLeaveApprovedCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcOnLeaveApproved != nil && mm_atomic.LoadUint64(&m.afterOnLeaveApprovedCounter) < 1 {
		return false
	}
	return true
}

// MinimockOnLeaveApprovedInspect logs each unmet expectation
func (m *TerminationHandlerMock) MinimockOnLeaveApprovedInspect() {
	for _, e := range m.OnLeaveApprovedMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to TerminationHandlerMock.OnLeaveApproved with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.OnLeaveApprovedMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterOnLeaveApprovedCounter) < 1 {
		if m.OnLeaveApprovedMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to TerminationHandlerMock.OnLeaveApproved")
		} else {
			m.t.Errorf("Expected call to TerminationHandlerMock.OnLeaveApproved with params: %#v", *m.OnLeaveApprovedMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcOnLeaveApproved != nil && mm_atomic.LoadUint64(&m.afterOnLeaveApprovedCounter) < 1 {
		m.t.Error("Expected call to TerminationHandlerMock.OnLeaveApproved")
	}
}

type mTerminationHandlerMockTerminating struct {
	mock               *TerminationHandlerMock
	defaultExpectation *TerminationHandlerMockTerminatingExpectation
	expectations       []*TerminationHandlerMockTerminatingExpectation
}

// TerminationHandlerMockTerminatingExpectation specifies expectation struct of the TerminationHandler.Terminating
type TerminationHandlerMockTerminatingExpectation struct {
	mock *TerminationHandlerMock

	results *TerminationHandlerMockTerminatingResults
	Counter uint64
}

// TerminationHandlerMockTerminatingResults contains results of the TerminationHandler.Terminating
type TerminationHandlerMockTerminatingResults struct {
	b1 bool
}

// Expect sets up expected params for TerminationHandler.Terminating
func (mmTerminating *mTerminationHandlerMockTerminating) Expect() *mTerminationHandlerMockTerminating {
	if mmTerminating.mock.funcTerminating != nil {
		mmTerminating.mock.t.Fatalf("TerminationHandlerMock.Terminating mock is already set by Set")
	}

	if mmTerminating.defaultExpectation == nil {
		mmTerminating.defaultExpectation = &TerminationHandlerMockTerminatingExpectation{}
	}

	return mmTerminating
}

// Inspect accepts an inspector function that has same arguments as the TerminationHandler.Terminating
func (mmTerminating *mTerminationHandlerMockTerminating) Inspect(f func()) *mTerminationHandlerMockTerminating {
	if mmTerminating.mock.inspectFuncTerminating != nil {
		mmTerminating.mock.t.Fatalf("Inspect function is already set for TerminationHandlerMock.Terminating")
	}

	mmTerminating.mock.inspectFuncTerminating = f

	return mmTerminating
}

// Return sets up results that will be returned by TerminationHandler.Terminating
func (mmTerminating *mTerminationHandlerMockTerminating) Return(b1 bool) *TerminationHandlerMock {
	if mmTerminating.mock.funcTerminating != nil {
		mmTerminating.mock.t.Fatalf("TerminationHandlerMock.Terminating mock is already set by Set")
	}

	if mmTerminating.defaultExpectation == nil {
		mmTerminating.defaultExpectation = &TerminationHandlerMockTerminatingExpectation{mock: mmTerminating.mock}
	}
	mmTerminating.defaultExpectation.results = &TerminationHandlerMockTerminatingResults{b1}
	return mmTerminating.mock
}

//Set uses given function f to mock the TerminationHandler.Terminating method
func (mmTerminating *mTerminationHandlerMockTerminating) Set(f func() (b1 bool)) *TerminationHandlerMock {
	if mmTerminating.defaultExpectation != nil {
		mmTerminating.mock.t.Fatalf("Default expectation is already set for the TerminationHandler.Terminating method")
	}

	if len(mmTerminating.expectations) > 0 {
		mmTerminating.mock.t.Fatalf("Some expectations are already set for the TerminationHandler.Terminating method")
	}

	mmTerminating.mock.funcTerminating = f
	return mmTerminating.mock
}

// Terminating implements network.TerminationHandler
func (mmTerminating *TerminationHandlerMock) Terminating() (b1 bool) {
	mm_atomic.AddUint64(&mmTerminating.beforeTerminatingCounter, 1)
	defer mm_atomic.AddUint64(&mmTerminating.afterTerminatingCounter, 1)

	if mmTerminating.inspectFuncTerminating != nil {
		mmTerminating.inspectFuncTerminating()
	}

	if mmTerminating.TerminatingMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmTerminating.TerminatingMock.defaultExpectation.Counter, 1)

		mm_results := mmTerminating.TerminatingMock.defaultExpectation.results
		if mm_results == nil {
			mmTerminating.t.Fatal("No results are set for the TerminationHandlerMock.Terminating")
		}
		return (*mm_results).b1
	}
	if mmTerminating.funcTerminating != nil {
		return mmTerminating.funcTerminating()
	}
	mmTerminating.t.Fatalf("Unexpected call to TerminationHandlerMock.Terminating.")
	return
}

// TerminatingAfterCounter returns a count of finished TerminationHandlerMock.Terminating invocations
func (mmTerminating *TerminationHandlerMock) TerminatingAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmTerminating.afterTerminatingCounter)
}

// TerminatingBeforeCounter returns a count of TerminationHandlerMock.Terminating invocations
func (mmTerminating *TerminationHandlerMock) TerminatingBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmTerminating.beforeTerminatingCounter)
}

// MinimockTerminatingDone returns true if the count of the Terminating invocations corresponds
// the number of defined expectations
func (m *TerminationHandlerMock) MinimockTerminatingDone() bool {
	for _, e := range m.TerminatingMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.TerminatingMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterTerminatingCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcTerminating != nil && mm_atomic.LoadUint64(&m.afterTerminatingCounter) < 1 {
		return false
	}
	return true
}

// MinimockTerminatingInspect logs each unmet expectation
func (m *TerminationHandlerMock) MinimockTerminatingInspect() {
	for _, e := range m.TerminatingMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to TerminationHandlerMock.Terminating")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.TerminatingMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterTerminatingCounter) < 1 {
		m.t.Error("Expected call to TerminationHandlerMock.Terminating")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcTerminating != nil && mm_atomic.LoadUint64(&m.afterTerminatingCounter) < 1 {
		m.t.Error("Expected call to TerminationHandlerMock.Terminating")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *TerminationHandlerMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockLeaveInspect()

		m.MinimockOnLeaveApprovedInspect()

		m.MinimockTerminatingInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *TerminationHandlerMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *TerminationHandlerMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockLeaveDone() &&
		m.MinimockOnLeaveApprovedDone() &&
		m.MinimockTerminatingDone()
}
