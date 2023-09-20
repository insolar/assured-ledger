package bilog

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	logm "github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/stretchr/testify/require"
)

func TestTextFormat(t *testing.T) {
	// TODO PLAT-44 parsing for text
	suite.Run(t, &SuiteTextualLog{logFormat: logcommon.TextFormat})
	suite.Run(t, &SuiteTextualLog{logFormat: logcommon.TextFormat, recycleBuf: true})
}

func TestJsonFormat(t *testing.T) {
	parseFn := func(t *testing.T, b []byte) map[string]interface{} {
		fields := make(map[string]interface{})
		err := json.Unmarshal(b, &fields)
		require.NoError(t, err, "unmarshal %s", b)
		return fields
	}
	suite.Run(t, &SuiteTextualLog{logFormat: logcommon.JSONFormat, parseFn: parseFn})
	suite.Run(t, &SuiteTextualLog{logFormat: logcommon.JSONFormat, parseFn: parseFn, recycleBuf: true})
}

func TestPbufFormat(t *testing.T) {
	// TODO PLAT-44 parsing for pbuf
	suite.Run(t, &SuiteTextualLog{logFormat: logcommon.PbufFormat})
	suite.Run(t, &SuiteTextualLog{logFormat: logcommon.PbufFormat, recycleBuf: true})
}

type SuiteTextualLog struct {
	suite.Suite
	logFormat  logcommon.LogFormat
	recycleBuf bool
	parseFn    func(*testing.T, []byte) map[string]interface{}
}

func (st SuiteTextualLog) newAdapter(level logcommon.Level) logm.Logger {
	zc := logcommon.Config{}

	var err error
	zc.BareOutput, err = logoutput.OpenLogBareOutput(logoutput.StdErrOutput, "", "")

	t := st.T()
	require.NoError(t, err)
	require.NotNil(t, zc.BareOutput.Writer)

	zc.Output = logcommon.OutputConfig{
		Format: st.logFormat,
	}
	zc.MsgFormat = logfmt.GetDefaultLogMsgFormatter()

	zb := logm.NewBuilder(NewFactory(nil, st.recycleBuf), zc, level).WithCaller(logcommon.CallerField)

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

	buf := bytes.Buffer{}
	lg, _ := st.newAdapter(logcommon.InfoLevel).Copy().
		WithOutput(&buf).
		WithCaller(logcommon.CallerFieldWithFuncName).
		WithMetrics(logcommon.LogMetricsWriteDelayField | logcommon.LogMetricsTimestamp).
		Build()

	const logstring = "msgstrval"
	lg.WithField("testfield", 200.200).Warnm(logRecord{logstring})
	fileLine, _ := logoutput.GetCallerFileNameWithLine(0, -1)

	if st.parseFn == nil {
		t.Skip("missing parser for", st.logFormat.String())
		return
	}

	c := st.parseFn(t, buf.Bytes())

	require.Equal(t, "warn", c["level"], "right message")
	require.Equal(t, logstring, c["s"], "right message")
	require.Equal(t, "bilog.logRecord", c["message"], "right message")
	require.Contains(t, c["caller"], fileLine, "right caller line")
	ltime, err := time.Parse(time.RFC3339Nano, c["time"].(string))
	require.NoError(t, err, "parseable time")
	ldur := time.Now().Sub(ltime)
	require.True(t, ldur >= 0, "worktime is not less than zero")
	require.True(t, ldur < time.Minute, "worktime should be less than a minute")
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
	require.Contains(t, s, "caller")
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
	require.Contains(t, s, "caller")
	buf.Reset()

	log = log.Copy().WithoutInheritedFields().WithField("test2", "value2").MustBuild()

	log.Error("test")
	s = buf.String()
	require.NotContains(t, s, "value0")
	require.NotContains(t, s, "value1")
	require.Contains(t, s, "value2")
	require.Contains(t, s, "caller")
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
	require.Contains(t, s, "caller")

	buf.Reset()
	log = log.Copy().WithoutInheritedDynFields().WithDynamicField("test2", func() interface{} { return "value2" }).MustBuild()

	log.Error("test")
	s = buf.String()
	require.Contains(t, s, "static0")
	require.NotContains(t, s, "value0")
	require.NotContains(t, s, "value1")
	require.Contains(t, s, "value2")
	require.Contains(t, s, "caller")

	buf.Reset()
	log = log.Copy().WithoutInheritedFields().WithDynamicField("test3", func() interface{} { return "value3" }).MustBuild()

	log.Error("test")
	s = buf.String()
	require.NotContains(t, s, "static0")
	require.NotContains(t, s, "value0")
	require.NotContains(t, s, "value1")
	require.NotContains(t, s, "value2")
	require.Contains(t, s, "value3")
	require.Contains(t, s, "caller")

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
	require.Contains(t, s, "caller")
}

func (st SuiteTextualLog) prepareSpecial(zc logcommon.Config, flushFn func()) (*bytes.Buffer, logm.Logger) {
	t := st.T()

	var buf bytes.Buffer
	zc.BareOutput = logcommon.BareOutput{
		Writer: &buf,
		FlushFn: func() error {
			if flushFn != nil {
				flushFn()
			}
			return nil
		},
	}
	zc.Output = logcommon.OutputConfig{Format: st.logFormat}
	zc.MsgFormat = logfmt.GetDefaultLogMsgFormatter()
	zc.Instruments.SkipFrameCountBaseline = 0

	zb := logm.NewBuilder(binLogFactory{}, zc, logcommon.InfoLevel)
	l, err := zb.Build()

	require.NoError(t, err)
	require.NotNil(t, l)

	return &buf, l
}

func (st SuiteTextualLog) TestPanic() {
	t := st.T()
	wg := sync.WaitGroup{}
	wg.Add(1)
	buf, log := st.prepareSpecial(logcommon.Config{}, wg.Done)

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

func (st SuiteTextualLog) TestFatal() {
	t := st.T()
	wg := sync.WaitGroup{}
	wg.Add(1)
	buf, log := st.prepareSpecial(logcommon.Config{}, func() {
		wg.Done()
		select {} // hand up to avoid os.Exit
	})

	log.Error("errorMsgText")
	go log.Fatal("fatalMsgText") // it will hang on flush
	wg.Wait()

	s := buf.String()
	require.Contains(t, s, "errorMsgText")
	require.Contains(t, s, "fatalMsgText")
}

func (st SuiteTextualLog) TestFatalSeverity() {
	t := st.T()
	wg := sync.WaitGroup{}
	wg.Add(1)
	buf, log := st.prepareSpecial(logcommon.Config{}, func() {
		wg.Done()
		select {} // hand up to avoid os.Exit
	})

	log.Error("errorMsgText")
	go log.Errorm(throw.Fatal("fatalMsgText")) // it will hang on flush
	wg.Wait()

	s := buf.String()
	require.Contains(t, s, "errorMsgText")
	require.Contains(t, s, "fatalMsgText")
}

func (st SuiteTextualLog) TestSeverityOverride() {
	t := st.T()
	buf, log := st.prepareSpecial(logcommon.Config{}, func() {
		t.FailNow()
	})

	log.Error("errorMsgText")
	log.Errorm(throw.WithSeverity(throw.Fatal("someMsgText"), throw.FraudSeverity))

	s := buf.String()
	require.Contains(t, s, "errorMsgText")
	require.Contains(t, s, "someMsgText")
}

func (st SuiteTextualLog) TestInternalError() {
	t := st.T()
	var capturedErr error

	buf, log := st.prepareSpecial(
		logcommon.Config{
			ErrorFn: func(err error) {
				require.Nil(t, capturedErr)
				require.NotNil(t, err)
				capturedErr = err
			},
		},
		nil)

	log.Errorm(struct {
		msg    string
		broken func() string
	}{
		msg: "errorMsgText",
		broken: func() string {
			panic("failFormat")
		},
	})

	s := buf.String()
	require.Equal(t, "", s)
	require.NotNil(t, capturedErr)
	require.Contains(t, capturedErr.Error(), "internal error (failFormat)")
}
