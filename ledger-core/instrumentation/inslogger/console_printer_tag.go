// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build convlogtxt testlogtxt

package inslogger

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/consprint"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"
)

func init() {
	logoutput.JSONConsoleWrapper = ConvertJSONConsoleOutput
}

func copyConsoleOutputConfig() consprint.Config {
	d := ConsoleWriterDefaults
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
	return consprint.NewConsolePrinter(in, copyConsoleOutputConfig())
}

func ConvertJSONTestingOutput(in logcommon.TestingLogger) logcommon.TestingLogger {
	return consprint.NewConsoleTestingPrinter(in, copyConsoleOutputConfig())
}
