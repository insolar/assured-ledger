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

// AtomicRecordStorageMock implements AtomicRecordStorage
type AtomicRecordStorageMock struct {
	t minimock.Tester

	funcForID          func(ctx context.Context, id insolar.ID) (m1 record.Material, err error)
	inspectFuncForID   func(ctx context.Context, id insolar.ID)
	afterForIDCounter  uint64
	beforeForIDCounter uint64
	ForIDMock          mAtomicRecordStorageMockForID

	funcSetAtomic          func(ctx context.Context, records ...record.Material) (err error)
	inspectFuncSetAtomic   func(ctx context.Context, records ...record.Material)
	afterSetAtomicCounter  uint64
	beforeSetAtomicCounter uint64
	SetAtomicMock          mAtomicRecordStorageMockSetAtomic
}

// NewAtomicRecordStorageMock returns a mock for AtomicRecordStorage
func NewAtomicRecordStorageMock(t minimock.Tester) *AtomicRecordStorageMock {
	m := &AtomicRecordStorageMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.ForIDMock = mAtomicRecordStorageMockForID{mock: m}
	m.ForIDMock.callArgs = []*AtomicRecordStorageMockForIDParams{}

	m.SetAtomicMock = mAtomicRecordStorageMockSetAtomic{mock: m}
	m.SetAtomicMock.callArgs = []*AtomicRecordStorageMockSetAtomicParams{}

	return m
}

type mAtomicRecordStorageMockForID struct {
	mock               *AtomicRecordStorageMock
	defaultExpectation *AtomicRecordStorageMockForIDExpectation
	expectations       []*AtomicRecordStorageMockForIDExpectation

	callArgs []*AtomicRecordStorageMockForIDParams
	mutex    sync.RWMutex
}

// AtomicRecordStorageMockForIDExpectation specifies expectation struct of the AtomicRecordStorage.ForID
type AtomicRecordStorageMockForIDExpectation struct {
	mock    *AtomicRecordStorageMock
	params  *AtomicRecordStorageMockForIDParams
	results *AtomicRecordStorageMockForIDResults
	Counter uint64
}

// AtomicRecordStorageMockForIDParams contains parameters of the AtomicRecordStorage.ForID
type AtomicRecordStorageMockForIDParams struct {
	ctx context.Context
	id  insolar.ID
}

// AtomicRecordStorageMockForIDResults contains results of the AtomicRecordStorage.ForID
type AtomicRecordStorageMockForIDResults struct {
	m1  record.Material
	err error
}

// Expect sets up expected params for AtomicRecordStorage.ForID
func (mmForID *mAtomicRecordStorageMockForID) Expect(ctx context.Context, id insolar.ID) *mAtomicRecordStorageMockForID {
	if mmForID.mock.funcForID != nil {
		mmForID.mock.t.Fatalf("AtomicRecordStorageMock.ForID mock is already set by Set")
	}

	if mmForID.defaultExpectation == nil {
		mmForID.defaultExpectation = &AtomicRecordStorageMockForIDExpectation{}
	}

	mmForID.defaultExpectation.params = &AtomicRecordStorageMockForIDParams{ctx, id}
	for _, e := range mmForID.expectations {
		if minimock.Equal(e.params, mmForID.defaultExpectation.params) {
			mmForID.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmForID.defaultExpectation.params)
		}
	}

	return mmForID
}

// Inspect accepts an inspector function that has same arguments as the AtomicRecordStorage.ForID
func (mmForID *mAtomicRecordStorageMockForID) Inspect(f func(ctx context.Context, id insolar.ID)) *mAtomicRecordStorageMockForID {
	if mmForID.mock.inspectFuncForID != nil {
		mmForID.mock.t.Fatalf("Inspect function is already set for AtomicRecordStorageMock.ForID")
	}

	mmForID.mock.inspectFuncForID = f

	return mmForID
}

// Return sets up results that will be returned by AtomicRecordStorage.ForID
func (mmForID *mAtomicRecordStorageMockForID) Return(m1 record.Material, err error) *AtomicRecordStorageMock {
	if mmForID.mock.funcForID != nil {
		mmForID.mock.t.Fatalf("AtomicRecordStorageMock.ForID mock is already set by Set")
	}

	if mmForID.defaultExpectation == nil {
		mmForID.defaultExpectation = &AtomicRecordStorageMockForIDExpectation{mock: mmForID.mock}
	}
	mmForID.defaultExpectation.results = &AtomicRecordStorageMockForIDResults{m1, err}
	return mmForID.mock
}

//Set uses given function f to mock the AtomicRecordStorage.ForID method
func (mmForID *mAtomicRecordStorageMockForID) Set(f func(ctx context.Context, id insolar.ID) (m1 record.Material, err error)) *AtomicRecordStorageMock {
	if mmForID.defaultExpectation != nil {
		mmForID.mock.t.Fatalf("Default expectation is already set for the AtomicRecordStorage.ForID method")
	}

	if len(mmForID.expectations) > 0 {
		mmForID.mock.t.Fatalf("Some expectations are already set for the AtomicRecordStorage.ForID method")
	}

	mmForID.mock.funcForID = f
	return mmForID.mock
}

// When sets expectation for the AtomicRecordStorage.ForID which will trigger the result defined by the following
// Then helper
func (mmForID *mAtomicRecordStorageMockForID) When(ctx context.Context, id insolar.ID) *AtomicRecordStorageMockForIDExpectation {
	if mmForID.mock.funcForID != nil {
		mmForID.mock.t.Fatalf("AtomicRecordStorageMock.ForID mock is already set by Set")
	}

	expectation := &AtomicRecordStorageMockForIDExpectation{
		mock:   mmForID.mock,
		params: &AtomicRecordStorageMockForIDParams{ctx, id},
	}
	mmForID.expectations = append(mmForID.expectations, expectation)
	return expectation
}

// Then sets up AtomicRecordStorage.ForID return parameters for the expectation previously defined by the When method
func (e *AtomicRecordStorageMockForIDExpectation) Then(m1 record.Material, err error) *AtomicRecordStorageMock {
	e.results = &AtomicRecordStorageMockForIDResults{m1, err}
	return e.mock
}

// ForID implements AtomicRecordStorage
func (mmForID *AtomicRecordStorageMock) ForID(ctx context.Context, id insolar.ID) (m1 record.Material, err error) {
	mm_atomic.AddUint64(&mmForID.beforeForIDCounter, 1)
	defer mm_atomic.AddUint64(&mmForID.afterForIDCounter, 1)

	if mmForID.inspectFuncForID != nil {
		mmForID.inspectFuncForID(ctx, id)
	}

	mm_params := &AtomicRecordStorageMockForIDParams{ctx, id}

	// Record call args
	mmForID.ForIDMock.mutex.Lock()
	mmForID.ForIDMock.callArgs = append(mmForID.ForIDMock.callArgs, mm_params)
	mmForID.ForIDMock.mutex.Unlock()

	for _, e := range mmForID.ForIDMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.m1, e.results.err
		}
	}

	if mmForID.ForIDMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmForID.ForIDMock.defaultExpectation.Counter, 1)
		mm_want := mmForID.ForIDMock.defaultExpectation.params
		mm_got := AtomicRecordStorageMockForIDParams{ctx, id}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmForID.t.Errorf("AtomicRecordStorageMock.ForID got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmForID.ForIDMock.defaultExpectation.results
		if mm_results == nil {
			mmForID.t.Fatal("No results are set for the AtomicRecordStorageMock.ForID")
		}
		return (*mm_results).m1, (*mm_results).err
	}
	if mmForID.funcForID != nil {
		return mmForID.funcForID(ctx, id)
	}
	mmForID.t.Fatalf("Unexpected call to AtomicRecordStorageMock.ForID. %v %v", ctx, id)
	return
}

// ForIDAfterCounter returns a count of finished AtomicRecordStorageMock.ForID invocations
func (mmForID *AtomicRecordStorageMock) ForIDAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmForID.afterForIDCounter)
}

// ForIDBeforeCounter returns a count of AtomicRecordStorageMock.ForID invocations
func (mmForID *AtomicRecordStorageMock) ForIDBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmForID.beforeForIDCounter)
}

// Calls returns a list of arguments used in each call to AtomicRecordStorageMock.ForID.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmForID *mAtomicRecordStorageMockForID) Calls() []*AtomicRecordStorageMockForIDParams {
	mmForID.mutex.RLock()

	argCopy := make([]*AtomicRecordStorageMockForIDParams, len(mmForID.callArgs))
	copy(argCopy, mmForID.callArgs)

	mmForID.mutex.RUnlock()

	return argCopy
}

// MinimockForIDDone returns true if the count of the ForID invocations corresponds
// the number of defined expectations
func (m *AtomicRecordStorageMock) MinimockForIDDone() bool {
	for _, e := range m.ForIDMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.ForIDMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterForIDCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcForID != nil && mm_atomic.LoadUint64(&m.afterForIDCounter) < 1 {
		return false
	}
	return true
}

// MinimockForIDInspect logs each unmet expectation
func (m *AtomicRecordStorageMock) MinimockForIDInspect() {
	for _, e := range m.ForIDMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to AtomicRecordStorageMock.ForID with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.ForIDMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterForIDCounter) < 1 {
		if m.ForIDMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to AtomicRecordStorageMock.ForID")
		} else {
			m.t.Errorf("Expected call to AtomicRecordStorageMock.ForID with params: %#v", *m.ForIDMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcForID != nil && mm_atomic.LoadUint64(&m.afterForIDCounter) < 1 {
		m.t.Error("Expected call to AtomicRecordStorageMock.ForID")
	}
}

type mAtomicRecordStorageMockSetAtomic struct {
	mock               *AtomicRecordStorageMock
	defaultExpectation *AtomicRecordStorageMockSetAtomicExpectation
	expectations       []*AtomicRecordStorageMockSetAtomicExpectation

	callArgs []*AtomicRecordStorageMockSetAtomicParams
	mutex    sync.RWMutex
}

// AtomicRecordStorageMockSetAtomicExpectation specifies expectation struct of the AtomicRecordStorage.SetAtomic
type AtomicRecordStorageMockSetAtomicExpectation struct {
	mock    *AtomicRecordStorageMock
	params  *AtomicRecordStorageMockSetAtomicParams
	results *AtomicRecordStorageMockSetAtomicResults
	Counter uint64
}

// AtomicRecordStorageMockSetAtomicParams contains parameters of the AtomicRecordStorage.SetAtomic
type AtomicRecordStorageMockSetAtomicParams struct {
	ctx     context.Context
	records []record.Material
}

// AtomicRecordStorageMockSetAtomicResults contains results of the AtomicRecordStorage.SetAtomic
type AtomicRecordStorageMockSetAtomicResults struct {
	err error
}

// Expect sets up expected params for AtomicRecordStorage.SetAtomic
func (mmSetAtomic *mAtomicRecordStorageMockSetAtomic) Expect(ctx context.Context, records ...record.Material) *mAtomicRecordStorageMockSetAtomic {
	if mmSetAtomic.mock.funcSetAtomic != nil {
		mmSetAtomic.mock.t.Fatalf("AtomicRecordStorageMock.SetAtomic mock is already set by Set")
	}

	if mmSetAtomic.defaultExpectation == nil {
		mmSetAtomic.defaultExpectation = &AtomicRecordStorageMockSetAtomicExpectation{}
	}

	mmSetAtomic.defaultExpectation.params = &AtomicRecordStorageMockSetAtomicParams{ctx, records}
	for _, e := range mmSetAtomic.expectations {
		if minimock.Equal(e.params, mmSetAtomic.defaultExpectation.params) {
			mmSetAtomic.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmSetAtomic.defaultExpectation.params)
		}
	}

	return mmSetAtomic
}

// Inspect accepts an inspector function that has same arguments as the AtomicRecordStorage.SetAtomic
func (mmSetAtomic *mAtomicRecordStorageMockSetAtomic) Inspect(f func(ctx context.Context, records ...record.Material)) *mAtomicRecordStorageMockSetAtomic {
	if mmSetAtomic.mock.inspectFuncSetAtomic != nil {
		mmSetAtomic.mock.t.Fatalf("Inspect function is already set for AtomicRecordStorageMock.SetAtomic")
	}

	mmSetAtomic.mock.inspectFuncSetAtomic = f

	return mmSetAtomic
}

// Return sets up results that will be returned by AtomicRecordStorage.SetAtomic
func (mmSetAtomic *mAtomicRecordStorageMockSetAtomic) Return(err error) *AtomicRecordStorageMock {
	if mmSetAtomic.mock.funcSetAtomic != nil {
		mmSetAtomic.mock.t.Fatalf("AtomicRecordStorageMock.SetAtomic mock is already set by Set")
	}

	if mmSetAtomic.defaultExpectation == nil {
		mmSetAtomic.defaultExpectation = &AtomicRecordStorageMockSetAtomicExpectation{mock: mmSetAtomic.mock}
	}
	mmSetAtomic.defaultExpectation.results = &AtomicRecordStorageMockSetAtomicResults{err}
	return mmSetAtomic.mock
}

//Set uses given function f to mock the AtomicRecordStorage.SetAtomic method
func (mmSetAtomic *mAtomicRecordStorageMockSetAtomic) Set(f func(ctx context.Context, records ...record.Material) (err error)) *AtomicRecordStorageMock {
	if mmSetAtomic.defaultExpectation != nil {
		mmSetAtomic.mock.t.Fatalf("Default expectation is already set for the AtomicRecordStorage.SetAtomic method")
	}

	if len(mmSetAtomic.expectations) > 0 {
		mmSetAtomic.mock.t.Fatalf("Some expectations are already set for the AtomicRecordStorage.SetAtomic method")
	}

	mmSetAtomic.mock.funcSetAtomic = f
	return mmSetAtomic.mock
}

// When sets expectation for the AtomicRecordStorage.SetAtomic which will trigger the result defined by the following
// Then helper
func (mmSetAtomic *mAtomicRecordStorageMockSetAtomic) When(ctx context.Context, records ...record.Material) *AtomicRecordStorageMockSetAtomicExpectation {
	if mmSetAtomic.mock.funcSetAtomic != nil {
		mmSetAtomic.mock.t.Fatalf("AtomicRecordStorageMock.SetAtomic mock is already set by Set")
	}

	expectation := &AtomicRecordStorageMockSetAtomicExpectation{
		mock:   mmSetAtomic.mock,
		params: &AtomicRecordStorageMockSetAtomicParams{ctx, records},
	}
	mmSetAtomic.expectations = append(mmSetAtomic.expectations, expectation)
	return expectation
}

// Then sets up AtomicRecordStorage.SetAtomic return parameters for the expectation previously defined by the When method
func (e *AtomicRecordStorageMockSetAtomicExpectation) Then(err error) *AtomicRecordStorageMock {
	e.results = &AtomicRecordStorageMockSetAtomicResults{err}
	return e.mock
}

// SetAtomic implements AtomicRecordStorage
func (mmSetAtomic *AtomicRecordStorageMock) SetAtomic(ctx context.Context, records ...record.Material) (err error) {
	mm_atomic.AddUint64(&mmSetAtomic.beforeSetAtomicCounter, 1)
	defer mm_atomic.AddUint64(&mmSetAtomic.afterSetAtomicCounter, 1)

	if mmSetAtomic.inspectFuncSetAtomic != nil {
		mmSetAtomic.inspectFuncSetAtomic(ctx, records...)
	}

	mm_params := &AtomicRecordStorageMockSetAtomicParams{ctx, records}

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
		mm_got := AtomicRecordStorageMockSetAtomicParams{ctx, records}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmSetAtomic.t.Errorf("AtomicRecordStorageMock.SetAtomic got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmSetAtomic.SetAtomicMock.defaultExpectation.results
		if mm_results == nil {
			mmSetAtomic.t.Fatal("No results are set for the AtomicRecordStorageMock.SetAtomic")
		}
		return (*mm_results).err
	}
	if mmSetAtomic.funcSetAtomic != nil {
		return mmSetAtomic.funcSetAtomic(ctx, records...)
	}
	mmSetAtomic.t.Fatalf("Unexpected call to AtomicRecordStorageMock.SetAtomic. %v %v", ctx, records)
	return
}

// SetAtomicAfterCounter returns a count of finished AtomicRecordStorageMock.SetAtomic invocations
func (mmSetAtomic *AtomicRecordStorageMock) SetAtomicAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSetAtomic.afterSetAtomicCounter)
}

// SetAtomicBeforeCounter returns a count of AtomicRecordStorageMock.SetAtomic invocations
func (mmSetAtomic *AtomicRecordStorageMock) SetAtomicBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSetAtomic.beforeSetAtomicCounter)
}

// Calls returns a list of arguments used in each call to AtomicRecordStorageMock.SetAtomic.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmSetAtomic *mAtomicRecordStorageMockSetAtomic) Calls() []*AtomicRecordStorageMockSetAtomicParams {
	mmSetAtomic.mutex.RLock()

	argCopy := make([]*AtomicRecordStorageMockSetAtomicParams, len(mmSetAtomic.callArgs))
	copy(argCopy, mmSetAtomic.callArgs)

	mmSetAtomic.mutex.RUnlock()

	return argCopy
}

// MinimockSetAtomicDone returns true if the count of the SetAtomic invocations corresponds
// the number of defined expectations
func (m *AtomicRecordStorageMock) MinimockSetAtomicDone() bool {
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
func (m *AtomicRecordStorageMock) MinimockSetAtomicInspect() {
	for _, e := range m.SetAtomicMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to AtomicRecordStorageMock.SetAtomic with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetAtomicMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetAtomicCounter) < 1 {
		if m.SetAtomicMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to AtomicRecordStorageMock.SetAtomic")
		} else {
			m.t.Errorf("Expected call to AtomicRecordStorageMock.SetAtomic with params: %#v", *m.SetAtomicMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSetAtomic != nil && mm_atomic.LoadUint64(&m.afterSetAtomicCounter) < 1 {
		m.t.Error("Expected call to AtomicRecordStorageMock.SetAtomic")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *AtomicRecordStorageMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockForIDInspect()

		m.MinimockSetAtomicInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *AtomicRecordStorageMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *AtomicRecordStorageMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockForIDDone() &&
		m.MinimockSetAtomicDone()
}
