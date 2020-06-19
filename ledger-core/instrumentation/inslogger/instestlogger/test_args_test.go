// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
		echoAll, emuMarks bool
	)
	cfg.OutputParams = "stub"
	cfg.OutputType = "stub"
	cfg.Formatter = "stub"

	argEchoAll = true
	argEmuMarks = true

	argOutFile = ""
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, nil)
	require.Equal(t, "stub", cfg.OutputParams)
	require.False(t, echoAll)
	require.False(t, emuMarks)

	argOutFile = "<test1>"
	_readTestLogConfig(&cfg, nil, nil, nil)
	require.Equal(t, "<test1>", cfg.OutputParams)

	argOutFile = "<test2>"
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, nil)
	require.Equal(t, "<test2>", cfg.OutputParams)
	require.True(t, echoAll)
	require.True(t, emuMarks)

	argEmuMarks = false
	argOutFile = "<test3>"
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, nil)
	require.Equal(t, "<test3>", cfg.OutputParams)
	require.True(t, echoAll)
	require.False(t, emuMarks)
}

func TestReadTestLogConfig_Args(t *testing.T) {
	argOutFile = ""

	var (
		cfg configuration.Log
		echoAll, emuMarks bool
	)
	cfg.OutputParams = "stub"
	cfg.OutputType = "stub"
	cfg.Formatter = "stub"

	cmdLine := flag.NewFlagSet("", flag.PanicOnError)
	initCmdOptions(cmdLine)

	require.NoError(t, cmdLine.Parse([]string{
		"-testlog.echo=1",
		"-testlog.marks=0",
	}))
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, cmdLine)

	require.Equal(t, "stub", cfg.OutputParams)
	require.False(t, echoAll)
	require.False(t, emuMarks)


	require.NoError(t, cmdLine.Parse([]string{
		"-testlog.out=test1",
		"-testlog.echo=1",
		"-testlog.marks=0",
	}))
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, cmdLine)

	require.Equal(t, "test1", cfg.OutputParams)
	require.True(t, echoAll)
	require.False(t, emuMarks)

	require.NoError(t, cmdLine.Parse([]string{
		"-testlog.out=test2",
		"-testlog.echo=0",
		"-testlog.marks=1",
	}))
	_readTestLogConfig(&cfg, &echoAll, &emuMarks, cmdLine)

	require.Equal(t, "test2", cfg.OutputParams)
	require.False(t, echoAll)
	require.True(t, emuMarks)
}
