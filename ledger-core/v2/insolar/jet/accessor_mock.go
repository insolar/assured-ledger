package jet

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

// AccessorMock implements Accessor
type AccessorMock struct {
	t minimock.Tester

	funcAll          func(ctx context.Context, pulse insolar.PulseNumber) (ja1 []insolar.JetID)
	inspectFuncAll   func(ctx context.Context, pulse insolar.PulseNumber)
	afterAllCounter  uint64
	beforeAllCounter uint64
	AllMock          mAccessorMockAll

	funcForID          func(ctx context.Context, pulse insolar.PulseNumber, recordID insolar.ID) (j1 insolar.JetID, b1 bool)
	inspectFuncForID   func(ctx context.Context, pulse insolar.PulseNumber, recordID insolar.ID)
	afterForIDCounter  uint64
	beforeForIDCounter uint64
	ForIDMock          mAccessorMockForID
}

// NewAccessorMock returns a mock for Accessor
func NewAccessorMock(t minimock.Tester) *AccessorMock {
	m := &AccessorMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.AllMock = mAccessorMockAll{mock: m}
	m.AllMock.callArgs = []*AccessorMockAllParams{}

	m.ForIDMock = mAccessorMockForID{mock: m}
	m.ForIDMock.callArgs = []*AccessorMockForIDParams{}

	return m
}

type mAccessorMockAll struct {
	mock               *AccessorMock
	defaultExpectation *AccessorMockAllExpectation
	expectations       []*AccessorMockAllExpectation

	callArgs []*AccessorMockAllParams
	mutex    sync.RWMutex
}

// AccessorMockAllExpectation specifies expectation struct of the Accessor.All
type AccessorMockAllExpectation struct {
	mock    *AccessorMock
	params  *AccessorMockAllParams
	results *AccessorMockAllResults
	Counter uint64
}

// AccessorMockAllParams contains parameters of the Accessor.All
type AccessorMockAllParams struct {
	ctx   context.Context
	pulse insolar.PulseNumber
}

// AccessorMockAllResults contains results of the Accessor.All
type AccessorMockAllResults struct {
	ja1 []insolar.JetID
}

// Expect sets up expected params for Accessor.All
func (mmAll *mAccessorMockAll) Expect(ctx context.Context, pulse insolar.PulseNumber) *mAccessorMockAll {
	if mmAll.mock.funcAll != nil {
		mmAll.mock.t.Fatalf("AccessorMock.All mock is already set by Set")
	}

	if mmAll.defaultExpectation == nil {
		mmAll.defaultExpectation = &AccessorMockAllExpectation{}
	}

	mmAll.defaultExpectation.params = &AccessorMockAllParams{ctx, pulse}
	for _, e := range mmAll.expectations {
		if minimock.Equal(e.params, mmAll.defaultExpectation.params) {
			mmAll.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmAll.defaultExpectation.params)
		}
	}

	return mmAll
}

// Inspect accepts an inspector function that has same arguments as the Accessor.All
func (mmAll *mAccessorMockAll) Inspect(f func(ctx context.Context, pulse insolar.PulseNumber)) *mAccessorMockAll {
	if mmAll.mock.inspectFuncAll != nil {
		mmAll.mock.t.Fatalf("Inspect function is already set for AccessorMock.All")
	}

	mmAll.mock.inspectFuncAll = f

	return mmAll
}

// Return sets up results that will be returned by Accessor.All
func (mmAll *mAccessorMockAll) Return(ja1 []insolar.JetID) *AccessorMock {
	if mmAll.mock.funcAll != nil {
		mmAll.mock.t.Fatalf("AccessorMock.All mock is already set by Set")
	}

	if mmAll.defaultExpectation == nil {
		mmAll.defaultExpectation = &AccessorMockAllExpectation{mock: mmAll.mock}
	}
	mmAll.defaultExpectation.results = &AccessorMockAllResults{ja1}
	return mmAll.mock
}

//Set uses given function f to mock the Accessor.All method
func (mmAll *mAccessorMockAll) Set(f func(ctx context.Context, pulse insolar.PulseNumber) (ja1 []insolar.JetID)) *AccessorMock {
	if mmAll.defaultExpectation != nil {
		mmAll.mock.t.Fatalf("Default expectation is already set for the Accessor.All method")
	}

	if len(mmAll.expectations) > 0 {
		mmAll.mock.t.Fatalf("Some expectations are already set for the Accessor.All method")
	}

	mmAll.mock.funcAll = f
	return mmAll.mock
}

// When sets expectation for the Accessor.All which will trigger the result defined by the following
// Then helper
func (mmAll *mAccessorMockAll) When(ctx context.Context, pulse insolar.PulseNumber) *AccessorMockAllExpectation {
	if mmAll.mock.funcAll != nil {
		mmAll.mock.t.Fatalf("AccessorMock.All mock is already set by Set")
	}

	expectation := &AccessorMockAllExpectation{
		mock:   mmAll.mock,
		params: &AccessorMockAllParams{ctx, pulse},
	}
	mmAll.expectations = append(mmAll.expectations, expectation)
	return expectation
}

// Then sets up Accessor.All return parameters for the expectation previously defined by the When method
func (e *AccessorMockAllExpectation) Then(ja1 []insolar.JetID) *AccessorMock {
	e.results = &AccessorMockAllResults{ja1}
	return e.mock
}

// All implements Accessor
func (mmAll *AccessorMock) All(ctx context.Context, pulse insolar.PulseNumber) (ja1 []insolar.JetID) {
	mm_atomic.AddUint64(&mmAll.beforeAllCounter, 1)
	defer mm_atomic.AddUint64(&mmAll.afterAllCounter, 1)

	if mmAll.inspectFuncAll != nil {
		mmAll.inspectFuncAll(ctx, pulse)
	}

	mm_params := &AccessorMockAllParams{ctx, pulse}

	// Record call args
	mmAll.AllMock.mutex.Lock()
	mmAll.AllMock.callArgs = append(mmAll.AllMock.callArgs, mm_params)
	mmAll.AllMock.mutex.Unlock()

	for _, e := range mmAll.AllMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.ja1
		}
	}

	if mmAll.AllMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmAll.AllMock.defaultExpectation.Counter, 1)
		mm_want := mmAll.AllMock.defaultExpectation.params
		mm_got := AccessorMockAllParams{ctx, pulse}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmAll.t.Errorf("AccessorMock.All got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmAll.AllMock.defaultExpectation.results
		if mm_results == nil {
			mmAll.t.Fatal("No results are set for the AccessorMock.All")
		}
		return (*mm_results).ja1
	}
	if mmAll.funcAll != nil {
		return mmAll.funcAll(ctx, pulse)
	}
	mmAll.t.Fatalf("Unexpected call to AccessorMock.All. %v %v", ctx, pulse)
	return
}

// AllAfterCounter returns a count of finished AccessorMock.All invocations
func (mmAll *AccessorMock) AllAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmAll.afterAllCounter)
}

// AllBeforeCounter returns a count of AccessorMock.All invocations
func (mmAll *AccessorMock) AllBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmAll.beforeAllCounter)
}

// Calls returns a list of arguments used in each call to AccessorMock.All.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmAll *mAccessorMockAll) Calls() []*AccessorMockAllParams {
	mmAll.mutex.RLock()

	argCopy := make([]*AccessorMockAllParams, len(mmAll.callArgs))
	copy(argCopy, mmAll.callArgs)

	mmAll.mutex.RUnlock()

	return argCopy
}

// MinimockAllDone returns true if the count of the All invocations corresponds
// the number of defined expectations
func (m *AccessorMock) MinimockAllDone() bool {
	for _, e := range m.AllMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.AllMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterAllCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcAll != nil && mm_atomic.LoadUint64(&m.afterAllCounter) < 1 {
		return false
	}
	return true
}

// MinimockAllInspect logs each unmet expectation
func (m *AccessorMock) MinimockAllInspect() {
	for _, e := range m.AllMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to AccessorMock.All with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.AllMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterAllCounter) < 1 {
		if m.AllMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to AccessorMock.All")
		} else {
			m.t.Errorf("Expected call to AccessorMock.All with params: %#v", *m.AllMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcAll != nil && mm_atomic.LoadUint64(&m.afterAllCounter) < 1 {
		m.t.Error("Expected call to AccessorMock.All")
	}
}

type mAccessorMockForID struct {
	mock               *AccessorMock
	defaultExpectation *AccessorMockForIDExpectation
	expectations       []*AccessorMockForIDExpectation

	callArgs []*AccessorMockForIDParams
	mutex    sync.RWMutex
}

// AccessorMockForIDExpectation specifies expectation struct of the Accessor.ForID
type AccessorMockForIDExpectation struct {
	mock    *AccessorMock
	params  *AccessorMockForIDParams
	results *AccessorMockForIDResults
	Counter uint64
}

// AccessorMockForIDParams contains parameters of the Accessor.ForID
type AccessorMockForIDParams struct {
	ctx      context.Context
	pulse    insolar.PulseNumber
	recordID insolar.ID
}

// AccessorMockForIDResults contains results of the Accessor.ForID
type AccessorMockForIDResults struct {
	j1 insolar.JetID
	b1 bool
}

// Expect sets up expected params for Accessor.ForID
func (mmForID *mAccessorMockForID) Expect(ctx context.Context, pulse insolar.PulseNumber, recordID insolar.ID) *mAccessorMockForID {
	if mmForID.mock.funcForID != nil {
		mmForID.mock.t.Fatalf("AccessorMock.ForID mock is already set by Set")
	}

	if mmForID.defaultExpectation == nil {
		mmForID.defaultExpectation = &AccessorMockForIDExpectation{}
	}

	mmForID.defaultExpectation.params = &AccessorMockForIDParams{ctx, pulse, recordID}
	for _, e := range mmForID.expectations {
		if minimock.Equal(e.params, mmForID.defaultExpectation.params) {
			mmForID.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmForID.defaultExpectation.params)
		}
	}

	return mmForID
}

// Inspect accepts an inspector function that has same arguments as the Accessor.ForID
func (mmForID *mAccessorMockForID) Inspect(f func(ctx context.Context, pulse insolar.PulseNumber, recordID insolar.ID)) *mAccessorMockForID {
	if mmForID.mock.inspectFuncForID != nil {
		mmForID.mock.t.Fatalf("Inspect function is already set for AccessorMock.ForID")
	}

	mmForID.mock.inspectFuncForID = f

	return mmForID
}

// Return sets up results that will be returned by Accessor.ForID
func (mmForID *mAccessorMockForID) Return(j1 insolar.JetID, b1 bool) *AccessorMock {
	if mmForID.mock.funcForID != nil {
		mmForID.mock.t.Fatalf("AccessorMock.ForID mock is already set by Set")
	}

	if mmForID.defaultExpectation == nil {
		mmForID.defaultExpectation = &AccessorMockForIDExpectation{mock: mmForID.mock}
	}
	mmForID.defaultExpectation.results = &AccessorMockForIDResults{j1, b1}
	return mmForID.mock
}

//Set uses given function f to mock the Accessor.ForID method
func (mmForID *mAccessorMockForID) Set(f func(ctx context.Context, pulse insolar.PulseNumber, recordID insolar.ID) (j1 insolar.JetID, b1 bool)) *AccessorMock {
	if mmForID.defaultExpectation != nil {
		mmForID.mock.t.Fatalf("Default expectation is already set for the Accessor.ForID method")
	}

	if len(mmForID.expectations) > 0 {
		mmForID.mock.t.Fatalf("Some expectations are already set for the Accessor.ForID method")
	}

	mmForID.mock.funcForID = f
	return mmForID.mock
}

// When sets expectation for the Accessor.ForID which will trigger the result defined by the following
// Then helper
func (mmForID *mAccessorMockForID) When(ctx context.Context, pulse insolar.PulseNumber, recordID insolar.ID) *AccessorMockForIDExpectation {
	if mmForID.mock.funcForID != nil {
		mmForID.mock.t.Fatalf("AccessorMock.ForID mock is already set by Set")
	}

	expectation := &AccessorMockForIDExpectation{
		mock:   mmForID.mock,
		params: &AccessorMockForIDParams{ctx, pulse, recordID},
	}
	mmForID.expectations = append(mmForID.expectations, expectation)
	return expectation
}

// Then sets up Accessor.ForID return parameters for the expectation previously defined by the When method
func (e *AccessorMockForIDExpectation) Then(j1 insolar.JetID, b1 bool) *AccessorMock {
	e.results = &AccessorMockForIDResults{j1, b1}
	return e.mock
}

// ForID implements Accessor
func (mmForID *AccessorMock) ForID(ctx context.Context, pulse insolar.PulseNumber, recordID insolar.ID) (j1 insolar.JetID, b1 bool) {
	mm_atomic.AddUint64(&mmForID.beforeForIDCounter, 1)
	defer mm_atomic.AddUint64(&mmForID.afterForIDCounter, 1)

	if mmForID.inspectFuncForID != nil {
		mmForID.inspectFuncForID(ctx, pulse, recordID)
	}

	mm_params := &AccessorMockForIDParams{ctx, pulse, recordID}

	// Record call args
	mmForID.ForIDMock.mutex.Lock()
	mmForID.ForIDMock.callArgs = append(mmForID.ForIDMock.callArgs, mm_params)
	mmForID.ForIDMock.mutex.Unlock()

	for _, e := range mmForID.ForIDMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.j1, e.results.b1
		}
	}

	if mmForID.ForIDMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmForID.ForIDMock.defaultExpectation.Counter, 1)
		mm_want := mmForID.ForIDMock.defaultExpectation.params
		mm_got := AccessorMockForIDParams{ctx, pulse, recordID}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmForID.t.Errorf("AccessorMock.ForID got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmForID.ForIDMock.defaultExpectation.results
		if mm_results == nil {
			mmForID.t.Fatal("No results are set for the AccessorMock.ForID")
		}
		return (*mm_results).j1, (*mm_results).b1
	}
	if mmForID.funcForID != nil {
		return mmForID.funcForID(ctx, pulse, recordID)
	}
	mmForID.t.Fatalf("Unexpected call to AccessorMock.ForID. %v %v %v", ctx, pulse, recordID)
	return
}

// ForIDAfterCounter returns a count of finished AccessorMock.ForID invocations
func (mmForID *AccessorMock) ForIDAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmForID.afterForIDCounter)
}

// ForIDBeforeCounter returns a count of AccessorMock.ForID invocations
func (mmForID *AccessorMock) ForIDBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmForID.beforeForIDCounter)
}

// Calls returns a list of arguments used in each call to AccessorMock.ForID.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmForID *mAccessorMockForID) Calls() []*AccessorMockForIDParams {
	mmForID.mutex.RLock()

	argCopy := make([]*AccessorMockForIDParams, len(mmForID.callArgs))
	copy(argCopy, mmForID.callArgs)

	mmForID.mutex.RUnlock()

	return argCopy
}

// MinimockForIDDone returns true if the count of the ForID invocations corresponds
// the number of defined expectations
func (m *AccessorMock) MinimockForIDDone() bool {
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
func (m *AccessorMock) MinimockForIDInspect() {
	for _, e := range m.ForIDMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to AccessorMock.ForID with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.ForIDMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterForIDCounter) < 1 {
		if m.ForIDMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to AccessorMock.ForID")
		} else {
			m.t.Errorf("Expected call to AccessorMock.ForID with params: %#v", *m.ForIDMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcForID != nil && mm_atomic.LoadUint64(&m.afterForIDCounter) < 1 {
		m.t.Error("Expected call to AccessorMock.ForID")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *AccessorMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockAllInspect()

		m.MinimockForIDInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *AccessorMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *AccessorMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockAllDone() &&
		m.MinimockForIDDone()
}
