package insrail

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

func lastNotInternalPC() uintptr {
	var lastNotInternalPC uintptr

	for i := 0; ; i++ {
		pc, file, _, ok := runtime.Caller(i)
		if !ok {
			return lastNotInternalPC
		}
		if strings.HasSuffix(file, "src/testing/testing.go") ||
			strings.Contains(file, "src/runtime") {

			continue
		} else {
			lastNotInternalPC = pc
		}
	}
}

func getTestingPackage() string {
	pc := lastNotInternalPC()
	if pc == 0 {
		panic(throw.IllegalState())
	}

	name := runtime.FuncForPC(pc).Name()

	name = trimLambdaSuffix(name)
	name = trimFunctionName(name)

	return name
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

func LogSkip(target Testing, jiraLink string) {
	target.Helper()

	skipList.Store(target.Name(), jiraLink)
	target.Skip(jiraLink)
}

func logCaseConstructor(target Testing, footer logCaseFooter) func() {
	return func() {
		if footer.Status == "" {
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
		}

		// this output will be made at end of the test, just before FAIL/PASS/SKIP mark
		global.Logger().Event(logcommon.NoLevel, footer)
	}
}

type Option interface {
	modifyConfig(target Testing, footer *logCaseFooter)
	applyAfter(target Testing)
}

type SkipOption struct {
	link string
}

func Skip(jiraLink string) Option {
	return SkipOption{link: jiraLink}
}

func (s SkipOption) modifyConfig(_ Testing, footer *logCaseFooter) {
	footer.Status = "SKIP"
	footer.SkippedLink = s.link
}

func (s SkipOption) applyAfter(target Testing) {
	target.Skip(s.link)
}

func LogCaseExt(target Testing, name string, opts ...Option) {
	target.Helper()

	header := logCaseHeader{
		ID:          name,
		TestName:    target.Name(),
		TestPackage: getTestingPackage(),
	}
	global.Logger().Event(logcommon.NoLevel, header)

	footer := header.ConstructFooter()
	for _, opt := range opts {
		opt.modifyConfig(target, &footer)
	}

	target.Cleanup(logCaseConstructor(target, footer))

	for _, opt := range opts {
		opt.applyAfter(target)
	}
}

func LogCase(target Testing, name string) {
	LogCaseExt(target, name)
}

func LogSkipCase(target Testing, name string, jiraLink string) {
	LogCaseExt(target, name, Skip(jiraLink))
}
