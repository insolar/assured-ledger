package cryptography

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"crypto"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
)

// ServiceMock implements Service
type ServiceMock struct {
	t minimock.Tester

	funcGetPublicKey          func() (p1 crypto.PublicKey, err error)
	inspectFuncGetPublicKey   func()
	afterGetPublicKeyCounter  uint64
	beforeGetPublicKeyCounter uint64
	GetPublicKeyMock          mServiceMockGetPublicKey

	funcSign          func(ba1 []byte) (sp1 *Signature, err error)
	inspectFuncSign   func(ba1 []byte)
	afterSignCounter  uint64
	beforeSignCounter uint64
	SignMock          mServiceMockSign

	funcVerify          func(p1 crypto.PublicKey, s1 Signature, ba1 []byte) (b1 bool)
	inspectFuncVerify   func(p1 crypto.PublicKey, s1 Signature, ba1 []byte)
	afterVerifyCounter  uint64
	beforeVerifyCounter uint64
	VerifyMock          mServiceMockVerify
}

// NewServiceMock returns a mock for Service
func NewServiceMock(t minimock.Tester) *ServiceMock {
	m := &ServiceMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.GetPublicKeyMock = mServiceMockGetPublicKey{mock: m}

	m.SignMock = mServiceMockSign{mock: m}
	m.SignMock.callArgs = []*ServiceMockSignParams{}

	m.VerifyMock = mServiceMockVerify{mock: m}
	m.VerifyMock.callArgs = []*ServiceMockVerifyParams{}

	return m
}

type mServiceMockGetPublicKey struct {
	mock               *ServiceMock
	defaultExpectation *ServiceMockGetPublicKeyExpectation
	expectations       []*ServiceMockGetPublicKeyExpectation
}

// ServiceMockGetPublicKeyExpectation specifies expectation struct of the Service.GetPublicKey
type ServiceMockGetPublicKeyExpectation struct {
	mock *ServiceMock

	results *ServiceMockGetPublicKeyResults
	Counter uint64
}

// ServiceMockGetPublicKeyResults contains results of the Service.GetPublicKey
type ServiceMockGetPublicKeyResults struct {
	p1  crypto.PublicKey
	err error
}

// Expect sets up expected params for Service.GetPublicKey
func (mmGetPublicKey *mServiceMockGetPublicKey) Expect() *mServiceMockGetPublicKey {
	if mmGetPublicKey.mock.funcGetPublicKey != nil {
		mmGetPublicKey.mock.t.Fatalf("ServiceMock.GetPublicKey mock is already set by Set")
	}

	if mmGetPublicKey.defaultExpectation == nil {
		mmGetPublicKey.defaultExpectation = &ServiceMockGetPublicKeyExpectation{}
	}

	return mmGetPublicKey
}

// Inspect accepts an inspector function that has same arguments as the Service.GetPublicKey
func (mmGetPublicKey *mServiceMockGetPublicKey) Inspect(f func()) *mServiceMockGetPublicKey {
	if mmGetPublicKey.mock.inspectFuncGetPublicKey != nil {
		mmGetPublicKey.mock.t.Fatalf("Inspect function is already set for ServiceMock.GetPublicKey")
	}

	mmGetPublicKey.mock.inspectFuncGetPublicKey = f

	return mmGetPublicKey
}

// Return sets up results that will be returned by Service.GetPublicKey
func (mmGetPublicKey *mServiceMockGetPublicKey) Return(p1 crypto.PublicKey, err error) *ServiceMock {
	if mmGetPublicKey.mock.funcGetPublicKey != nil {
		mmGetPublicKey.mock.t.Fatalf("ServiceMock.GetPublicKey mock is already set by Set")
	}

	if mmGetPublicKey.defaultExpectation == nil {
		mmGetPublicKey.defaultExpectation = &ServiceMockGetPublicKeyExpectation{mock: mmGetPublicKey.mock}
	}
	mmGetPublicKey.defaultExpectation.results = &ServiceMockGetPublicKeyResults{p1, err}
	return mmGetPublicKey.mock
}

//Set uses given function f to mock the Service.GetPublicKey method
func (mmGetPublicKey *mServiceMockGetPublicKey) Set(f func() (p1 crypto.PublicKey, err error)) *ServiceMock {
	if mmGetPublicKey.defaultExpectation != nil {
		mmGetPublicKey.mock.t.Fatalf("Default expectation is already set for the Service.GetPublicKey method")
	}

	if len(mmGetPublicKey.expectations) > 0 {
		mmGetPublicKey.mock.t.Fatalf("Some expectations are already set for the Service.GetPublicKey method")
	}

	mmGetPublicKey.mock.funcGetPublicKey = f
	return mmGetPublicKey.mock
}

// GetPublicKey implements Service
func (mmGetPublicKey *ServiceMock) GetPublicKey() (p1 crypto.PublicKey, err error) {
	mm_atomic.AddUint64(&mmGetPublicKey.beforeGetPublicKeyCounter, 1)
	defer mm_atomic.AddUint64(&mmGetPublicKey.afterGetPublicKeyCounter, 1)

	if mmGetPublicKey.inspectFuncGetPublicKey != nil {
		mmGetPublicKey.inspectFuncGetPublicKey()
	}

	if mmGetPublicKey.GetPublicKeyMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetPublicKey.GetPublicKeyMock.defaultExpectation.Counter, 1)

		mm_results := mmGetPublicKey.GetPublicKeyMock.defaultExpectation.results
		if mm_results == nil {
			mmGetPublicKey.t.Fatal("No results are set for the ServiceMock.GetPublicKey")
		}
		return (*mm_results).p1, (*mm_results).err
	}
	if mmGetPublicKey.funcGetPublicKey != nil {
		return mmGetPublicKey.funcGetPublicKey()
	}
	mmGetPublicKey.t.Fatalf("Unexpected call to ServiceMock.GetPublicKey.")
	return
}

// GetPublicKeyAfterCounter returns a count of finished ServiceMock.GetPublicKey invocations
func (mmGetPublicKey *ServiceMock) GetPublicKeyAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetPublicKey.afterGetPublicKeyCounter)
}

// GetPublicKeyBeforeCounter returns a count of ServiceMock.GetPublicKey invocations
func (mmGetPublicKey *ServiceMock) GetPublicKeyBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetPublicKey.beforeGetPublicKeyCounter)
}

// MinimockGetPublicKeyDone returns true if the count of the GetPublicKey invocations corresponds
// the number of defined expectations
func (m *ServiceMock) MinimockGetPublicKeyDone() bool {
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
func (m *ServiceMock) MinimockGetPublicKeyInspect() {
	for _, e := range m.GetPublicKeyMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to ServiceMock.GetPublicKey")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetPublicKeyMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetPublicKeyCounter) < 1 {
		m.t.Error("Expected call to ServiceMock.GetPublicKey")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetPublicKey != nil && mm_atomic.LoadUint64(&m.afterGetPublicKeyCounter) < 1 {
		m.t.Error("Expected call to ServiceMock.GetPublicKey")
	}
}

type mServiceMockSign struct {
	mock               *ServiceMock
	defaultExpectation *ServiceMockSignExpectation
	expectations       []*ServiceMockSignExpectation

	callArgs []*ServiceMockSignParams
	mutex    sync.RWMutex
}

// ServiceMockSignExpectation specifies expectation struct of the Service.Sign
type ServiceMockSignExpectation struct {
	mock    *ServiceMock
	params  *ServiceMockSignParams
	results *ServiceMockSignResults
	Counter uint64
}

// ServiceMockSignParams contains parameters of the Service.Sign
type ServiceMockSignParams struct {
	ba1 []byte
}

// ServiceMockSignResults contains results of the Service.Sign
type ServiceMockSignResults struct {
	sp1 *Signature
	err error
}

// Expect sets up expected params for Service.Sign
func (mmSign *mServiceMockSign) Expect(ba1 []byte) *mServiceMockSign {
	if mmSign.mock.funcSign != nil {
		mmSign.mock.t.Fatalf("ServiceMock.Sign mock is already set by Set")
	}

	if mmSign.defaultExpectation == nil {
		mmSign.defaultExpectation = &ServiceMockSignExpectation{}
	}

	mmSign.defaultExpectation.params = &ServiceMockSignParams{ba1}
	for _, e := range mmSign.expectations {
		if minimock.Equal(e.params, mmSign.defaultExpectation.params) {
			mmSign.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmSign.defaultExpectation.params)
		}
	}

	return mmSign
}

// Inspect accepts an inspector function that has same arguments as the Service.Sign
func (mmSign *mServiceMockSign) Inspect(f func(ba1 []byte)) *mServiceMockSign {
	if mmSign.mock.inspectFuncSign != nil {
		mmSign.mock.t.Fatalf("Inspect function is already set for ServiceMock.Sign")
	}

	mmSign.mock.inspectFuncSign = f

	return mmSign
}

// Return sets up results that will be returned by Service.Sign
func (mmSign *mServiceMockSign) Return(sp1 *Signature, err error) *ServiceMock {
	if mmSign.mock.funcSign != nil {
		mmSign.mock.t.Fatalf("ServiceMock.Sign mock is already set by Set")
	}

	if mmSign.defaultExpectation == nil {
		mmSign.defaultExpectation = &ServiceMockSignExpectation{mock: mmSign.mock}
	}
	mmSign.defaultExpectation.results = &ServiceMockSignResults{sp1, err}
	return mmSign.mock
}

//Set uses given function f to mock the Service.Sign method
func (mmSign *mServiceMockSign) Set(f func(ba1 []byte) (sp1 *Signature, err error)) *ServiceMock {
	if mmSign.defaultExpectation != nil {
		mmSign.mock.t.Fatalf("Default expectation is already set for the Service.Sign method")
	}

	if len(mmSign.expectations) > 0 {
		mmSign.mock.t.Fatalf("Some expectations are already set for the Service.Sign method")
	}

	mmSign.mock.funcSign = f
	return mmSign.mock
}

// When sets expectation for the Service.Sign which will trigger the result defined by the following
// Then helper
func (mmSign *mServiceMockSign) When(ba1 []byte) *ServiceMockSignExpectation {
	if mmSign.mock.funcSign != nil {
		mmSign.mock.t.Fatalf("ServiceMock.Sign mock is already set by Set")
	}

	expectation := &ServiceMockSignExpectation{
		mock:   mmSign.mock,
		params: &ServiceMockSignParams{ba1},
	}
	mmSign.expectations = append(mmSign.expectations, expectation)
	return expectation
}

// Then sets up Service.Sign return parameters for the expectation previously defined by the When method
func (e *ServiceMockSignExpectation) Then(sp1 *Signature, err error) *ServiceMock {
	e.results = &ServiceMockSignResults{sp1, err}
	return e.mock
}

// Sign implements Service
func (mmSign *ServiceMock) Sign(ba1 []byte) (sp1 *Signature, err error) {
	mm_atomic.AddUint64(&mmSign.beforeSignCounter, 1)
	defer mm_atomic.AddUint64(&mmSign.afterSignCounter, 1)

	if mmSign.inspectFuncSign != nil {
		mmSign.inspectFuncSign(ba1)
	}

	mm_params := &ServiceMockSignParams{ba1}

	// Record call args
	mmSign.SignMock.mutex.Lock()
	mmSign.SignMock.callArgs = append(mmSign.SignMock.callArgs, mm_params)
	mmSign.SignMock.mutex.Unlock()

	for _, e := range mmSign.SignMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.sp1, e.results.err
		}
	}

	if mmSign.SignMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmSign.SignMock.defaultExpectation.Counter, 1)
		mm_want := mmSign.SignMock.defaultExpectation.params
		mm_got := ServiceMockSignParams{ba1}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmSign.t.Errorf("ServiceMock.Sign got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmSign.SignMock.defaultExpectation.results
		if mm_results == nil {
			mmSign.t.Fatal("No results are set for the ServiceMock.Sign")
		}
		return (*mm_results).sp1, (*mm_results).err
	}
	if mmSign.funcSign != nil {
		return mmSign.funcSign(ba1)
	}
	mmSign.t.Fatalf("Unexpected call to ServiceMock.Sign. %v", ba1)
	return
}

// SignAfterCounter returns a count of finished ServiceMock.Sign invocations
func (mmSign *ServiceMock) SignAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSign.afterSignCounter)
}

// SignBeforeCounter returns a count of ServiceMock.Sign invocations
func (mmSign *ServiceMock) SignBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSign.beforeSignCounter)
}

// Calls returns a list of arguments used in each call to ServiceMock.Sign.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmSign *mServiceMockSign) Calls() []*ServiceMockSignParams {
	mmSign.mutex.RLock()

	argCopy := make([]*ServiceMockSignParams, len(mmSign.callArgs))
	copy(argCopy, mmSign.callArgs)

	mmSign.mutex.RUnlock()

	return argCopy
}

// MinimockSignDone returns true if the count of the Sign invocations corresponds
// the number of defined expectations
func (m *ServiceMock) MinimockSignDone() bool {
	for _, e := range m.SignMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SignMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSignCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSign != nil && mm_atomic.LoadUint64(&m.afterSignCounter) < 1 {
		return false
	}
	return true
}

// MinimockSignInspect logs each unmet expectation
func (m *ServiceMock) MinimockSignInspect() {
	for _, e := range m.SignMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ServiceMock.Sign with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SignMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSignCounter) < 1 {
		if m.SignMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ServiceMock.Sign")
		} else {
			m.t.Errorf("Expected call to ServiceMock.Sign with params: %#v", *m.SignMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSign != nil && mm_atomic.LoadUint64(&m.afterSignCounter) < 1 {
		m.t.Error("Expected call to ServiceMock.Sign")
	}
}

type mServiceMockVerify struct {
	mock               *ServiceMock
	defaultExpectation *ServiceMockVerifyExpectation
	expectations       []*ServiceMockVerifyExpectation

	callArgs []*ServiceMockVerifyParams
	mutex    sync.RWMutex
}

// ServiceMockVerifyExpectation specifies expectation struct of the Service.Verify
type ServiceMockVerifyExpectation struct {
	mock    *ServiceMock
	params  *ServiceMockVerifyParams
	results *ServiceMockVerifyResults
	Counter uint64
}

// ServiceMockVerifyParams contains parameters of the Service.Verify
type ServiceMockVerifyParams struct {
	p1  crypto.PublicKey
	s1  Signature
	ba1 []byte
}

// ServiceMockVerifyResults contains results of the Service.Verify
type ServiceMockVerifyResults struct {
	b1 bool
}

// Expect sets up expected params for Service.Verify
func (mmVerify *mServiceMockVerify) Expect(p1 crypto.PublicKey, s1 Signature, ba1 []byte) *mServiceMockVerify {
	if mmVerify.mock.funcVerify != nil {
		mmVerify.mock.t.Fatalf("ServiceMock.Verify mock is already set by Set")
	}

	if mmVerify.defaultExpectation == nil {
		mmVerify.defaultExpectation = &ServiceMockVerifyExpectation{}
	}

	mmVerify.defaultExpectation.params = &ServiceMockVerifyParams{p1, s1, ba1}
	for _, e := range mmVerify.expectations {
		if minimock.Equal(e.params, mmVerify.defaultExpectation.params) {
			mmVerify.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmVerify.defaultExpectation.params)
		}
	}

	return mmVerify
}

// Inspect accepts an inspector function that has same arguments as the Service.Verify
func (mmVerify *mServiceMockVerify) Inspect(f func(p1 crypto.PublicKey, s1 Signature, ba1 []byte)) *mServiceMockVerify {
	if mmVerify.mock.inspectFuncVerify != nil {
		mmVerify.mock.t.Fatalf("Inspect function is already set for ServiceMock.Verify")
	}

	mmVerify.mock.inspectFuncVerify = f

	return mmVerify
}

// Return sets up results that will be returned by Service.Verify
func (mmVerify *mServiceMockVerify) Return(b1 bool) *ServiceMock {
	if mmVerify.mock.funcVerify != nil {
		mmVerify.mock.t.Fatalf("ServiceMock.Verify mock is already set by Set")
	}

	if mmVerify.defaultExpectation == nil {
		mmVerify.defaultExpectation = &ServiceMockVerifyExpectation{mock: mmVerify.mock}
	}
	mmVerify.defaultExpectation.results = &ServiceMockVerifyResults{b1}
	return mmVerify.mock
}

//Set uses given function f to mock the Service.Verify method
func (mmVerify *mServiceMockVerify) Set(f func(p1 crypto.PublicKey, s1 Signature, ba1 []byte) (b1 bool)) *ServiceMock {
	if mmVerify.defaultExpectation != nil {
		mmVerify.mock.t.Fatalf("Default expectation is already set for the Service.Verify method")
	}

	if len(mmVerify.expectations) > 0 {
		mmVerify.mock.t.Fatalf("Some expectations are already set for the Service.Verify method")
	}

	mmVerify.mock.funcVerify = f
	return mmVerify.mock
}

// When sets expectation for the Service.Verify which will trigger the result defined by the following
// Then helper
func (mmVerify *mServiceMockVerify) When(p1 crypto.PublicKey, s1 Signature, ba1 []byte) *ServiceMockVerifyExpectation {
	if mmVerify.mock.funcVerify != nil {
		mmVerify.mock.t.Fatalf("ServiceMock.Verify mock is already set by Set")
	}

	expectation := &ServiceMockVerifyExpectation{
		mock:   mmVerify.mock,
		params: &ServiceMockVerifyParams{p1, s1, ba1},
	}
	mmVerify.expectations = append(mmVerify.expectations, expectation)
	return expectation
}

// Then sets up Service.Verify return parameters for the expectation previously defined by the When method
func (e *ServiceMockVerifyExpectation) Then(b1 bool) *ServiceMock {
	e.results = &ServiceMockVerifyResults{b1}
	return e.mock
}

// Verify implements Service
func (mmVerify *ServiceMock) Verify(p1 crypto.PublicKey, s1 Signature, ba1 []byte) (b1 bool) {
	mm_atomic.AddUint64(&mmVerify.beforeVerifyCounter, 1)
	defer mm_atomic.AddUint64(&mmVerify.afterVerifyCounter, 1)

	if mmVerify.inspectFuncVerify != nil {
		mmVerify.inspectFuncVerify(p1, s1, ba1)
	}

	mm_params := &ServiceMockVerifyParams{p1, s1, ba1}

	// Record call args
	mmVerify.VerifyMock.mutex.Lock()
	mmVerify.VerifyMock.callArgs = append(mmVerify.VerifyMock.callArgs, mm_params)
	mmVerify.VerifyMock.mutex.Unlock()

	for _, e := range mmVerify.VerifyMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.b1
		}
	}

	if mmVerify.VerifyMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmVerify.VerifyMock.defaultExpectation.Counter, 1)
		mm_want := mmVerify.VerifyMock.defaultExpectation.params
		mm_got := ServiceMockVerifyParams{p1, s1, ba1}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmVerify.t.Errorf("ServiceMock.Verify got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmVerify.VerifyMock.defaultExpectation.results
		if mm_results == nil {
			mmVerify.t.Fatal("No results are set for the ServiceMock.Verify")
		}
		return (*mm_results).b1
	}
	if mmVerify.funcVerify != nil {
		return mmVerify.funcVerify(p1, s1, ba1)
	}
	mmVerify.t.Fatalf("Unexpected call to ServiceMock.Verify. %v %v %v", p1, s1, ba1)
	return
}

// VerifyAfterCounter returns a count of finished ServiceMock.Verify invocations
func (mmVerify *ServiceMock) VerifyAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmVerify.afterVerifyCounter)
}

// VerifyBeforeCounter returns a count of ServiceMock.Verify invocations
func (mmVerify *ServiceMock) VerifyBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmVerify.beforeVerifyCounter)
}

// Calls returns a list of arguments used in each call to ServiceMock.Verify.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmVerify *mServiceMockVerify) Calls() []*ServiceMockVerifyParams {
	mmVerify.mutex.RLock()

	argCopy := make([]*ServiceMockVerifyParams, len(mmVerify.callArgs))
	copy(argCopy, mmVerify.callArgs)

	mmVerify.mutex.RUnlock()

	return argCopy
}

// MinimockVerifyDone returns true if the count of the Verify invocations corresponds
// the number of defined expectations
func (m *ServiceMock) MinimockVerifyDone() bool {
	for _, e := range m.VerifyMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.VerifyMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterVerifyCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcVerify != nil && mm_atomic.LoadUint64(&m.afterVerifyCounter) < 1 {
		return false
	}
	return true
}

// MinimockVerifyInspect logs each unmet expectation
func (m *ServiceMock) MinimockVerifyInspect() {
	for _, e := range m.VerifyMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ServiceMock.Verify with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.VerifyMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterVerifyCounter) < 1 {
		if m.VerifyMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ServiceMock.Verify")
		} else {
			m.t.Errorf("Expected call to ServiceMock.Verify with params: %#v", *m.VerifyMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcVerify != nil && mm_atomic.LoadUint64(&m.afterVerifyCounter) < 1 {
		m.t.Error("Expected call to ServiceMock.Verify")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *ServiceMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockGetPublicKeyInspect()

		m.MinimockSignInspect()

		m.MinimockVerifyInspect()
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
		m.MinimockGetPublicKeyDone() &&
		m.MinimockSignDone() &&
		m.MinimockVerifyDone()
}
