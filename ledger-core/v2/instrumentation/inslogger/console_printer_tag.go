// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build convlogtxt

package inslogger

import (
	"os"

	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger/consprint"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logoutput"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func init() {
	if !ConsoleWriterDefaults.Enable {
		return
	}

	if _, ok := logoutput.BareStdErr.Writer.(*os.File); !ok {
		// avoid cases of unexpected use
		panic(throw.IllegalState())
	}

	d := ConsoleWriterDefaults
	cfg := consprint.Config{
		NoColor: d.NoColor,
		TimeFormat: d.TimeFormat,
		PartsOrder: d.PartsOrder,

		FormatTimestamp: d.FormatTimestamp,
		FormatLevel: d.FormatLevel,
		FormatCaller: d.FormatCaller,
		FormatMessage: d.FormatMessage,
		FormatFieldName: d.FormatFieldName,
		FormatFieldValue: d.FormatFieldValue,
		FormatErrFieldName: d.FormatErrFieldName,
		FormatErrFieldValue: d.FormatErrFieldValue,
	}
	logoutput.BareStdErr.Writer = consprint.NewConsolePrinter(logoutput.BareStdErr.Writer, cfg)
}
