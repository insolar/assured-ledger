// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
