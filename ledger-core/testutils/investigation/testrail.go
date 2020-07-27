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

	"github.com/insolar/assured-ledger/ledger-core/log"
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

type logCaseHeader struct {
	*log.Msg    `txt:"testrail"`
	ID          string
	TestPackage string
	TestName    string
}

func (h logCaseHeader) ConstructFooter() logCaseFooter {
	return logCaseFooter{
		ID:          h.ID,
		TestPackage: h.TestPackage,
		TestName:    h.TestName,
	}
}

type logCaseFooter struct {
	*log.Msg    `txt:"testrail"`
	ID          string
	TestPackage string
	TestName    string
	Status      string
	SkippedLink string `opt:""`
}

func LogCaseExt(target Testing, name string, skipDepth int) {
	target.Helper()

	if skipDepth < 0 {
		panic(throw.IllegalValue())
	}

	header := logCaseHeader{
		ID:          name,
		TestName:    target.Name(),
		TestPackage: getParentPackage(skipDepth),
	}
	global.Logger().Event(logcommon.NoLevel, header)

	target.Cleanup(func() {
		footer := header.ConstructFooter()

		switch {
		case target.Skipped():
			footer.Status = "SKIP"

			if skippedLink, present := skipList.Load(target.Name()); present {
				footer.SkippedLink = skippedLink.(string)
			}
		case target.Failed():
			footer.Status = "FAIL"
		case checkPanicInStack():
			footer.Status = "FAIL"
		default:
			footer.Status = "PASS"
		}

		// this output will be made at end of the test, just before FAIL/PASS/SKIP mark
		global.Logger().Event(logcommon.NoLevel, footer)
	})
}

func LogCase(target Testing, name string) {
	target.Helper()

	LogCaseExt(target, name, 1)
}
