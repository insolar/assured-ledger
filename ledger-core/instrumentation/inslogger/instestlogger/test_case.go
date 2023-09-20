package instestlogger

import (
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func LogCase(target logcommon.TestingLogger, name string) {
	const TestCasePrefix = ""

	target.Log(TestCasePrefix + name)

	if isGlobalBasedOn(target) {
		// initialization was already done
		// for this logger, so can output now
		global.Logger().Event(logcommon.NoLevel, TestCasePrefix + name)
		return
	}

	t, ok := target.(markerT)
	if !ok {
		panic(throw.IllegalValue())
	}

	t.Cleanup(func() {
		// this output will be made at end of the test, just before FAIL/PASS/SKIP mark
		if global.IsInitialized() {
			global.Logger().Event(logcommon.NoLevel, TestCasePrefix + name)
		}
	})
}

