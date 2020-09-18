// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package global

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"

	"github.com/stretchr/testify/require"
)

type logRecord struct {
	s string
}

func (x logRecord) GetLogObjectMarshaller() logfmt.LogObjectMarshaller {
	return &mar{x: &x}
}

type mar struct {
	x *logRecord
}

func (m mar) MarshalLogObject(lw logfmt.LogObjectWriter, lmc logfmt.LogObjectMetricCollector) (string, bool) {
	return m.x.s, false
}

type LogObject struct {
	*log.Msg
	s string
}

func reflectedRandom(t reflect.Kind) reflect.Value {
	switch t {
	// BEGIN OF GENERATED PART (struct_test_types_gen)
	case reflect.Bool:
		return reflect.ValueOf(true)
	case reflect.Int:
		return reflect.ValueOf(int(9))
	case reflect.Int8:
		return reflect.ValueOf(int8(9))
	case reflect.Int16:
		return reflect.ValueOf(int16(9))
	case reflect.Int32:
		return reflect.ValueOf(int32(9))
	case reflect.Int64:
		return reflect.ValueOf(int64(9))
	case reflect.Uint:
		return reflect.ValueOf(uint(9))
	case reflect.Uint8:
		return reflect.ValueOf(uint8(9))
	case reflect.Uint16:
		return reflect.ValueOf(uint16(9))
	case reflect.Uint32:
		return reflect.ValueOf(uint32(9))
	case reflect.Uint64:
		return reflect.ValueOf(uint64(9))
	case reflect.Uintptr:
		return reflect.ValueOf(uintptr(9))
	case reflect.Float32:
		return reflect.ValueOf(float32(9))
	case reflect.Float64:
		return reflect.ValueOf(float64(9))
	case reflect.Complex64:
		return reflect.ValueOf(complex64(9))
	case reflect.Complex128:
		return reflect.ValueOf(complex128(9))
	case reflect.String:
		return reflect.ValueOf(logstring)
	// END OF GENERATED PART
	default:
		return reflect.ValueOf(nil)
	}
}

var logstring = "msgstrval"

func TestLogFieldsMarshaler(t *testing.T) {
	for _, obj := range []interface{}{
		logRecord{s: logstring}, &logRecord{s: logstring},
		mar{x: &logRecord{s: logstring}}, &mar{x: &logRecord{s: logstring}},
		logstring, &logstring, func() string { return logstring },
	} {
		buf := bytes.Buffer{}
		lg, _ := Logger().Copy().WithOutput(&buf).Build()

		lg.WithField("testfield", 200.200).Warn(obj)
		fileLine, _ := logoutput.GetCallerFileNameWithLine(0, -1)

		c := make(map[string]interface{})
		err := json.Unmarshal(buf.Bytes(), &c)
		require.NoError(t, err, "unmarshal %s", buf.Bytes())

		require.Equal(t, "warn", c["level"], "right message")
		require.Equal(t, logstring, c["message"], "right message")
		require.Contains(t, c["caller"], fileLine, "right caller line")
		// TODO: PLAT-830
		// ltime, err := time.Parse(time.RFC3339Nano, c["time"].(string))
		// require.NoError(t, err, "parseable time")
		// ldur := time.Since(ltime)
		// assert.True(t, ldur >= 0, "worktime is not less than zero")
		// assert.True(t, ldur < time.Second, "worktime is less than a second")
		assert.Equal(t, 200.200, c["testfield"], "customfield")
		assert.NotNil(t, c["writeDuration"], "duration exists")
	}
}

func TestLogLevels(t *testing.T) {
	buf := bytes.Buffer{}
	lg, _ := Logger().Copy().WithOutput(&buf).Build()

	lg.Copy().WithLevel(log.FatalLevel).MustBuild().Warn(logstring)
	require.Nil(t, buf.Bytes(), "do not log warns at panic level")

	lg.Warn(logstring)
	require.NotNil(t, buf.Bytes(), "previous logger saves it's level")

	if false {
		_ = SetFilter(log.PanicLevel)
		lg.Warn(logstring)
		require.Nil(t, buf.String(), "do not log warns at global panic level")
	}
}

func TestLogOther(t *testing.T) {
	buf := bytes.Buffer{}
	lg, _ := Logger().Copy().WithOutput(&buf).Build()
	c := make(map[string]interface{})
	lg.Warn(nil)
	require.NoError(t, json.Unmarshal(buf.Bytes(), &c))
	require.Equal(t, "<nil>", c["message"], "nil")

	buf.Reset()
	lg.Warn(100.1)
	require.NoError(t, json.Unmarshal(buf.Bytes(), &c))
	require.Equal(t, "100.1", c["message"], "nil")

	buf.Reset()
	lg.Warn(LogObject{s: logstring})
	require.NoError(t, json.Unmarshal(buf.Bytes(), &c))
	require.Equal(t, fmt.Sprintf("%s", logstring), c["s"], "nil")

	buf.Reset()
	lg.Warn(&LogObject{s: logstring})
	require.NoError(t, json.Unmarshal(buf.Bytes(), &c))
	require.Equal(t, fmt.Sprintf("%s", logstring), c["s"], "nil")

	// ???
	buf.Reset()
	lg.Warn(struct{ s string }{logstring})
	require.NoError(t, json.Unmarshal(buf.Bytes(), &c))
	require.Equal(t, fmt.Sprintf("%s", logstring), c["s"], "nil")

}

// BEGIN OF GENERATED PART (type_formats_gen)
var types = map[string]string{
	"complex64":  "%f",
	"float64":    "%f",
	"int16":      "%d",
	"bool":       "%t",
	"uint32":     "%d",
	"complex128": "%f",
	"string":     "%s",
	"uint8":      "%d",
	"int32":      "%d",
	"uint":       "%d",
	"float32":    "%f",
	"int8":       "%d",
	"int":        "%d",
	"uint64":     "%d",
	"uint16":     "%d",
	"int64":      "%d",
	"uintptr":    "%d",
}

// END OF GENERATED PART

func TestLogValueGetters(t *testing.T) {
	for _, fmtOpt := range []struct {
		fmtType    logcommon.LogFormat
		pairSep    string
		complexFmt string
		quotedStr  bool
	}{
		{logcommon.JSONFormat, `":`, "[%.0f,%.0f]", true},
		{logcommon.TextFormat, `=`, "(%.0f+%.0fi)", false},
	} {
		t.Run(fmtOpt.fmtType.String(), func(t *testing.T) {
			_testLogValueGetters(t, fmtOpt.fmtType, fmtOpt.pairSep, fmtOpt.complexFmt, fmtOpt.quotedStr)
		})
	}
}

func _testLogValueGetters(t *testing.T, fmtType logcommon.LogFormat, pairSep, complexFmt string, quotedStr bool) {
	for ft := range types {
		for _, tag := range []string{"fmt+opt", "raw+opt", "fmt", "raw", "skip", "txt", "opt"} {
			buf := bytes.Buffer{}
			lg, _ := Logger().Copy().WithFormat(fmtType).WithOutput(&buf).Build()
			plr := struct {
				msg string

				// BEGIN OF GENERATED PART (struct_test_types_gen)
				F_uintptr_fmt_opt uintptr `fmt+opt:"<<%d>>"`
				F_uintptr_raw_opt uintptr `raw+opt:"<<%d>>"`
				F_uintptr_fmt     uintptr `fmt:"<<%d>>"`
				F_uintptr_raw     uintptr `raw:"<<%d>>"`
				F_uintptr_skip    uintptr `skip:"<<%d>>"`
				F_uintptr_txt     uintptr `txt:"<<%d>>"`
				F_uintptr_opt     uintptr `opt:"<<%d>>"`

				F_complex64_fmt_opt complex64 `fmt+opt:"<<%f>>"`
				F_complex64_raw_opt complex64 `raw+opt:"<<%f>>"`
				F_complex64_fmt     complex64 `fmt:"<<%f>>"`
				F_complex64_raw     complex64 `raw:"<<%f>>"`
				F_complex64_skip    complex64 `skip:"<<%f>>"`
				F_complex64_txt     complex64 `txt:"<<%f>>"`
				F_complex64_opt     complex64 `opt:"<<%f>>"`

				F_string_fmt_opt string `fmt+opt:"<<%s>>"`
				F_string_raw_opt string `raw+opt:"<<%s>>"`
				F_string_fmt     string `fmt:"<<%s>>"`
				F_string_raw     string `raw:"<<%s>>"`
				F_string_skip    string `skip:"<<%s>>"`
				F_string_txt     string `txt:"<<%s>>"`
				F_string_opt     string `opt:"<<%s>>"`

				F_bool_fmt_opt bool `fmt+opt:"<<%t>>"`
				F_bool_raw_opt bool `raw+opt:"<<%t>>"`
				F_bool_fmt     bool `fmt:"<<%t>>"`
				F_bool_raw     bool `raw:"<<%t>>"`
				F_bool_skip    bool `skip:"<<%t>>"`
				F_bool_txt     bool `txt:"<<%t>>"`
				F_bool_opt     bool `opt:"<<%t>>"`

				F_int32_fmt_opt int32 `fmt+opt:"<<%d>>"`
				F_int32_raw_opt int32 `raw+opt:"<<%d>>"`
				F_int32_fmt     int32 `fmt:"<<%d>>"`
				F_int32_raw     int32 `raw:"<<%d>>"`
				F_int32_skip    int32 `skip:"<<%d>>"`
				F_int32_txt     int32 `txt:"<<%d>>"`
				F_int32_opt     int32 `opt:"<<%d>>"`

				F_uint16_fmt_opt uint16 `fmt+opt:"<<%d>>"`
				F_uint16_raw_opt uint16 `raw+opt:"<<%d>>"`
				F_uint16_fmt     uint16 `fmt:"<<%d>>"`
				F_uint16_raw     uint16 `raw:"<<%d>>"`
				F_uint16_skip    uint16 `skip:"<<%d>>"`
				F_uint16_txt     uint16 `txt:"<<%d>>"`
				F_uint16_opt     uint16 `opt:"<<%d>>"`

				F_uint64_fmt_opt uint64 `fmt+opt:"<<%d>>"`
				F_uint64_raw_opt uint64 `raw+opt:"<<%d>>"`
				F_uint64_fmt     uint64 `fmt:"<<%d>>"`
				F_uint64_raw     uint64 `raw:"<<%d>>"`
				F_uint64_skip    uint64 `skip:"<<%d>>"`
				F_uint64_txt     uint64 `txt:"<<%d>>"`
				F_uint64_opt     uint64 `opt:"<<%d>>"`

				F_uint8_fmt_opt uint8 `fmt+opt:"<<%d>>"`
				F_uint8_raw_opt uint8 `raw+opt:"<<%d>>"`
				F_uint8_fmt     uint8 `fmt:"<<%d>>"`
				F_uint8_raw     uint8 `raw:"<<%d>>"`
				F_uint8_skip    uint8 `skip:"<<%d>>"`
				F_uint8_txt     uint8 `txt:"<<%d>>"`
				F_uint8_opt     uint8 `opt:"<<%d>>"`

				F_int_fmt_opt int `fmt+opt:"<<%d>>"`
				F_int_raw_opt int `raw+opt:"<<%d>>"`
				F_int_fmt     int `fmt:"<<%d>>"`
				F_int_raw     int `raw:"<<%d>>"`
				F_int_skip    int `skip:"<<%d>>"`
				F_int_txt     int `txt:"<<%d>>"`
				F_int_opt     int `opt:"<<%d>>"`

				F_int64_fmt_opt int64 `fmt+opt:"<<%d>>"`
				F_int64_raw_opt int64 `raw+opt:"<<%d>>"`
				F_int64_fmt     int64 `fmt:"<<%d>>"`
				F_int64_raw     int64 `raw:"<<%d>>"`
				F_int64_skip    int64 `skip:"<<%d>>"`
				F_int64_txt     int64 `txt:"<<%d>>"`
				F_int64_opt     int64 `opt:"<<%d>>"`

				F_int8_fmt_opt int8 `fmt+opt:"<<%d>>"`
				F_int8_raw_opt int8 `raw+opt:"<<%d>>"`
				F_int8_fmt     int8 `fmt:"<<%d>>"`
				F_int8_raw     int8 `raw:"<<%d>>"`
				F_int8_skip    int8 `skip:"<<%d>>"`
				F_int8_txt     int8 `txt:"<<%d>>"`
				F_int8_opt     int8 `opt:"<<%d>>"`

				F_complex128_fmt_opt complex128 `fmt+opt:"<<%f>>"`
				F_complex128_raw_opt complex128 `raw+opt:"<<%f>>"`
				F_complex128_fmt     complex128 `fmt:"<<%f>>"`
				F_complex128_raw     complex128 `raw:"<<%f>>"`
				F_complex128_skip    complex128 `skip:"<<%f>>"`
				F_complex128_txt     complex128 `txt:"<<%f>>"`
				F_complex128_opt     complex128 `opt:"<<%f>>"`

				F_float64_fmt_opt float64 `fmt+opt:"<<%f>>"`
				F_float64_raw_opt float64 `raw+opt:"<<%f>>"`
				F_float64_fmt     float64 `fmt:"<<%f>>"`
				F_float64_raw     float64 `raw:"<<%f>>"`
				F_float64_skip    float64 `skip:"<<%f>>"`
				F_float64_txt     float64 `txt:"<<%f>>"`
				F_float64_opt     float64 `opt:"<<%f>>"`

				F_float32_fmt_opt float32 `fmt+opt:"<<%f>>"`
				F_float32_raw_opt float32 `raw+opt:"<<%f>>"`
				F_float32_fmt     float32 `fmt:"<<%f>>"`
				F_float32_raw     float32 `raw:"<<%f>>"`
				F_float32_skip    float32 `skip:"<<%f>>"`
				F_float32_txt     float32 `txt:"<<%f>>"`
				F_float32_opt     float32 `opt:"<<%f>>"`

				F_int16_fmt_opt int16 `fmt+opt:"<<%d>>"`
				F_int16_raw_opt int16 `raw+opt:"<<%d>>"`
				F_int16_fmt     int16 `fmt:"<<%d>>"`
				F_int16_raw     int16 `raw:"<<%d>>"`
				F_int16_skip    int16 `skip:"<<%d>>"`
				F_int16_txt     int16 `txt:"<<%d>>"`
				F_int16_opt     int16 `opt:"<<%d>>"`

				F_uint32_fmt_opt uint32 `fmt+opt:"<<%d>>"`
				F_uint32_raw_opt uint32 `raw+opt:"<<%d>>"`
				F_uint32_fmt     uint32 `fmt:"<<%d>>"`
				F_uint32_raw     uint32 `raw:"<<%d>>"`
				F_uint32_skip    uint32 `skip:"<<%d>>"`
				F_uint32_txt     uint32 `txt:"<<%d>>"`
				F_uint32_opt     uint32 `opt:"<<%d>>"`

				F_uint_fmt_opt uint `fmt+opt:"<<%d>>"`
				F_uint_raw_opt uint `raw+opt:"<<%d>>"`
				F_uint_fmt     uint `fmt:"<<%d>>"`
				F_uint_raw     uint `raw:"<<%d>>"`
				F_uint_skip    uint `skip:"<<%d>>"`
				F_uint_txt     uint `txt:"<<%d>>"`
				F_uint_opt     uint `opt:"<<%d>>"`
				// END OF GENERATED PART
			}{}

			fname := fmt.Sprintf("F_%s_%s", ft, strings.ReplaceAll(tag, "+", "_"))
			v := reflect.ValueOf(&plr).Elem()
			f := v.FieldByName(fname)
			saved := reflectedRandom(f.Type().Kind())
			f.Set(saved)
			lg.Warn(plr)

			format := "<<" + types[ft] + ">>"
			//if ft == "string" {
			mustHave := fmt.Sprintf(format, saved.Interface())

			switch {
			case tag == "txt":
				mustHave = format
			case tag == "opt":
				switch ft {
				case "string":
					mustHave = saved.String()
					if quotedStr {
						mustHave = fmt.Sprintf(`"%s"`, mustHave)
					}
				case "complex64", "complex128":
					c := saved.Complex()
					mustHave = fmt.Sprintf(complexFmt, real(c), imag(c))
				default:
					mustHave = fmt.Sprint(saved.Interface())
				}
			}

			if quotedStr && !strings.Contains(tag, "raw") && tag != "opt" {
				mustHave = `"` + mustHave
			}

			s := buf.String()

			fname += pairSep
			if tag == "skip" {
				assert.NotContains(t, s, fname)
			} else {
				assert.Contains(t, s, fname+mustHave)
			}
		}
	}
}

func TestLogAwkwardValueGetters(t *testing.T) {
	buf := bytes.Buffer{}
	lg, _ := Logger().Copy().WithOutput(&buf).Build()
	plr := struct {
		f    func() string
		notf func() (string, string)
	}{}
	plr.f = func() string {
		return logstring
	}
	plr.notf = func() (s string, s2 string) {
		return "", ""
	}
	lg.Warn(plr)
	c := make(map[string]interface{})
	require.NoError(t, json.Unmarshal(buf.Bytes(), &c))
	require.Equal(t, logstring, c["f"])
	require.Equal(t, "global.TestLogAwkwardValueGetters.func2", c["notf"])

	plr2 := struct {
		msg *string
		inf interface{}
	}{}
	plr2.msg = &logstring
	plr2.inf = logstring
	buf.Reset()
	lg.Warn(plr2)
	c = make(map[string]interface{})
	require.NoError(t, json.Unmarshal(buf.Bytes(), &c))
	require.Equal(t, logstring, c["message"])
	require.Equal(t, logstring, c["inf"])

}
