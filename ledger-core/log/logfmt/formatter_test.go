package logfmt

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/args"
)

type stringerStruct struct {
	s string
}

func (v stringerStruct) String() string {
	return "wrong" // must take LogString() first
}

func (v stringerStruct) LogString() string {
	return v.s
}

type stringerRefStruct struct {
	s string
}

func (p *stringerRefStruct) String() string {
	return p.s
}

type stubStruct struct {
}

const sampleStructAsString = `f0:  99:int,f1:999:int,f2:test_raw,f3:test2:string,f4:nil,f5:stringer_test:ptr,` +
	`f6:func_result:func,f7:stringerVal:struct,f8:stringerRef:ptr,f9:nil,f10:{}:struct,` +
	`f11:logfmt.createSampleStruct.func2:func,` +
	`msg:message title`

func createSampleStruct() interface{} {
	s := "test2"
	return struct {
		msg string
		f0  int `fmt:"%4d"`
		f1  int
		f2  string `raw:"%s"`
		f3  *string
		f4  *string
		f5  interface{}
		f6  func() string
		f7  stringerStruct
		f8  *stringerRefStruct
		f9  *stringerRefStruct
		f10 stubStruct // no special handling
		f11 func()
	}{
		"message title",
		99, 999, "test_raw", &s, nil,
		args.LazyFmt("stringer_test"),
		func() string { return "func_result" },
		stringerStruct{"stringerVal"},
		&stringerRefStruct{"stringerRef"},
		nil,
		stubStruct{},
		func() { /* something else */ },
	}
}

func TestTryLogObject_Many(t *testing.T) {
	f := GetDefaultLogMsgFormatter()

	require.Equal(t,
		"{message title 99} 888",
		f.testTryLogObject(struct {
			msg string
			f1  int
		}{
			"message title",
			99,
		}, 888))
}

func TestTryLogObject_Str(t *testing.T) {
	f := GetDefaultLogMsgFormatter()

	require.Equal(t, "text", f.testTryLogObject("text"))
	s := "text"
	require.Equal(t, "text", f.testTryLogObject(s))
	require.Equal(t, "text", f.testTryLogObject(&s))
	ps := &s
	require.Equal(t, "text", f.testTryLogObject(ps))
	ps = nil
	require.Equal(t, "<nil>", f.testTryLogObject(ps))

	require.Equal(t, "text", f.testTryLogObject(func() string { return "text" }))
}

func TestStruct_NamelessFields(t *testing.T) {
	f := GetDefaultLogMsgFormatter()

	require.Equal(t, "int:1:int,msg:some text", f.testTryLogObject(
		struct {
			string
			int
		}{
			"some text", 1,
		}))

	require.Equal(t, "int:1:int,Msg:another text:string,msg:some text", f.testTryLogObject(
		struct {
			string
			int
			Msg string
		}{
			"some text", 1, "another text",
		}))

	require.Equal(t, "int:1:int,string:some text:string,msg:another text", f.testTryLogObject(
		struct {
			int
			Msg string
			string
		}{
			1, "another text", "some text",
		}))

	require.Equal(t, "_txtTag1:tag text:string,string:another text:string,msg:Msg text", f.testTryLogObject(
		struct {
			Msg string
			_   string `txt:"tag text"`
			string
		}{
			"Msg text", "", "another text",
		}))
}

func TestTryLogObject_SingleUnnamed(t *testing.T) {
	f := GetDefaultLogMsgFormatter()

	require.Equal(t, sampleStructAsString, f.testTryLogObject(createSampleStruct()))
}

func TestTryLogObject_SingleNamed(t *testing.T) {
	f := GetDefaultLogMsgFormatter()

	type SomeType struct {
		i int
	}

	require.Equal(t,
		"{7}",
		f.testTryLogObject(SomeType{7}))
}

var _ LogObject = SomeLogObjectValue{}

type SomeLogObjectValue struct {
	IntVal int
	Msg    string
}

func (SomeLogObjectValue) GetLogObjectMarshaller() LogObjectMarshaller {
	return nil
}

var _ LogObject = &SomeLogObjectPtr{}

type SomeLogObjectPtr struct {
	IntVal int
	Msg    string
}

func (*SomeLogObjectPtr) GetLogObjectMarshaller() LogObjectMarshaller {
	return nil
}

var _ LogObject = SomeLogObjectWithTemplate{}
var _ LogObject = &SomeLogObjectWithTemplate{}

type SomeLogObjectWithTemplate struct {
	*MsgTemplate
	IntVal int
}

type SomeLogObjectWithTemplateAndMsg struct {
	*MsgTemplate `txt:"TemplateAndMsg"`
	IntVal       int
}

type SomeLogObjectWithTemplateAndMsg2 struct {
	*MsgTemplate
	IntVal int
	_      struct{} `txt:"TemplateAndMsg"`
}

func TestTryLogObject_SingleLogObject(t *testing.T) {
	f := GetDefaultLogMsgFormatter()

	require.Equal(t,
		"IntVal:7:int,msg:msgText",
		f.testTryLogObject(SomeLogObjectValue{7, "msgText"}))

	require.Equal(t,
		"IntVal:7:int,msg:msgText",
		f.testTryLogObject(&SomeLogObjectPtr{7, "msgText"}))

	require.Equal(t,
		"{7 msgText}",
		f.testTryLogObject(SomeLogObjectPtr{7, "msgText"})) // function has ptr receiver and is not taken

	require.Equal(t,
		"IntVal:7:int,msg:logfmt.SomeLogObjectWithTemplate",
		f.testTryLogObject(&SomeLogObjectWithTemplate{nil, 7}))

	require.Equal(t,
		"IntVal:7:int,msg:logfmt.SomeLogObjectWithTemplate",
		f.testTryLogObject(SomeLogObjectWithTemplate{nil, 7}))

	require.Equal(t,
		"IntVal:7:int,msg:TemplateAndMsg",
		f.testTryLogObject(SomeLogObjectWithTemplateAndMsg{nil, 7}))

	require.Equal(t,
		"IntVal:7:int,msg:TemplateAndMsg",
		f.testTryLogObject(SomeLogObjectWithTemplateAndMsg2{nil, 7, struct{}{}}))
}

type SomeLogObjectWithMsg struct {
	*MsgTemplate
	IntVal  int    `opt:""`
	Message string `txt:"fixedObjectMessage"`
}

func TestTryLogObject_ConstMsg(t *testing.T) {
	f := GetDefaultLogMsgFormatter()

	require.Equal(t,
		"Data:0:int,msg:inlineConstantText",
		f.testTryLogObject(struct {
			Data int
			Msg  string `txt:"inlineConstantText"`
		}{}))

	require.Equal(t,
		"IntVal:78:int,msg:fixedObjectMessage",
		f.testTryLogObject(SomeLogObjectWithMsg{IntVal: 78}))
}

func TestTryLogObject_Optionals(t *testing.T) {
	f := GetDefaultLogMsgFormatter()

	require.Equal(t,
		"msg:fixedObjectMessage",
		f.testTryLogObject(SomeLogObjectWithMsg{}))

	require.Equal(t,
		"msg:inlineConstantText",
		f.testTryLogObject(struct {
			Data int    `anotherTag:"" opt:""`
			Msg  string `txt:"inlineConstantText"`
		}{}))
}

type ForcedLogObjectValue struct {
	IntVal  int
	SomeTxt string
}

func TestTryLogObject_ForcedStruct(t *testing.T) {
	f := GetDefaultLogMsgFormatter()

	require.Equal(t,
		"{7 abc}", // regular fmt.Sprint
		f.testTryLogObject(ForcedLogObjectValue{7, "abc"}))

	require.Equal(t,
		"IntVal:7:int,SomeTxt:abc:string,msg:logfmt.ForcedLogObjectValue",
		f.testLogObject(ForcedLogObjectValue{7, "abc"}))
}

func (v MsgFormatConfig) testTryLogObject(a ...interface{}) string {
	if len(a) != 1 {
		return v.FmtLogObject(a...)
	}
	return v._logObject(v.FmtLogStructOrObject(a[0]))
}

func (v MsgFormatConfig) testLogObject(a interface{}) string {
	return v._logObject(v.FmtLogStruct(a))
}

func (v MsgFormatConfig) _logObject(m LogObjectMarshaller, s string) string {
	if m == nil {
		return s
	}
	o := output{}
	msg, _ := m.MarshalLogObject(&o, nil)
	o.buf.WriteString("msg:")
	o.buf.WriteString(msg)
	return o.buf.String()
}

var _ LogObjectWriter = &output{}

type output struct {
	buf strings.Builder
}

func (p *output) AddIntField(key string, v int64, fmt LogFieldFormat) {
	p.addField(key, v, fmt)
}

func (p *output) AddUintField(key string, v uint64, fmt LogFieldFormat) {
	p.addField(key, v, fmt)
}

func (p *output) AddBoolField(key string, v bool, fmt LogFieldFormat) {
	p.addField(key, v, fmt)
}

func (p *output) AddFloatField(key string, v float64, fmt LogFieldFormat) {
	p.addField(key, v, fmt)
}

func (p *output) AddComplexField(key string, v complex128, fmt LogFieldFormat) {
	p.addField(key, v, fmt)
}

func (p *output) AddStrField(key string, v string, fmt LogFieldFormat) {
	p.addField(key, v, fmt)
}

func (p *output) addField(k string, v interface{}, fmtStr LogFieldFormat) {
	switch {
	case fmtStr.HasFmt:
		p._addField(k, fmt.Sprintf(fmtStr.Fmt, v), fmtStr.Kind.String())
	default:
		p._addField(k, v, fmtStr.Kind.String())
	}
}

func (p *output) _addField(k string, v interface{}, tName string) {
	p.buf.WriteString(fmt.Sprintf("%s:%v:%s,", k, v, tName))
}

func (p *output) AddIntfField(k string, v interface{}, fmtStr LogFieldFormat) {
	switch {
	case v == nil:
		p.buf.WriteString(fmt.Sprintf("%s:nil,", k))
	case fmtStr.HasFmt:
		p.buf.WriteString(fmt.Sprintf("%s:%v:%T,", k, fmt.Sprintf(fmtStr.Fmt, v), v))
	default:
		p.buf.WriteString(fmt.Sprintf("%s:%v:%T,", k, v, v))
	}
}

func (p *output) AddRawJSONField(k string, v interface{}, fFmt LogFieldFormat) {
	p.buf.WriteString(fmt.Sprintf("%s:%s,", k, fmt.Sprintf(fFmt.Fmt, v)))
}

func (p *output) AddTimeField(k string, v time.Time, fFmt LogFieldFormat) {
	switch {
	case v.IsZero():
		p.buf.WriteString(fmt.Sprintf("%s:0:time,", k))
	case fFmt.HasFmt:
		p.buf.WriteString(fmt.Sprintf("%s:%s:time", k, v.Format(fFmt.Fmt)))
	default:
		p.buf.WriteString(fmt.Sprintf("%s:%s:time", k, v.String()))
	}
}

func (p *output) AddErrorField(msg string, stack throw.StackTrace, severity throw.Severity, hasPanic bool) {
	if msg != "" {
		p.buf.WriteString(fmt.Sprintf("%s:%s,", "errorMsg", msg))
	}
	if stack == nil {
		return
	}
	pn := ""
	if hasPanic {
		pn = "panic:"
	}
	st := stack.StackTraceAsText()
	p.buf.WriteString(fmt.Sprintf("%s:%s%s,", "errorStack", pn, st))
}
