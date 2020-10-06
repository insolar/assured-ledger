// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/log"
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

func AssertAnyTranscriptEqual(t TestingT, expected, actual *rms.Any, logger log.Logger) bool {
	t.Helper()

	ok := assert.True(t, expected.Equal(actual))
	if !ok {
		logger.Infof("expected: %#v", expected)
		logger.Infof("actual: %#v", actual)

		diff := diff(expected, actual)
		logger.Infof("Diff: %v", diff)
	}

	return ok
}

// diff returns a diff of both values as long as both are of the same type and
// are a struct, map, slice, array or string. Otherwise it returns an empty string.
func diff(expected interface{}, actual interface{}) string {
	if expected == nil || actual == nil {
		return ""
	}

	et, ek := typeAndKind(expected)
	at, _ := typeAndKind(actual)

	if et != at {
		return ""
	}

	if ek != reflect.Struct && ek != reflect.Map && ek != reflect.Slice && ek != reflect.Array && ek != reflect.String {
		return ""
	}

	var e, a string
	if et != reflect.TypeOf("") {
		e = spewConfig.Sdump(expected)
		a = spewConfig.Sdump(actual)
	} else {
		e = reflect.ValueOf(expected).String()
		a = reflect.ValueOf(actual).String()
	}

	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(e),
		B:        difflib.SplitLines(a),
		FromFile: "Expected",
		FromDate: "",
		ToFile:   "Actual",
		ToDate:   "",
		Context:  1,
	})

	return "\nDiff:\n" + diff
}

var spewConfig = spew.ConfigState{
	Indent:                  " ",
	DisablePointerAddresses: true,
	DisableCapacities:       true,
	SortKeys:                true,
}

func typeAndKind(v interface{}) (reflect.Type, reflect.Kind) {
	t := reflect.TypeOf(v)
	k := t.Kind()

	if k == reflect.Ptr {
		t = t.Elem()
		k = t.Kind()
	}
	return t, k
}
