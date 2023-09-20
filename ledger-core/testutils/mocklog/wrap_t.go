package mocklog

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/args"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Tester interface {
// 	minimock.Tester
//	logcommon.TestingLogger

	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Error(...interface{})
	Errorf(format string, args ...interface{})
	FailNow()

	Helper()
	Log(...interface{})
}

func T(t Tester) Tester {
	switch t.(type) {
	case nil:
		return nil
	case stackReport:
		return t
	}
	return stackReport{t}
}

type stackReport struct {
	t Tester
}

func (v stackReport) FailNow() {
	v.t.FailNow()
}

func (v stackReport) Helper() {
	v.t.Helper()
}

func (v stackReport) Log(args ...interface{}) {
	v.t.Helper()
	v.t.Log(args...)
}

func (v stackReport) Error(a ...interface{}) {
	v.t.Helper()
	v.t.Error(args.AppendIntfArgs(a, v.stackText())...)
}

func (v stackReport) Errorf(format string, args ...interface{}) {
	v.t.Helper()
	v.t.Error(fmt.Sprintf(format, args...), v.stackText())
}

func (v stackReport) Fatal(a ...interface{}) {
	v.t.Helper()
	v.t.Fatal(args.AppendIntfArgs(a, v.stackText())...)
}

func (v stackReport) Fatalf(format string, args ...interface{}) {
	v.t.Helper()
	v.t.Fatal(fmt.Sprintf(format, args...), v.stackText())
}

func (v stackReport) stackText() string {
	return throw.JoinStackText("", throw.CaptureStack(1))
}
