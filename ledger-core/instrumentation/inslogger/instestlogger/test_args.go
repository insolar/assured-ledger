// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package instestlogger

import (
	"flag"
	"os"
	"strconv"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"
)


// readTestLogConfig MUST be in a separate, test-only package to avoid polluting cmd line with test args
func readTestLogConfig(cfg *configuration.Log, echoAll, emuMarks *bool) {
	if !flag.Parsed() {
		flag.Parse()
	}

	switch {
	case argOutFile != "":
		cfg.OutputParams = argOutFile

		if echoAll != nil {
			*echoAll = argEchoAll
		}
		if emuMarks != nil {
			*emuMarks = argEmuMarks
		}

	case os.Getenv("TESTLOG_OUT") != "":
		cfg.OutputParams = os.Getenv("TESTLOG_OUT")

		if v := os.Getenv("TESTLOG_ECHO"); v != "" && echoAll != nil {
			if b, err := strconv.ParseBool(v); err == nil {
				*echoAll = b
			}
		}

		if v := os.Getenv("TESTLOG_MARKS"); v != "" && emuMarks != nil {
			if b, err := strconv.ParseBool(v); err == nil {
				*emuMarks = b
			}
		}

	default:
		return
	}

	cfg.OutputType = logoutput.FileOutput.String()
	cfg.Formatter = logcommon.JSONFormat.String()
}

var argEchoAll bool
var argEmuMarks bool
var argOutFile string

func init() {
	flag.BoolVar(&argEchoAll, "testlog.echo", false, "copy all log messages to console")
	flag.BoolVar(&argEmuMarks, "testlog.marks", false, "emulate test run/pass/fail/skip marks")
	flag.StringVar(&argOutFile, "testlog.out", "", "output file for json log")
}
