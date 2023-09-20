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
func readTestLogConfig(cfg *configuration.Log, echoAll, emuMarks, prettyPrintJSON *bool) {
	_readTestLogConfig(cfg, echoAll, emuMarks, prettyPrintJSON, flag.CommandLine)
}

func _readTestLogConfig(cfg *configuration.Log, echoAll, emuMarks, prettyPrintJSON *bool, cmdLine *flag.FlagSet) {
	if !flag.Parsed() {
		flag.Parse()
	}

	switch {
	case argOutFile != "":
		cfg.OutputParams = argOutFile
		readLogFmt(argOutFmt, cfg, prettyPrintJSON)

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

			readLogFmt(m["TESTLOG_TXT"], cfg, prettyPrintJSON)

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
}

const testlogJSON = 0
const testlogConvertJSON = 1
const testlogText = 2

func _readTxtFmt(v string) (int, bool) {
	switch v {
	case "0", "f", "F", "false", "FALSE", "False", "json":
		return testlogJSON, true
	case "1", "t", "T", "true", "TRUE", "True", "conv", "convert":
		return testlogConvertJSON, true
	case "text":
		return testlogText, true
	}
	return testlogJSON, false
}

func readLogFmt(v string, cfg *configuration.Log, prettyPrintJSON *bool) {
	cfg.Formatter = logcommon.JSONFormat.String()

	switch mode, ok := _readTxtFmt(v); {
	case !ok:
	case mode == testlogConvertJSON:
		if prettyPrintJSON != nil {
			*prettyPrintJSON = true
		}
	case mode == testlogText:
		cfg.Formatter = logcommon.TextFormat.String()
		switch cfg.Adapter {
		case "zerolog", "":
			cfg.Adapter = "bilog"
		}
	default:
		if prettyPrintJSON != nil {
			*prettyPrintJSON = false
		}
	}
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
var argOutFmt string
var argOutFile string

func initCmdOptions(cmdLine *flag.FlagSet) {
	cmdLine.BoolVar(&argEchoAll, "testlog.echo", false, "copy all log messages to console")
	cmdLine.BoolVar(&argEmuMarks, "testlog.marks", false, "emulate test run/pass/fail/skip marks")
	cmdLine.StringVar(&argOutFmt, "testlog.txt", "", "conversion to txt")
	cmdLine.StringVar(&argOutFile, "testlog.out", "", "output file for json log")
}

func init() {
	initCmdOptions(flag.CommandLine)
}
