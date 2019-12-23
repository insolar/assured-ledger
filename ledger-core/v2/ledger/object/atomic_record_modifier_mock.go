package object

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/ledger/object.AtomicRecordModifier -o ./atomic_record_modifier_mock.go

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
)

// AtomicRecordModifierMock implements AtomicRecordModifier
type AtomicRecordModifierMock struct {
	t minimock.Tester

	funcSetAtomic          func(ctx context.Context, records ...record.Material) (err error)
	inspectFuncSetAtomic   func(ctx context.Context, records ...record.Material)
	afterSetAtomicCounter  uint64
	beforeSetAtomicCounter uint64
	SetAtomicMock          mAtomicRecordModifierMockSetAtomic
}

// NewAtomicRecordModifierMock returns a mock for AtomicRecordModifier
func NewAtomicRecordModifierMock(t minimock.Tester) *AtomicRecordModifierMock {
	m := &AtomicRecordModifierMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.SetAtomicMock = mAtomicRecordModifierMockSetAtomic{mock: m}
	m.SetAtomicMock.callArgs = []*AtomicRecordModifierMockSetAtomicParams{}

	return m
}

type mAtomicRecordModifierMockSetAtomic struct {
	mock               *AtomicRecordModifierMock
	defaultExpectation *AtomicRecordModifierMockSetAtomicExpectation
	expectations       []*AtomicRecordModifierMockSetAtomicExpectation

	callArgs []*AtomicRecordModifierMockSetAtomicParams
	mutex    sync.RWMutex
}

// AtomicRecordModifierMockSetAtomicExpectation specifies expectation struct of the AtomicRecordModifier.SetAtomic
type AtomicRecordModifierMockSetAtomicExpectation struct {
	mock    *AtomicRecordModifierMock
	params  *AtomicRecordModifierMockSetAtomicParams
	results *AtomicRecordModifierMockSetAtomicResults
	Counter uint64
}

// AtomicRecordModifierMockSetAtomicParams contains parameters of the AtomicRecordModifier.SetAtomic
type AtomicRecordModifierMockSetAtomicParams struct {
	ctx     context.Context
	records []record.Material
}

// AtomicRecordModifierMockSetAtomicResults contains results of the AtomicRecordModifier.SetAtomic
type AtomicRecordModifierMockSetAtomicResults struct {
	err error
}

// Expect sets up expected params for AtomicRecordModifier.SetAtomic
func (mmSetAtomic *mAtomicRecordModifierMockSetAtomic) Expect(ctx context.Context, records ...record.Material) *mAtomicRecordModifierMockSetAtomic {
	if mmSetAtomic.mock.funcSetAtomic != nil {
		mmSetAtomic.mock.t.Fatalf("AtomicRecordModifierMock.SetAtomic mock is already set by Set")
	}

	if mmSetAtomic.defaultExpectation == nil {
		mmSetAtomic.defaultExpectation = &AtomicRecordModifierMockSetAtomicExpectation{}
	}

	mmSetAtomic.defaultExpectation.params = &AtomicRecordModifierMockSetAtomicParams{ctx, records}
	for _, e := range mmSetAtomic.expectations {
		if minimock.Equal(e.params, mmSetAtomic.defaultExpectation.params) {
			mmSetAtomic.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmSetAtomic.defaultExpectation.params)
		}
	}

	return mmSetAtomic
}

// Inspect accepts an inspector function that has same arguments as the AtomicRecordModifier.SetAtomic
func (mmSetAtomic *mAtomicRecordModifierMockSetAtomic) Inspect(f func(ctx context.Context, records ...record.Material)) *mAtomicRecordModifierMockSetAtomic {
	if mmSetAtomic.mock.inspectFuncSetAtomic != nil {
		mmSetAtomic.mock.t.Fatalf("Inspect function is already set for AtomicRecordModifierMock.SetAtomic")
	}

	mmSetAtomic.mock.inspectFuncSetAtomic = f

	return mmSetAtomic
}

// Return sets up results that will be returned by AtomicRecordModifier.SetAtomic
func (mmSetAtomic *mAtomicRecordModifierMockSetAtomic) Return(err error) *AtomicRecordModifierMock {
	if mmSetAtomic.mock.funcSetAtomic != nil {
		mmSetAtomic.mock.t.Fatalf("AtomicRecordModifierMock.SetAtomic mock is already set by Set")
	}

	if mmSetAtomic.defaultExpectation == nil {
		mmSetAtomic.defaultExpectation = &AtomicRecordModifierMockSetAtomicExpectation{mock: mmSetAtomic.mock}
	}
	mmSetAtomic.defaultExpectation.results = &AtomicRecordModifierMockSetAtomicResults{err}
	return mmSetAtomic.mock
}

//Set uses given function f to mock the AtomicRecordModifier.SetAtomic method
func (mmSetAtomic *mAtomicRecordModifierMockSetAtomic) Set(f func(ctx context.Context, records ...record.Material) (err error)) *AtomicRecordModifierMock {
	if mmSetAtomic.defaultExpectation != nil {
		mmSetAtomic.mock.t.Fatalf("Default expectation is already set for the AtomicRecordModifier.SetAtomic method")
	}

	if len(mmSetAtomic.expectations) > 0 {
		mmSetAtomic.mock.t.Fatalf("Some expectations are already set for the AtomicRecordModifier.SetAtomic method")
	}

	mmSetAtomic.mock.funcSetAtomic = f
	return mmSetAtomic.mock
}

// When sets expectation for the AtomicRecordModifier.SetAtomic which will trigger the result defined by the following
// Then helper
func (mmSetAtomic *mAtomicRecordModifierMockSetAtomic) When(ctx context.Context, records ...record.Material) *AtomicRecordModifierMockSetAtomicExpectation {
	if mmSetAtomic.mock.funcSetAtomic != nil {
		mmSetAtomic.mock.t.Fatalf("AtomicRecordModifierMock.SetAtomic mock is already set by Set")
	}

	expectation := &AtomicRecordModifierMockSetAtomicExpectation{
		mock:   mmSetAtomic.mock,
		params: &AtomicRecordModifierMockSetAtomicParams{ctx, records},
	}
	mmSetAtomic.expectations = append(mmSetAtomic.expectations, expectation)
	return expectation
}

// Then sets up AtomicRecordModifier.SetAtomic return parameters for the expectation previously defined by the When method
func (e *AtomicRecordModifierMockSetAtomicExpectation) Then(err error) *AtomicRecordModifierMock {
	e.results = &AtomicRecordModifierMockSetAtomicResults{err}
	return e.mock
}

// SetAtomic implements AtomicRecordModifier
func (mmSetAtomic *AtomicRecordModifierMock) SetAtomic(ctx context.Context, records ...record.Material) (err error) {
	mm_atomic.AddUint64(&mmSetAtomic.beforeSetAtomicCounter, 1)
	defer mm_atomic.AddUint64(&mmSetAtomic.afterSetAtomicCounter, 1)

	if mmSetAtomic.inspectFuncSetAtomic != nil {
		mmSetAtomic.inspectFuncSetAtomic(ctx, records...)
	}

	mm_params := &AtomicRecordModifierMockSetAtomicParams{ctx, records}

	// Record call args
	mmSetAtomic.SetAtomicMock.mutex.Lock()
	mmSetAtomic.SetAtomicMock.callArgs = append(mmSetAtomic.SetAtomicMock.callArgs, mm_params)
	mmSetAtomic.SetAtomicMock.mutex.Unlock()

	for _, e := range mmSetAtomic.SetAtomicMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmSetAtomic.SetAtomicMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmSetAtomic.SetAtomicMock.defaultExpectation.Counter, 1)
		mm_want := mmSetAtomic.SetAtomicMock.defaultExpectation.params
		mm_got := AtomicRecordModifierMockSetAtomicParams{ctx, records}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmSetAtomic.t.Errorf("AtomicRecordModifierMock.SetAtomic got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmSetAtomic.SetAtomicMock.defaultExpectation.results
		if mm_results == nil {
			mmSetAtomic.t.Fatal("No results are set for the AtomicRecordModifierMock.SetAtomic")
		}
		return (*mm_results).err
	}
	if mmSetAtomic.funcSetAtomic != nil {
		return mmSetAtomic.funcSetAtomic(ctx, records...)
	}
	mmSetAtomic.t.Fatalf("Unexpected call to AtomicRecordModifierMock.SetAtomic. %v %v", ctx, records)
	return
}

// SetAtomicAfterCounter returns a count of finished AtomicRecordModifierMock.SetAtomic invocations
func (mmSetAtomic *AtomicRecordModifierMock) SetAtomicAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSetAtomic.afterSetAtomicCounter)
}

// SetAtomicBeforeCounter returns a count of AtomicRecordModifierMock.SetAtomic invocations
func (mmSetAtomic *AtomicRecordModifierMock) SetAtomicBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSetAtomic.beforeSetAtomicCounter)
}

// Calls returns a list of arguments used in each call to AtomicRecordModifierMock.SetAtomic.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmSetAtomic *mAtomicRecordModifierMockSetAtomic) Calls() []*AtomicRecordModifierMockSetAtomicParams {
	mmSetAtomic.mutex.RLock()

	argCopy := make([]*AtomicRecordModifierMockSetAtomicParams, len(mmSetAtomic.callArgs))
	copy(argCopy, mmSetAtomic.callArgs)

	mmSetAtomic.mutex.RUnlock()

	return argCopy
}

// MinimockSetAtomicDone returns true if the count of the SetAtomic invocations corresponds
// the number of defined expectations
func (m *AtomicRecordModifierMock) MinimockSetAtomicDone() bool {
	for _, e := range m.SetAtomicMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetAtomicMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetAtomicCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSetAtomic != nil && mm_atomic.LoadUint64(&m.afterSetAtomicCounter) < 1 {
		return false
	}
	return true
}

// MinimockSetAtomicInspect logs each unmet expectation
func (m *AtomicRecordModifierMock) MinimockSetAtomicInspect() {
	for _, e := range m.SetAtomicMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to AtomicRecordModifierMock.SetAtomic with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetAtomicMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetAtomicCounter) < 1 {
		if m.SetAtomicMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to AtomicRecordModifierMock.SetAtomic")
		} else {
			m.t.Errorf("Expected call to AtomicRecordModifierMock.SetAtomic with params: %#v", *m.SetAtomicMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSetAtomic != nil && mm_atomic.LoadUint64(&m.afterSetAtomicCounter) < 1 {
		m.t.Error("Expected call to AtomicRecordModifierMock.SetAtomic")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *AtomicRecordModifierMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockSetAtomicInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *AtomicRecordModifierMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *AtomicRecordModifierMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockSetAtomicDone()
}
