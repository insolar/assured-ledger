package instestlogger

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
)

func TestReadArgsMap(t *testing.T) {
	m := readArgsMap([]string{
		`TESTLOG_X=a`,
		`TESTLOG_Y=a b`,
		`TESTLOG_Z="a z"`,
		`TESTLOG_B="b a`,
		`TESTLOG_A=1`,
		`TESTLOG_A`,
		`ELSE`,
		`TESTLOG`,
	})

	require.Equal(t, 5, len(m))
	require.Equal(t, `a`, m[`TESTLOG_X`])
	require.Equal(t, `a b`, m[`TESTLOG_Y`])
	require.Equal(t, `a z`, m[`TESTLOG_Z`])
	require.Equal(t, `"b a`, m[`TESTLOG_B`])
	require.Equal(t, `1`, m[`TESTLOG_A`])
}

func TestReadTestLogConfig_Options(t *testing.T) {
	var (
		cfg configuration.Log
		echoAll, emuMarks, prettyPrintJSON bool
	)
	cfg.OutputParams = "stub"
	cfg.OutputType = "stub"
	cfg.Formatter = "stub"

	argEchoAll = true
	argEmuMarks = true
	argOutFmt = ""

	argOutFile = ""
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, &prettyPrintJSON, nil)
	require.Equal(t, "stub", cfg.OutputParams)
	require.Equal(t, "stub", cfg.Formatter)
	require.False(t, echoAll)
	require.False(t, emuMarks)
	require.False(t, prettyPrintJSON)

	argOutFile = "<test1>"
	_readTestLogConfig(&cfg, nil, nil, nil, nil)
	require.Equal(t, "<test1>", cfg.OutputParams)
	require.Equal(t, "json", cfg.Formatter)

	argOutFile = "<test2>"
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, &prettyPrintJSON, nil)
	require.Equal(t, "<test2>", cfg.OutputParams)
	require.Equal(t, "json", cfg.Formatter)
	require.True(t, echoAll)
	require.True(t, emuMarks)
	require.False(t, prettyPrintJSON)

	argEmuMarks = false
	argOutFile = "<test3>"
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, &prettyPrintJSON, nil)
	require.Equal(t, "<test3>", cfg.OutputParams)
	require.Equal(t, "json", cfg.Formatter)
	require.True(t, echoAll)
	require.False(t, emuMarks)
	require.False(t, prettyPrintJSON)

	argOutFile = "<test4>"
	argOutFmt = "0"
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, &prettyPrintJSON, nil)
	require.Equal(t, "<test4>", cfg.OutputParams)
	require.Equal(t, "json", cfg.Formatter)
	require.False(t, prettyPrintJSON)

	argOutFile = "<test5>"
	argOutFmt = "1"
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, &prettyPrintJSON, nil)
	require.Equal(t, "<test5>", cfg.OutputParams)
	require.Equal(t, "json", cfg.Formatter)
	require.True(t, prettyPrintJSON)

	prettyPrintJSON = false
	argOutFile = "<test6>"
	argOutFmt = "text"
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, &prettyPrintJSON, nil)
	require.Equal(t, "<test6>", cfg.OutputParams)
	require.Equal(t, "text", cfg.Formatter)
	require.False(t, prettyPrintJSON)

	cmdLine := flag.NewFlagSet("", flag.PanicOnError)
	initCmdOptions(cmdLine)
	require.NoError(t, cmdLine.Parse([]string{
		"-testlog.out=test2",
		"-testlog.echo=0",
		"-testlog.marks=1",
	}))
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, &prettyPrintJSON, cmdLine)

	require.Equal(t, "test2", cfg.OutputParams)
	require.False(t, echoAll)
	require.True(t, emuMarks)
}

func TestReadTestLogConfig_Args(t *testing.T) {
	argOutFile = ""

	var (
		cfg configuration.Log
		echoAll, emuMarks, prettyPrintJSON bool
	)
	cfg.OutputParams = "stub"
	cfg.OutputType = "stub"
	cfg.Formatter = "stub"

	cmdLine := flag.NewFlagSet("", flag.PanicOnError)

	require.NoError(t, cmdLine.Parse([]string{
		"TESTLOG_ECHO=1",
		"TESTLOG_MARKS=0",
	}))
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, &prettyPrintJSON, cmdLine)

	require.Equal(t, "stub", cfg.OutputParams)
	require.Equal(t, "stub", cfg.Formatter)
	require.False(t, echoAll)
	require.False(t, emuMarks)
	require.False(t, prettyPrintJSON)

	require.NoError(t, cmdLine.Parse([]string{
		"TESTLOG_OUT=test1",
		"TESTLOG_ECHO=1",
		"TESTLOG_MARKS=0",
	}))
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, &prettyPrintJSON, cmdLine)

	require.Equal(t, "test1", cfg.OutputParams)
	require.Equal(t, "json", cfg.Formatter)
	require.True(t, echoAll)
	require.False(t, emuMarks)
	require.False(t, prettyPrintJSON)

	require.NoError(t, cmdLine.Parse([]string{
		"TESTLOG_OUT=test2",
		"TESTLOG_ECHO=0",
		"TESTLOG_MARKS=1",
		"TESTLOG_TXT=0",
	}))
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, &prettyPrintJSON, cmdLine)

	require.Equal(t, "test2", cfg.OutputParams)
	require.Equal(t, "json", cfg.Formatter)
	require.False(t, echoAll)
	require.True(t, emuMarks)
	require.False(t, prettyPrintJSON)

	require.NoError(t, cmdLine.Parse([]string{
		"TESTLOG_OUT=test3",
		"TESTLOG_ECHO=0",
		"TESTLOG_MARKS=0",
		"TESTLOG_TXT=1",
	}))
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, &prettyPrintJSON, cmdLine)

	require.Equal(t, "test3", cfg.OutputParams)
	require.Equal(t, "json", cfg.Formatter)
	require.False(t, echoAll)
	require.False(t, emuMarks)
	require.True(t, prettyPrintJSON)

	prettyPrintJSON = false
	require.NoError(t, cmdLine.Parse([]string{
		"TESTLOG_OUT=test4",
		"TESTLOG_ECHO=0",
		"TESTLOG_MARKS=0",
		"TESTLOG_TXT=text",
	}))
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, &prettyPrintJSON, cmdLine)

	require.Equal(t, "test4", cfg.OutputParams)
	require.Equal(t, "text", cfg.Formatter)
	require.False(t, echoAll)
	require.False(t, emuMarks)
	require.False(t, prettyPrintJSON)
}
