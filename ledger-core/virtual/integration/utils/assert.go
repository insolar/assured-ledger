package utils

import (
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type TestingT interface {
	Helper()
	Logf(format string, args ...interface{})
	assert.TestingT
}

func AssertVCallRequestEqual(t TestingT, expected, actual *rms.VCallRequest, msgAndArgs ...interface{}) bool {
	t.Helper()

	ok := assert.True(t, expected.Equal(actual), msgAndArgs...)
	if !ok {
		t.Logf("expected: %#v", expected)
		t.Logf("actual:   %#v", actual)
	}

	return ok
}

func AssertVStateReportsEqual(t TestingT, expected, actual *rms.VStateReport, msgAndArgs ...interface{}) bool {
	t.Helper()

	ok := assert.True(t, expected.Equal(actual), msgAndArgs...)
	if !ok {
		t.Logf("expected: %#v", expected)
		t.Logf("actual:   %#v", actual)
	}

	return ok
}

func AssertCallDelegationTokenEqual(t TestingT, expected, actual *rms.CallDelegationToken, msgAndArgs ...interface{}) bool {
	t.Helper()

	ok := assert.True(t, expected.Equal(actual), msgAndArgs...)
	if !ok {
		t.Logf("expected: %#v", expected)
		t.Logf("actual:   %#v", actual)
	}

	return ok
}

//nolint:golint
func AssertVStateReport_ProvidedContentBodyEqual(t TestingT, expected, actual *rms.VStateReport_ProvidedContentBody, msgAndArgs ...interface{}) bool {
	t.Helper()

	ok := assert.True(t, expected.Equal(actual), msgAndArgs...)
	if !ok {
		t.Logf("expected: %#v", expected)
		t.Logf("actual:   %#v", actual)
	}

	return ok
}
