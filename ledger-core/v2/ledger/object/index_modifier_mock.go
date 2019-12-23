package object

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
)

// IndexModifierMock implements IndexModifier
type IndexModifierMock struct {
	t minimock.Tester

	funcSetIndex          func(ctx context.Context, pn insolar.PulseNumber, index record.Index) (err error)
	inspectFuncSetIndex   func(ctx context.Context, pn insolar.PulseNumber, index record.Index)
	afterSetIndexCounter  uint64
	beforeSetIndexCounter uint64
	SetIndexMock          mIndexModifierMockSetIndex

	funcUpdateLastKnownPulse          func(ctx context.Context, pn insolar.PulseNumber) (err error)
	inspectFuncUpdateLastKnownPulse   func(ctx context.Context, pn insolar.PulseNumber)
	afterUpdateLastKnownPulseCounter  uint64
	beforeUpdateLastKnownPulseCounter uint64
	UpdateLastKnownPulseMock          mIndexModifierMockUpdateLastKnownPulse
}

// NewIndexModifierMock returns a mock for IndexModifier
func NewIndexModifierMock(t minimock.Tester) *IndexModifierMock {
	m := &IndexModifierMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.SetIndexMock = mIndexModifierMockSetIndex{mock: m}
	m.SetIndexMock.callArgs = []*IndexModifierMockSetIndexParams{}

	m.UpdateLastKnownPulseMock = mIndexModifierMockUpdateLastKnownPulse{mock: m}
	m.UpdateLastKnownPulseMock.callArgs = []*IndexModifierMockUpdateLastKnownPulseParams{}

	return m
}

type mIndexModifierMockSetIndex struct {
	mock               *IndexModifierMock
	defaultExpectation *IndexModifierMockSetIndexExpectation
	expectations       []*IndexModifierMockSetIndexExpectation

	callArgs []*IndexModifierMockSetIndexParams
	mutex    sync.RWMutex
}

// IndexModifierMockSetIndexExpectation specifies expectation struct of the IndexModifier.SetIndex
type IndexModifierMockSetIndexExpectation struct {
	mock    *IndexModifierMock
	params  *IndexModifierMockSetIndexParams
	results *IndexModifierMockSetIndexResults
	Counter uint64
}

// IndexModifierMockSetIndexParams contains parameters of the IndexModifier.SetIndex
type IndexModifierMockSetIndexParams struct {
	ctx   context.Context
	pn    insolar.PulseNumber
	index record.Index
}

// IndexModifierMockSetIndexResults contains results of the IndexModifier.SetIndex
type IndexModifierMockSetIndexResults struct {
	err error
}

// Expect sets up expected params for IndexModifier.SetIndex
func (mmSetIndex *mIndexModifierMockSetIndex) Expect(ctx context.Context, pn insolar.PulseNumber, index record.Index) *mIndexModifierMockSetIndex {
	if mmSetIndex.mock.funcSetIndex != nil {
		mmSetIndex.mock.t.Fatalf("IndexModifierMock.SetIndex mock is already set by Set")
	}

	if mmSetIndex.defaultExpectation == nil {
		mmSetIndex.defaultExpectation = &IndexModifierMockSetIndexExpectation{}
	}

	mmSetIndex.defaultExpectation.params = &IndexModifierMockSetIndexParams{ctx, pn, index}
	for _, e := range mmSetIndex.expectations {
		if minimock.Equal(e.params, mmSetIndex.defaultExpectation.params) {
			mmSetIndex.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmSetIndex.defaultExpectation.params)
		}
	}

	return mmSetIndex
}

// Inspect accepts an inspector function that has same arguments as the IndexModifier.SetIndex
func (mmSetIndex *mIndexModifierMockSetIndex) Inspect(f func(ctx context.Context, pn insolar.PulseNumber, index record.Index)) *mIndexModifierMockSetIndex {
	if mmSetIndex.mock.inspectFuncSetIndex != nil {
		mmSetIndex.mock.t.Fatalf("Inspect function is already set for IndexModifierMock.SetIndex")
	}

	mmSetIndex.mock.inspectFuncSetIndex = f

	return mmSetIndex
}

// Return sets up results that will be returned by IndexModifier.SetIndex
func (mmSetIndex *mIndexModifierMockSetIndex) Return(err error) *IndexModifierMock {
	if mmSetIndex.mock.funcSetIndex != nil {
		mmSetIndex.mock.t.Fatalf("IndexModifierMock.SetIndex mock is already set by Set")
	}

	if mmSetIndex.defaultExpectation == nil {
		mmSetIndex.defaultExpectation = &IndexModifierMockSetIndexExpectation{mock: mmSetIndex.mock}
	}
	mmSetIndex.defaultExpectation.results = &IndexModifierMockSetIndexResults{err}
	return mmSetIndex.mock
}

//Set uses given function f to mock the IndexModifier.SetIndex method
func (mmSetIndex *mIndexModifierMockSetIndex) Set(f func(ctx context.Context, pn insolar.PulseNumber, index record.Index) (err error)) *IndexModifierMock {
	if mmSetIndex.defaultExpectation != nil {
		mmSetIndex.mock.t.Fatalf("Default expectation is already set for the IndexModifier.SetIndex method")
	}

	if len(mmSetIndex.expectations) > 0 {
		mmSetIndex.mock.t.Fatalf("Some expectations are already set for the IndexModifier.SetIndex method")
	}

	mmSetIndex.mock.funcSetIndex = f
	return mmSetIndex.mock
}

// When sets expectation for the IndexModifier.SetIndex which will trigger the result defined by the following
// Then helper
func (mmSetIndex *mIndexModifierMockSetIndex) When(ctx context.Context, pn insolar.PulseNumber, index record.Index) *IndexModifierMockSetIndexExpectation {
	if mmSetIndex.mock.funcSetIndex != nil {
		mmSetIndex.mock.t.Fatalf("IndexModifierMock.SetIndex mock is already set by Set")
	}

	expectation := &IndexModifierMockSetIndexExpectation{
		mock:   mmSetIndex.mock,
		params: &IndexModifierMockSetIndexParams{ctx, pn, index},
	}
	mmSetIndex.expectations = append(mmSetIndex.expectations, expectation)
	return expectation
}

// Then sets up IndexModifier.SetIndex return parameters for the expectation previously defined by the When method
func (e *IndexModifierMockSetIndexExpectation) Then(err error) *IndexModifierMock {
	e.results = &IndexModifierMockSetIndexResults{err}
	return e.mock
}

// SetIndex implements IndexModifier
func (mmSetIndex *IndexModifierMock) SetIndex(ctx context.Context, pn insolar.PulseNumber, index record.Index) (err error) {
	mm_atomic.AddUint64(&mmSetIndex.beforeSetIndexCounter, 1)
	defer mm_atomic.AddUint64(&mmSetIndex.afterSetIndexCounter, 1)

	if mmSetIndex.inspectFuncSetIndex != nil {
		mmSetIndex.inspectFuncSetIndex(ctx, pn, index)
	}

	mm_params := &IndexModifierMockSetIndexParams{ctx, pn, index}

	// Record call args
	mmSetIndex.SetIndexMock.mutex.Lock()
	mmSetIndex.SetIndexMock.callArgs = append(mmSetIndex.SetIndexMock.callArgs, mm_params)
	mmSetIndex.SetIndexMock.mutex.Unlock()

	for _, e := range mmSetIndex.SetIndexMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmSetIndex.SetIndexMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmSetIndex.SetIndexMock.defaultExpectation.Counter, 1)
		mm_want := mmSetIndex.SetIndexMock.defaultExpectation.params
		mm_got := IndexModifierMockSetIndexParams{ctx, pn, index}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmSetIndex.t.Errorf("IndexModifierMock.SetIndex got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmSetIndex.SetIndexMock.defaultExpectation.results
		if mm_results == nil {
			mmSetIndex.t.Fatal("No results are set for the IndexModifierMock.SetIndex")
		}
		return (*mm_results).err
	}
	if mmSetIndex.funcSetIndex != nil {
		return mmSetIndex.funcSetIndex(ctx, pn, index)
	}
	mmSetIndex.t.Fatalf("Unexpected call to IndexModifierMock.SetIndex. %v %v %v", ctx, pn, index)
	return
}

// SetIndexAfterCounter returns a count of finished IndexModifierMock.SetIndex invocations
func (mmSetIndex *IndexModifierMock) SetIndexAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSetIndex.afterSetIndexCounter)
}

// SetIndexBeforeCounter returns a count of IndexModifierMock.SetIndex invocations
func (mmSetIndex *IndexModifierMock) SetIndexBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSetIndex.beforeSetIndexCounter)
}

// Calls returns a list of arguments used in each call to IndexModifierMock.SetIndex.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmSetIndex *mIndexModifierMockSetIndex) Calls() []*IndexModifierMockSetIndexParams {
	mmSetIndex.mutex.RLock()

	argCopy := make([]*IndexModifierMockSetIndexParams, len(mmSetIndex.callArgs))
	copy(argCopy, mmSetIndex.callArgs)

	mmSetIndex.mutex.RUnlock()

	return argCopy
}

// MinimockSetIndexDone returns true if the count of the SetIndex invocations corresponds
// the number of defined expectations
func (m *IndexModifierMock) MinimockSetIndexDone() bool {
	for _, e := range m.SetIndexMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetIndexMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetIndexCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSetIndex != nil && mm_atomic.LoadUint64(&m.afterSetIndexCounter) < 1 {
		return false
	}
	return true
}

// MinimockSetIndexInspect logs each unmet expectation
func (m *IndexModifierMock) MinimockSetIndexInspect() {
	for _, e := range m.SetIndexMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to IndexModifierMock.SetIndex with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetIndexMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetIndexCounter) < 1 {
		if m.SetIndexMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to IndexModifierMock.SetIndex")
		} else {
			m.t.Errorf("Expected call to IndexModifierMock.SetIndex with params: %#v", *m.SetIndexMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSetIndex != nil && mm_atomic.LoadUint64(&m.afterSetIndexCounter) < 1 {
		m.t.Error("Expected call to IndexModifierMock.SetIndex")
	}
}

type mIndexModifierMockUpdateLastKnownPulse struct {
	mock               *IndexModifierMock
	defaultExpectation *IndexModifierMockUpdateLastKnownPulseExpectation
	expectations       []*IndexModifierMockUpdateLastKnownPulseExpectation

	callArgs []*IndexModifierMockUpdateLastKnownPulseParams
	mutex    sync.RWMutex
}

// IndexModifierMockUpdateLastKnownPulseExpectation specifies expectation struct of the IndexModifier.UpdateLastKnownPulse
type IndexModifierMockUpdateLastKnownPulseExpectation struct {
	mock    *IndexModifierMock
	params  *IndexModifierMockUpdateLastKnownPulseParams
	results *IndexModifierMockUpdateLastKnownPulseResults
	Counter uint64
}

// IndexModifierMockUpdateLastKnownPulseParams contains parameters of the IndexModifier.UpdateLastKnownPulse
type IndexModifierMockUpdateLastKnownPulseParams struct {
	ctx context.Context
	pn  insolar.PulseNumber
}

// IndexModifierMockUpdateLastKnownPulseResults contains results of the IndexModifier.UpdateLastKnownPulse
type IndexModifierMockUpdateLastKnownPulseResults struct {
	err error
}

// Expect sets up expected params for IndexModifier.UpdateLastKnownPulse
func (mmUpdateLastKnownPulse *mIndexModifierMockUpdateLastKnownPulse) Expect(ctx context.Context, pn insolar.PulseNumber) *mIndexModifierMockUpdateLastKnownPulse {
	if mmUpdateLastKnownPulse.mock.funcUpdateLastKnownPulse != nil {
		mmUpdateLastKnownPulse.mock.t.Fatalf("IndexModifierMock.UpdateLastKnownPulse mock is already set by Set")
	}

	if mmUpdateLastKnownPulse.defaultExpectation == nil {
		mmUpdateLastKnownPulse.defaultExpectation = &IndexModifierMockUpdateLastKnownPulseExpectation{}
	}

	mmUpdateLastKnownPulse.defaultExpectation.params = &IndexModifierMockUpdateLastKnownPulseParams{ctx, pn}
	for _, e := range mmUpdateLastKnownPulse.expectations {
		if minimock.Equal(e.params, mmUpdateLastKnownPulse.defaultExpectation.params) {
			mmUpdateLastKnownPulse.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmUpdateLastKnownPulse.defaultExpectation.params)
		}
	}

	return mmUpdateLastKnownPulse
}

// Inspect accepts an inspector function that has same arguments as the IndexModifier.UpdateLastKnownPulse
func (mmUpdateLastKnownPulse *mIndexModifierMockUpdateLastKnownPulse) Inspect(f func(ctx context.Context, pn insolar.PulseNumber)) *mIndexModifierMockUpdateLastKnownPulse {
	if mmUpdateLastKnownPulse.mock.inspectFuncUpdateLastKnownPulse != nil {
		mmUpdateLastKnownPulse.mock.t.Fatalf("Inspect function is already set for IndexModifierMock.UpdateLastKnownPulse")
	}

	mmUpdateLastKnownPulse.mock.inspectFuncUpdateLastKnownPulse = f

	return mmUpdateLastKnownPulse
}

// Return sets up results that will be returned by IndexModifier.UpdateLastKnownPulse
func (mmUpdateLastKnownPulse *mIndexModifierMockUpdateLastKnownPulse) Return(err error) *IndexModifierMock {
	if mmUpdateLastKnownPulse.mock.funcUpdateLastKnownPulse != nil {
		mmUpdateLastKnownPulse.mock.t.Fatalf("IndexModifierMock.UpdateLastKnownPulse mock is already set by Set")
	}

	if mmUpdateLastKnownPulse.defaultExpectation == nil {
		mmUpdateLastKnownPulse.defaultExpectation = &IndexModifierMockUpdateLastKnownPulseExpectation{mock: mmUpdateLastKnownPulse.mock}
	}
	mmUpdateLastKnownPulse.defaultExpectation.results = &IndexModifierMockUpdateLastKnownPulseResults{err}
	return mmUpdateLastKnownPulse.mock
}

//Set uses given function f to mock the IndexModifier.UpdateLastKnownPulse method
func (mmUpdateLastKnownPulse *mIndexModifierMockUpdateLastKnownPulse) Set(f func(ctx context.Context, pn insolar.PulseNumber) (err error)) *IndexModifierMock {
	if mmUpdateLastKnownPulse.defaultExpectation != nil {
		mmUpdateLastKnownPulse.mock.t.Fatalf("Default expectation is already set for the IndexModifier.UpdateLastKnownPulse method")
	}

	if len(mmUpdateLastKnownPulse.expectations) > 0 {
		mmUpdateLastKnownPulse.mock.t.Fatalf("Some expectations are already set for the IndexModifier.UpdateLastKnownPulse method")
	}

	mmUpdateLastKnownPulse.mock.funcUpdateLastKnownPulse = f
	return mmUpdateLastKnownPulse.mock
}

// When sets expectation for the IndexModifier.UpdateLastKnownPulse which will trigger the result defined by the following
// Then helper
func (mmUpdateLastKnownPulse *mIndexModifierMockUpdateLastKnownPulse) When(ctx context.Context, pn insolar.PulseNumber) *IndexModifierMockUpdateLastKnownPulseExpectation {
	if mmUpdateLastKnownPulse.mock.funcUpdateLastKnownPulse != nil {
		mmUpdateLastKnownPulse.mock.t.Fatalf("IndexModifierMock.UpdateLastKnownPulse mock is already set by Set")
	}

	expectation := &IndexModifierMockUpdateLastKnownPulseExpectation{
		mock:   mmUpdateLastKnownPulse.mock,
		params: &IndexModifierMockUpdateLastKnownPulseParams{ctx, pn},
	}
	mmUpdateLastKnownPulse.expectations = append(mmUpdateLastKnownPulse.expectations, expectation)
	return expectation
}

// Then sets up IndexModifier.UpdateLastKnownPulse return parameters for the expectation previously defined by the When method
func (e *IndexModifierMockUpdateLastKnownPulseExpectation) Then(err error) *IndexModifierMock {
	e.results = &IndexModifierMockUpdateLastKnownPulseResults{err}
	return e.mock
}

// UpdateLastKnownPulse implements IndexModifier
func (mmUpdateLastKnownPulse *IndexModifierMock) UpdateLastKnownPulse(ctx context.Context, pn insolar.PulseNumber) (err error) {
	mm_atomic.AddUint64(&mmUpdateLastKnownPulse.beforeUpdateLastKnownPulseCounter, 1)
	defer mm_atomic.AddUint64(&mmUpdateLastKnownPulse.afterUpdateLastKnownPulseCounter, 1)

	if mmUpdateLastKnownPulse.inspectFuncUpdateLastKnownPulse != nil {
		mmUpdateLastKnownPulse.inspectFuncUpdateLastKnownPulse(ctx, pn)
	}

	mm_params := &IndexModifierMockUpdateLastKnownPulseParams{ctx, pn}

	// Record call args
	mmUpdateLastKnownPulse.UpdateLastKnownPulseMock.mutex.Lock()
	mmUpdateLastKnownPulse.UpdateLastKnownPulseMock.callArgs = append(mmUpdateLastKnownPulse.UpdateLastKnownPulseMock.callArgs, mm_params)
	mmUpdateLastKnownPulse.UpdateLastKnownPulseMock.mutex.Unlock()

	for _, e := range mmUpdateLastKnownPulse.UpdateLastKnownPulseMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmUpdateLastKnownPulse.UpdateLastKnownPulseMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmUpdateLastKnownPulse.UpdateLastKnownPulseMock.defaultExpectation.Counter, 1)
		mm_want := mmUpdateLastKnownPulse.UpdateLastKnownPulseMock.defaultExpectation.params
		mm_got := IndexModifierMockUpdateLastKnownPulseParams{ctx, pn}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmUpdateLastKnownPulse.t.Errorf("IndexModifierMock.UpdateLastKnownPulse got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmUpdateLastKnownPulse.UpdateLastKnownPulseMock.defaultExpectation.results
		if mm_results == nil {
			mmUpdateLastKnownPulse.t.Fatal("No results are set for the IndexModifierMock.UpdateLastKnownPulse")
		}
		return (*mm_results).err
	}
	if mmUpdateLastKnownPulse.funcUpdateLastKnownPulse != nil {
		return mmUpdateLastKnownPulse.funcUpdateLastKnownPulse(ctx, pn)
	}
	mmUpdateLastKnownPulse.t.Fatalf("Unexpected call to IndexModifierMock.UpdateLastKnownPulse. %v %v", ctx, pn)
	return
}

// UpdateLastKnownPulseAfterCounter returns a count of finished IndexModifierMock.UpdateLastKnownPulse invocations
func (mmUpdateLastKnownPulse *IndexModifierMock) UpdateLastKnownPulseAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmUpdateLastKnownPulse.afterUpdateLastKnownPulseCounter)
}

// UpdateLastKnownPulseBeforeCounter returns a count of IndexModifierMock.UpdateLastKnownPulse invocations
func (mmUpdateLastKnownPulse *IndexModifierMock) UpdateLastKnownPulseBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmUpdateLastKnownPulse.beforeUpdateLastKnownPulseCounter)
}

// Calls returns a list of arguments used in each call to IndexModifierMock.UpdateLastKnownPulse.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmUpdateLastKnownPulse *mIndexModifierMockUpdateLastKnownPulse) Calls() []*IndexModifierMockUpdateLastKnownPulseParams {
	mmUpdateLastKnownPulse.mutex.RLock()

	argCopy := make([]*IndexModifierMockUpdateLastKnownPulseParams, len(mmUpdateLastKnownPulse.callArgs))
	copy(argCopy, mmUpdateLastKnownPulse.callArgs)

	mmUpdateLastKnownPulse.mutex.RUnlock()

	return argCopy
}

// MinimockUpdateLastKnownPulseDone returns true if the count of the UpdateLastKnownPulse invocations corresponds
// the number of defined expectations
func (m *IndexModifierMock) MinimockUpdateLastKnownPulseDone() bool {
	for _, e := range m.UpdateLastKnownPulseMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.UpdateLastKnownPulseMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterUpdateLastKnownPulseCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcUpdateLastKnownPulse != nil && mm_atomic.LoadUint64(&m.afterUpdateLastKnownPulseCounter) < 1 {
		return false
	}
	return true
}

// MinimockUpdateLastKnownPulseInspect logs each unmet expectation
func (m *IndexModifierMock) MinimockUpdateLastKnownPulseInspect() {
	for _, e := range m.UpdateLastKnownPulseMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to IndexModifierMock.UpdateLastKnownPulse with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.UpdateLastKnownPulseMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterUpdateLastKnownPulseCounter) < 1 {
		if m.UpdateLastKnownPulseMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to IndexModifierMock.UpdateLastKnownPulse")
		} else {
			m.t.Errorf("Expected call to IndexModifierMock.UpdateLastKnownPulse with params: %#v", *m.UpdateLastKnownPulseMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcUpdateLastKnownPulse != nil && mm_atomic.LoadUint64(&m.afterUpdateLastKnownPulseCounter) < 1 {
		m.t.Error("Expected call to IndexModifierMock.UpdateLastKnownPulse")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *IndexModifierMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockSetIndexInspect()

		m.MinimockUpdateLastKnownPulseInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *IndexModifierMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *IndexModifierMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockSetIndexDone() &&
		m.MinimockUpdateLastKnownPulseDone()
}
