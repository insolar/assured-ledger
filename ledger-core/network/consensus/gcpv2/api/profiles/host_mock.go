package profiles

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

// HostMock implements Host
type HostMock struct {
	t minimock.Tester

	funcGetDefaultEndpoint          func() (o1 endpoints.Outbound)
	inspectFuncGetDefaultEndpoint   func()
	afterGetDefaultEndpointCounter  uint64
	beforeGetDefaultEndpointCounter uint64
	GetDefaultEndpointMock          mHostMockGetDefaultEndpoint

	funcGetPublicKeyStore          func() (p1 cryptkit.PublicKeyStore)
	inspectFuncGetPublicKeyStore   func()
	afterGetPublicKeyStoreCounter  uint64
	beforeGetPublicKeyStoreCounter uint64
	GetPublicKeyStoreMock          mHostMockGetPublicKeyStore

	funcIsAcceptableHost          func(from endpoints.Inbound) (b1 bool)
	inspectFuncIsAcceptableHost   func(from endpoints.Inbound)
	afterIsAcceptableHostCounter  uint64
	beforeIsAcceptableHostCounter uint64
	IsAcceptableHostMock          mHostMockIsAcceptableHost
}

// NewHostMock returns a mock for Host
func NewHostMock(t minimock.Tester) *HostMock {
	m := &HostMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.GetDefaultEndpointMock = mHostMockGetDefaultEndpoint{mock: m}

	m.GetPublicKeyStoreMock = mHostMockGetPublicKeyStore{mock: m}

	m.IsAcceptableHostMock = mHostMockIsAcceptableHost{mock: m}
	m.IsAcceptableHostMock.callArgs = []*HostMockIsAcceptableHostParams{}

	return m
}

type mHostMockGetDefaultEndpoint struct {
	mock               *HostMock
	defaultExpectation *HostMockGetDefaultEndpointExpectation
	expectations       []*HostMockGetDefaultEndpointExpectation
}

// HostMockGetDefaultEndpointExpectation specifies expectation struct of the Host.GetDefaultEndpoint
type HostMockGetDefaultEndpointExpectation struct {
	mock *HostMock

	results *HostMockGetDefaultEndpointResults
	Counter uint64
}

// HostMockGetDefaultEndpointResults contains results of the Host.GetDefaultEndpoint
type HostMockGetDefaultEndpointResults struct {
	o1 endpoints.Outbound
}

// Expect sets up expected params for Host.GetDefaultEndpoint
func (mmGetDefaultEndpoint *mHostMockGetDefaultEndpoint) Expect() *mHostMockGetDefaultEndpoint {
	if mmGetDefaultEndpoint.mock.funcGetDefaultEndpoint != nil {
		mmGetDefaultEndpoint.mock.t.Fatalf("HostMock.GetDefaultEndpoint mock is already set by Set")
	}

	if mmGetDefaultEndpoint.defaultExpectation == nil {
		mmGetDefaultEndpoint.defaultExpectation = &HostMockGetDefaultEndpointExpectation{}
	}

	return mmGetDefaultEndpoint
}

// Inspect accepts an inspector function that has same arguments as the Host.GetDefaultEndpoint
func (mmGetDefaultEndpoint *mHostMockGetDefaultEndpoint) Inspect(f func()) *mHostMockGetDefaultEndpoint {
	if mmGetDefaultEndpoint.mock.inspectFuncGetDefaultEndpoint != nil {
		mmGetDefaultEndpoint.mock.t.Fatalf("Inspect function is already set for HostMock.GetDefaultEndpoint")
	}

	mmGetDefaultEndpoint.mock.inspectFuncGetDefaultEndpoint = f

	return mmGetDefaultEndpoint
}

// Return sets up results that will be returned by Host.GetDefaultEndpoint
func (mmGetDefaultEndpoint *mHostMockGetDefaultEndpoint) Return(o1 endpoints.Outbound) *HostMock {
	if mmGetDefaultEndpoint.mock.funcGetDefaultEndpoint != nil {
		mmGetDefaultEndpoint.mock.t.Fatalf("HostMock.GetDefaultEndpoint mock is already set by Set")
	}

	if mmGetDefaultEndpoint.defaultExpectation == nil {
		mmGetDefaultEndpoint.defaultExpectation = &HostMockGetDefaultEndpointExpectation{mock: mmGetDefaultEndpoint.mock}
	}
	mmGetDefaultEndpoint.defaultExpectation.results = &HostMockGetDefaultEndpointResults{o1}
	return mmGetDefaultEndpoint.mock
}

//Set uses given function f to mock the Host.GetDefaultEndpoint method
func (mmGetDefaultEndpoint *mHostMockGetDefaultEndpoint) Set(f func() (o1 endpoints.Outbound)) *HostMock {
	if mmGetDefaultEndpoint.defaultExpectation != nil {
		mmGetDefaultEndpoint.mock.t.Fatalf("Default expectation is already set for the Host.GetDefaultEndpoint method")
	}

	if len(mmGetDefaultEndpoint.expectations) > 0 {
		mmGetDefaultEndpoint.mock.t.Fatalf("Some expectations are already set for the Host.GetDefaultEndpoint method")
	}

	mmGetDefaultEndpoint.mock.funcGetDefaultEndpoint = f
	return mmGetDefaultEndpoint.mock
}

// GetDefaultEndpoint implements Host
func (mmGetDefaultEndpoint *HostMock) GetDefaultEndpoint() (o1 endpoints.Outbound) {
	mm_atomic.AddUint64(&mmGetDefaultEndpoint.beforeGetDefaultEndpointCounter, 1)
	defer mm_atomic.AddUint64(&mmGetDefaultEndpoint.afterGetDefaultEndpointCounter, 1)

	if mmGetDefaultEndpoint.inspectFuncGetDefaultEndpoint != nil {
		mmGetDefaultEndpoint.inspectFuncGetDefaultEndpoint()
	}

	if mmGetDefaultEndpoint.GetDefaultEndpointMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetDefaultEndpoint.GetDefaultEndpointMock.defaultExpectation.Counter, 1)

		mm_results := mmGetDefaultEndpoint.GetDefaultEndpointMock.defaultExpectation.results
		if mm_results == nil {
			mmGetDefaultEndpoint.t.Fatal("No results are set for the HostMock.GetDefaultEndpoint")
		}
		return (*mm_results).o1
	}
	if mmGetDefaultEndpoint.funcGetDefaultEndpoint != nil {
		return mmGetDefaultEndpoint.funcGetDefaultEndpoint()
	}
	mmGetDefaultEndpoint.t.Fatalf("Unexpected call to HostMock.GetDefaultEndpoint.")
	return
}

// GetDefaultEndpointAfterCounter returns a count of finished HostMock.GetDefaultEndpoint invocations
func (mmGetDefaultEndpoint *HostMock) GetDefaultEndpointAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetDefaultEndpoint.afterGetDefaultEndpointCounter)
}

// GetDefaultEndpointBeforeCounter returns a count of HostMock.GetDefaultEndpoint invocations
func (mmGetDefaultEndpoint *HostMock) GetDefaultEndpointBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetDefaultEndpoint.beforeGetDefaultEndpointCounter)
}

// MinimockGetDefaultEndpointDone returns true if the count of the GetDefaultEndpoint invocations corresponds
// the number of defined expectations
func (m *HostMock) MinimockGetDefaultEndpointDone() bool {
	for _, e := range m.GetDefaultEndpointMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetDefaultEndpointMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetDefaultEndpointCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetDefaultEndpoint != nil && mm_atomic.LoadUint64(&m.afterGetDefaultEndpointCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetDefaultEndpointInspect logs each unmet expectation
func (m *HostMock) MinimockGetDefaultEndpointInspect() {
	for _, e := range m.GetDefaultEndpointMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to HostMock.GetDefaultEndpoint")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetDefaultEndpointMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetDefaultEndpointCounter) < 1 {
		m.t.Error("Expected call to HostMock.GetDefaultEndpoint")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetDefaultEndpoint != nil && mm_atomic.LoadUint64(&m.afterGetDefaultEndpointCounter) < 1 {
		m.t.Error("Expected call to HostMock.GetDefaultEndpoint")
	}
}

type mHostMockGetPublicKeyStore struct {
	mock               *HostMock
	defaultExpectation *HostMockGetPublicKeyStoreExpectation
	expectations       []*HostMockGetPublicKeyStoreExpectation
}

// HostMockGetPublicKeyStoreExpectation specifies expectation struct of the Host.GetPublicKeyStore
type HostMockGetPublicKeyStoreExpectation struct {
	mock *HostMock

	results *HostMockGetPublicKeyStoreResults
	Counter uint64
}

// HostMockGetPublicKeyStoreResults contains results of the Host.GetPublicKeyStore
type HostMockGetPublicKeyStoreResults struct {
	p1 cryptkit.PublicKeyStore
}

// Expect sets up expected params for Host.GetPublicKeyStore
func (mmGetPublicKeyStore *mHostMockGetPublicKeyStore) Expect() *mHostMockGetPublicKeyStore {
	if mmGetPublicKeyStore.mock.funcGetPublicKeyStore != nil {
		mmGetPublicKeyStore.mock.t.Fatalf("HostMock.GetPublicKeyStore mock is already set by Set")
	}

	if mmGetPublicKeyStore.defaultExpectation == nil {
		mmGetPublicKeyStore.defaultExpectation = &HostMockGetPublicKeyStoreExpectation{}
	}

	return mmGetPublicKeyStore
}

// Inspect accepts an inspector function that has same arguments as the Host.GetPublicKeyStore
func (mmGetPublicKeyStore *mHostMockGetPublicKeyStore) Inspect(f func()) *mHostMockGetPublicKeyStore {
	if mmGetPublicKeyStore.mock.inspectFuncGetPublicKeyStore != nil {
		mmGetPublicKeyStore.mock.t.Fatalf("Inspect function is already set for HostMock.GetPublicKeyStore")
	}

	mmGetPublicKeyStore.mock.inspectFuncGetPublicKeyStore = f

	return mmGetPublicKeyStore
}

// Return sets up results that will be returned by Host.GetPublicKeyStore
func (mmGetPublicKeyStore *mHostMockGetPublicKeyStore) Return(p1 cryptkit.PublicKeyStore) *HostMock {
	if mmGetPublicKeyStore.mock.funcGetPublicKeyStore != nil {
		mmGetPublicKeyStore.mock.t.Fatalf("HostMock.GetPublicKeyStore mock is already set by Set")
	}

	if mmGetPublicKeyStore.defaultExpectation == nil {
		mmGetPublicKeyStore.defaultExpectation = &HostMockGetPublicKeyStoreExpectation{mock: mmGetPublicKeyStore.mock}
	}
	mmGetPublicKeyStore.defaultExpectation.results = &HostMockGetPublicKeyStoreResults{p1}
	return mmGetPublicKeyStore.mock
}

//Set uses given function f to mock the Host.GetPublicKeyStore method
func (mmGetPublicKeyStore *mHostMockGetPublicKeyStore) Set(f func() (p1 cryptkit.PublicKeyStore)) *HostMock {
	if mmGetPublicKeyStore.defaultExpectation != nil {
		mmGetPublicKeyStore.mock.t.Fatalf("Default expectation is already set for the Host.GetPublicKeyStore method")
	}

	if len(mmGetPublicKeyStore.expectations) > 0 {
		mmGetPublicKeyStore.mock.t.Fatalf("Some expectations are already set for the Host.GetPublicKeyStore method")
	}

	mmGetPublicKeyStore.mock.funcGetPublicKeyStore = f
	return mmGetPublicKeyStore.mock
}

// GetPublicKeyStore implements Host
func (mmGetPublicKeyStore *HostMock) GetPublicKeyStore() (p1 cryptkit.PublicKeyStore) {
	mm_atomic.AddUint64(&mmGetPublicKeyStore.beforeGetPublicKeyStoreCounter, 1)
	defer mm_atomic.AddUint64(&mmGetPublicKeyStore.afterGetPublicKeyStoreCounter, 1)

	if mmGetPublicKeyStore.inspectFuncGetPublicKeyStore != nil {
		mmGetPublicKeyStore.inspectFuncGetPublicKeyStore()
	}

	if mmGetPublicKeyStore.GetPublicKeyStoreMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetPublicKeyStore.GetPublicKeyStoreMock.defaultExpectation.Counter, 1)

		mm_results := mmGetPublicKeyStore.GetPublicKeyStoreMock.defaultExpectation.results
		if mm_results == nil {
			mmGetPublicKeyStore.t.Fatal("No results are set for the HostMock.GetPublicKeyStore")
		}
		return (*mm_results).p1
	}
	if mmGetPublicKeyStore.funcGetPublicKeyStore != nil {
		return mmGetPublicKeyStore.funcGetPublicKeyStore()
	}
	mmGetPublicKeyStore.t.Fatalf("Unexpected call to HostMock.GetPublicKeyStore.")
	return
}

// GetPublicKeyStoreAfterCounter returns a count of finished HostMock.GetPublicKeyStore invocations
func (mmGetPublicKeyStore *HostMock) GetPublicKeyStoreAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetPublicKeyStore.afterGetPublicKeyStoreCounter)
}

// GetPublicKeyStoreBeforeCounter returns a count of HostMock.GetPublicKeyStore invocations
func (mmGetPublicKeyStore *HostMock) GetPublicKeyStoreBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetPublicKeyStore.beforeGetPublicKeyStoreCounter)
}

// MinimockGetPublicKeyStoreDone returns true if the count of the GetPublicKeyStore invocations corresponds
// the number of defined expectations
func (m *HostMock) MinimockGetPublicKeyStoreDone() bool {
	for _, e := range m.GetPublicKeyStoreMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetPublicKeyStoreMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetPublicKeyStoreCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetPublicKeyStore != nil && mm_atomic.LoadUint64(&m.afterGetPublicKeyStoreCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetPublicKeyStoreInspect logs each unmet expectation
func (m *HostMock) MinimockGetPublicKeyStoreInspect() {
	for _, e := range m.GetPublicKeyStoreMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to HostMock.GetPublicKeyStore")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetPublicKeyStoreMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetPublicKeyStoreCounter) < 1 {
		m.t.Error("Expected call to HostMock.GetPublicKeyStore")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetPublicKeyStore != nil && mm_atomic.LoadUint64(&m.afterGetPublicKeyStoreCounter) < 1 {
		m.t.Error("Expected call to HostMock.GetPublicKeyStore")
	}
}

type mHostMockIsAcceptableHost struct {
	mock               *HostMock
	defaultExpectation *HostMockIsAcceptableHostExpectation
	expectations       []*HostMockIsAcceptableHostExpectation

	callArgs []*HostMockIsAcceptableHostParams
	mutex    sync.RWMutex
}

// HostMockIsAcceptableHostExpectation specifies expectation struct of the Host.IsAcceptableHost
type HostMockIsAcceptableHostExpectation struct {
	mock    *HostMock
	params  *HostMockIsAcceptableHostParams
	results *HostMockIsAcceptableHostResults
	Counter uint64
}

// HostMockIsAcceptableHostParams contains parameters of the Host.IsAcceptableHost
type HostMockIsAcceptableHostParams struct {
	from endpoints.Inbound
}

// HostMockIsAcceptableHostResults contains results of the Host.IsAcceptableHost
type HostMockIsAcceptableHostResults struct {
	b1 bool
}

// Expect sets up expected params for Host.IsAcceptableHost
func (mmIsAcceptableHost *mHostMockIsAcceptableHost) Expect(from endpoints.Inbound) *mHostMockIsAcceptableHost {
	if mmIsAcceptableHost.mock.funcIsAcceptableHost != nil {
		mmIsAcceptableHost.mock.t.Fatalf("HostMock.IsAcceptableHost mock is already set by Set")
	}

	if mmIsAcceptableHost.defaultExpectation == nil {
		mmIsAcceptableHost.defaultExpectation = &HostMockIsAcceptableHostExpectation{}
	}

	mmIsAcceptableHost.defaultExpectation.params = &HostMockIsAcceptableHostParams{from}
	for _, e := range mmIsAcceptableHost.expectations {
		if minimock.Equal(e.params, mmIsAcceptableHost.defaultExpectation.params) {
			mmIsAcceptableHost.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmIsAcceptableHost.defaultExpectation.params)
		}
	}

	return mmIsAcceptableHost
}

// Inspect accepts an inspector function that has same arguments as the Host.IsAcceptableHost
func (mmIsAcceptableHost *mHostMockIsAcceptableHost) Inspect(f func(from endpoints.Inbound)) *mHostMockIsAcceptableHost {
	if mmIsAcceptableHost.mock.inspectFuncIsAcceptableHost != nil {
		mmIsAcceptableHost.mock.t.Fatalf("Inspect function is already set for HostMock.IsAcceptableHost")
	}

	mmIsAcceptableHost.mock.inspectFuncIsAcceptableHost = f

	return mmIsAcceptableHost
}

// Return sets up results that will be returned by Host.IsAcceptableHost
func (mmIsAcceptableHost *mHostMockIsAcceptableHost) Return(b1 bool) *HostMock {
	if mmIsAcceptableHost.mock.funcIsAcceptableHost != nil {
		mmIsAcceptableHost.mock.t.Fatalf("HostMock.IsAcceptableHost mock is already set by Set")
	}

	if mmIsAcceptableHost.defaultExpectation == nil {
		mmIsAcceptableHost.defaultExpectation = &HostMockIsAcceptableHostExpectation{mock: mmIsAcceptableHost.mock}
	}
	mmIsAcceptableHost.defaultExpectation.results = &HostMockIsAcceptableHostResults{b1}
	return mmIsAcceptableHost.mock
}

//Set uses given function f to mock the Host.IsAcceptableHost method
func (mmIsAcceptableHost *mHostMockIsAcceptableHost) Set(f func(from endpoints.Inbound) (b1 bool)) *HostMock {
	if mmIsAcceptableHost.defaultExpectation != nil {
		mmIsAcceptableHost.mock.t.Fatalf("Default expectation is already set for the Host.IsAcceptableHost method")
	}

	if len(mmIsAcceptableHost.expectations) > 0 {
		mmIsAcceptableHost.mock.t.Fatalf("Some expectations are already set for the Host.IsAcceptableHost method")
	}

	mmIsAcceptableHost.mock.funcIsAcceptableHost = f
	return mmIsAcceptableHost.mock
}

// When sets expectation for the Host.IsAcceptableHost which will trigger the result defined by the following
// Then helper
func (mmIsAcceptableHost *mHostMockIsAcceptableHost) When(from endpoints.Inbound) *HostMockIsAcceptableHostExpectation {
	if mmIsAcceptableHost.mock.funcIsAcceptableHost != nil {
		mmIsAcceptableHost.mock.t.Fatalf("HostMock.IsAcceptableHost mock is already set by Set")
	}

	expectation := &HostMockIsAcceptableHostExpectation{
		mock:   mmIsAcceptableHost.mock,
		params: &HostMockIsAcceptableHostParams{from},
	}
	mmIsAcceptableHost.expectations = append(mmIsAcceptableHost.expectations, expectation)
	return expectation
}

// Then sets up Host.IsAcceptableHost return parameters for the expectation previously defined by the When method
func (e *HostMockIsAcceptableHostExpectation) Then(b1 bool) *HostMock {
	e.results = &HostMockIsAcceptableHostResults{b1}
	return e.mock
}

// IsAcceptableHost implements Host
func (mmIsAcceptableHost *HostMock) IsAcceptableHost(from endpoints.Inbound) (b1 bool) {
	mm_atomic.AddUint64(&mmIsAcceptableHost.beforeIsAcceptableHostCounter, 1)
	defer mm_atomic.AddUint64(&mmIsAcceptableHost.afterIsAcceptableHostCounter, 1)

	if mmIsAcceptableHost.inspectFuncIsAcceptableHost != nil {
		mmIsAcceptableHost.inspectFuncIsAcceptableHost(from)
	}

	mm_params := &HostMockIsAcceptableHostParams{from}

	// Record call args
	mmIsAcceptableHost.IsAcceptableHostMock.mutex.Lock()
	mmIsAcceptableHost.IsAcceptableHostMock.callArgs = append(mmIsAcceptableHost.IsAcceptableHostMock.callArgs, mm_params)
	mmIsAcceptableHost.IsAcceptableHostMock.mutex.Unlock()

	for _, e := range mmIsAcceptableHost.IsAcceptableHostMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.b1
		}
	}

	if mmIsAcceptableHost.IsAcceptableHostMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmIsAcceptableHost.IsAcceptableHostMock.defaultExpectation.Counter, 1)
		mm_want := mmIsAcceptableHost.IsAcceptableHostMock.defaultExpectation.params
		mm_got := HostMockIsAcceptableHostParams{from}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmIsAcceptableHost.t.Errorf("HostMock.IsAcceptableHost got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmIsAcceptableHost.IsAcceptableHostMock.defaultExpectation.results
		if mm_results == nil {
			mmIsAcceptableHost.t.Fatal("No results are set for the HostMock.IsAcceptableHost")
		}
		return (*mm_results).b1
	}
	if mmIsAcceptableHost.funcIsAcceptableHost != nil {
		return mmIsAcceptableHost.funcIsAcceptableHost(from)
	}
	mmIsAcceptableHost.t.Fatalf("Unexpected call to HostMock.IsAcceptableHost. %v", from)
	return
}

// IsAcceptableHostAfterCounter returns a count of finished HostMock.IsAcceptableHost invocations
func (mmIsAcceptableHost *HostMock) IsAcceptableHostAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmIsAcceptableHost.afterIsAcceptableHostCounter)
}

// IsAcceptableHostBeforeCounter returns a count of HostMock.IsAcceptableHost invocations
func (mmIsAcceptableHost *HostMock) IsAcceptableHostBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmIsAcceptableHost.beforeIsAcceptableHostCounter)
}

// Calls returns a list of arguments used in each call to HostMock.IsAcceptableHost.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmIsAcceptableHost *mHostMockIsAcceptableHost) Calls() []*HostMockIsAcceptableHostParams {
	mmIsAcceptableHost.mutex.RLock()

	argCopy := make([]*HostMockIsAcceptableHostParams, len(mmIsAcceptableHost.callArgs))
	copy(argCopy, mmIsAcceptableHost.callArgs)

	mmIsAcceptableHost.mutex.RUnlock()

	return argCopy
}

// MinimockIsAcceptableHostDone returns true if the count of the IsAcceptableHost invocations corresponds
// the number of defined expectations
func (m *HostMock) MinimockIsAcceptableHostDone() bool {
	for _, e := range m.IsAcceptableHostMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.IsAcceptableHostMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterIsAcceptableHostCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcIsAcceptableHost != nil && mm_atomic.LoadUint64(&m.afterIsAcceptableHostCounter) < 1 {
		return false
	}
	return true
}

// MinimockIsAcceptableHostInspect logs each unmet expectation
func (m *HostMock) MinimockIsAcceptableHostInspect() {
	for _, e := range m.IsAcceptableHostMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to HostMock.IsAcceptableHost with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.IsAcceptableHostMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterIsAcceptableHostCounter) < 1 {
		if m.IsAcceptableHostMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to HostMock.IsAcceptableHost")
		} else {
			m.t.Errorf("Expected call to HostMock.IsAcceptableHost with params: %#v", *m.IsAcceptableHostMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcIsAcceptableHost != nil && mm_atomic.LoadUint64(&m.afterIsAcceptableHostCounter) < 1 {
		m.t.Error("Expected call to HostMock.IsAcceptableHost")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *HostMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockGetDefaultEndpointInspect()

		m.MinimockGetPublicKeyStoreInspect()

		m.MinimockIsAcceptableHostInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *HostMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *HostMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockGetDefaultEndpointDone() &&
		m.MinimockGetPublicKeyStoreDone() &&
		m.MinimockIsAcceptableHostDone()
}
