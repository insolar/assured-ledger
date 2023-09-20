package zlog

import (
	"bytes"
	"sync"
	"testing"

	logm "github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"

	"github.com/stretchr/testify/require"
)

func newZerologAdapter(level logcommon.Level) (logm.Logger, error) {
	zc := logcommon.Config{}

	var err error
	zc.BareOutput, err = logoutput.OpenLogBareOutput(logoutput.StdErrOutput, "", "")
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
	zc.Instruments.SkipFrameCountBaseline = ZerologSkipFrameCount

	zb := logm.NewBuilder(NewFactory(), zc, level).WithCaller(logcommon.CallerField)
	return zb.Build()
}

func TestZeroLogAdapter_CallerInfoWithFunc(t *testing.T) {
	log, err := newZerologAdapter(logcommon.InfoLevel)
	require.NoError(t, err)
	require.NotNil(t, log)

	var buf bytes.Buffer
	log, err = log.Copy().WithOutput(&buf).WithCaller(logcommon.CallerFieldWithFuncName).Build()
	require.NoError(t, err)

	log.Error("test")
	fileName, _ := logoutput.GetCallerFileNameWithLine(0, -1)

	s := buf.String()
	require.Contains(t, s, fileName)
	require.Contains(t, s, "TestZeroLogAdapter_CallerInfoWithFunc")
}

func TestZeroLogAdapter_CallerInfo(t *testing.T) {
	log, err := newZerologAdapter(logcommon.InfoLevel)

	require.NoError(t, err)
	require.NotNil(t, log)

	var buf bytes.Buffer
	log, err = log.Copy().WithOutput(&buf).WithCaller(logcommon.CallerField).Build()
	require.NoError(t, err)

	log.Error("test")
	fileName, _ := logoutput.GetCallerFileNameWithLine(0, -1)

	s := buf.String()
	require.Contains(t, s, fileName)
}

func TestZeroLogAdapter_InheritFields(t *testing.T) {
	log, err := newZerologAdapter(logcommon.InfoLevel)

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
	require.Contains(t, s, "caller")
}

func TestZeroLogAdapter_ChangeLevel(t *testing.T) {
	log, err := newZerologAdapter(logcommon.InfoLevel)

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

func TestZeroLogAdapter_BuildFields(t *testing.T) {
	log, err := newZerologAdapter(logcommon.InfoLevel)

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
	require.Contains(t, s, "caller")
	buf.Reset()

	log, err = log.Copy().WithoutInheritedFields().WithField("test2", "value2").Build()
	require.NoError(t, err)

	log.Error("test")
	s = buf.String()
	require.NotContains(t, s, "value0")
	require.NotContains(t, s, "value1")
	require.Contains(t, s, "caller")
	require.Contains(t, s, "value2")
}

func TestZeroLogAdapter_BuildDynFields(t *testing.T) {
	log, err := newZerologAdapter(logcommon.InfoLevel)

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
	require.Contains(t, s, "caller")

	buf.Reset()
	log, err = log.Copy().WithoutInheritedDynFields().WithDynamicField("test2", func() interface{} { return "value2" }).Build()
	require.NoError(t, err)

	log.Error("test")
	s = buf.String()
	require.Contains(t, s, "static0")
	require.NotContains(t, s, "value0")
	require.NotContains(t, s, "value1")
	require.Contains(t, s, "value2")
	require.Contains(t, s, "caller")

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
	require.Contains(t, s, "caller")

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
	require.Contains(t, s, "caller")
}

func TestZeroLogAdapter_Fatal(t *testing.T) {
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

	zb := logm.NewBuilder(zerologFactory{}, zc, logcommon.InfoLevel)
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

func TestZeroLogAdapter_Panic(t *testing.T) {
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

	zb := logm.NewBuilder(zerologFactory{}, zc, logcommon.InfoLevel)
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
