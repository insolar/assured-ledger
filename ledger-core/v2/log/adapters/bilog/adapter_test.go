///
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
///

package bilog

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	logm "github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logoutput"

	"github.com/stretchr/testify/require"
)

func TestTextFormat(t *testing.T) {
	suite.Run(t, &SuiteTextualLog{logFormat: logcommon.TextFormat})
	// TODO PLAT-44 parsing for text
}

func TestJsonFormat(t *testing.T) {
	suite.Run(t, &SuiteTextualLog{logFormat: logcommon.JsonFormat,
		parseFn: func(t *testing.T, b []byte) map[string]interface{} {
			fields := make(map[string]interface{})
			err := json.Unmarshal(b, &fields)
			require.NoError(t, err, "unmarshal %s", b)
			return fields
		},
	})
}

func TestPbufFormat(t *testing.T) {
	suite.Run(t, &SuiteTextualLog{logFormat: logcommon.PbufFormat})
	// TODO PLAT-44 parsing for pbuf
}

type SuiteTextualLog struct {
	suite.Suite
	logFormat logcommon.LogFormat
	parseFn   func(*testing.T, []byte) map[string]interface{}
}

func (st SuiteTextualLog) newAdapter(level logcommon.Level) logm.Logger {
	zc := logcommon.Config{}

	var err error
	zc.BareOutput, err = logoutput.OpenLogBareOutput(logoutput.StdErrOutput, "")

	t := st.T()
	require.NoError(t, err)
	require.NotNil(t, zc.BareOutput.Writer)

	zc.Output = logcommon.OutputConfig{
		Format: st.logFormat,
	}
	zc.MsgFormat = logfmt.GetDefaultLogMsgFormatter()

	zb := logm.NewBuilder(NewFactory(nil, false), zc, level)

	l, err := zb.Build()
	require.NoError(t, err)
	require.False(t, l.IsZero())
	return l
}

type logRecord struct {
	s string
}

func (st SuiteTextualLog) TestFields() {
	t := st.T()

	if st.parseFn == nil {
		t.Skip("missing parser for", st.logFormat.String())
		return
	}

	t.Log()
	buf := bytes.Buffer{}
	lg, _ := st.newAdapter(logcommon.InfoLevel).Copy().
		WithOutput(&buf).
		WithCaller(logcommon.CallerFieldWithFuncName).
		WithMetrics(logcommon.LogMetricsWriteDelayField | logcommon.LogMetricsTimestamp).
		Build()

	const logstring = "msgstrval"
	lg.WithField("testfield", 200.200).Warnm(logRecord{logstring})
	fileLine, _ := logoutput.GetCallerFileNameWithLine(0, -1)

	c := st.parseFn(t, buf.Bytes())

	require.Equal(t, "warn", c["level"], "right message")
	require.Equal(t, logstring, c["s"], "right message")
	require.Equal(t, "bilog.logRecord", c["message"], "right message")
	require.Contains(t, c["caller"], fileLine, "right caller line")
	ltime, err := time.Parse(time.RFC3339Nano, c["time"].(string))
	require.NoError(t, err, "parseable time")
	ldur := time.Now().Sub(ltime)
	require.True(t, ldur >= 0, "worktime is not less than zero")
	require.True(t, ldur < time.Second, "worktime lesser than second")
	require.Equal(t, 200.200, c["testfield"], "customfield")
	require.NotNil(t, c["writeDuration"], "duration exists")
}

func (st SuiteTextualLog) TestCallerInfoWithFunc() {
	t := st.T()
	log := st.newAdapter(logcommon.InfoLevel)

	var buf bytes.Buffer
	log = log.Copy().WithOutput(&buf).WithCaller(logcommon.CallerFieldWithFuncName).MustBuild()

	log.Error("test")
	fileName, funcName := logoutput.GetCallerFileNameWithLine(0, -1)

	s := buf.String()
	require.Contains(t, s, fileName)
	require.Contains(t, s, funcName)
}

func (st SuiteTextualLog) TestCallerInfo() {
	t := st.T()
	log := st.newAdapter(logcommon.InfoLevel)

	var buf bytes.Buffer
	log = log.Copy().WithOutput(&buf).WithCaller(logcommon.CallerField).MustBuild()

	log.Error("test")
	fileName, _ := logoutput.GetCallerFileNameWithLine(0, -1)

	s := buf.String()
	require.Contains(t, s, fileName)
}

func (st SuiteTextualLog) TestInheritFields() {
	t := st.T()
	log := st.newAdapter(logcommon.InfoLevel)

	var buf bytes.Buffer
	log = log.Copy().WithOutput(&buf).WithCaller(logcommon.CallerField).WithField("field1", "value1").MustBuild()

	log = log.WithField("field2", "value2")

	var buf2 bytes.Buffer
	log = log.Copy().WithOutput(&buf2).MustBuild()

	log.Error("test")

	s := buf2.String()
	require.Contains(t, s, "value1")
	require.Contains(t, s, "value2")
}

func (st SuiteTextualLog) TestChangeLevel() {
	t := st.T()
	log := st.newAdapter(logcommon.InfoLevel)
	require.True(t, log.Is(logcommon.InfoLevel))

	prevLog := log
	log = log.Copy().WithLevel(logcommon.InfoLevel).MustBuild()
	require.Equal(t, prevLog, log)
	require.True(t, log.Is(logcommon.InfoLevel))

	log = log.Copy().WithLevel(logcommon.DebugLevel).MustBuild()
	require.True(t, log.Is(logcommon.DebugLevel))
}

func (st SuiteTextualLog) TestBuildFields() {
	t := st.T()
	log := st.newAdapter(logcommon.InfoLevel)
	require.True(t, log.Is(logcommon.InfoLevel))

	log = log.Copy().WithField("test0", "value0").MustBuild()

	var buf bytes.Buffer
	log = log.Copy().WithOutput(&buf).MustBuild()
	log = log.Copy().WithField("test1", "value1").MustBuild()

	log.Error("test")

	s := buf.String()
	require.Contains(t, s, "value0")
	require.Contains(t, s, "value1")
	buf.Reset()

	log = log.Copy().WithoutInheritedFields().WithField("test2", "value2").MustBuild()

	log.Error("test")
	s = buf.String()
	require.NotContains(t, s, "value0")
	require.NotContains(t, s, "value1")
	require.Contains(t, s, "value2")
}

func (st SuiteTextualLog) TestBuildDynFields() {
	t := st.T()
	log := st.newAdapter(logcommon.InfoLevel)

	log = log.Copy().
		WithDynamicField("test0", func() interface{} { return "value0" }).
		WithField("test00", "static0").
		MustBuild()

	var buf bytes.Buffer
	log = log.Copy().WithOutput(&buf).MustBuild()

	log = log.Copy().WithDynamicField("test1", func() interface{} { return "value1" }).MustBuild()

	log.Error("test")

	s := buf.String()
	require.Contains(t, s, "static0")
	require.Contains(t, s, "value0")
	require.Contains(t, s, "value1")

	buf.Reset()
	log = log.Copy().WithoutInheritedDynFields().WithDynamicField("test2", func() interface{} { return "value2" }).MustBuild()

	log.Error("test")
	s = buf.String()
	require.Contains(t, s, "static0")
	require.NotContains(t, s, "value0")
	require.NotContains(t, s, "value1")
	require.Contains(t, s, "value2")

	buf.Reset()
	log = log.Copy().WithoutInheritedFields().WithDynamicField("test3", func() interface{} { return "value3" }).MustBuild()

	log.Error("test")
	s = buf.String()
	require.NotContains(t, s, "static0")
	require.NotContains(t, s, "value0")
	require.NotContains(t, s, "value1")
	require.NotContains(t, s, "value2")
	require.Contains(t, s, "value3")

	buf.Reset()
	log = log.Copy().WithoutInheritedFields().WithDynamicField("test3", func() interface{} { return "value-3" }).MustBuild()

	log.Error("test")
	s = buf.String()
	require.NotContains(t, s, "static0")
	require.NotContains(t, s, "value0")
	require.NotContains(t, s, "value1")
	require.NotContains(t, s, "value2")
	require.NotContains(t, s, "value3")
	require.Contains(t, s, "value-3")
}

func (st SuiteTextualLog) TestFatal() {
	t := st.T()
	zc := logcommon.Config{}

	var buf bytes.Buffer
	wg := sync.WaitGroup{}
	wg.Add(1)
	zc.BareOutput = logcommon.BareOutput{
		Writer: &buf,
		FlushFn: func() error {
			wg.Done()
			select {} // hang up to stop zerolog's call to os.Exit
		},
	}
	zc.Output = logcommon.OutputConfig{Format: logcommon.TextFormat}
	zc.MsgFormat = logfmt.GetDefaultLogMsgFormatter()
	zc.Instruments.SkipFrameCountBaseline = 0

	zb := logm.NewBuilder(binLogFactory{}, zc, logcommon.InfoLevel)
	log, err := zb.Build()

	require.NoError(t, err)
	require.NotNil(t, log)

	log.Error("errorMsgText")
	go log.Fatal("fatalMsgText") // it will hang on flush
	wg.Wait()

	s := buf.String()
	require.Contains(t, s, "errorMsgText")
	require.Contains(t, s, "fatalMsgText")
}

func (st SuiteTextualLog) TestPanic() {
	t := st.T()
	zc := logcommon.Config{}

	var buf bytes.Buffer
	wg := sync.WaitGroup{}
	wg.Add(1)
	zc.BareOutput = logcommon.BareOutput{
		Writer: &buf,
		FlushFn: func() error {
			wg.Done()
			return nil
		},
	}
	zc.Output = logcommon.OutputConfig{Format: logcommon.TextFormat}
	zc.MsgFormat = logfmt.GetDefaultLogMsgFormatter()
	zc.Instruments.SkipFrameCountBaseline = 0

	zb := logm.NewBuilder(binLogFactory{}, zc, logcommon.InfoLevel)
	log, err := zb.Build()

	require.NoError(t, err)
	require.NotNil(t, log)

	log.Error("errorMsgText")
	require.PanicsWithValue(t, "panicMsgText", func() {
		log.Panic("panicMsgText")
	})
	wg.Wait()
	log.Error("errorNextMsgText")

	s := buf.String()
	require.Contains(t, s, "errorMsgText")
	require.Contains(t, s, "panicMsgText")
	require.Contains(t, s, "errorNextMsgText")
}
