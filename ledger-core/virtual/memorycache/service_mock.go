package memorycache

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

// ServiceMock implements Service
type ServiceMock struct {
	t minimock.Tester

	funcGet          func(ctx context.Context, objectReference reference.Global) (o1 descriptor.Object, err error)
	inspectFuncGet   func(ctx context.Context, objectReference reference.Global)
	afterGetCounter  uint64
	beforeGetCounter uint64
	GetMock          mServiceMockGet

	funcSet          func(ctx context.Context, objectDescriptor descriptor.Object) (g1 reference.Global, err error)
	inspectFuncSet   func(ctx context.Context, objectDescriptor descriptor.Object)
	afterSetCounter  uint64
	beforeSetCounter uint64
	SetMock          mServiceMockSet
}

// NewServiceMock returns a mock for Service
func NewServiceMock(t minimock.Tester) *ServiceMock {
	m := &ServiceMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.GetMock = mServiceMockGet{mock: m}
	m.GetMock.callArgs = []*ServiceMockGetParams{}

	m.SetMock = mServiceMockSet{mock: m}
	m.SetMock.callArgs = []*ServiceMockSetParams{}

	return m
}

type mServiceMockGet struct {
	mock               *ServiceMock
	defaultExpectation *ServiceMockGetExpectation
	expectations       []*ServiceMockGetExpectation

	callArgs []*ServiceMockGetParams
	mutex    sync.RWMutex
}

// ServiceMockGetExpectation specifies expectation struct of the Service.Get
type ServiceMockGetExpectation struct {
	mock    *ServiceMock
	params  *ServiceMockGetParams
	results *ServiceMockGetResults
	Counter uint64
}

// ServiceMockGetParams contains parameters of the Service.Get
type ServiceMockGetParams struct {
	ctx             context.Context
	objectReference reference.Global
}

// ServiceMockGetResults contains results of the Service.Get
type ServiceMockGetResults struct {
	o1  descriptor.Object
	err error
}

// Expect sets up expected params for Service.Get
func (mmGet *mServiceMockGet) Expect(ctx context.Context, objectReference reference.Global) *mServiceMockGet {
	if mmGet.mock.funcGet != nil {
		mmGet.mock.t.Fatalf("ServiceMock.Get mock is already set by Set")
	}

	if mmGet.defaultExpectation == nil {
		mmGet.defaultExpectation = &ServiceMockGetExpectation{}
	}

	mmGet.defaultExpectation.params = &ServiceMockGetParams{ctx, objectReference}
	for _, e := range mmGet.expectations {
		if minimock.Equal(e.params, mmGet.defaultExpectation.params) {
			mmGet.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmGet.defaultExpectation.params)
		}
	}

	return mmGet
}

// Inspect accepts an inspector function that has same arguments as the Service.Get
func (mmGet *mServiceMockGet) Inspect(f func(ctx context.Context, objectReference reference.Global)) *mServiceMockGet {
	if mmGet.mock.inspectFuncGet != nil {
		mmGet.mock.t.Fatalf("Inspect function is already set for ServiceMock.Get")
	}

	mmGet.mock.inspectFuncGet = f

	return mmGet
}

// Return sets up results that will be returned by Service.Get
func (mmGet *mServiceMockGet) Return(o1 descriptor.Object, err error) *ServiceMock {
	if mmGet.mock.funcGet != nil {
		mmGet.mock.t.Fatalf("ServiceMock.Get mock is already set by Set")
	}

	if mmGet.defaultExpectation == nil {
		mmGet.defaultExpectation = &ServiceMockGetExpectation{mock: mmGet.mock}
	}
	mmGet.defaultExpectation.results = &ServiceMockGetResults{o1, err}
	return mmGet.mock
}

//Set uses given function f to mock the Service.Get method
func (mmGet *mServiceMockGet) Set(f func(ctx context.Context, objectReference reference.Global) (o1 descriptor.Object, err error)) *ServiceMock {
	if mmGet.defaultExpectation != nil {
		mmGet.mock.t.Fatalf("Default expectation is already set for the Service.Get method")
	}

	if len(mmGet.expectations) > 0 {
		mmGet.mock.t.Fatalf("Some expectations are already set for the Service.Get method")
	}

	mmGet.mock.funcGet = f
	return mmGet.mock
}

// When sets expectation for the Service.Get which will trigger the result defined by the following
// Then helper
func (mmGet *mServiceMockGet) When(ctx context.Context, objectReference reference.Global) *ServiceMockGetExpectation {
	if mmGet.mock.funcGet != nil {
		mmGet.mock.t.Fatalf("ServiceMock.Get mock is already set by Set")
	}

	expectation := &ServiceMockGetExpectation{
		mock:   mmGet.mock,
		params: &ServiceMockGetParams{ctx, objectReference},
	}
	mmGet.expectations = append(mmGet.expectations, expectation)
	return expectation
}

// Then sets up Service.Get return parameters for the expectation previously defined by the When method
func (e *ServiceMockGetExpectation) Then(o1 descriptor.Object, err error) *ServiceMock {
	e.results = &ServiceMockGetResults{o1, err}
	return e.mock
}

// Get implements Service
func (mmGet *ServiceMock) Get(ctx context.Context, objectReference reference.Global) (o1 descriptor.Object, err error) {
	mm_atomic.AddUint64(&mmGet.beforeGetCounter, 1)
	defer mm_atomic.AddUint64(&mmGet.afterGetCounter, 1)

	if mmGet.inspectFuncGet != nil {
		mmGet.inspectFuncGet(ctx, objectReference)
	}

	mm_params := &ServiceMockGetParams{ctx, objectReference}

	// Record call args
	mmGet.GetMock.mutex.Lock()
	mmGet.GetMock.callArgs = append(mmGet.GetMock.callArgs, mm_params)
	mmGet.GetMock.mutex.Unlock()

	for _, e := range mmGet.GetMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.o1, e.results.err
		}
	}

	if mmGet.GetMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGet.GetMock.defaultExpectation.Counter, 1)
		mm_want := mmGet.GetMock.defaultExpectation.params
		mm_got := ServiceMockGetParams{ctx, objectReference}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmGet.t.Errorf("ServiceMock.Get got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmGet.GetMock.defaultExpectation.results
		if mm_results == nil {
			mmGet.t.Fatal("No results are set for the ServiceMock.Get")
		}
		return (*mm_results).o1, (*mm_results).err
	}
	if mmGet.funcGet != nil {
		return mmGet.funcGet(ctx, objectReference)
	}
	mmGet.t.Fatalf("Unexpected call to ServiceMock.Get. %v %v", ctx, objectReference)
	return
}

// GetAfterCounter returns a count of finished ServiceMock.Get invocations
func (mmGet *ServiceMock) GetAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGet.afterGetCounter)
}

// GetBeforeCounter returns a count of ServiceMock.Get invocations
func (mmGet *ServiceMock) GetBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGet.beforeGetCounter)
}

// Calls returns a list of arguments used in each call to ServiceMock.Get.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmGet *mServiceMockGet) Calls() []*ServiceMockGetParams {
	mmGet.mutex.RLock()

	argCopy := make([]*ServiceMockGetParams, len(mmGet.callArgs))
	copy(argCopy, mmGet.callArgs)

	mmGet.mutex.RUnlock()

	return argCopy
}

// MinimockGetDone returns true if the count of the Get invocations corresponds
// the number of defined expectations
func (m *ServiceMock) MinimockGetDone() bool {
	for _, e := range m.GetMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGet != nil && mm_atomic.LoadUint64(&m.afterGetCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetInspect logs each unmet expectation
func (m *ServiceMock) MinimockGetInspect() {
	for _, e := range m.GetMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ServiceMock.Get with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetCounter) < 1 {
		if m.GetMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ServiceMock.Get")
		} else {
			m.t.Errorf("Expected call to ServiceMock.Get with params: %#v", *m.GetMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGet != nil && mm_atomic.LoadUint64(&m.afterGetCounter) < 1 {
		m.t.Error("Expected call to ServiceMock.Get")
	}
}

type mServiceMockSet struct {
	mock               *ServiceMock
	defaultExpectation *ServiceMockSetExpectation
	expectations       []*ServiceMockSetExpectation

	callArgs []*ServiceMockSetParams
	mutex    sync.RWMutex
}

// ServiceMockSetExpectation specifies expectation struct of the Service.Set
type ServiceMockSetExpectation struct {
	mock    *ServiceMock
	params  *ServiceMockSetParams
	results *ServiceMockSetResults
	Counter uint64
}

// ServiceMockSetParams contains parameters of the Service.Set
type ServiceMockSetParams struct {
	ctx              context.Context
	objectDescriptor descriptor.Object
}

// ServiceMockSetResults contains results of the Service.Set
type ServiceMockSetResults struct {
	g1  reference.Global
	err error
}

// Expect sets up expected params for Service.Set
func (mmSet *mServiceMockSet) Expect(ctx context.Context, objectDescriptor descriptor.Object) *mServiceMockSet {
	if mmSet.mock.funcSet != nil {
		mmSet.mock.t.Fatalf("ServiceMock.Set mock is already set by Set")
	}

	if mmSet.defaultExpectation == nil {
		mmSet.defaultExpectation = &ServiceMockSetExpectation{}
	}

	mmSet.defaultExpectation.params = &ServiceMockSetParams{ctx, objectDescriptor}
	for _, e := range mmSet.expectations {
		if minimock.Equal(e.params, mmSet.defaultExpectation.params) {
			mmSet.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmSet.defaultExpectation.params)
		}
	}

	return mmSet
}

// Inspect accepts an inspector function that has same arguments as the Service.Set
func (mmSet *mServiceMockSet) Inspect(f func(ctx context.Context, objectDescriptor descriptor.Object)) *mServiceMockSet {
	if mmSet.mock.inspectFuncSet != nil {
		mmSet.mock.t.Fatalf("Inspect function is already set for ServiceMock.Set")
	}

	mmSet.mock.inspectFuncSet = f

	return mmSet
}

// Return sets up results that will be returned by Service.Set
func (mmSet *mServiceMockSet) Return(g1 reference.Global, err error) *ServiceMock {
	if mmSet.mock.funcSet != nil {
		mmSet.mock.t.Fatalf("ServiceMock.Set mock is already set by Set")
	}

	if mmSet.defaultExpectation == nil {
		mmSet.defaultExpectation = &ServiceMockSetExpectation{mock: mmSet.mock}
	}
	mmSet.defaultExpectation.results = &ServiceMockSetResults{g1, err}
	return mmSet.mock
}

//Set uses given function f to mock the Service.Set method
func (mmSet *mServiceMockSet) Set(f func(ctx context.Context, objectDescriptor descriptor.Object) (g1 reference.Global, err error)) *ServiceMock {
	if mmSet.defaultExpectation != nil {
		mmSet.mock.t.Fatalf("Default expectation is already set for the Service.Set method")
	}

	if len(mmSet.expectations) > 0 {
		mmSet.mock.t.Fatalf("Some expectations are already set for the Service.Set method")
	}

	mmSet.mock.funcSet = f
	return mmSet.mock
}

// When sets expectation for the Service.Set which will trigger the result defined by the following
// Then helper
func (mmSet *mServiceMockSet) When(ctx context.Context, objectDescriptor descriptor.Object) *ServiceMockSetExpectation {
	if mmSet.mock.funcSet != nil {
		mmSet.mock.t.Fatalf("ServiceMock.Set mock is already set by Set")
	}

	expectation := &ServiceMockSetExpectation{
		mock:   mmSet.mock,
		params: &ServiceMockSetParams{ctx, objectDescriptor},
	}
	mmSet.expectations = append(mmSet.expectations, expectation)
	return expectation
}

// Then sets up Service.Set return parameters for the expectation previously defined by the When method
func (e *ServiceMockSetExpectation) Then(g1 reference.Global, err error) *ServiceMock {
	e.results = &ServiceMockSetResults{g1, err}
	return e.mock
}

// Set implements Service
func (mmSet *ServiceMock) Set(ctx context.Context, objectDescriptor descriptor.Object) (g1 reference.Global, err error) {
	mm_atomic.AddUint64(&mmSet.beforeSetCounter, 1)
	defer mm_atomic.AddUint64(&mmSet.afterSetCounter, 1)

	if mmSet.inspectFuncSet != nil {
		mmSet.inspectFuncSet(ctx, objectDescriptor)
	}

	mm_params := &ServiceMockSetParams{ctx, objectDescriptor}

	// Record call args
	mmSet.SetMock.mutex.Lock()
	mmSet.SetMock.callArgs = append(mmSet.SetMock.callArgs, mm_params)
	mmSet.SetMock.mutex.Unlock()

	for _, e := range mmSet.SetMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.g1, e.results.err
		}
	}

	if mmSet.SetMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmSet.SetMock.defaultExpectation.Counter, 1)
		mm_want := mmSet.SetMock.defaultExpectation.params
		mm_got := ServiceMockSetParams{ctx, objectDescriptor}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmSet.t.Errorf("ServiceMock.Set got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmSet.SetMock.defaultExpectation.results
		if mm_results == nil {
			mmSet.t.Fatal("No results are set for the ServiceMock.Set")
		}
		return (*mm_results).g1, (*mm_results).err
	}
	if mmSet.funcSet != nil {
		return mmSet.funcSet(ctx, objectDescriptor)
	}
	mmSet.t.Fatalf("Unexpected call to ServiceMock.Set. %v %v", ctx, objectDescriptor)
	return
}

// SetAfterCounter returns a count of finished ServiceMock.Set invocations
func (mmSet *ServiceMock) SetAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSet.afterSetCounter)
}

// SetBeforeCounter returns a count of ServiceMock.Set invocations
func (mmSet *ServiceMock) SetBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSet.beforeSetCounter)
}

// Calls returns a list of arguments used in each call to ServiceMock.Set.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmSet *mServiceMockSet) Calls() []*ServiceMockSetParams {
	mmSet.mutex.RLock()

	argCopy := make([]*ServiceMockSetParams, len(mmSet.callArgs))
	copy(argCopy, mmSet.callArgs)

	mmSet.mutex.RUnlock()

	return argCopy
}

// MinimockSetDone returns true if the count of the Set invocations corresponds
// the number of defined expectations
func (m *ServiceMock) MinimockSetDone() bool {
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
func (m *ServiceMock) MinimockSetInspect() {
	for _, e := range m.SetMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ServiceMock.Set with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetCounter) < 1 {
		if m.SetMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ServiceMock.Set")
		} else {
			m.t.Errorf("Expected call to ServiceMock.Set with params: %#v", *m.SetMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSet != nil && mm_atomic.LoadUint64(&m.afterSetCounter) < 1 {
		m.t.Error("Expected call to ServiceMock.Set")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *ServiceMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockGetInspect()

		m.MinimockSetInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *ServiceMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *ServiceMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockGetDone() &&
		m.MinimockSetDone()
}
