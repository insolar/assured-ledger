package adapter

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

// MemoryCacheMock implements MemoryCache
type MemoryCacheMock struct {
	t minimock.Tester

	funcPrepareAsync          func(e1 smachine.ExecutionContext, fn AsyncCallFunc) (a1 smachine.AsyncCallRequester)
	inspectFuncPrepareAsync   func(e1 smachine.ExecutionContext, fn AsyncCallFunc)
	afterPrepareAsyncCounter  uint64
	beforePrepareAsyncCounter uint64
	PrepareAsyncMock          mMemoryCacheMockPrepareAsync

	funcPrepareNotify          func(e1 smachine.ExecutionContext, fn CallFunc) (n1 smachine.NotifyRequester)
	inspectFuncPrepareNotify   func(e1 smachine.ExecutionContext, fn CallFunc)
	afterPrepareNotifyCounter  uint64
	beforePrepareNotifyCounter uint64
	PrepareNotifyMock          mMemoryCacheMockPrepareNotify

	funcPrepareSync          func(e1 smachine.ExecutionContext, fn CallFunc) (s1 smachine.SyncCallRequester)
	inspectFuncPrepareSync   func(e1 smachine.ExecutionContext, fn CallFunc)
	afterPrepareSyncCounter  uint64
	beforePrepareSyncCounter uint64
	PrepareSyncMock          mMemoryCacheMockPrepareSync
}

// NewMemoryCacheMock returns a mock for MemoryCache
func NewMemoryCacheMock(t minimock.Tester) *MemoryCacheMock {
	m := &MemoryCacheMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.PrepareAsyncMock = mMemoryCacheMockPrepareAsync{mock: m}
	m.PrepareAsyncMock.callArgs = []*MemoryCacheMockPrepareAsyncParams{}

	m.PrepareNotifyMock = mMemoryCacheMockPrepareNotify{mock: m}
	m.PrepareNotifyMock.callArgs = []*MemoryCacheMockPrepareNotifyParams{}

	m.PrepareSyncMock = mMemoryCacheMockPrepareSync{mock: m}
	m.PrepareSyncMock.callArgs = []*MemoryCacheMockPrepareSyncParams{}

	return m
}

type mMemoryCacheMockPrepareAsync struct {
	mock               *MemoryCacheMock
	defaultExpectation *MemoryCacheMockPrepareAsyncExpectation
	expectations       []*MemoryCacheMockPrepareAsyncExpectation

	callArgs []*MemoryCacheMockPrepareAsyncParams
	mutex    sync.RWMutex
}

// MemoryCacheMockPrepareAsyncExpectation specifies expectation struct of the MemoryCache.PrepareAsync
type MemoryCacheMockPrepareAsyncExpectation struct {
	mock    *MemoryCacheMock
	params  *MemoryCacheMockPrepareAsyncParams
	results *MemoryCacheMockPrepareAsyncResults
	Counter uint64
}

// MemoryCacheMockPrepareAsyncParams contains parameters of the MemoryCache.PrepareAsync
type MemoryCacheMockPrepareAsyncParams struct {
	e1 smachine.ExecutionContext
	fn AsyncCallFunc
}

// MemoryCacheMockPrepareAsyncResults contains results of the MemoryCache.PrepareAsync
type MemoryCacheMockPrepareAsyncResults struct {
	a1 smachine.AsyncCallRequester
}

// Expect sets up expected params for MemoryCache.PrepareAsync
func (mmPrepareAsync *mMemoryCacheMockPrepareAsync) Expect(e1 smachine.ExecutionContext, fn AsyncCallFunc) *mMemoryCacheMockPrepareAsync {
	if mmPrepareAsync.mock.funcPrepareAsync != nil {
		mmPrepareAsync.mock.t.Fatalf("MemoryCacheMock.PrepareAsync mock is already set by Set")
	}

	if mmPrepareAsync.defaultExpectation == nil {
		mmPrepareAsync.defaultExpectation = &MemoryCacheMockPrepareAsyncExpectation{}
	}

	mmPrepareAsync.defaultExpectation.params = &MemoryCacheMockPrepareAsyncParams{e1, fn}
	for _, e := range mmPrepareAsync.expectations {
		if minimock.Equal(e.params, mmPrepareAsync.defaultExpectation.params) {
			mmPrepareAsync.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmPrepareAsync.defaultExpectation.params)
		}
	}

	return mmPrepareAsync
}

// Inspect accepts an inspector function that has same arguments as the MemoryCache.PrepareAsync
func (mmPrepareAsync *mMemoryCacheMockPrepareAsync) Inspect(f func(e1 smachine.ExecutionContext, fn AsyncCallFunc)) *mMemoryCacheMockPrepareAsync {
	if mmPrepareAsync.mock.inspectFuncPrepareAsync != nil {
		mmPrepareAsync.mock.t.Fatalf("Inspect function is already set for MemoryCacheMock.PrepareAsync")
	}

	mmPrepareAsync.mock.inspectFuncPrepareAsync = f

	return mmPrepareAsync
}

// Return sets up results that will be returned by MemoryCache.PrepareAsync
func (mmPrepareAsync *mMemoryCacheMockPrepareAsync) Return(a1 smachine.AsyncCallRequester) *MemoryCacheMock {
	if mmPrepareAsync.mock.funcPrepareAsync != nil {
		mmPrepareAsync.mock.t.Fatalf("MemoryCacheMock.PrepareAsync mock is already set by Set")
	}

	if mmPrepareAsync.defaultExpectation == nil {
		mmPrepareAsync.defaultExpectation = &MemoryCacheMockPrepareAsyncExpectation{mock: mmPrepareAsync.mock}
	}
	mmPrepareAsync.defaultExpectation.results = &MemoryCacheMockPrepareAsyncResults{a1}
	return mmPrepareAsync.mock
}

//Set uses given function f to mock the MemoryCache.PrepareAsync method
func (mmPrepareAsync *mMemoryCacheMockPrepareAsync) Set(f func(e1 smachine.ExecutionContext, fn AsyncCallFunc) (a1 smachine.AsyncCallRequester)) *MemoryCacheMock {
	if mmPrepareAsync.defaultExpectation != nil {
		mmPrepareAsync.mock.t.Fatalf("Default expectation is already set for the MemoryCache.PrepareAsync method")
	}

	if len(mmPrepareAsync.expectations) > 0 {
		mmPrepareAsync.mock.t.Fatalf("Some expectations are already set for the MemoryCache.PrepareAsync method")
	}

	mmPrepareAsync.mock.funcPrepareAsync = f
	return mmPrepareAsync.mock
}

// When sets expectation for the MemoryCache.PrepareAsync which will trigger the result defined by the following
// Then helper
func (mmPrepareAsync *mMemoryCacheMockPrepareAsync) When(e1 smachine.ExecutionContext, fn AsyncCallFunc) *MemoryCacheMockPrepareAsyncExpectation {
	if mmPrepareAsync.mock.funcPrepareAsync != nil {
		mmPrepareAsync.mock.t.Fatalf("MemoryCacheMock.PrepareAsync mock is already set by Set")
	}

	expectation := &MemoryCacheMockPrepareAsyncExpectation{
		mock:   mmPrepareAsync.mock,
		params: &MemoryCacheMockPrepareAsyncParams{e1, fn},
	}
	mmPrepareAsync.expectations = append(mmPrepareAsync.expectations, expectation)
	return expectation
}

// Then sets up MemoryCache.PrepareAsync return parameters for the expectation previously defined by the When method
func (e *MemoryCacheMockPrepareAsyncExpectation) Then(a1 smachine.AsyncCallRequester) *MemoryCacheMock {
	e.results = &MemoryCacheMockPrepareAsyncResults{a1}
	return e.mock
}

// PrepareAsync implements MemoryCache
func (mmPrepareAsync *MemoryCacheMock) PrepareAsync(e1 smachine.ExecutionContext, fn AsyncCallFunc) (a1 smachine.AsyncCallRequester) {
	mm_atomic.AddUint64(&mmPrepareAsync.beforePrepareAsyncCounter, 1)
	defer mm_atomic.AddUint64(&mmPrepareAsync.afterPrepareAsyncCounter, 1)

	if mmPrepareAsync.inspectFuncPrepareAsync != nil {
		mmPrepareAsync.inspectFuncPrepareAsync(e1, fn)
	}

	mm_params := &MemoryCacheMockPrepareAsyncParams{e1, fn}

	// Record call args
	mmPrepareAsync.PrepareAsyncMock.mutex.Lock()
	mmPrepareAsync.PrepareAsyncMock.callArgs = append(mmPrepareAsync.PrepareAsyncMock.callArgs, mm_params)
	mmPrepareAsync.PrepareAsyncMock.mutex.Unlock()

	for _, e := range mmPrepareAsync.PrepareAsyncMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.a1
		}
	}

	if mmPrepareAsync.PrepareAsyncMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmPrepareAsync.PrepareAsyncMock.defaultExpectation.Counter, 1)
		mm_want := mmPrepareAsync.PrepareAsyncMock.defaultExpectation.params
		mm_got := MemoryCacheMockPrepareAsyncParams{e1, fn}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmPrepareAsync.t.Errorf("MemoryCacheMock.PrepareAsync got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmPrepareAsync.PrepareAsyncMock.defaultExpectation.results
		if mm_results == nil {
			mmPrepareAsync.t.Fatal("No results are set for the MemoryCacheMock.PrepareAsync")
		}
		return (*mm_results).a1
	}
	if mmPrepareAsync.funcPrepareAsync != nil {
		return mmPrepareAsync.funcPrepareAsync(e1, fn)
	}
	mmPrepareAsync.t.Fatalf("Unexpected call to MemoryCacheMock.PrepareAsync. %v %v", e1, fn)
	return
}

// PrepareAsyncAfterCounter returns a count of finished MemoryCacheMock.PrepareAsync invocations
func (mmPrepareAsync *MemoryCacheMock) PrepareAsyncAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareAsync.afterPrepareAsyncCounter)
}

// PrepareAsyncBeforeCounter returns a count of MemoryCacheMock.PrepareAsync invocations
func (mmPrepareAsync *MemoryCacheMock) PrepareAsyncBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareAsync.beforePrepareAsyncCounter)
}

// Calls returns a list of arguments used in each call to MemoryCacheMock.PrepareAsync.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmPrepareAsync *mMemoryCacheMockPrepareAsync) Calls() []*MemoryCacheMockPrepareAsyncParams {
	mmPrepareAsync.mutex.RLock()

	argCopy := make([]*MemoryCacheMockPrepareAsyncParams, len(mmPrepareAsync.callArgs))
	copy(argCopy, mmPrepareAsync.callArgs)

	mmPrepareAsync.mutex.RUnlock()

	return argCopy
}

// MinimockPrepareAsyncDone returns true if the count of the PrepareAsync invocations corresponds
// the number of defined expectations
func (m *MemoryCacheMock) MinimockPrepareAsyncDone() bool {
	for _, e := range m.PrepareAsyncMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareAsyncMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareAsyncCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareAsync != nil && mm_atomic.LoadUint64(&m.afterPrepareAsyncCounter) < 1 {
		return false
	}
	return true
}

// MinimockPrepareAsyncInspect logs each unmet expectation
func (m *MemoryCacheMock) MinimockPrepareAsyncInspect() {
	for _, e := range m.PrepareAsyncMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to MemoryCacheMock.PrepareAsync with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareAsyncMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareAsyncCounter) < 1 {
		if m.PrepareAsyncMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to MemoryCacheMock.PrepareAsync")
		} else {
			m.t.Errorf("Expected call to MemoryCacheMock.PrepareAsync with params: %#v", *m.PrepareAsyncMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareAsync != nil && mm_atomic.LoadUint64(&m.afterPrepareAsyncCounter) < 1 {
		m.t.Error("Expected call to MemoryCacheMock.PrepareAsync")
	}
}

type mMemoryCacheMockPrepareNotify struct {
	mock               *MemoryCacheMock
	defaultExpectation *MemoryCacheMockPrepareNotifyExpectation
	expectations       []*MemoryCacheMockPrepareNotifyExpectation

	callArgs []*MemoryCacheMockPrepareNotifyParams
	mutex    sync.RWMutex
}

// MemoryCacheMockPrepareNotifyExpectation specifies expectation struct of the MemoryCache.PrepareNotify
type MemoryCacheMockPrepareNotifyExpectation struct {
	mock    *MemoryCacheMock
	params  *MemoryCacheMockPrepareNotifyParams
	results *MemoryCacheMockPrepareNotifyResults
	Counter uint64
}

// MemoryCacheMockPrepareNotifyParams contains parameters of the MemoryCache.PrepareNotify
type MemoryCacheMockPrepareNotifyParams struct {
	e1 smachine.ExecutionContext
	fn CallFunc
}

// MemoryCacheMockPrepareNotifyResults contains results of the MemoryCache.PrepareNotify
type MemoryCacheMockPrepareNotifyResults struct {
	n1 smachine.NotifyRequester
}

// Expect sets up expected params for MemoryCache.PrepareNotify
func (mmPrepareNotify *mMemoryCacheMockPrepareNotify) Expect(e1 smachine.ExecutionContext, fn CallFunc) *mMemoryCacheMockPrepareNotify {
	if mmPrepareNotify.mock.funcPrepareNotify != nil {
		mmPrepareNotify.mock.t.Fatalf("MemoryCacheMock.PrepareNotify mock is already set by Set")
	}

	if mmPrepareNotify.defaultExpectation == nil {
		mmPrepareNotify.defaultExpectation = &MemoryCacheMockPrepareNotifyExpectation{}
	}

	mmPrepareNotify.defaultExpectation.params = &MemoryCacheMockPrepareNotifyParams{e1, fn}
	for _, e := range mmPrepareNotify.expectations {
		if minimock.Equal(e.params, mmPrepareNotify.defaultExpectation.params) {
			mmPrepareNotify.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmPrepareNotify.defaultExpectation.params)
		}
	}

	return mmPrepareNotify
}

// Inspect accepts an inspector function that has same arguments as the MemoryCache.PrepareNotify
func (mmPrepareNotify *mMemoryCacheMockPrepareNotify) Inspect(f func(e1 smachine.ExecutionContext, fn CallFunc)) *mMemoryCacheMockPrepareNotify {
	if mmPrepareNotify.mock.inspectFuncPrepareNotify != nil {
		mmPrepareNotify.mock.t.Fatalf("Inspect function is already set for MemoryCacheMock.PrepareNotify")
	}

	mmPrepareNotify.mock.inspectFuncPrepareNotify = f

	return mmPrepareNotify
}

// Return sets up results that will be returned by MemoryCache.PrepareNotify
func (mmPrepareNotify *mMemoryCacheMockPrepareNotify) Return(n1 smachine.NotifyRequester) *MemoryCacheMock {
	if mmPrepareNotify.mock.funcPrepareNotify != nil {
		mmPrepareNotify.mock.t.Fatalf("MemoryCacheMock.PrepareNotify mock is already set by Set")
	}

	if mmPrepareNotify.defaultExpectation == nil {
		mmPrepareNotify.defaultExpectation = &MemoryCacheMockPrepareNotifyExpectation{mock: mmPrepareNotify.mock}
	}
	mmPrepareNotify.defaultExpectation.results = &MemoryCacheMockPrepareNotifyResults{n1}
	return mmPrepareNotify.mock
}

//Set uses given function f to mock the MemoryCache.PrepareNotify method
func (mmPrepareNotify *mMemoryCacheMockPrepareNotify) Set(f func(e1 smachine.ExecutionContext, fn CallFunc) (n1 smachine.NotifyRequester)) *MemoryCacheMock {
	if mmPrepareNotify.defaultExpectation != nil {
		mmPrepareNotify.mock.t.Fatalf("Default expectation is already set for the MemoryCache.PrepareNotify method")
	}

	if len(mmPrepareNotify.expectations) > 0 {
		mmPrepareNotify.mock.t.Fatalf("Some expectations are already set for the MemoryCache.PrepareNotify method")
	}

	mmPrepareNotify.mock.funcPrepareNotify = f
	return mmPrepareNotify.mock
}

// When sets expectation for the MemoryCache.PrepareNotify which will trigger the result defined by the following
// Then helper
func (mmPrepareNotify *mMemoryCacheMockPrepareNotify) When(e1 smachine.ExecutionContext, fn CallFunc) *MemoryCacheMockPrepareNotifyExpectation {
	if mmPrepareNotify.mock.funcPrepareNotify != nil {
		mmPrepareNotify.mock.t.Fatalf("MemoryCacheMock.PrepareNotify mock is already set by Set")
	}

	expectation := &MemoryCacheMockPrepareNotifyExpectation{
		mock:   mmPrepareNotify.mock,
		params: &MemoryCacheMockPrepareNotifyParams{e1, fn},
	}
	mmPrepareNotify.expectations = append(mmPrepareNotify.expectations, expectation)
	return expectation
}

// Then sets up MemoryCache.PrepareNotify return parameters for the expectation previously defined by the When method
func (e *MemoryCacheMockPrepareNotifyExpectation) Then(n1 smachine.NotifyRequester) *MemoryCacheMock {
	e.results = &MemoryCacheMockPrepareNotifyResults{n1}
	return e.mock
}

// PrepareNotify implements MemoryCache
func (mmPrepareNotify *MemoryCacheMock) PrepareNotify(e1 smachine.ExecutionContext, fn CallFunc) (n1 smachine.NotifyRequester) {
	mm_atomic.AddUint64(&mmPrepareNotify.beforePrepareNotifyCounter, 1)
	defer mm_atomic.AddUint64(&mmPrepareNotify.afterPrepareNotifyCounter, 1)

	if mmPrepareNotify.inspectFuncPrepareNotify != nil {
		mmPrepareNotify.inspectFuncPrepareNotify(e1, fn)
	}

	mm_params := &MemoryCacheMockPrepareNotifyParams{e1, fn}

	// Record call args
	mmPrepareNotify.PrepareNotifyMock.mutex.Lock()
	mmPrepareNotify.PrepareNotifyMock.callArgs = append(mmPrepareNotify.PrepareNotifyMock.callArgs, mm_params)
	mmPrepareNotify.PrepareNotifyMock.mutex.Unlock()

	for _, e := range mmPrepareNotify.PrepareNotifyMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.n1
		}
	}

	if mmPrepareNotify.PrepareNotifyMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmPrepareNotify.PrepareNotifyMock.defaultExpectation.Counter, 1)
		mm_want := mmPrepareNotify.PrepareNotifyMock.defaultExpectation.params
		mm_got := MemoryCacheMockPrepareNotifyParams{e1, fn}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmPrepareNotify.t.Errorf("MemoryCacheMock.PrepareNotify got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmPrepareNotify.PrepareNotifyMock.defaultExpectation.results
		if mm_results == nil {
			mmPrepareNotify.t.Fatal("No results are set for the MemoryCacheMock.PrepareNotify")
		}
		return (*mm_results).n1
	}
	if mmPrepareNotify.funcPrepareNotify != nil {
		return mmPrepareNotify.funcPrepareNotify(e1, fn)
	}
	mmPrepareNotify.t.Fatalf("Unexpected call to MemoryCacheMock.PrepareNotify. %v %v", e1, fn)
	return
}

// PrepareNotifyAfterCounter returns a count of finished MemoryCacheMock.PrepareNotify invocations
func (mmPrepareNotify *MemoryCacheMock) PrepareNotifyAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareNotify.afterPrepareNotifyCounter)
}

// PrepareNotifyBeforeCounter returns a count of MemoryCacheMock.PrepareNotify invocations
func (mmPrepareNotify *MemoryCacheMock) PrepareNotifyBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareNotify.beforePrepareNotifyCounter)
}

// Calls returns a list of arguments used in each call to MemoryCacheMock.PrepareNotify.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmPrepareNotify *mMemoryCacheMockPrepareNotify) Calls() []*MemoryCacheMockPrepareNotifyParams {
	mmPrepareNotify.mutex.RLock()

	argCopy := make([]*MemoryCacheMockPrepareNotifyParams, len(mmPrepareNotify.callArgs))
	copy(argCopy, mmPrepareNotify.callArgs)

	mmPrepareNotify.mutex.RUnlock()

	return argCopy
}

// MinimockPrepareNotifyDone returns true if the count of the PrepareNotify invocations corresponds
// the number of defined expectations
func (m *MemoryCacheMock) MinimockPrepareNotifyDone() bool {
	for _, e := range m.PrepareNotifyMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareNotifyMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareNotifyCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareNotify != nil && mm_atomic.LoadUint64(&m.afterPrepareNotifyCounter) < 1 {
		return false
	}
	return true
}

// MinimockPrepareNotifyInspect logs each unmet expectation
func (m *MemoryCacheMock) MinimockPrepareNotifyInspect() {
	for _, e := range m.PrepareNotifyMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to MemoryCacheMock.PrepareNotify with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareNotifyMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareNotifyCounter) < 1 {
		if m.PrepareNotifyMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to MemoryCacheMock.PrepareNotify")
		} else {
			m.t.Errorf("Expected call to MemoryCacheMock.PrepareNotify with params: %#v", *m.PrepareNotifyMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareNotify != nil && mm_atomic.LoadUint64(&m.afterPrepareNotifyCounter) < 1 {
		m.t.Error("Expected call to MemoryCacheMock.PrepareNotify")
	}
}

type mMemoryCacheMockPrepareSync struct {
	mock               *MemoryCacheMock
	defaultExpectation *MemoryCacheMockPrepareSyncExpectation
	expectations       []*MemoryCacheMockPrepareSyncExpectation

	callArgs []*MemoryCacheMockPrepareSyncParams
	mutex    sync.RWMutex
}

// MemoryCacheMockPrepareSyncExpectation specifies expectation struct of the MemoryCache.PrepareSync
type MemoryCacheMockPrepareSyncExpectation struct {
	mock    *MemoryCacheMock
	params  *MemoryCacheMockPrepareSyncParams
	results *MemoryCacheMockPrepareSyncResults
	Counter uint64
}

// MemoryCacheMockPrepareSyncParams contains parameters of the MemoryCache.PrepareSync
type MemoryCacheMockPrepareSyncParams struct {
	e1 smachine.ExecutionContext
	fn CallFunc
}

// MemoryCacheMockPrepareSyncResults contains results of the MemoryCache.PrepareSync
type MemoryCacheMockPrepareSyncResults struct {
	s1 smachine.SyncCallRequester
}

// Expect sets up expected params for MemoryCache.PrepareSync
func (mmPrepareSync *mMemoryCacheMockPrepareSync) Expect(e1 smachine.ExecutionContext, fn CallFunc) *mMemoryCacheMockPrepareSync {
	if mmPrepareSync.mock.funcPrepareSync != nil {
		mmPrepareSync.mock.t.Fatalf("MemoryCacheMock.PrepareSync mock is already set by Set")
	}

	if mmPrepareSync.defaultExpectation == nil {
		mmPrepareSync.defaultExpectation = &MemoryCacheMockPrepareSyncExpectation{}
	}

	mmPrepareSync.defaultExpectation.params = &MemoryCacheMockPrepareSyncParams{e1, fn}
	for _, e := range mmPrepareSync.expectations {
		if minimock.Equal(e.params, mmPrepareSync.defaultExpectation.params) {
			mmPrepareSync.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmPrepareSync.defaultExpectation.params)
		}
	}

	return mmPrepareSync
}

// Inspect accepts an inspector function that has same arguments as the MemoryCache.PrepareSync
func (mmPrepareSync *mMemoryCacheMockPrepareSync) Inspect(f func(e1 smachine.ExecutionContext, fn CallFunc)) *mMemoryCacheMockPrepareSync {
	if mmPrepareSync.mock.inspectFuncPrepareSync != nil {
		mmPrepareSync.mock.t.Fatalf("Inspect function is already set for MemoryCacheMock.PrepareSync")
	}

	mmPrepareSync.mock.inspectFuncPrepareSync = f

	return mmPrepareSync
}

// Return sets up results that will be returned by MemoryCache.PrepareSync
func (mmPrepareSync *mMemoryCacheMockPrepareSync) Return(s1 smachine.SyncCallRequester) *MemoryCacheMock {
	if mmPrepareSync.mock.funcPrepareSync != nil {
		mmPrepareSync.mock.t.Fatalf("MemoryCacheMock.PrepareSync mock is already set by Set")
	}

	if mmPrepareSync.defaultExpectation == nil {
		mmPrepareSync.defaultExpectation = &MemoryCacheMockPrepareSyncExpectation{mock: mmPrepareSync.mock}
	}
	mmPrepareSync.defaultExpectation.results = &MemoryCacheMockPrepareSyncResults{s1}
	return mmPrepareSync.mock
}

//Set uses given function f to mock the MemoryCache.PrepareSync method
func (mmPrepareSync *mMemoryCacheMockPrepareSync) Set(f func(e1 smachine.ExecutionContext, fn CallFunc) (s1 smachine.SyncCallRequester)) *MemoryCacheMock {
	if mmPrepareSync.defaultExpectation != nil {
		mmPrepareSync.mock.t.Fatalf("Default expectation is already set for the MemoryCache.PrepareSync method")
	}

	if len(mmPrepareSync.expectations) > 0 {
		mmPrepareSync.mock.t.Fatalf("Some expectations are already set for the MemoryCache.PrepareSync method")
	}

	mmPrepareSync.mock.funcPrepareSync = f
	return mmPrepareSync.mock
}

// When sets expectation for the MemoryCache.PrepareSync which will trigger the result defined by the following
// Then helper
func (mmPrepareSync *mMemoryCacheMockPrepareSync) When(e1 smachine.ExecutionContext, fn CallFunc) *MemoryCacheMockPrepareSyncExpectation {
	if mmPrepareSync.mock.funcPrepareSync != nil {
		mmPrepareSync.mock.t.Fatalf("MemoryCacheMock.PrepareSync mock is already set by Set")
	}

	expectation := &MemoryCacheMockPrepareSyncExpectation{
		mock:   mmPrepareSync.mock,
		params: &MemoryCacheMockPrepareSyncParams{e1, fn},
	}
	mmPrepareSync.expectations = append(mmPrepareSync.expectations, expectation)
	return expectation
}

// Then sets up MemoryCache.PrepareSync return parameters for the expectation previously defined by the When method
func (e *MemoryCacheMockPrepareSyncExpectation) Then(s1 smachine.SyncCallRequester) *MemoryCacheMock {
	e.results = &MemoryCacheMockPrepareSyncResults{s1}
	return e.mock
}

// PrepareSync implements MemoryCache
func (mmPrepareSync *MemoryCacheMock) PrepareSync(e1 smachine.ExecutionContext, fn CallFunc) (s1 smachine.SyncCallRequester) {
	mm_atomic.AddUint64(&mmPrepareSync.beforePrepareSyncCounter, 1)
	defer mm_atomic.AddUint64(&mmPrepareSync.afterPrepareSyncCounter, 1)

	if mmPrepareSync.inspectFuncPrepareSync != nil {
		mmPrepareSync.inspectFuncPrepareSync(e1, fn)
	}

	mm_params := &MemoryCacheMockPrepareSyncParams{e1, fn}

	// Record call args
	mmPrepareSync.PrepareSyncMock.mutex.Lock()
	mmPrepareSync.PrepareSyncMock.callArgs = append(mmPrepareSync.PrepareSyncMock.callArgs, mm_params)
	mmPrepareSync.PrepareSyncMock.mutex.Unlock()

	for _, e := range mmPrepareSync.PrepareSyncMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.s1
		}
	}

	if mmPrepareSync.PrepareSyncMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmPrepareSync.PrepareSyncMock.defaultExpectation.Counter, 1)
		mm_want := mmPrepareSync.PrepareSyncMock.defaultExpectation.params
		mm_got := MemoryCacheMockPrepareSyncParams{e1, fn}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmPrepareSync.t.Errorf("MemoryCacheMock.PrepareSync got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmPrepareSync.PrepareSyncMock.defaultExpectation.results
		if mm_results == nil {
			mmPrepareSync.t.Fatal("No results are set for the MemoryCacheMock.PrepareSync")
		}
		return (*mm_results).s1
	}
	if mmPrepareSync.funcPrepareSync != nil {
		return mmPrepareSync.funcPrepareSync(e1, fn)
	}
	mmPrepareSync.t.Fatalf("Unexpected call to MemoryCacheMock.PrepareSync. %v %v", e1, fn)
	return
}

// PrepareSyncAfterCounter returns a count of finished MemoryCacheMock.PrepareSync invocations
func (mmPrepareSync *MemoryCacheMock) PrepareSyncAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareSync.afterPrepareSyncCounter)
}

// PrepareSyncBeforeCounter returns a count of MemoryCacheMock.PrepareSync invocations
func (mmPrepareSync *MemoryCacheMock) PrepareSyncBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareSync.beforePrepareSyncCounter)
}

// Calls returns a list of arguments used in each call to MemoryCacheMock.PrepareSync.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmPrepareSync *mMemoryCacheMockPrepareSync) Calls() []*MemoryCacheMockPrepareSyncParams {
	mmPrepareSync.mutex.RLock()

	argCopy := make([]*MemoryCacheMockPrepareSyncParams, len(mmPrepareSync.callArgs))
	copy(argCopy, mmPrepareSync.callArgs)

	mmPrepareSync.mutex.RUnlock()

	return argCopy
}

// MinimockPrepareSyncDone returns true if the count of the PrepareSync invocations corresponds
// the number of defined expectations
func (m *MemoryCacheMock) MinimockPrepareSyncDone() bool {
	for _, e := range m.PrepareSyncMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareSyncMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareSyncCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareSync != nil && mm_atomic.LoadUint64(&m.afterPrepareSyncCounter) < 1 {
		return false
	}
	return true
}

// MinimockPrepareSyncInspect logs each unmet expectation
func (m *MemoryCacheMock) MinimockPrepareSyncInspect() {
	for _, e := range m.PrepareSyncMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to MemoryCacheMock.PrepareSync with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareSyncMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareSyncCounter) < 1 {
		if m.PrepareSyncMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to MemoryCacheMock.PrepareSync")
		} else {
			m.t.Errorf("Expected call to MemoryCacheMock.PrepareSync with params: %#v", *m.PrepareSyncMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareSync != nil && mm_atomic.LoadUint64(&m.afterPrepareSyncCounter) < 1 {
		m.t.Error("Expected call to MemoryCacheMock.PrepareSync")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *MemoryCacheMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockPrepareAsyncInspect()

		m.MinimockPrepareNotifyInspect()

		m.MinimockPrepareSyncInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *MemoryCacheMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *MemoryCacheMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockPrepareAsyncDone() &&
		m.MinimockPrepareNotifyDone() &&
		m.MinimockPrepareSyncDone()
}
