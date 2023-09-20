package inslogger

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger(cfg configuration.Log) (log.Logger, error) {
	return NewLog(cfg)
}

func TestLog_NewLog_Config(t *testing.T) {
	invalidtests := map[string]configuration.Log{
		"InvalidAdapter":   {Level: "Debug", Adapter: "invalid", Formatter: "text"},
		"InvalidLevel":     {Level: "Invalid", Adapter: "zerolog", Formatter: "text"},
		"InvalidFormatter": {Level: "Debug", Adapter: "zerolog", Formatter: "invalid"},
	}

	for name, test := range invalidtests {
		t.Run(name, func(t *testing.T) {
			logger, err := newTestLogger(test)
			assert.Error(t, err)
			assert.True(t, logger.IsZero())
		})
	}

	validtests := map[string]configuration.Log{
		"WithAdapter": {Level: "Debug", Adapter: "zerolog", Formatter: "text"},
	}
	for name, test := range validtests {
		t.Run(name, func(t *testing.T) {
			logger, err := newTestLogger(test)
			assert.NoError(t, err)
			assert.False(t, logger.IsZero())
		})
	}
}

func TestLog_AddFields(t *testing.T) {
	errtxt1 := "~CHECK_ERROR_OUTPUT_WITH_FIELDS~"
	errtxt2 := "~CHECK_ERROR_OUTPUT_WITHOUT_FIELDS~"

	var (
		fieldname  = "TraceID"
		fieldvalue = "Trace100500"
	)
	tt := []struct {
		name    string
		fieldfn func(la log.Logger) log.Logger
	}{
		{
			name: "WithFields",
			fieldfn: func(la log.Logger) log.Logger {
				fields := map[string]interface{}{fieldname: fieldvalue}
				return la.WithFields(fields)
			},
		},
		{
			name: "WithField",
			fieldfn: func(la log.Logger) log.Logger {
				return la.WithField(fieldname, fieldvalue)
			},
		},
	}

	for _, tItem := range tt {
		t.Run(tItem.name, func(t *testing.T) {
			la, err := newTestLogger(configuration.NewLog())
			assert.NoError(t, err)

			var b bytes.Buffer
			logger, err := la.Copy().WithOutput(&b).Build()
			assert.NoError(t, err)

			tItem.fieldfn(logger).Error(errtxt1)
			logger.Error(errtxt2)

			var recitems []string
			for {
				line, err := b.ReadBytes('\n')
				if err != nil && err != io.EOF {
					require.NoError(t, err)
				}

				recitems = append(recitems, string(line))
				if err == io.EOF {
					break
				}
			}
			assert.Contains(t, recitems[0], errtxt1)
			assert.Contains(t, recitems[1], errtxt2)
			assert.Contains(t, recitems[0], fieldvalue)
			assert.NotContains(t, recitems[1], fieldvalue)
		})
	}
}

var adapters = []string{"zerolog"}

func TestLog_Timestamp(t *testing.T) {
	for _, adapter := range adapters {
		adapter := adapter
		t.Run(adapter, func(t *testing.T) {
			logger, err := newTestLogger(configuration.Log{Level: "info", Adapter: adapter, Formatter: "json"})
			require.NoError(t, err)
			require.NotNil(t, logger)

			var buf bytes.Buffer
			logger, err = logger.Copy().WithOutput(&buf).Build()
			require.NoError(t, err)

			logger.Error("test")

			s := buf.String()
			assert.Regexp(t, regexp.MustCompile("[0-9][0-9]:[0-9][0-9]:[0-9][0-9]"), s)
		})
	}
}

func TestLog_WriteDuration(t *testing.T) {
	for _, adapter := range adapters {
		adapter := adapter
		t.Run(adapter, func(t *testing.T) {
			logger, err := newTestLogger(configuration.Log{Level: "info", Adapter: adapter, Formatter: "json"})
			require.NoError(t, err)
			require.NotNil(t, logger)

			var buf bytes.Buffer
			logger, err = logger.Copy().WithOutput(&buf).WithMetrics(logcommon.LogMetricsResetMode).Build()
			require.NoError(t, err)

			logger2, err := logger.Copy().WithMetrics(logcommon.LogMetricsWriteDelayField).Build()
			require.NoError(t, err)

			logger3, err := logger.Copy().WithMetrics(logcommon.LogMetricsResetMode).Build()
			require.NoError(t, err)

			logger.Error("test")
			assert.NotContains(t, buf.String(), `,"writeDuration":"`)
			logger3.Error("test")
			assert.NotContains(t, buf.String(), `,"writeDuration":"`)
			logger2.Error("test2")
			s := buf.String()
			assert.Contains(t, s, `,"writeDuration":"`)
		})
	}
}

func TestLog_DynField(t *testing.T) {
	for _, adapter := range adapters {
		adapter := adapter
		t.Run(adapter, func(t *testing.T) {
			logger, err := newTestLogger(configuration.Log{Level: "info", Adapter: adapter, Formatter: "json"})
			require.NoError(t, err)
			require.NotNil(t, logger)

			const skipConstant = "---skip---"
			dynFieldValue := skipConstant
			var buf bytes.Buffer
			logger, err = logger.Copy().WithOutput(&buf).WithDynamicField("dynField1", func() interface{} {
				if dynFieldValue == skipConstant {
					return nil
				}
				return dynFieldValue
			}).Build()
			require.NoError(t, err)

			logger.Error("test1")
			assert.NotContains(t, buf.String(), `"dynField1":`)
			dynFieldValue = ""
			logger.Error("test2")
			assert.Contains(t, buf.String(), `"dynField1":""`)
			dynFieldValue = "some text"
			logger.Error("test3")
			assert.Contains(t, buf.String(), `"dynField1":"some text"`)
		})
	}
}

func TestMain(m *testing.M) {
	l, err := global.Logger().Copy().WithFormat(logcommon.JSONFormat).WithLevel(log.DebugLevel).Build()
	if err != nil {
		panic(err)
	}
	global.SetLogger(l)
	_ = global.SetFilter(log.DebugLevel)
	exitCode := m.Run()
	os.Exit(exitCode)
}
