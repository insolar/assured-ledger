package network

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
)

// OriginProviderMock implements network.OriginProvider
type OriginProviderMock struct {
	t minimock.Tester

	funcGetOrigin          func() (n1 node.NetworkNode)
	inspectFuncGetOrigin   func()
	afterGetOriginCounter  uint64
	beforeGetOriginCounter uint64
	GetOriginMock          mOriginProviderMockGetOrigin
}

// NewOriginProviderMock returns a mock for network.OriginProvider
func NewOriginProviderMock(t minimock.Tester) *OriginProviderMock {
	m := &OriginProviderMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.GetOriginMock = mOriginProviderMockGetOrigin{mock: m}

	return m
}

type mOriginProviderMockGetOrigin struct {
	mock               *OriginProviderMock
	defaultExpectation *OriginProviderMockGetOriginExpectation
	expectations       []*OriginProviderMockGetOriginExpectation
}

// OriginProviderMockGetOriginExpectation specifies expectation struct of the OriginProvider.GetOrigin
type OriginProviderMockGetOriginExpectation struct {
	mock *OriginProviderMock

	results *OriginProviderMockGetOriginResults
	Counter uint64
}

// OriginProviderMockGetOriginResults contains results of the OriginProvider.GetOrigin
type OriginProviderMockGetOriginResults struct {
	n1 node.NetworkNode
}

// Expect sets up expected params for OriginProvider.GetOrigin
func (mmGetOrigin *mOriginProviderMockGetOrigin) Expect() *mOriginProviderMockGetOrigin {
	if mmGetOrigin.mock.funcGetOrigin != nil {
		mmGetOrigin.mock.t.Fatalf("OriginProviderMock.GetOrigin mock is already set by Set")
	}

	if mmGetOrigin.defaultExpectation == nil {
		mmGetOrigin.defaultExpectation = &OriginProviderMockGetOriginExpectation{}
	}

	return mmGetOrigin
}

// Inspect accepts an inspector function that has same arguments as the OriginProvider.GetOrigin
func (mmGetOrigin *mOriginProviderMockGetOrigin) Inspect(f func()) *mOriginProviderMockGetOrigin {
	if mmGetOrigin.mock.inspectFuncGetOrigin != nil {
		mmGetOrigin.mock.t.Fatalf("Inspect function is already set for OriginProviderMock.GetOrigin")
	}

	mmGetOrigin.mock.inspectFuncGetOrigin = f

	return mmGetOrigin
}

// Return sets up results that will be returned by OriginProvider.GetOrigin
func (mmGetOrigin *mOriginProviderMockGetOrigin) Return(n1 node.NetworkNode) *OriginProviderMock {
	if mmGetOrigin.mock.funcGetOrigin != nil {
		mmGetOrigin.mock.t.Fatalf("OriginProviderMock.GetOrigin mock is already set by Set")
	}

	if mmGetOrigin.defaultExpectation == nil {
		mmGetOrigin.defaultExpectation = &OriginProviderMockGetOriginExpectation{mock: mmGetOrigin.mock}
	}
	mmGetOrigin.defaultExpectation.results = &OriginProviderMockGetOriginResults{n1}
	return mmGetOrigin.mock
}

//Set uses given function f to mock the OriginProvider.GetOrigin method
func (mmGetOrigin *mOriginProviderMockGetOrigin) Set(f func() (n1 node.NetworkNode)) *OriginProviderMock {
	if mmGetOrigin.defaultExpectation != nil {
		mmGetOrigin.mock.t.Fatalf("Default expectation is already set for the OriginProvider.GetOrigin method")
	}

	if len(mmGetOrigin.expectations) > 0 {
		mmGetOrigin.mock.t.Fatalf("Some expectations are already set for the OriginProvider.GetOrigin method")
	}

	mmGetOrigin.mock.funcGetOrigin = f
	return mmGetOrigin.mock
}

// GetOrigin implements network.OriginProvider
func (mmGetOrigin *OriginProviderMock) GetOrigin() (n1 node.NetworkNode) {
	mm_atomic.AddUint64(&mmGetOrigin.beforeGetOriginCounter, 1)
	defer mm_atomic.AddUint64(&mmGetOrigin.afterGetOriginCounter, 1)

	if mmGetOrigin.inspectFuncGetOrigin != nil {
		mmGetOrigin.inspectFuncGetOrigin()
	}

	if mmGetOrigin.GetOriginMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetOrigin.GetOriginMock.defaultExpectation.Counter, 1)

		mm_results := mmGetOrigin.GetOriginMock.defaultExpectation.results
		if mm_results == nil {
			mmGetOrigin.t.Fatal("No results are set for the OriginProviderMock.GetOrigin")
		}
		return (*mm_results).n1
	}
	if mmGetOrigin.funcGetOrigin != nil {
		return mmGetOrigin.funcGetOrigin()
	}
	mmGetOrigin.t.Fatalf("Unexpected call to OriginProviderMock.GetOrigin.")
	return
}

// GetOriginAfterCounter returns a count of finished OriginProviderMock.GetOrigin invocations
func (mmGetOrigin *OriginProviderMock) GetOriginAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetOrigin.afterGetOriginCounter)
}

// GetOriginBeforeCounter returns a count of OriginProviderMock.GetOrigin invocations
func (mmGetOrigin *OriginProviderMock) GetOriginBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetOrigin.beforeGetOriginCounter)
}

// MinimockGetOriginDone returns true if the count of the GetOrigin invocations corresponds
// the number of defined expectations
func (m *OriginProviderMock) MinimockGetOriginDone() bool {
	for _, e := range m.GetOriginMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetOriginMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetOriginCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetOrigin != nil && mm_atomic.LoadUint64(&m.afterGetOriginCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetOriginInspect logs each unmet expectation
func (m *OriginProviderMock) MinimockGetOriginInspect() {
	for _, e := range m.GetOriginMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to OriginProviderMock.GetOrigin")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetOriginMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetOriginCounter) < 1 {
		m.t.Error("Expected call to OriginProviderMock.GetOrigin")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetOrigin != nil && mm_atomic.LoadUint64(&m.afterGetOriginCounter) < 1 {
		m.t.Error("Expected call to OriginProviderMock.GetOrigin")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *OriginProviderMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockGetOriginInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *OriginProviderMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *OriginProviderMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockGetOriginDone()
}
