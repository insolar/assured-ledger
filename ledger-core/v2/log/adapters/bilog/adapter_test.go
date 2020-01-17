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
	"sync"
	"testing"

	logm "github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logoutput"

	"github.com/stretchr/testify/require"
)

func newAdapter(level logcommon.Level) (logm.Logger, error) {
	zc := logcommon.Config{}

	var err error
	zc.BareOutput, err = logoutput.OpenLogBareOutput(logoutput.StdErrOutput, "")
	if err != nil {
		return logm.Logger{}, err
	}
	if zc.BareOutput.Writer == nil {
		panic("output is nil")
	}

	zc.Output = logcommon.OutputConfig{
		Format: logcommon.TextFormat,
	}
	zc.MsgFormat = logfmt.GetDefaultLogMsgFormatter()

	zb := NewBuilder(zc, level, nil)
	return zb.Build()
}

func TestLogAdapter_CallerInfoWithFunc(t *testing.T) {
	log, err := newAdapter(logcommon.InfoLevel)
	require.NoError(t, err)
	require.NotNil(t, log)

	var buf bytes.Buffer
	log, err = log.Copy().WithOutput(&buf).WithCaller(logcommon.CallerFieldWithFuncName).Build()
	require.NoError(t, err)

	log.Error("test")

	s := buf.String()
	require.Contains(t, s, "adapter_test.go:62")
	require.Contains(t, s, "TestLogAdapter_CallerInfoWithFunc")
}

func TestLogAdapter_CallerInfo(t *testing.T) {
	log, err := newAdapter(logcommon.InfoLevel)

	require.NoError(t, err)
	require.NotNil(t, log)

	var buf bytes.Buffer
	log, err = log.Copy().WithOutput(&buf).WithCaller(logcommon.CallerField).Build()
	require.NoError(t, err)

	log.Error("test")

	s := buf.String()
	require.Contains(t, s, "adapter_test.go:79")
}

func TestLogAdapter_InheritFields(t *testing.T) {
	log, err := newAdapter(logcommon.InfoLevel)

	require.NoError(t, err)
	require.NotNil(t, log)

	var buf bytes.Buffer
	log, err = log.Copy().WithOutput(&buf).WithCaller(logcommon.CallerField).WithField("field1", "value1").Build()
	require.NoError(t, err)

	log = log.WithField("field2", "value2")

	var buf2 bytes.Buffer
	log, err = log.Copy().WithOutput(&buf2).Build()
	require.NoError(t, err)

	log.Error("test")

	s := buf2.String()
	require.Contains(t, s, "value1")
	require.Contains(t, s, "value2")
}

func TestLogAdapter_ChangeLevel(t *testing.T) {
	log, err := newAdapter(logcommon.InfoLevel)

	require.NoError(t, err)
	require.NotNil(t, log)
	require.True(t, log.Is(logcommon.InfoLevel))

	prevLog := log
	log, err = log.Copy().WithLevel(logcommon.InfoLevel).Build()
	require.NoError(t, err)
	require.Equal(t, prevLog, log)
	require.True(t, log.Is(logcommon.InfoLevel))

	log, err = log.Copy().WithLevel(logcommon.DebugLevel).Build()
	require.NoError(t, err)
	require.True(t, log.Is(logcommon.DebugLevel))
}

func TestLogAdapter_BuildFields(t *testing.T) {
	log, err := newAdapter(logcommon.InfoLevel)

	require.NoError(t, err)
	require.NotNil(t, log)
	require.True(t, log.Is(logcommon.InfoLevel))

	log, err = log.Copy().WithField("test0", "value0").Build()
	require.NoError(t, err)

	var buf bytes.Buffer
	log, err = log.Copy().WithOutput(&buf).Build()
	require.NoError(t, err)

	log, err = log.Copy().WithField("test1", "value1").Build()
	require.NoError(t, err)

	log.Error("test")

	s := buf.String()
	require.Contains(t, s, "value0")
	require.Contains(t, s, "value1")
	buf.Reset()

	log, err = log.Copy().WithoutInheritedFields().WithField("test2", "value2").Build()
	require.NoError(t, err)

	log.Error("test")
	s = buf.String()
	require.NotContains(t, s, "value0")
	require.NotContains(t, s, "value1")
	require.Contains(t, s, "value2")
}

func TestLogAdapter_BuildDynFields(t *testing.T) {
	log, err := newAdapter(logcommon.InfoLevel)

	require.NoError(t, err)
	require.NotNil(t, log)
	require.True(t, log.Is(logcommon.InfoLevel))

	log, err = log.Copy().
		WithDynamicField("test0", func() interface{} { return "value0" }).
		WithField("test00", "static0").
		Build()
	require.NoError(t, err)

	var buf bytes.Buffer
	log, err = log.Copy().WithOutput(&buf).Build()
	require.NoError(t, err)

	log, err = log.Copy().WithDynamicField("test1", func() interface{} { return "value1" }).Build()
	require.NoError(t, err)

	log.Error("test")

	s := buf.String()
	require.Contains(t, s, "static0")
	require.Contains(t, s, "value0")
	require.Contains(t, s, "value1")

	buf.Reset()
	log, err = log.Copy().WithoutInheritedDynFields().WithDynamicField("test2", func() interface{} { return "value2" }).Build()
	require.NoError(t, err)

	log.Error("test")
	s = buf.String()
	require.Contains(t, s, "static0")
	require.NotContains(t, s, "value0")
	require.NotContains(t, s, "value1")
	require.Contains(t, s, "value2")

	buf.Reset()
	log, err = log.Copy().WithoutInheritedFields().WithDynamicField("test3", func() interface{} { return "value3" }).Build()
	require.NoError(t, err)

	log.Error("test")
	s = buf.String()
	require.NotContains(t, s, "static0")
	require.NotContains(t, s, "value0")
	require.NotContains(t, s, "value1")
	require.NotContains(t, s, "value2")
	require.Contains(t, s, "value3")

	buf.Reset()
	log, err = log.Copy().WithoutInheritedFields().WithDynamicField("test3", func() interface{} { return "value-3" }).Build()
	require.NoError(t, err)

	log.Error("test")
	s = buf.String()
	require.NotContains(t, s, "static0")
	require.NotContains(t, s, "value0")
	require.NotContains(t, s, "value1")
	require.NotContains(t, s, "value2")
	require.NotContains(t, s, "value3")
	require.Contains(t, s, "value-3")
}

func TestLogAdapter_Fatal(t *testing.T) {
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

func TestLogAdapter_Panic(t *testing.T) {
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
