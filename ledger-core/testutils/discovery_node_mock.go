package testutils

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"crypto"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// DiscoveryNodeMock implements nodeinfo.DiscoveryNode
type DiscoveryNodeMock struct {
	t minimock.Tester

	funcGetHost          func() (s1 string)
	inspectFuncGetHost   func()
	afterGetHostCounter  uint64
	beforeGetHostCounter uint64
	GetHostMock          mDiscoveryNodeMockGetHost

	funcGetNodeRef          func() (g1 reference.Global)
	inspectFuncGetNodeRef   func()
	afterGetNodeRefCounter  uint64
	beforeGetNodeRefCounter uint64
	GetNodeRefMock          mDiscoveryNodeMockGetNodeRef

	funcGetPublicKey          func() (p1 crypto.PublicKey)
	inspectFuncGetPublicKey   func()
	afterGetPublicKeyCounter  uint64
	beforeGetPublicKeyCounter uint64
	GetPublicKeyMock          mDiscoveryNodeMockGetPublicKey
}

// NewDiscoveryNodeMock returns a mock for nodeinfo.DiscoveryNode
func NewDiscoveryNodeMock(t minimock.Tester) *DiscoveryNodeMock {
	m := &DiscoveryNodeMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.GetHostMock = mDiscoveryNodeMockGetHost{mock: m}

	m.GetNodeRefMock = mDiscoveryNodeMockGetNodeRef{mock: m}

	m.GetPublicKeyMock = mDiscoveryNodeMockGetPublicKey{mock: m}

	return m
}

type mDiscoveryNodeMockGetHost struct {
	mock               *DiscoveryNodeMock
	defaultExpectation *DiscoveryNodeMockGetHostExpectation
	expectations       []*DiscoveryNodeMockGetHostExpectation
}

// DiscoveryNodeMockGetHostExpectation specifies expectation struct of the DiscoveryNode.GetHost
type DiscoveryNodeMockGetHostExpectation struct {
	mock *DiscoveryNodeMock

	results *DiscoveryNodeMockGetHostResults
	Counter uint64
}

// DiscoveryNodeMockGetHostResults contains results of the DiscoveryNode.GetHost
type DiscoveryNodeMockGetHostResults struct {
	s1 string
}

// Expect sets up expected params for DiscoveryNode.GetHost
func (mmGetHost *mDiscoveryNodeMockGetHost) Expect() *mDiscoveryNodeMockGetHost {
	if mmGetHost.mock.funcGetHost != nil {
		mmGetHost.mock.t.Fatalf("DiscoveryNodeMock.GetHost mock is already set by Set")
	}

	if mmGetHost.defaultExpectation == nil {
		mmGetHost.defaultExpectation = &DiscoveryNodeMockGetHostExpectation{}
	}

	return mmGetHost
}

// Inspect accepts an inspector function that has same arguments as the DiscoveryNode.GetHost
func (mmGetHost *mDiscoveryNodeMockGetHost) Inspect(f func()) *mDiscoveryNodeMockGetHost {
	if mmGetHost.mock.inspectFuncGetHost != nil {
		mmGetHost.mock.t.Fatalf("Inspect function is already set for DiscoveryNodeMock.GetHost")
	}

	mmGetHost.mock.inspectFuncGetHost = f

	return mmGetHost
}

// Return sets up results that will be returned by DiscoveryNode.GetHost
func (mmGetHost *mDiscoveryNodeMockGetHost) Return(s1 string) *DiscoveryNodeMock {
	if mmGetHost.mock.funcGetHost != nil {
		mmGetHost.mock.t.Fatalf("DiscoveryNodeMock.GetHost mock is already set by Set")
	}

	if mmGetHost.defaultExpectation == nil {
		mmGetHost.defaultExpectation = &DiscoveryNodeMockGetHostExpectation{mock: mmGetHost.mock}
	}
	mmGetHost.defaultExpectation.results = &DiscoveryNodeMockGetHostResults{s1}
	return mmGetHost.mock
}

//Set uses given function f to mock the DiscoveryNode.GetHost method
func (mmGetHost *mDiscoveryNodeMockGetHost) Set(f func() (s1 string)) *DiscoveryNodeMock {
	if mmGetHost.defaultExpectation != nil {
		mmGetHost.mock.t.Fatalf("Default expectation is already set for the DiscoveryNode.GetHost method")
	}

	if len(mmGetHost.expectations) > 0 {
		mmGetHost.mock.t.Fatalf("Some expectations are already set for the DiscoveryNode.GetHost method")
	}

	mmGetHost.mock.funcGetHost = f
	return mmGetHost.mock
}

// GetHost implements nodeinfo.DiscoveryNode
func (mmGetHost *DiscoveryNodeMock) GetHost() (s1 string) {
	mm_atomic.AddUint64(&mmGetHost.beforeGetHostCounter, 1)
	defer mm_atomic.AddUint64(&mmGetHost.afterGetHostCounter, 1)

	if mmGetHost.inspectFuncGetHost != nil {
		mmGetHost.inspectFuncGetHost()
	}

	if mmGetHost.GetHostMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetHost.GetHostMock.defaultExpectation.Counter, 1)

		mm_results := mmGetHost.GetHostMock.defaultExpectation.results
		if mm_results == nil {
			mmGetHost.t.Fatal("No results are set for the DiscoveryNodeMock.GetHost")
		}
		return (*mm_results).s1
	}
	if mmGetHost.funcGetHost != nil {
		return mmGetHost.funcGetHost()
	}
	mmGetHost.t.Fatalf("Unexpected call to DiscoveryNodeMock.GetHost.")
	return
}

// GetHostAfterCounter returns a count of finished DiscoveryNodeMock.GetHost invocations
func (mmGetHost *DiscoveryNodeMock) GetHostAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetHost.afterGetHostCounter)
}

// GetHostBeforeCounter returns a count of DiscoveryNodeMock.GetHost invocations
func (mmGetHost *DiscoveryNodeMock) GetHostBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetHost.beforeGetHostCounter)
}

// MinimockGetHostDone returns true if the count of the GetHost invocations corresponds
// the number of defined expectations
func (m *DiscoveryNodeMock) MinimockGetHostDone() bool {
	for _, e := range m.GetHostMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetHostMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetHostCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetHost != nil && mm_atomic.LoadUint64(&m.afterGetHostCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetHostInspect logs each unmet expectation
func (m *DiscoveryNodeMock) MinimockGetHostInspect() {
	for _, e := range m.GetHostMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to DiscoveryNodeMock.GetHost")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetHostMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetHostCounter) < 1 {
		m.t.Error("Expected call to DiscoveryNodeMock.GetHost")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetHost != nil && mm_atomic.LoadUint64(&m.afterGetHostCounter) < 1 {
		m.t.Error("Expected call to DiscoveryNodeMock.GetHost")
	}
}

type mDiscoveryNodeMockGetNodeRef struct {
	mock               *DiscoveryNodeMock
	defaultExpectation *DiscoveryNodeMockGetNodeRefExpectation
	expectations       []*DiscoveryNodeMockGetNodeRefExpectation
}

// DiscoveryNodeMockGetNodeRefExpectation specifies expectation struct of the DiscoveryNode.GetNodeRef
type DiscoveryNodeMockGetNodeRefExpectation struct {
	mock *DiscoveryNodeMock

	results *DiscoveryNodeMockGetNodeRefResults
	Counter uint64
}

// DiscoveryNodeMockGetNodeRefResults contains results of the DiscoveryNode.GetNodeRef
type DiscoveryNodeMockGetNodeRefResults struct {
	g1 reference.Global
}

// Expect sets up expected params for DiscoveryNode.GetNodeRef
func (mmGetNodeRef *mDiscoveryNodeMockGetNodeRef) Expect() *mDiscoveryNodeMockGetNodeRef {
	if mmGetNodeRef.mock.funcGetNodeRef != nil {
		mmGetNodeRef.mock.t.Fatalf("DiscoveryNodeMock.GetNodeRef mock is already set by Set")
	}

	if mmGetNodeRef.defaultExpectation == nil {
		mmGetNodeRef.defaultExpectation = &DiscoveryNodeMockGetNodeRefExpectation{}
	}

	return mmGetNodeRef
}

// Inspect accepts an inspector function that has same arguments as the DiscoveryNode.GetNodeRef
func (mmGetNodeRef *mDiscoveryNodeMockGetNodeRef) Inspect(f func()) *mDiscoveryNodeMockGetNodeRef {
	if mmGetNodeRef.mock.inspectFuncGetNodeRef != nil {
		mmGetNodeRef.mock.t.Fatalf("Inspect function is already set for DiscoveryNodeMock.GetNodeRef")
	}

	mmGetNodeRef.mock.inspectFuncGetNodeRef = f

	return mmGetNodeRef
}

// Return sets up results that will be returned by DiscoveryNode.GetNodeRef
func (mmGetNodeRef *mDiscoveryNodeMockGetNodeRef) Return(g1 reference.Global) *DiscoveryNodeMock {
	if mmGetNodeRef.mock.funcGetNodeRef != nil {
		mmGetNodeRef.mock.t.Fatalf("DiscoveryNodeMock.GetNodeRef mock is already set by Set")
	}

	if mmGetNodeRef.defaultExpectation == nil {
		mmGetNodeRef.defaultExpectation = &DiscoveryNodeMockGetNodeRefExpectation{mock: mmGetNodeRef.mock}
	}
	mmGetNodeRef.defaultExpectation.results = &DiscoveryNodeMockGetNodeRefResults{g1}
	return mmGetNodeRef.mock
}

//Set uses given function f to mock the DiscoveryNode.GetNodeRef method
func (mmGetNodeRef *mDiscoveryNodeMockGetNodeRef) Set(f func() (g1 reference.Global)) *DiscoveryNodeMock {
	if mmGetNodeRef.defaultExpectation != nil {
		mmGetNodeRef.mock.t.Fatalf("Default expectation is already set for the DiscoveryNode.GetNodeRef method")
	}

	if len(mmGetNodeRef.expectations) > 0 {
		mmGetNodeRef.mock.t.Fatalf("Some expectations are already set for the DiscoveryNode.GetNodeRef method")
	}

	mmGetNodeRef.mock.funcGetNodeRef = f
	return mmGetNodeRef.mock
}

// GetNodeRef implements nodeinfo.DiscoveryNode
func (mmGetNodeRef *DiscoveryNodeMock) GetNodeRef() (g1 reference.Global) {
	mm_atomic.AddUint64(&mmGetNodeRef.beforeGetNodeRefCounter, 1)
	defer mm_atomic.AddUint64(&mmGetNodeRef.afterGetNodeRefCounter, 1)

	if mmGetNodeRef.inspectFuncGetNodeRef != nil {
		mmGetNodeRef.inspectFuncGetNodeRef()
	}

	if mmGetNodeRef.GetNodeRefMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetNodeRef.GetNodeRefMock.defaultExpectation.Counter, 1)

		mm_results := mmGetNodeRef.GetNodeRefMock.defaultExpectation.results
		if mm_results == nil {
			mmGetNodeRef.t.Fatal("No results are set for the DiscoveryNodeMock.GetNodeRef")
		}
		return (*mm_results).g1
	}
	if mmGetNodeRef.funcGetNodeRef != nil {
		return mmGetNodeRef.funcGetNodeRef()
	}
	mmGetNodeRef.t.Fatalf("Unexpected call to DiscoveryNodeMock.GetNodeRef.")
	return
}

// GetNodeRefAfterCounter returns a count of finished DiscoveryNodeMock.GetNodeRef invocations
func (mmGetNodeRef *DiscoveryNodeMock) GetNodeRefAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetNodeRef.afterGetNodeRefCounter)
}

// GetNodeRefBeforeCounter returns a count of DiscoveryNodeMock.GetNodeRef invocations
func (mmGetNodeRef *DiscoveryNodeMock) GetNodeRefBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetNodeRef.beforeGetNodeRefCounter)
}

// MinimockGetNodeRefDone returns true if the count of the GetNodeRef invocations corresponds
// the number of defined expectations
func (m *DiscoveryNodeMock) MinimockGetNodeRefDone() bool {
	for _, e := range m.GetNodeRefMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetNodeRefMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetNodeRefCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetNodeRef != nil && mm_atomic.LoadUint64(&m.afterGetNodeRefCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetNodeRefInspect logs each unmet expectation
func (m *DiscoveryNodeMock) MinimockGetNodeRefInspect() {
	for _, e := range m.GetNodeRefMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to DiscoveryNodeMock.GetNodeRef")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetNodeRefMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetNodeRefCounter) < 1 {
		m.t.Error("Expected call to DiscoveryNodeMock.GetNodeRef")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetNodeRef != nil && mm_atomic.LoadUint64(&m.afterGetNodeRefCounter) < 1 {
		m.t.Error("Expected call to DiscoveryNodeMock.GetNodeRef")
	}
}

type mDiscoveryNodeMockGetPublicKey struct {
	mock               *DiscoveryNodeMock
	defaultExpectation *DiscoveryNodeMockGetPublicKeyExpectation
	expectations       []*DiscoveryNodeMockGetPublicKeyExpectation
}

// DiscoveryNodeMockGetPublicKeyExpectation specifies expectation struct of the DiscoveryNode.GetPublicKey
type DiscoveryNodeMockGetPublicKeyExpectation struct {
	mock *DiscoveryNodeMock

	results *DiscoveryNodeMockGetPublicKeyResults
	Counter uint64
}

// DiscoveryNodeMockGetPublicKeyResults contains results of the DiscoveryNode.GetPublicKey
type DiscoveryNodeMockGetPublicKeyResults struct {
	p1 crypto.PublicKey
}

// Expect sets up expected params for DiscoveryNode.GetPublicKey
func (mmGetPublicKey *mDiscoveryNodeMockGetPublicKey) Expect() *mDiscoveryNodeMockGetPublicKey {
	if mmGetPublicKey.mock.funcGetPublicKey != nil {
		mmGetPublicKey.mock.t.Fatalf("DiscoveryNodeMock.GetPublicKey mock is already set by Set")
	}

	if mmGetPublicKey.defaultExpectation == nil {
		mmGetPublicKey.defaultExpectation = &DiscoveryNodeMockGetPublicKeyExpectation{}
	}

	return mmGetPublicKey
}

// Inspect accepts an inspector function that has same arguments as the DiscoveryNode.GetPublicKey
func (mmGetPublicKey *mDiscoveryNodeMockGetPublicKey) Inspect(f func()) *mDiscoveryNodeMockGetPublicKey {
	if mmGetPublicKey.mock.inspectFuncGetPublicKey != nil {
		mmGetPublicKey.mock.t.Fatalf("Inspect function is already set for DiscoveryNodeMock.GetPublicKey")
	}

	mmGetPublicKey.mock.inspectFuncGetPublicKey = f

	return mmGetPublicKey
}

// Return sets up results that will be returned by DiscoveryNode.GetPublicKey
func (mmGetPublicKey *mDiscoveryNodeMockGetPublicKey) Return(p1 crypto.PublicKey) *DiscoveryNodeMock {
	if mmGetPublicKey.mock.funcGetPublicKey != nil {
		mmGetPublicKey.mock.t.Fatalf("DiscoveryNodeMock.GetPublicKey mock is already set by Set")
	}

	if mmGetPublicKey.defaultExpectation == nil {
		mmGetPublicKey.defaultExpectation = &DiscoveryNodeMockGetPublicKeyExpectation{mock: mmGetPublicKey.mock}
	}
	mmGetPublicKey.defaultExpectation.results = &DiscoveryNodeMockGetPublicKeyResults{p1}
	return mmGetPublicKey.mock
}

//Set uses given function f to mock the DiscoveryNode.GetPublicKey method
func (mmGetPublicKey *mDiscoveryNodeMockGetPublicKey) Set(f func() (p1 crypto.PublicKey)) *DiscoveryNodeMock {
	if mmGetPublicKey.defaultExpectation != nil {
		mmGetPublicKey.mock.t.Fatalf("Default expectation is already set for the DiscoveryNode.GetPublicKey method")
	}

	if len(mmGetPublicKey.expectations) > 0 {
		mmGetPublicKey.mock.t.Fatalf("Some expectations are already set for the DiscoveryNode.GetPublicKey method")
	}

	mmGetPublicKey.mock.funcGetPublicKey = f
	return mmGetPublicKey.mock
}

// GetPublicKey implements nodeinfo.DiscoveryNode
func (mmGetPublicKey *DiscoveryNodeMock) GetPublicKey() (p1 crypto.PublicKey) {
	mm_atomic.AddUint64(&mmGetPublicKey.beforeGetPublicKeyCounter, 1)
	defer mm_atomic.AddUint64(&mmGetPublicKey.afterGetPublicKeyCounter, 1)

	if mmGetPublicKey.inspectFuncGetPublicKey != nil {
		mmGetPublicKey.inspectFuncGetPublicKey()
	}

	if mmGetPublicKey.GetPublicKeyMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetPublicKey.GetPublicKeyMock.defaultExpectation.Counter, 1)

		mm_results := mmGetPublicKey.GetPublicKeyMock.defaultExpectation.results
		if mm_results == nil {
			mmGetPublicKey.t.Fatal("No results are set for the DiscoveryNodeMock.GetPublicKey")
		}
		return (*mm_results).p1
	}
	if mmGetPublicKey.funcGetPublicKey != nil {
		return mmGetPublicKey.funcGetPublicKey()
	}
	mmGetPublicKey.t.Fatalf("Unexpected call to DiscoveryNodeMock.GetPublicKey.")
	return
}

// GetPublicKeyAfterCounter returns a count of finished DiscoveryNodeMock.GetPublicKey invocations
func (mmGetPublicKey *DiscoveryNodeMock) GetPublicKeyAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetPublicKey.afterGetPublicKeyCounter)
}

// GetPublicKeyBeforeCounter returns a count of DiscoveryNodeMock.GetPublicKey invocations
func (mmGetPublicKey *DiscoveryNodeMock) GetPublicKeyBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetPublicKey.beforeGetPublicKeyCounter)
}

// MinimockGetPublicKeyDone returns true if the count of the GetPublicKey invocations corresponds
// the number of defined expectations
func (m *DiscoveryNodeMock) MinimockGetPublicKeyDone() bool {
	for _, e := range m.GetPublicKeyMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetPublicKeyMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetPublicKeyCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetPublicKey != nil && mm_atomic.LoadUint64(&m.afterGetPublicKeyCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetPublicKeyInspect logs each unmet expectation
func (m *DiscoveryNodeMock) MinimockGetPublicKeyInspect() {
	for _, e := range m.GetPublicKeyMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to DiscoveryNodeMock.GetPublicKey")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetPublicKeyMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetPublicKeyCounter) < 1 {
		m.t.Error("Expected call to DiscoveryNodeMock.GetPublicKey")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetPublicKey != nil && mm_atomic.LoadUint64(&m.afterGetPublicKeyCounter) < 1 {
		m.t.Error("Expected call to DiscoveryNodeMock.GetPublicKey")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *DiscoveryNodeMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockGetHostInspect()

		m.MinimockGetNodeRefInspect()

		m.MinimockGetPublicKeyInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *DiscoveryNodeMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *DiscoveryNodeMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockGetHostDone() &&
		m.MinimockGetNodeRefDone() &&
		m.MinimockGetPublicKeyDone()
}
