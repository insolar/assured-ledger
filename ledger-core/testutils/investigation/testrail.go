// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package investigation

import (
	"bytes"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var skipList = sync.Map{}

type Testing interface {
	Name() string
	Failed() bool
	Skipped() bool

	Log(args ...interface{})
	Skip(args ...interface{})

	Helper()
	Cleanup(func())
}

var (
	panicSymptom1 = []byte("/usr/local/go/src/runtime/panic.go")
	panicSymptom2 = []byte("panic(")
)

func checkPanicInStack() bool {
	buf := make([]byte, 65536)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, 2*len(buf))
	}

	return bytes.Contains(buf, panicSymptom1) && bytes.Contains(buf, panicSymptom2)
}

var trimLambdaSuffixRe = regexp.MustCompile(`(.*)\.func(?:[\d]+\.)*[\d]+`)

func trimLambdaSuffix(fullName string) string {
	pos := trimLambdaSuffixRe.FindStringSubmatch(fullName)
	if len(pos) > 0 {
		return pos[1]
	}
	return fullName
}

func trimFunctionName(fullName string) string {
	nameParts := strings.Split(fullName, ".")
	return strings.Join(nameParts[:len(nameParts)-1], ".")
}

func getParentPackage(skipDepth int) string {
	// skip that function and LogCase function
	pc, _, _, ok := runtime.Caller(2 + skipDepth)
	if !ok {
		return ""
	}
	name := runtime.FuncForPC(pc).Name()

	name = trimLambdaSuffix(name)
	name = trimFunctionName(name)

	return name
}

func LogSkip(target Testing, jiraLink string) {
	target.Helper()

	skipList.Store(target.Name(), jiraLink)
	target.Skip(jiraLink)
}

func LogCaseExt(target Testing, name string, skipDepth int) {
	target.Helper()

	if skipDepth < 0 {
		panic(throw.IllegalValue())
	}

	const TestCasePrefix = ""

	if global.IsInitialized() {
		global.Logger().Event(logcommon.NoLevel, name)
	} else {
		target.Log(TestCasePrefix + name)
	}

	testName := target.Name()
	if packageName := getParentPackage(skipDepth); packageName != "" {
		testName = packageName + "." + testName
	}

	target.Cleanup(func() {
		result := struct {
			TestName    string `opt:""`
			ID          string
			Status      string
			SkippedLink string `opt:""`
		}{}
		result.ID = TestCasePrefix + name
		result.TestName = testName

		if target.Skipped() {
			result.Status = "SKIP"

			if skippedLink, present := skipList.Load(target.Name()); present {
				result.SkippedLink = skippedLink.(string)
			}
		} else if target.Failed() {
			result.Status = "FAIL"
		} else if checkPanicInStack() {
			result.Status = "FAIL"
		} else {
			result.Status = "PASS"
		}

		// this output will be made at end of the test, just before FAIL/PASS/SKIP mark
		if global.IsInitialized() {
			global.Logger().Event(logcommon.NoLevel, result)
		} else {
			output := []interface{}{result.ID, result.TestName, result.Status}
			if result.SkippedLink != "" {
				output = append(output, "\""+result.SkippedLink+"\"")
			}
			target.Log(output...)
		}
	})
}

func LogCase(target Testing, name string) {
	target.Helper()

	LogCaseExt(target, name, 1)
}
