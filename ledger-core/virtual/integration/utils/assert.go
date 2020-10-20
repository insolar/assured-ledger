// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/messagediff"
)

type TestingT interface {
	Helper()
	Logf(format string, args ...interface{})
	assert.TestingT
}

func AssertVCallRequestEqual(t TestingT, expected, actual *rms.VCallRequest, msgAndArgs ...interface{}) bool {
	// t.Helper()

	ok := assert.True(t, expected.Equal(actual), msgAndArgs...)
	if !ok {

		diff, _ := messagediff.PrettyDiff(expected, actual)
		t.Logf("Diff: \n%v", diff)
	}

	return true
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

// nolint:golint
func AssertVStateReport_ProvidedContentBodyEqual(t TestingT, expected, actual *rms.VStateReport_ProvidedContentBody, msgAndArgs ...interface{}) bool {
	t.Helper()

	ok := assert.True(t, expected.Equal(actual), msgAndArgs...)
	if !ok {
		t.Logf("expected: %#v", expected)
		t.Logf("actual:   %#v", actual)
	}

	return ok
}

func AssertAnyTranscriptEqual(t TestingT, expected, actual *rms.Any) bool {
	t.Helper()

	ok := assert.True(t, expected.Equal(actual))

	if !ok {

		diff, _ := messagediff.PrettyDiff(expected.Get(), actual.Get())
		t.Logf("Diff: %#v", diff)

	}

	return ok
}
