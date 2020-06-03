package introspector

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	context "context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	mm_introproto "github.com/insolar/assured-ledger/ledger-core/instrumentation/introspector/introproto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
)

// PublisherServerMock implements introproto.PublisherServer
type PublisherServerMock struct {
	t minimock.Tester

	funcGetMessagesFilters          func(ctx context.Context, ep1 *mm_introproto.EmptyArgs) (ap1 *mm_introproto.AllMessageFilterStats, err error)
	inspectFuncGetMessagesFilters   func(ctx context.Context, ep1 *mm_introproto.EmptyArgs)
	afterGetMessagesFiltersCounter  uint64
	beforeGetMessagesFiltersCounter uint64
	GetMessagesFiltersMock          mPublisherServerMockGetMessagesFilters

	funcGetMessagesStat          func(ctx context.Context, ep1 *mm_introproto.EmptyArgs) (ap1 *mm_introproto.AllMessageStatByType, err error)
	inspectFuncGetMessagesStat   func(ctx context.Context, ep1 *mm_introproto.EmptyArgs)
	afterGetMessagesStatCounter  uint64
	beforeGetMessagesStatCounter uint64
	GetMessagesStatMock          mPublisherServerMockGetMessagesStat

	funcSetMessagesFilter          func(ctx context.Context, mp1 *mm_introproto.MessageFilterByType) (mp2 *mm_introproto.MessageFilterByType, err error)
	inspectFuncSetMessagesFilter   func(ctx context.Context, mp1 *mm_introproto.MessageFilterByType)
	afterSetMessagesFilterCounter  uint64
	beforeSetMessagesFilterCounter uint64
	SetMessagesFilterMock          mPublisherServerMockSetMessagesFilter
}

// NewPublisherServerMock returns a mock for introproto.PublisherServer
func NewPublisherServerMock(t minimock.Tester) *PublisherServerMock {
	m := &PublisherServerMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.GetMessagesFiltersMock = mPublisherServerMockGetMessagesFilters{mock: m}
	m.GetMessagesFiltersMock.callArgs = []*PublisherServerMockGetMessagesFiltersParams{}

	m.GetMessagesStatMock = mPublisherServerMockGetMessagesStat{mock: m}
	m.GetMessagesStatMock.callArgs = []*PublisherServerMockGetMessagesStatParams{}

	m.SetMessagesFilterMock = mPublisherServerMockSetMessagesFilter{mock: m}
	m.SetMessagesFilterMock.callArgs = []*PublisherServerMockSetMessagesFilterParams{}

	return m
}

type mPublisherServerMockGetMessagesFilters struct {
	mock               *PublisherServerMock
	defaultExpectation *PublisherServerMockGetMessagesFiltersExpectation
	expectations       []*PublisherServerMockGetMessagesFiltersExpectation

	callArgs []*PublisherServerMockGetMessagesFiltersParams
	mutex    sync.RWMutex
}

// PublisherServerMockGetMessagesFiltersExpectation specifies expectation struct of the PublisherServer.GetMessagesFilters
type PublisherServerMockGetMessagesFiltersExpectation struct {
	mock    *PublisherServerMock
	params  *PublisherServerMockGetMessagesFiltersParams
	results *PublisherServerMockGetMessagesFiltersResults
	Counter uint64
}

// PublisherServerMockGetMessagesFiltersParams contains parameters of the PublisherServer.GetMessagesFilters
type PublisherServerMockGetMessagesFiltersParams struct {
	ctx context.Context
	ep1 *mm_introproto.EmptyArgs
}

// PublisherServerMockGetMessagesFiltersResults contains results of the PublisherServer.GetMessagesFilters
type PublisherServerMockGetMessagesFiltersResults struct {
	ap1 *mm_introproto.AllMessageFilterStats
	err error
}

// Expect sets up expected params for PublisherServer.GetMessagesFilters
func (mmGetMessagesFilters *mPublisherServerMockGetMessagesFilters) Expect(ctx context.Context, ep1 *mm_introproto.EmptyArgs) *mPublisherServerMockGetMessagesFilters {
	if mmGetMessagesFilters.mock.funcGetMessagesFilters != nil {
		mmGetMessagesFilters.mock.t.Fatalf("PublisherServerMock.GetMessagesFilters mock is already set by Set")
	}

	if mmGetMessagesFilters.defaultExpectation == nil {
		mmGetMessagesFilters.defaultExpectation = &PublisherServerMockGetMessagesFiltersExpectation{}
	}

	mmGetMessagesFilters.defaultExpectation.params = &PublisherServerMockGetMessagesFiltersParams{ctx, ep1}
	for _, e := range mmGetMessagesFilters.expectations {
		if minimock.Equal(e.params, mmGetMessagesFilters.defaultExpectation.params) {
			mmGetMessagesFilters.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmGetMessagesFilters.defaultExpectation.params)
		}
	}

	return mmGetMessagesFilters
}

// Inspect accepts an inspector function that has same arguments as the PublisherServer.GetMessagesFilters
func (mmGetMessagesFilters *mPublisherServerMockGetMessagesFilters) Inspect(f func(ctx context.Context, ep1 *mm_introproto.EmptyArgs)) *mPublisherServerMockGetMessagesFilters {
	if mmGetMessagesFilters.mock.inspectFuncGetMessagesFilters != nil {
		mmGetMessagesFilters.mock.t.Fatalf("Inspect function is already set for PublisherServerMock.GetMessagesFilters")
	}

	mmGetMessagesFilters.mock.inspectFuncGetMessagesFilters = f

	return mmGetMessagesFilters
}

// Return sets up results that will be returned by PublisherServer.GetMessagesFilters
func (mmGetMessagesFilters *mPublisherServerMockGetMessagesFilters) Return(ap1 *mm_introproto.AllMessageFilterStats, err error) *PublisherServerMock {
	if mmGetMessagesFilters.mock.funcGetMessagesFilters != nil {
		mmGetMessagesFilters.mock.t.Fatalf("PublisherServerMock.GetMessagesFilters mock is already set by Set")
	}

	if mmGetMessagesFilters.defaultExpectation == nil {
		mmGetMessagesFilters.defaultExpectation = &PublisherServerMockGetMessagesFiltersExpectation{mock: mmGetMessagesFilters.mock}
	}
	mmGetMessagesFilters.defaultExpectation.results = &PublisherServerMockGetMessagesFiltersResults{ap1, err}
	return mmGetMessagesFilters.mock
}

//Set uses given function f to mock the PublisherServer.GetMessagesFilters method
func (mmGetMessagesFilters *mPublisherServerMockGetMessagesFilters) Set(f func(ctx context.Context, ep1 *mm_introproto.EmptyArgs) (ap1 *mm_introproto.AllMessageFilterStats, err error)) *PublisherServerMock {
	if mmGetMessagesFilters.defaultExpectation != nil {
		mmGetMessagesFilters.mock.t.Fatalf("Default expectation is already set for the PublisherServer.GetMessagesFilters method")
	}

	if len(mmGetMessagesFilters.expectations) > 0 {
		mmGetMessagesFilters.mock.t.Fatalf("Some expectations are already set for the PublisherServer.GetMessagesFilters method")
	}

	mmGetMessagesFilters.mock.funcGetMessagesFilters = f
	return mmGetMessagesFilters.mock
}

// When sets expectation for the PublisherServer.GetMessagesFilters which will trigger the result defined by the following
// Then helper
func (mmGetMessagesFilters *mPublisherServerMockGetMessagesFilters) When(ctx context.Context, ep1 *mm_introproto.EmptyArgs) *PublisherServerMockGetMessagesFiltersExpectation {
	if mmGetMessagesFilters.mock.funcGetMessagesFilters != nil {
		mmGetMessagesFilters.mock.t.Fatalf("PublisherServerMock.GetMessagesFilters mock is already set by Set")
	}

	expectation := &PublisherServerMockGetMessagesFiltersExpectation{
		mock:   mmGetMessagesFilters.mock,
		params: &PublisherServerMockGetMessagesFiltersParams{ctx, ep1},
	}
	mmGetMessagesFilters.expectations = append(mmGetMessagesFilters.expectations, expectation)
	return expectation
}

// Then sets up PublisherServer.GetMessagesFilters return parameters for the expectation previously defined by the When method
func (e *PublisherServerMockGetMessagesFiltersExpectation) Then(ap1 *mm_introproto.AllMessageFilterStats, err error) *PublisherServerMock {
	e.results = &PublisherServerMockGetMessagesFiltersResults{ap1, err}
	return e.mock
}

// GetMessagesFilters implements introproto.PublisherServer
func (mmGetMessagesFilters *PublisherServerMock) GetMessagesFilters(ctx context.Context, ep1 *mm_introproto.EmptyArgs) (ap1 *mm_introproto.AllMessageFilterStats, err error) {
	mm_atomic.AddUint64(&mmGetMessagesFilters.beforeGetMessagesFiltersCounter, 1)
	defer mm_atomic.AddUint64(&mmGetMessagesFilters.afterGetMessagesFiltersCounter, 1)

	if mmGetMessagesFilters.inspectFuncGetMessagesFilters != nil {
		mmGetMessagesFilters.inspectFuncGetMessagesFilters(ctx, ep1)
	}

	mm_params := &PublisherServerMockGetMessagesFiltersParams{ctx, ep1}

	// Record call args
	mmGetMessagesFilters.GetMessagesFiltersMock.mutex.Lock()
	mmGetMessagesFilters.GetMessagesFiltersMock.callArgs = append(mmGetMessagesFilters.GetMessagesFiltersMock.callArgs, mm_params)
	mmGetMessagesFilters.GetMessagesFiltersMock.mutex.Unlock()

	for _, e := range mmGetMessagesFilters.GetMessagesFiltersMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.ap1, e.results.err
		}
	}

	if mmGetMessagesFilters.GetMessagesFiltersMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetMessagesFilters.GetMessagesFiltersMock.defaultExpectation.Counter, 1)
		mm_want := mmGetMessagesFilters.GetMessagesFiltersMock.defaultExpectation.params
		mm_got := PublisherServerMockGetMessagesFiltersParams{ctx, ep1}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmGetMessagesFilters.t.Errorf("PublisherServerMock.GetMessagesFilters got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmGetMessagesFilters.GetMessagesFiltersMock.defaultExpectation.results
		if mm_results == nil {
			mmGetMessagesFilters.t.Fatal("No results are set for the PublisherServerMock.GetMessagesFilters")
		}
		return (*mm_results).ap1, (*mm_results).err
	}
	if mmGetMessagesFilters.funcGetMessagesFilters != nil {
		return mmGetMessagesFilters.funcGetMessagesFilters(ctx, ep1)
	}
	mmGetMessagesFilters.t.Fatalf("Unexpected call to PublisherServerMock.GetMessagesFilters. %v %v", ctx, ep1)
	return
}

// GetMessagesFiltersAfterCounter returns a count of finished PublisherServerMock.GetMessagesFilters invocations
func (mmGetMessagesFilters *PublisherServerMock) GetMessagesFiltersAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetMessagesFilters.afterGetMessagesFiltersCounter)
}

// GetMessagesFiltersBeforeCounter returns a count of PublisherServerMock.GetMessagesFilters invocations
func (mmGetMessagesFilters *PublisherServerMock) GetMessagesFiltersBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetMessagesFilters.beforeGetMessagesFiltersCounter)
}

// Calls returns a list of arguments used in each call to PublisherServerMock.GetMessagesFilters.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmGetMessagesFilters *mPublisherServerMockGetMessagesFilters) Calls() []*PublisherServerMockGetMessagesFiltersParams {
	mmGetMessagesFilters.mutex.RLock()

	argCopy := make([]*PublisherServerMockGetMessagesFiltersParams, len(mmGetMessagesFilters.callArgs))
	copy(argCopy, mmGetMessagesFilters.callArgs)

	mmGetMessagesFilters.mutex.RUnlock()

	return argCopy
}

// MinimockGetMessagesFiltersDone returns true if the count of the GetMessagesFilters invocations corresponds
// the number of defined expectations
func (m *PublisherServerMock) MinimockGetMessagesFiltersDone() bool {
	for _, e := range m.GetMessagesFiltersMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetMessagesFiltersMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetMessagesFiltersCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetMessagesFilters != nil && mm_atomic.LoadUint64(&m.afterGetMessagesFiltersCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetMessagesFiltersInspect logs each unmet expectation
func (m *PublisherServerMock) MinimockGetMessagesFiltersInspect() {
	for _, e := range m.GetMessagesFiltersMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to PublisherServerMock.GetMessagesFilters with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetMessagesFiltersMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetMessagesFiltersCounter) < 1 {
		if m.GetMessagesFiltersMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to PublisherServerMock.GetMessagesFilters")
		} else {
			m.t.Errorf("Expected call to PublisherServerMock.GetMessagesFilters with params: %#v", *m.GetMessagesFiltersMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetMessagesFilters != nil && mm_atomic.LoadUint64(&m.afterGetMessagesFiltersCounter) < 1 {
		m.t.Error("Expected call to PublisherServerMock.GetMessagesFilters")
	}
}

type mPublisherServerMockGetMessagesStat struct {
	mock               *PublisherServerMock
	defaultExpectation *PublisherServerMockGetMessagesStatExpectation
	expectations       []*PublisherServerMockGetMessagesStatExpectation

	callArgs []*PublisherServerMockGetMessagesStatParams
	mutex    sync.RWMutex
}

// PublisherServerMockGetMessagesStatExpectation specifies expectation struct of the PublisherServer.GetMessagesStat
type PublisherServerMockGetMessagesStatExpectation struct {
	mock    *PublisherServerMock
	params  *PublisherServerMockGetMessagesStatParams
	results *PublisherServerMockGetMessagesStatResults
	Counter uint64
}

// PublisherServerMockGetMessagesStatParams contains parameters of the PublisherServer.GetMessagesStat
type PublisherServerMockGetMessagesStatParams struct {
	ctx context.Context
	ep1 *mm_introproto.EmptyArgs
}

// PublisherServerMockGetMessagesStatResults contains results of the PublisherServer.GetMessagesStat
type PublisherServerMockGetMessagesStatResults struct {
	ap1 *mm_introproto.AllMessageStatByType
	err error
}

// Expect sets up expected params for PublisherServer.GetMessagesStat
func (mmGetMessagesStat *mPublisherServerMockGetMessagesStat) Expect(ctx context.Context, ep1 *mm_introproto.EmptyArgs) *mPublisherServerMockGetMessagesStat {
	if mmGetMessagesStat.mock.funcGetMessagesStat != nil {
		mmGetMessagesStat.mock.t.Fatalf("PublisherServerMock.GetMessagesStat mock is already set by Set")
	}

	if mmGetMessagesStat.defaultExpectation == nil {
		mmGetMessagesStat.defaultExpectation = &PublisherServerMockGetMessagesStatExpectation{}
	}

	mmGetMessagesStat.defaultExpectation.params = &PublisherServerMockGetMessagesStatParams{ctx, ep1}
	for _, e := range mmGetMessagesStat.expectations {
		if minimock.Equal(e.params, mmGetMessagesStat.defaultExpectation.params) {
			mmGetMessagesStat.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmGetMessagesStat.defaultExpectation.params)
		}
	}

	return mmGetMessagesStat
}

// Inspect accepts an inspector function that has same arguments as the PublisherServer.GetMessagesStat
func (mmGetMessagesStat *mPublisherServerMockGetMessagesStat) Inspect(f func(ctx context.Context, ep1 *mm_introproto.EmptyArgs)) *mPublisherServerMockGetMessagesStat {
	if mmGetMessagesStat.mock.inspectFuncGetMessagesStat != nil {
		mmGetMessagesStat.mock.t.Fatalf("Inspect function is already set for PublisherServerMock.GetMessagesStat")
	}

	mmGetMessagesStat.mock.inspectFuncGetMessagesStat = f

	return mmGetMessagesStat
}

// Return sets up results that will be returned by PublisherServer.GetMessagesStat
func (mmGetMessagesStat *mPublisherServerMockGetMessagesStat) Return(ap1 *mm_introproto.AllMessageStatByType, err error) *PublisherServerMock {
	if mmGetMessagesStat.mock.funcGetMessagesStat != nil {
		mmGetMessagesStat.mock.t.Fatalf("PublisherServerMock.GetMessagesStat mock is already set by Set")
	}

	if mmGetMessagesStat.defaultExpectation == nil {
		mmGetMessagesStat.defaultExpectation = &PublisherServerMockGetMessagesStatExpectation{mock: mmGetMessagesStat.mock}
	}
	mmGetMessagesStat.defaultExpectation.results = &PublisherServerMockGetMessagesStatResults{ap1, err}
	return mmGetMessagesStat.mock
}

//Set uses given function f to mock the PublisherServer.GetMessagesStat method
func (mmGetMessagesStat *mPublisherServerMockGetMessagesStat) Set(f func(ctx context.Context, ep1 *mm_introproto.EmptyArgs) (ap1 *mm_introproto.AllMessageStatByType, err error)) *PublisherServerMock {
	if mmGetMessagesStat.defaultExpectation != nil {
		mmGetMessagesStat.mock.t.Fatalf("Default expectation is already set for the PublisherServer.GetMessagesStat method")
	}

	if len(mmGetMessagesStat.expectations) > 0 {
		mmGetMessagesStat.mock.t.Fatalf("Some expectations are already set for the PublisherServer.GetMessagesStat method")
	}

	mmGetMessagesStat.mock.funcGetMessagesStat = f
	return mmGetMessagesStat.mock
}

// When sets expectation for the PublisherServer.GetMessagesStat which will trigger the result defined by the following
// Then helper
func (mmGetMessagesStat *mPublisherServerMockGetMessagesStat) When(ctx context.Context, ep1 *mm_introproto.EmptyArgs) *PublisherServerMockGetMessagesStatExpectation {
	if mmGetMessagesStat.mock.funcGetMessagesStat != nil {
		mmGetMessagesStat.mock.t.Fatalf("PublisherServerMock.GetMessagesStat mock is already set by Set")
	}

	expectation := &PublisherServerMockGetMessagesStatExpectation{
		mock:   mmGetMessagesStat.mock,
		params: &PublisherServerMockGetMessagesStatParams{ctx, ep1},
	}
	mmGetMessagesStat.expectations = append(mmGetMessagesStat.expectations, expectation)
	return expectation
}

// Then sets up PublisherServer.GetMessagesStat return parameters for the expectation previously defined by the When method
func (e *PublisherServerMockGetMessagesStatExpectation) Then(ap1 *mm_introproto.AllMessageStatByType, err error) *PublisherServerMock {
	e.results = &PublisherServerMockGetMessagesStatResults{ap1, err}
	return e.mock
}

// GetMessagesStat implements introproto.PublisherServer
func (mmGetMessagesStat *PublisherServerMock) GetMessagesStat(ctx context.Context, ep1 *mm_introproto.EmptyArgs) (ap1 *mm_introproto.AllMessageStatByType, err error) {
	mm_atomic.AddUint64(&mmGetMessagesStat.beforeGetMessagesStatCounter, 1)
	defer mm_atomic.AddUint64(&mmGetMessagesStat.afterGetMessagesStatCounter, 1)

	if mmGetMessagesStat.inspectFuncGetMessagesStat != nil {
		mmGetMessagesStat.inspectFuncGetMessagesStat(ctx, ep1)
	}

	mm_params := &PublisherServerMockGetMessagesStatParams{ctx, ep1}

	// Record call args
	mmGetMessagesStat.GetMessagesStatMock.mutex.Lock()
	mmGetMessagesStat.GetMessagesStatMock.callArgs = append(mmGetMessagesStat.GetMessagesStatMock.callArgs, mm_params)
	mmGetMessagesStat.GetMessagesStatMock.mutex.Unlock()

	for _, e := range mmGetMessagesStat.GetMessagesStatMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.ap1, e.results.err
		}
	}

	if mmGetMessagesStat.GetMessagesStatMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetMessagesStat.GetMessagesStatMock.defaultExpectation.Counter, 1)
		mm_want := mmGetMessagesStat.GetMessagesStatMock.defaultExpectation.params
		mm_got := PublisherServerMockGetMessagesStatParams{ctx, ep1}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmGetMessagesStat.t.Errorf("PublisherServerMock.GetMessagesStat got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmGetMessagesStat.GetMessagesStatMock.defaultExpectation.results
		if mm_results == nil {
			mmGetMessagesStat.t.Fatal("No results are set for the PublisherServerMock.GetMessagesStat")
		}
		return (*mm_results).ap1, (*mm_results).err
	}
	if mmGetMessagesStat.funcGetMessagesStat != nil {
		return mmGetMessagesStat.funcGetMessagesStat(ctx, ep1)
	}
	mmGetMessagesStat.t.Fatalf("Unexpected call to PublisherServerMock.GetMessagesStat. %v %v", ctx, ep1)
	return
}

// GetMessagesStatAfterCounter returns a count of finished PublisherServerMock.GetMessagesStat invocations
func (mmGetMessagesStat *PublisherServerMock) GetMessagesStatAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetMessagesStat.afterGetMessagesStatCounter)
}

// GetMessagesStatBeforeCounter returns a count of PublisherServerMock.GetMessagesStat invocations
func (mmGetMessagesStat *PublisherServerMock) GetMessagesStatBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetMessagesStat.beforeGetMessagesStatCounter)
}

// Calls returns a list of arguments used in each call to PublisherServerMock.GetMessagesStat.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmGetMessagesStat *mPublisherServerMockGetMessagesStat) Calls() []*PublisherServerMockGetMessagesStatParams {
	mmGetMessagesStat.mutex.RLock()

	argCopy := make([]*PublisherServerMockGetMessagesStatParams, len(mmGetMessagesStat.callArgs))
	copy(argCopy, mmGetMessagesStat.callArgs)

	mmGetMessagesStat.mutex.RUnlock()

	return argCopy
}

// MinimockGetMessagesStatDone returns true if the count of the GetMessagesStat invocations corresponds
// the number of defined expectations
func (m *PublisherServerMock) MinimockGetMessagesStatDone() bool {
	for _, e := range m.GetMessagesStatMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetMessagesStatMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetMessagesStatCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetMessagesStat != nil && mm_atomic.LoadUint64(&m.afterGetMessagesStatCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetMessagesStatInspect logs each unmet expectation
func (m *PublisherServerMock) MinimockGetMessagesStatInspect() {
	for _, e := range m.GetMessagesStatMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to PublisherServerMock.GetMessagesStat with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetMessagesStatMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetMessagesStatCounter) < 1 {
		if m.GetMessagesStatMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to PublisherServerMock.GetMessagesStat")
		} else {
			m.t.Errorf("Expected call to PublisherServerMock.GetMessagesStat with params: %#v", *m.GetMessagesStatMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetMessagesStat != nil && mm_atomic.LoadUint64(&m.afterGetMessagesStatCounter) < 1 {
		m.t.Error("Expected call to PublisherServerMock.GetMessagesStat")
	}
}

type mPublisherServerMockSetMessagesFilter struct {
	mock               *PublisherServerMock
	defaultExpectation *PublisherServerMockSetMessagesFilterExpectation
	expectations       []*PublisherServerMockSetMessagesFilterExpectation

	callArgs []*PublisherServerMockSetMessagesFilterParams
	mutex    sync.RWMutex
}

// PublisherServerMockSetMessagesFilterExpectation specifies expectation struct of the PublisherServer.SetMessagesFilter
type PublisherServerMockSetMessagesFilterExpectation struct {
	mock    *PublisherServerMock
	params  *PublisherServerMockSetMessagesFilterParams
	results *PublisherServerMockSetMessagesFilterResults
	Counter uint64
}

// PublisherServerMockSetMessagesFilterParams contains parameters of the PublisherServer.SetMessagesFilter
type PublisherServerMockSetMessagesFilterParams struct {
	ctx context.Context
	mp1 *mm_introproto.MessageFilterByType
}

// PublisherServerMockSetMessagesFilterResults contains results of the PublisherServer.SetMessagesFilter
type PublisherServerMockSetMessagesFilterResults struct {
	mp2 *mm_introproto.MessageFilterByType
	err error
}

// Expect sets up expected params for PublisherServer.SetMessagesFilter
func (mmSetMessagesFilter *mPublisherServerMockSetMessagesFilter) Expect(ctx context.Context, mp1 *mm_introproto.MessageFilterByType) *mPublisherServerMockSetMessagesFilter {
	if mmSetMessagesFilter.mock.funcSetMessagesFilter != nil {
		mmSetMessagesFilter.mock.t.Fatalf("PublisherServerMock.SetMessagesFilter mock is already set by Set")
	}

	if mmSetMessagesFilter.defaultExpectation == nil {
		mmSetMessagesFilter.defaultExpectation = &PublisherServerMockSetMessagesFilterExpectation{}
	}

	mmSetMessagesFilter.defaultExpectation.params = &PublisherServerMockSetMessagesFilterParams{ctx, mp1}
	for _, e := range mmSetMessagesFilter.expectations {
		if minimock.Equal(e.params, mmSetMessagesFilter.defaultExpectation.params) {
			mmSetMessagesFilter.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmSetMessagesFilter.defaultExpectation.params)
		}
	}

	return mmSetMessagesFilter
}

// Inspect accepts an inspector function that has same arguments as the PublisherServer.SetMessagesFilter
func (mmSetMessagesFilter *mPublisherServerMockSetMessagesFilter) Inspect(f func(ctx context.Context, mp1 *mm_introproto.MessageFilterByType)) *mPublisherServerMockSetMessagesFilter {
	if mmSetMessagesFilter.mock.inspectFuncSetMessagesFilter != nil {
		mmSetMessagesFilter.mock.t.Fatalf("Inspect function is already set for PublisherServerMock.SetMessagesFilter")
	}

	mmSetMessagesFilter.mock.inspectFuncSetMessagesFilter = f

	return mmSetMessagesFilter
}

// Return sets up results that will be returned by PublisherServer.SetMessagesFilter
func (mmSetMessagesFilter *mPublisherServerMockSetMessagesFilter) Return(mp2 *mm_introproto.MessageFilterByType, err error) *PublisherServerMock {
	if mmSetMessagesFilter.mock.funcSetMessagesFilter != nil {
		mmSetMessagesFilter.mock.t.Fatalf("PublisherServerMock.SetMessagesFilter mock is already set by Set")
	}

	if mmSetMessagesFilter.defaultExpectation == nil {
		mmSetMessagesFilter.defaultExpectation = &PublisherServerMockSetMessagesFilterExpectation{mock: mmSetMessagesFilter.mock}
	}
	mmSetMessagesFilter.defaultExpectation.results = &PublisherServerMockSetMessagesFilterResults{mp2, err}
	return mmSetMessagesFilter.mock
}

//Set uses given function f to mock the PublisherServer.SetMessagesFilter method
func (mmSetMessagesFilter *mPublisherServerMockSetMessagesFilter) Set(f func(ctx context.Context, mp1 *mm_introproto.MessageFilterByType) (mp2 *mm_introproto.MessageFilterByType, err error)) *PublisherServerMock {
	if mmSetMessagesFilter.defaultExpectation != nil {
		mmSetMessagesFilter.mock.t.Fatalf("Default expectation is already set for the PublisherServer.SetMessagesFilter method")
	}

	if len(mmSetMessagesFilter.expectations) > 0 {
		mmSetMessagesFilter.mock.t.Fatalf("Some expectations are already set for the PublisherServer.SetMessagesFilter method")
	}

	mmSetMessagesFilter.mock.funcSetMessagesFilter = f
	return mmSetMessagesFilter.mock
}

// When sets expectation for the PublisherServer.SetMessagesFilter which will trigger the result defined by the following
// Then helper
func (mmSetMessagesFilter *mPublisherServerMockSetMessagesFilter) When(ctx context.Context, mp1 *mm_introproto.MessageFilterByType) *PublisherServerMockSetMessagesFilterExpectation {
	if mmSetMessagesFilter.mock.funcSetMessagesFilter != nil {
		mmSetMessagesFilter.mock.t.Fatalf("PublisherServerMock.SetMessagesFilter mock is already set by Set")
	}

	expectation := &PublisherServerMockSetMessagesFilterExpectation{
		mock:   mmSetMessagesFilter.mock,
		params: &PublisherServerMockSetMessagesFilterParams{ctx, mp1},
	}
	mmSetMessagesFilter.expectations = append(mmSetMessagesFilter.expectations, expectation)
	return expectation
}

// Then sets up PublisherServer.SetMessagesFilter return parameters for the expectation previously defined by the When method
func (e *PublisherServerMockSetMessagesFilterExpectation) Then(mp2 *mm_introproto.MessageFilterByType, err error) *PublisherServerMock {
	e.results = &PublisherServerMockSetMessagesFilterResults{mp2, err}
	return e.mock
}

// SetMessagesFilter implements introproto.PublisherServer
func (mmSetMessagesFilter *PublisherServerMock) SetMessagesFilter(ctx context.Context, mp1 *mm_introproto.MessageFilterByType) (mp2 *mm_introproto.MessageFilterByType, err error) {
	mm_atomic.AddUint64(&mmSetMessagesFilter.beforeSetMessagesFilterCounter, 1)
	defer mm_atomic.AddUint64(&mmSetMessagesFilter.afterSetMessagesFilterCounter, 1)

	if mmSetMessagesFilter.inspectFuncSetMessagesFilter != nil {
		mmSetMessagesFilter.inspectFuncSetMessagesFilter(ctx, mp1)
	}

	mm_params := &PublisherServerMockSetMessagesFilterParams{ctx, mp1}

	// Record call args
	mmSetMessagesFilter.SetMessagesFilterMock.mutex.Lock()
	mmSetMessagesFilter.SetMessagesFilterMock.callArgs = append(mmSetMessagesFilter.SetMessagesFilterMock.callArgs, mm_params)
	mmSetMessagesFilter.SetMessagesFilterMock.mutex.Unlock()

	for _, e := range mmSetMessagesFilter.SetMessagesFilterMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.mp2, e.results.err
		}
	}

	if mmSetMessagesFilter.SetMessagesFilterMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmSetMessagesFilter.SetMessagesFilterMock.defaultExpectation.Counter, 1)
		mm_want := mmSetMessagesFilter.SetMessagesFilterMock.defaultExpectation.params
		mm_got := PublisherServerMockSetMessagesFilterParams{ctx, mp1}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmSetMessagesFilter.t.Errorf("PublisherServerMock.SetMessagesFilter got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmSetMessagesFilter.SetMessagesFilterMock.defaultExpectation.results
		if mm_results == nil {
			mmSetMessagesFilter.t.Fatal("No results are set for the PublisherServerMock.SetMessagesFilter")
		}
		return (*mm_results).mp2, (*mm_results).err
	}
	if mmSetMessagesFilter.funcSetMessagesFilter != nil {
		return mmSetMessagesFilter.funcSetMessagesFilter(ctx, mp1)
	}
	mmSetMessagesFilter.t.Fatalf("Unexpected call to PublisherServerMock.SetMessagesFilter. %v %v", ctx, mp1)
	return
}

// SetMessagesFilterAfterCounter returns a count of finished PublisherServerMock.SetMessagesFilter invocations
func (mmSetMessagesFilter *PublisherServerMock) SetMessagesFilterAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSetMessagesFilter.afterSetMessagesFilterCounter)
}

// SetMessagesFilterBeforeCounter returns a count of PublisherServerMock.SetMessagesFilter invocations
func (mmSetMessagesFilter *PublisherServerMock) SetMessagesFilterBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSetMessagesFilter.beforeSetMessagesFilterCounter)
}

// Calls returns a list of arguments used in each call to PublisherServerMock.SetMessagesFilter.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmSetMessagesFilter *mPublisherServerMockSetMessagesFilter) Calls() []*PublisherServerMockSetMessagesFilterParams {
	mmSetMessagesFilter.mutex.RLock()

	argCopy := make([]*PublisherServerMockSetMessagesFilterParams, len(mmSetMessagesFilter.callArgs))
	copy(argCopy, mmSetMessagesFilter.callArgs)

	mmSetMessagesFilter.mutex.RUnlock()

	return argCopy
}

// MinimockSetMessagesFilterDone returns true if the count of the SetMessagesFilter invocations corresponds
// the number of defined expectations
func (m *PublisherServerMock) MinimockSetMessagesFilterDone() bool {
	for _, e := range m.SetMessagesFilterMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetMessagesFilterMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetMessagesFilterCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSetMessagesFilter != nil && mm_atomic.LoadUint64(&m.afterSetMessagesFilterCounter) < 1 {
		return false
	}
	return true
}

// MinimockSetMessagesFilterInspect logs each unmet expectation
func (m *PublisherServerMock) MinimockSetMessagesFilterInspect() {
	for _, e := range m.SetMessagesFilterMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to PublisherServerMock.SetMessagesFilter with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetMessagesFilterMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetMessagesFilterCounter) < 1 {
		if m.SetMessagesFilterMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to PublisherServerMock.SetMessagesFilter")
		} else {
			m.t.Errorf("Expected call to PublisherServerMock.SetMessagesFilter with params: %#v", *m.SetMessagesFilterMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSetMessagesFilter != nil && mm_atomic.LoadUint64(&m.afterSetMessagesFilterCounter) < 1 {
		m.t.Error("Expected call to PublisherServerMock.SetMessagesFilter")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *PublisherServerMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockGetMessagesFiltersInspect()

		m.MinimockGetMessagesStatInspect()

		m.MinimockSetMessagesFilterInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *PublisherServerMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *PublisherServerMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockGetMessagesFiltersDone() &&
		m.MinimockGetMessagesStatDone() &&
		m.MinimockSetMessagesFilterDone()
}
