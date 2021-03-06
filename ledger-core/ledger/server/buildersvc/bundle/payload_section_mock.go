package bundle

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/ledger"
)

// PayloadSectionMock implements PayloadSection
type PayloadSectionMock struct {
	t minimock.Tester

	funcAllocatePayloadStorage          func(size int, extID ledger.ExtensionID) (p1 PayloadReceptacle, s1 ledger.StorageLocator, err error)
	inspectFuncAllocatePayloadStorage   func(size int, extID ledger.ExtensionID)
	afterAllocatePayloadStorageCounter  uint64
	beforeAllocatePayloadStorageCounter uint64
	AllocatePayloadStorageMock          mPayloadSectionMockAllocatePayloadStorage
}

// NewPayloadSectionMock returns a mock for PayloadSection
func NewPayloadSectionMock(t minimock.Tester) *PayloadSectionMock {
	m := &PayloadSectionMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.AllocatePayloadStorageMock = mPayloadSectionMockAllocatePayloadStorage{mock: m}
	m.AllocatePayloadStorageMock.callArgs = []*PayloadSectionMockAllocatePayloadStorageParams{}

	return m
}

type mPayloadSectionMockAllocatePayloadStorage struct {
	mock               *PayloadSectionMock
	defaultExpectation *PayloadSectionMockAllocatePayloadStorageExpectation
	expectations       []*PayloadSectionMockAllocatePayloadStorageExpectation

	callArgs []*PayloadSectionMockAllocatePayloadStorageParams
	mutex    sync.RWMutex
}

// PayloadSectionMockAllocatePayloadStorageExpectation specifies expectation struct of the PayloadSection.AllocatePayloadStorage
type PayloadSectionMockAllocatePayloadStorageExpectation struct {
	mock    *PayloadSectionMock
	params  *PayloadSectionMockAllocatePayloadStorageParams
	results *PayloadSectionMockAllocatePayloadStorageResults
	Counter uint64
}

// PayloadSectionMockAllocatePayloadStorageParams contains parameters of the PayloadSection.AllocatePayloadStorage
type PayloadSectionMockAllocatePayloadStorageParams struct {
	size  int
	extID ledger.ExtensionID
}

// PayloadSectionMockAllocatePayloadStorageResults contains results of the PayloadSection.AllocatePayloadStorage
type PayloadSectionMockAllocatePayloadStorageResults struct {
	p1  PayloadReceptacle
	s1  ledger.StorageLocator
	err error
}

// Expect sets up expected params for PayloadSection.AllocatePayloadStorage
func (mmAllocatePayloadStorage *mPayloadSectionMockAllocatePayloadStorage) Expect(size int, extID ledger.ExtensionID) *mPayloadSectionMockAllocatePayloadStorage {
	if mmAllocatePayloadStorage.mock.funcAllocatePayloadStorage != nil {
		mmAllocatePayloadStorage.mock.t.Fatalf("PayloadSectionMock.AllocatePayloadStorage mock is already set by Set")
	}

	if mmAllocatePayloadStorage.defaultExpectation == nil {
		mmAllocatePayloadStorage.defaultExpectation = &PayloadSectionMockAllocatePayloadStorageExpectation{}
	}

	mmAllocatePayloadStorage.defaultExpectation.params = &PayloadSectionMockAllocatePayloadStorageParams{size, extID}
	for _, e := range mmAllocatePayloadStorage.expectations {
		if minimock.Equal(e.params, mmAllocatePayloadStorage.defaultExpectation.params) {
			mmAllocatePayloadStorage.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmAllocatePayloadStorage.defaultExpectation.params)
		}
	}

	return mmAllocatePayloadStorage
}

// Inspect accepts an inspector function that has same arguments as the PayloadSection.AllocatePayloadStorage
func (mmAllocatePayloadStorage *mPayloadSectionMockAllocatePayloadStorage) Inspect(f func(size int, extID ledger.ExtensionID)) *mPayloadSectionMockAllocatePayloadStorage {
	if mmAllocatePayloadStorage.mock.inspectFuncAllocatePayloadStorage != nil {
		mmAllocatePayloadStorage.mock.t.Fatalf("Inspect function is already set for PayloadSectionMock.AllocatePayloadStorage")
	}

	mmAllocatePayloadStorage.mock.inspectFuncAllocatePayloadStorage = f

	return mmAllocatePayloadStorage
}

// Return sets up results that will be returned by PayloadSection.AllocatePayloadStorage
func (mmAllocatePayloadStorage *mPayloadSectionMockAllocatePayloadStorage) Return(p1 PayloadReceptacle, s1 ledger.StorageLocator, err error) *PayloadSectionMock {
	if mmAllocatePayloadStorage.mock.funcAllocatePayloadStorage != nil {
		mmAllocatePayloadStorage.mock.t.Fatalf("PayloadSectionMock.AllocatePayloadStorage mock is already set by Set")
	}

	if mmAllocatePayloadStorage.defaultExpectation == nil {
		mmAllocatePayloadStorage.defaultExpectation = &PayloadSectionMockAllocatePayloadStorageExpectation{mock: mmAllocatePayloadStorage.mock}
	}
	mmAllocatePayloadStorage.defaultExpectation.results = &PayloadSectionMockAllocatePayloadStorageResults{p1, s1, err}
	return mmAllocatePayloadStorage.mock
}

//Set uses given function f to mock the PayloadSection.AllocatePayloadStorage method
func (mmAllocatePayloadStorage *mPayloadSectionMockAllocatePayloadStorage) Set(f func(size int, extID ledger.ExtensionID) (p1 PayloadReceptacle, s1 ledger.StorageLocator, err error)) *PayloadSectionMock {
	if mmAllocatePayloadStorage.defaultExpectation != nil {
		mmAllocatePayloadStorage.mock.t.Fatalf("Default expectation is already set for the PayloadSection.AllocatePayloadStorage method")
	}

	if len(mmAllocatePayloadStorage.expectations) > 0 {
		mmAllocatePayloadStorage.mock.t.Fatalf("Some expectations are already set for the PayloadSection.AllocatePayloadStorage method")
	}

	mmAllocatePayloadStorage.mock.funcAllocatePayloadStorage = f
	return mmAllocatePayloadStorage.mock
}

// When sets expectation for the PayloadSection.AllocatePayloadStorage which will trigger the result defined by the following
// Then helper
func (mmAllocatePayloadStorage *mPayloadSectionMockAllocatePayloadStorage) When(size int, extID ledger.ExtensionID) *PayloadSectionMockAllocatePayloadStorageExpectation {
	if mmAllocatePayloadStorage.mock.funcAllocatePayloadStorage != nil {
		mmAllocatePayloadStorage.mock.t.Fatalf("PayloadSectionMock.AllocatePayloadStorage mock is already set by Set")
	}

	expectation := &PayloadSectionMockAllocatePayloadStorageExpectation{
		mock:   mmAllocatePayloadStorage.mock,
		params: &PayloadSectionMockAllocatePayloadStorageParams{size, extID},
	}
	mmAllocatePayloadStorage.expectations = append(mmAllocatePayloadStorage.expectations, expectation)
	return expectation
}

// Then sets up PayloadSection.AllocatePayloadStorage return parameters for the expectation previously defined by the When method
func (e *PayloadSectionMockAllocatePayloadStorageExpectation) Then(p1 PayloadReceptacle, s1 ledger.StorageLocator, err error) *PayloadSectionMock {
	e.results = &PayloadSectionMockAllocatePayloadStorageResults{p1, s1, err}
	return e.mock
}

// AllocatePayloadStorage implements PayloadSection
func (mmAllocatePayloadStorage *PayloadSectionMock) AllocatePayloadStorage(size int, extID ledger.ExtensionID) (p1 PayloadReceptacle, s1 ledger.StorageLocator, err error) {
	mm_atomic.AddUint64(&mmAllocatePayloadStorage.beforeAllocatePayloadStorageCounter, 1)
	defer mm_atomic.AddUint64(&mmAllocatePayloadStorage.afterAllocatePayloadStorageCounter, 1)

	if mmAllocatePayloadStorage.inspectFuncAllocatePayloadStorage != nil {
		mmAllocatePayloadStorage.inspectFuncAllocatePayloadStorage(size, extID)
	}

	mm_params := &PayloadSectionMockAllocatePayloadStorageParams{size, extID}

	// Record call args
	mmAllocatePayloadStorage.AllocatePayloadStorageMock.mutex.Lock()
	mmAllocatePayloadStorage.AllocatePayloadStorageMock.callArgs = append(mmAllocatePayloadStorage.AllocatePayloadStorageMock.callArgs, mm_params)
	mmAllocatePayloadStorage.AllocatePayloadStorageMock.mutex.Unlock()

	for _, e := range mmAllocatePayloadStorage.AllocatePayloadStorageMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.p1, e.results.s1, e.results.err
		}
	}

	if mmAllocatePayloadStorage.AllocatePayloadStorageMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmAllocatePayloadStorage.AllocatePayloadStorageMock.defaultExpectation.Counter, 1)
		mm_want := mmAllocatePayloadStorage.AllocatePayloadStorageMock.defaultExpectation.params
		mm_got := PayloadSectionMockAllocatePayloadStorageParams{size, extID}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmAllocatePayloadStorage.t.Errorf("PayloadSectionMock.AllocatePayloadStorage got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmAllocatePayloadStorage.AllocatePayloadStorageMock.defaultExpectation.results
		if mm_results == nil {
			mmAllocatePayloadStorage.t.Fatal("No results are set for the PayloadSectionMock.AllocatePayloadStorage")
		}
		return (*mm_results).p1, (*mm_results).s1, (*mm_results).err
	}
	if mmAllocatePayloadStorage.funcAllocatePayloadStorage != nil {
		return mmAllocatePayloadStorage.funcAllocatePayloadStorage(size, extID)
	}
	mmAllocatePayloadStorage.t.Fatalf("Unexpected call to PayloadSectionMock.AllocatePayloadStorage. %v %v", size, extID)
	return
}

// AllocatePayloadStorageAfterCounter returns a count of finished PayloadSectionMock.AllocatePayloadStorage invocations
func (mmAllocatePayloadStorage *PayloadSectionMock) AllocatePayloadStorageAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmAllocatePayloadStorage.afterAllocatePayloadStorageCounter)
}

// AllocatePayloadStorageBeforeCounter returns a count of PayloadSectionMock.AllocatePayloadStorage invocations
func (mmAllocatePayloadStorage *PayloadSectionMock) AllocatePayloadStorageBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmAllocatePayloadStorage.beforeAllocatePayloadStorageCounter)
}

// Calls returns a list of arguments used in each call to PayloadSectionMock.AllocatePayloadStorage.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmAllocatePayloadStorage *mPayloadSectionMockAllocatePayloadStorage) Calls() []*PayloadSectionMockAllocatePayloadStorageParams {
	mmAllocatePayloadStorage.mutex.RLock()

	argCopy := make([]*PayloadSectionMockAllocatePayloadStorageParams, len(mmAllocatePayloadStorage.callArgs))
	copy(argCopy, mmAllocatePayloadStorage.callArgs)

	mmAllocatePayloadStorage.mutex.RUnlock()

	return argCopy
}

// MinimockAllocatePayloadStorageDone returns true if the count of the AllocatePayloadStorage invocations corresponds
// the number of defined expectations
func (m *PayloadSectionMock) MinimockAllocatePayloadStorageDone() bool {
	for _, e := range m.AllocatePayloadStorageMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.AllocatePayloadStorageMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterAllocatePayloadStorageCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcAllocatePayloadStorage != nil && mm_atomic.LoadUint64(&m.afterAllocatePayloadStorageCounter) < 1 {
		return false
	}
	return true
}

// MinimockAllocatePayloadStorageInspect logs each unmet expectation
func (m *PayloadSectionMock) MinimockAllocatePayloadStorageInspect() {
	for _, e := range m.AllocatePayloadStorageMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to PayloadSectionMock.AllocatePayloadStorage with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.AllocatePayloadStorageMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterAllocatePayloadStorageCounter) < 1 {
		if m.AllocatePayloadStorageMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to PayloadSectionMock.AllocatePayloadStorage")
		} else {
			m.t.Errorf("Expected call to PayloadSectionMock.AllocatePayloadStorage with params: %#v", *m.AllocatePayloadStorageMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcAllocatePayloadStorage != nil && mm_atomic.LoadUint64(&m.afterAllocatePayloadStorageCounter) < 1 {
		m.t.Error("Expected call to PayloadSectionMock.AllocatePayloadStorage")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *PayloadSectionMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockAllocatePayloadStorageInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *PayloadSectionMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *PayloadSectionMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockAllocatePayloadStorageDone()
}
