// +build convlogtxt copylogtxt

package prettylog

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/consprint"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"
)

const ConvertJSONLogForConsole = true

func init() {
	logoutput.JSONConsoleWrapper = ConvertJSONConsoleOutput
}

func copyConsoleOutputConfig() consprint.Config {
	d := Defaults
	return consprint.Config{
		NoColor:    d.NoColor,
		TimeFormat: d.TimeFormat,
		PartsOrder: d.PartsOrder,

		FormatTimestamp:     d.FormatTimestamp,
		FormatLevel:         d.FormatLevel,
		FormatCaller:        d.FormatCaller,
		FormatMessage:       d.FormatMessage,
		FormatFieldName:     d.FormatFieldName,
		FormatFieldValue:    d.FormatFieldValue,
		FormatErrFieldName:  d.FormatErrFieldName,
		FormatErrFieldValue: d.FormatErrFieldValue,
	}
}

func ConvertJSONConsoleOutput(in io.Writer) io.Writer {
	if in == nil {
		return nil
	}
	return consprint.NewConsolePrinter(in, copyConsoleOutputConfig())
}

func ConvertJSONTestingOutput(in logcommon.TestingLogger) logcommon.TestingLogger {
	if in == nil {
		return nil
	}
	return consprint.NewConsoleTestingPrinter(in, copyConsoleOutputConfig())
}
