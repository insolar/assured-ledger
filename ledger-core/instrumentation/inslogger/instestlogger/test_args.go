// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package instestlogger

import (
	"flag"
	"strconv"
	"strings"


	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"
)


// readTestLogConfig MUST be in a separate, test-only package to avoid polluting cmd line with test args
func readTestLogConfig(cfg *configuration.Log, echoAll, emuMarks *bool) {
	_readTestLogConfig(cfg, echoAll, emuMarks, flag.CommandLine)
}

func _readTestLogConfig(cfg *configuration.Log, echoAll, emuMarks *bool, cmdLine *flag.FlagSet) {
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

	case cmdLine != nil && cmdLine.NArg() > 0:
		m := readArgsMap(cmdLine.Args())
		if outFile := m["TESTLOG_OUT"]; outFile != "" {
			cfg.OutputParams = outFile

			if v := m["TESTLOG_ECHO"]; v != "" && echoAll != nil {
				if b, err := strconv.ParseBool(v); err == nil {
					*echoAll = b
				}
			}

			if v := m["TESTLOG_MARKS"]; v != "" && emuMarks != nil {
				if b, err := strconv.ParseBool(v); err == nil {
					*emuMarks = b
				}
			}

			break
		}
		return

	default:
		return
	}

	cfg.OutputType = logoutput.FileOutput.String()
	cfg.Formatter = logcommon.JSONFormat.String()
}

func readArgsMap(args []string) (m map[string]string) {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "TESTLOG_") {
			continue
		}
		sep := strings.IndexByte(arg, '=')
		if sep < 0 {
			continue
		}
		if m == nil {
			m = map[string]string{}
		}
		n := arg[:sep]
		v := arg[sep+1:]
		if vv, err := strconv.Unquote(v); err == nil {
			v = vv
		}
		m[n] = v
	}
	return
}

var argEchoAll bool
var argEmuMarks bool
var argOutFile string

func initCmdOptions(cmdLine *flag.FlagSet) {
	cmdLine.BoolVar(&argEchoAll, "testlog.echo", false, "copy all log messages to console")
	cmdLine.BoolVar(&argEmuMarks, "testlog.marks", false, "emulate test run/pass/fail/skip marks")
	cmdLine.StringVar(&argOutFile, "testlog.out", "", "output file for json log")
}

func init() {
	initCmdOptions(flag.CommandLine)
}
