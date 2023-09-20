// +build !convlogtxt,!copylogtxt

package prettylog

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
)

const ConvertJSONLogForConsole = false

func ConvertJSONConsoleOutput(in io.Writer) io.Writer {
	return in
}

func ConvertJSONTestingOutput(in logcommon.TestingLogger) logcommon.TestingLogger {
	return in
}
